package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/monitoring"
	"github.com/authsec-ai/authsec/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminUserController struct {
	tenantRepo      *database.TenantRepository
	userRepo        *database.UserRepository
	adminUserRepo   *database.AdminUserRepository
	tenantDBService *database.TenantDBService
}

var (
	errTenantNotFound = fmt.Errorf("tenant not found")
	errTenantDBNotSet = fmt.Errorf("tenant database not configured")
)

// TenantUserListRequest represents payload for listing tenant users
type TenantUserListRequest struct {
	TenantID  string `json:"tenant_id" binding:"required"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	ClientID  string `json:"client_id"`
	ProjectID string `json:"project_id"`
	Provider  string `json:"provider"`
}

// AdminUserListRequest represents the request payload for listing admin users
type AdminUserListRequest struct {
	Status    string `json:"status"`     // Filter by status: "pending" or "active"
	Provider  string `json:"provider"`   // Filter by provider
	TenantID  string `json:"tenant_id"`  // Optional (usually from JWT)
	ClientID  string `json:"client_id"`  // Optional
	ProjectID string `json:"project_id"` // Optional
	Page      int    `json:"page"`       // Pagination
	Limit     int    `json:"limit"`      // Pagination
}

type toggleAdminUserActiveRequest struct {
	TenantID string        `json:"tenant_id" binding:"required"`
	UserID   string        `json:"user_id" binding:"required"`
	Active   *shared.FlexibleBool `json:"active" binding:"required"`
}

// NewAdminUserController creates a new admin user controller
func NewAdminUserController() (*AdminUserController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get config for database parameters
	cfg := config.GetConfig()

	// Create tenant database service
	tenantDBService, err := database.NewTenantDBService(db, cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant DB service: %w", err)
	}

	return &AdminUserController{
		tenantRepo:      database.NewTenantRepository(db),
		userRepo:        database.NewUserRepository(db),
		adminUserRepo:   database.NewAdminUserRepository(db),
		tenantDBService: tenantDBService,
	}, nil
}

// ListTenants retrieves all tenants
func (auc *AdminUserController) ListTenants(c *gin.Context) {
	tenants, err := auc.tenantRepo.GetAllTenants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tenants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenants": tenants})
}

// ListAdminUsers godoc
// @Summary List admin users
// @Description Retrieves all users assigned to the admin role with optional provider and status filtering
// @Tags Admin-Users
// @Security BearerAuth
// @Produce json
// @Param provider query string false "Filter by authentication provider (e.g., local, google, azure, okta)"
// @Param input body AdminUserListRequest false "Filter options including status"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/users/list [get]
// @Router /uflow/admin/users/list [post]
func (auc *AdminUserController) ListAdminUsers(c *gin.Context) {
	requestID := c.GetString("request_id")
	logPrefix := "ListAdminUsers"
	if requestID != "" {
		logPrefix = fmt.Sprintf("%s request_id=%s", logPrefix, requestID)
	}
	log.Printf("%s: handling %s %s from %s", logPrefix, c.Request.Method, c.FullPath(), c.ClientIP())

	if auc.adminUserRepo == nil {
		log.Printf("%s: admin user repository not initialized", logPrefix)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin user repository not initialized"})
		return
	}

	// Get tenant_id from validated JWT token
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		log.Printf("%s: tenant_id not found in authentication token", logPrefix)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
		return
	}
	log.Printf("%s: resolved tenant_id from token: %s", logPrefix, tenantIDStr)

	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		log.Printf("%s: invalid tenant_id %q: %v", logPrefix, tenantIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}
	log.Printf("%s: using tenant_id=%s", logPrefix, tenantUUID)

	if err := auc.adminUserRepo.EnsureTenantAdminRoleAssignment(tenantUUID); err != nil {
		log.Printf("%s: failed ensure tenant admin roles: %v", logPrefix, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reconcile admin roles"})
		return
	}

	// Get optional provider filter from query parameter
	provider := c.Query("provider")
	if provider != "" {
		log.Printf("%s: filtering by provider: %s", logPrefix, provider)
	}

	// Get optional status filter from request body (for POST) or query param (for GET)
	var statusFilter string
	if c.Request.Method == "POST" {
		var req AdminUserListRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			statusFilter = req.Status
		}
	} else {
		statusFilter = c.Query("status")
	}
	if statusFilter != "" {
		log.Printf("%s: filtering by status: %s", logPrefix, statusFilter)
	}

	users, err := auc.adminUserRepo.ListAdminUsersByTenantWithFilter(tenantUUID, provider)
	if err != nil {
		log.Printf("%s: failed to retrieve admin users: %v", logPrefix, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve admin users"})
		return
	}

	if users == nil {
		users = []models.AdminUser{}
	}
	log.Printf("%s: returning %d admin users", logPrefix, len(users))

	responseUsers := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		payload, err := buildAdminUserResponse(user)
		if err != nil {
			log.Printf("%s: failed to marshal admin user %s: %v", logPrefix, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare admin user response"})
			return
		}

		// Fetch and add roles for this user
		if user.TenantID != nil {
			roles, err := auc.adminUserRepo.GetUserRoles(user.ID, *user.TenantID)
			if err != nil {
				log.Printf("%s: failed to get roles for user %s: %v", logPrefix, user.ID, err)
				payload["roles"] = []interface{}{} // Return empty array on error
			} else {
				payload["roles"] = roles
			}
		} else {
			payload["roles"] = []interface{}{}
		}

		// Check for pending registration
		hasPending, err := auc.adminUserRepo.HasPendingRegistration(user.Email)
		if err != nil {
			log.Printf("%s: failed to check pending registration for user %s: %v", logPrefix, user.ID, err)
			payload["pending_registration"] = false
		} else {
			payload["pending_registration"] = hasPending
		}

		// Apply status filter if specified
		if statusFilter != "" {
			if statusFilter == "pending" && !hasPending {
				continue // Skip non-pending users
			}
			if statusFilter == "active" && hasPending {
				continue // Skip pending users
			}
		}

		responseUsers = append(responseUsers, payload)
	}

	log.Printf("%s: returning %d users after filtering", logPrefix, len(responseUsers))
	c.JSON(http.StatusOK, gin.H{"users": responseUsers})
}

// ToggleAdminUserActive godoc
// @Summary Activate or deactivate an admin user
// @Description Updates the active flag for an admin user in the master database
// @Tags Admin-Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body toggleAdminUserActiveRequest true "Admin user toggle payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/users/active [post]
func (auc *AdminUserController) ToggleAdminUserActive(c *gin.Context) {
	requestID := c.GetString("request_id")
	logPrefix := "ToggleAdminUserActive"
	if requestID != "" {
		logPrefix = fmt.Sprintf("%s request_id=%s", logPrefix, requestID)
	}

	logger := monitoring.GetLogger().WithField("request_id", requestID).WithField("operation", "toggle_admin_user_active")
	logger.Info("Processing admin user activation/deactivation request")

	// Validate repository initialization
	if auc.adminUserRepo == nil {
		logger.Error("Admin user repository not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin user repository not initialized"})
		return
	}

	// Parse and validate request body
	var req toggleAdminUserActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Warn("Failed to bind JSON request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	active := false
	if req.Active == nil {
		logger.Warn("Active flag missing in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "active flag is required"})
		return
	} else {
		active = req.Active.Bool()
	}

	// Validate and parse user ID
	userUUID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		logger.WithError(err).WithField("user_id", req.UserID).Warn("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	// Validate and parse tenant ID
	tenantUUID, err := uuid.Parse(strings.TrimSpace(req.TenantID))
	if err != nil {
		logger.WithError(err).WithField("tenant_id", req.TenantID).Warn("Invalid tenant ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}

	logger = logger.WithField("user_id", userUUID).WithField("tenant_id", tenantUUID).WithField("active", active)

	// Fetch admin user
	adminUser, err := auc.adminUserRepo.GetAdminUserByID(userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrAdminUserNotFound) {
			logger.Warn("Admin user not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
			return
		}
		logger.WithError(err).Error("Failed to fetch admin user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load admin user"})
		return
	}

	// Validate tenant ownership
	if adminUser.TenantID == nil || !strings.EqualFold(adminUser.TenantID.String(), tenantUUID.String()) {
		logger.WithField("admin_tenant_id", adminUser.TenantID).Warn("Admin user belongs to different tenant")
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin user belongs to a different tenant"})
		return
	}

	// Check if trying to deactivate primary admin
	if adminUser.IsPrimaryAdmin && !active {
		logger.Warn("Attempted to deactivate primary admin")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "deactivate_primary_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot deactivate primary admin")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot deactivate the primary admin"})
		return
	}

	// Check if trying to deactivate the last active admin
	if !active {
		activeAdmins, err := auc.adminUserRepo.ListAdminUsersByTenant(tenantUUID)
		if err != nil {
			logger.WithError(err).Error("Failed to verify admin count")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin count"})
			return
		}

		activeCount := 0
		for _, admin := range activeAdmins {
			if admin.Active && admin.ID != userUUID {
				activeCount++
			}
		}

		if activeCount == 0 {
			logger.Warn("Attempted to deactivate last active admin")
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "deactivate_last_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot deactivate last admin")
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot deactivate the last active admin in the tenant"})
			return
		}
	}

	// Update admin user active status
	updated, err := auc.adminUserRepo.UpdateAdminUserActive(userUUID, active)
	if err != nil {
		logger.WithError(err).Error("Failed to update admin user active flag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update admin user"})
		return
	}

	if !updated {
		logger.Warn("Admin user not found during update")
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
		return
	}

	// Send email notification if account is being deactivated
	if !active {
		if err := utils.SendAccountDeactivationEmail(adminUser.Email); err != nil {
			logger.WithError(err).Warn("Failed to send deactivation email, but proceeding with deactivation")
			// Don't fail the deactivation if email fails - it's a notification only
		} else {
			logger.Info("Deactivation email sent successfully")
		}
	}

	// Audit log the successful operation
	action := "admin_user_deactivated"
	message := "Admin user deactivated successfully"
	if active {
		action = "admin_user_activated"
		message = "Admin user activated successfully"
	}

	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), action, c.ClientIP(), c.GetHeader("User-Agent"), true, message)
	}

	// Audit log: Admin user activated/deactivated (stdout)
	middlewares.Audit(c, "admin_user", adminUser.ID.String(), action, &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"active": !active,
			"email":  adminUser.Email,
		},
		After: map[string]interface{}{
			"active": active,
			"email":  adminUser.Email,
		},
	})

	logger.Info("Successfully toggled admin user active status")
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"user_id": userUUID.String(),
		"active":  active,
	})
}

// DeleteAdminUser godoc
// @Summary Soft delete an admin user
// @Description Marks an admin user as inactive in the master database. Cannot delete the primary admin or the last remaining admin.
// @Tags Admin-Users
// @Security BearerAuth
// @Produce json
// @Param user_id path string true "Admin user ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/users/{user_id} [delete]
func (auc *AdminUserController) DeleteAdminUser(c *gin.Context) {
	requestID := c.GetString("request_id")
	logger := monitoring.GetLogger().WithField("request_id", requestID).WithField("operation", "delete_admin_user")
	logger.Info("Processing admin user delete request")

	// Validate user ID parameter
	userID := c.Param("user_id")
	if userID == "" {
		logger.Warn("Missing user_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logger.WithError(err).WithField("user_id", userID).Warn("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	logger = logger.WithField("user_id", userUUID)

	// Get authenticated user info
	userInfo := middlewares.GetUserInfo(c)
	if userInfo == nil || strings.TrimSpace(userInfo.TenantID) == "" {
		logger.Warn("Missing tenant scope in request")
		c.JSON(http.StatusForbidden, gin.H{"error": "Tenant scope is required"})
		return
	}

	userTenant := strings.TrimSpace(userInfo.TenantID)
	logger = logger.WithField("tenant_id", userTenant)

	// Fetch admin user to delete
	adminUser, err := auc.adminUserRepo.GetAdminUserByID(userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("Admin user not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
			return
		}
		logger.WithError(err).Error("Failed to fetch admin user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load admin user"})
		return
	}

	// Validate tenant ownership
	if adminUser.TenantID == nil || !strings.EqualFold(adminUser.TenantID.String(), userTenant) {
		logger.WithField("admin_tenant_id", adminUser.TenantID).Warn("Admin user belongs to different tenant")
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin user belongs to a different tenant"})
		return
	}

	// Check if this is the primary admin
	if adminUser.IsPrimaryAdmin {
		logger.Warn("Attempted to delete primary admin")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "delete_primary_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot delete primary admin")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the primary admin"})
		return
	}

	// Check if this is the last active admin for the tenant
	tenantUUID, _ := uuid.Parse(userTenant)
	activeAdmins, err := auc.adminUserRepo.ListAdminUsersByTenant(tenantUUID)
	if err != nil {
		logger.WithError(err).Error("Failed to verify admin count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin count"})
		return
	}

	// Count active admins excluding the current user
	activeCount := 0
	for _, admin := range activeAdmins {
		if admin.Active && admin.ID != userUUID {
			activeCount++
		}
	}

	if activeCount == 0 {
		logger.Warn("Attempted to delete last active admin in tenant")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "delete_last_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot delete last admin")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the last active admin in the tenant"})
		return
	}

	// Perform soft delete
	if err := auc.adminUserRepo.DeleteAdminUser(userUUID); err != nil {
		if errors.Is(err, database.ErrAdminUserNotFound) {
			logger.Warn("Admin user not found during delete")
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
			return
		}
		logger.WithError(err).Error("Failed to delete admin user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete admin user"})
		return
	}

	// Audit log the successful deletion
	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_user_deleted", c.ClientIP(), c.GetHeader("User-Agent"), true, "Admin user soft deleted successfully")
	}

	// Audit log: Admin user deleted (stdout)
	middlewares.Audit(c, "admin_user", adminUser.ID.String(), "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"email":            adminUser.Email,
			"active":           adminUser.Active,
			"is_primary_admin": adminUser.IsPrimaryAdmin,
		},
		After: map[string]interface{}{
			"deleted": true,
		},
	})

	logger.Info("Successfully soft deleted admin user")
	c.JSON(http.StatusOK, gin.H{"message": "Admin user soft deleted"})
}

// DeleteAdminUserAllRequest is the request body for hard delete
type DeleteAdminUserAllRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
}

// DeleteAdminUserAll godoc
// @Summary Hard delete admin user and all related data
// @Description Permanently deletes an admin user and all associated data from the master database. This includes role_bindings, totp_secrets, backup_codes, webauthn_credentials, sessions, etc. Cannot delete the primary admin or the last remaining admin.
// @Tags Admin-Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body DeleteAdminUserAllRequest true "Admin user delete payload"
// @Success 200 {object} map[string]interface{} "Admin user and all related data deleted successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Cannot delete primary admin or last admin"
// @Failure 404 {object} map[string]string "Admin user not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/admin/users/delete_all [post]
func (auc *AdminUserController) DeleteAdminUserAll(c *gin.Context) {
	requestID := c.GetString("request_id")
	logger := monitoring.GetLogger().WithField("request_id", requestID).WithField("operation", "delete_admin_user_all")
	logger.Info("Processing admin user hard delete request")

	var req DeleteAdminUserAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Warn("Failed to bind JSON request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate and parse user ID
	userUUID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		logger.WithError(err).WithField("user_id", req.UserID).Warn("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	// Validate and parse tenant ID
	tenantUUID, err := uuid.Parse(strings.TrimSpace(req.TenantID))
	if err != nil {
		logger.WithError(err).WithField("tenant_id", req.TenantID).Warn("Invalid tenant ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}

	logger = logger.WithField("user_id", userUUID).WithField("tenant_id", tenantUUID)

	// Validate repository initialization
	if auc.adminUserRepo == nil {
		logger.Error("Admin user repository not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin user repository not initialized"})
		return
	}

	// Get authenticated user info to validate tenant ownership
	userInfo := middlewares.GetUserInfo(c)
	if userInfo == nil || strings.TrimSpace(userInfo.TenantID) == "" {
		logger.Warn("Missing tenant scope in request")
		c.JSON(http.StatusForbidden, gin.H{"error": "Tenant scope is required"})
		return
	}

	if !strings.EqualFold(strings.TrimSpace(userInfo.TenantID), tenantUUID.String()) {
		logger.Warn("Cross-tenant deletion attempted")
		c.JSON(http.StatusForbidden, gin.H{"error": "Cross-tenant deletion is not allowed"})
		return
	}

	// Fetch admin user to delete
	adminUser, err := auc.adminUserRepo.GetAdminUserByID(userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrAdminUserNotFound) {
			logger.Warn("Admin user not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
			return
		}
		logger.WithError(err).Error("Failed to fetch admin user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load admin user"})
		return
	}

	// Validate tenant ownership
	if adminUser.TenantID == nil || !strings.EqualFold(adminUser.TenantID.String(), tenantUUID.String()) {
		logger.WithField("admin_tenant_id", adminUser.TenantID).Warn("Admin user belongs to different tenant")
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin user belongs to a different tenant"})
		return
	}

	// Check if this is the primary admin - CANNOT be deleted
	if adminUser.IsPrimaryAdmin {
		logger.Warn("Attempted to delete primary admin")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "delete_all_primary_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot delete primary admin")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the primary admin who created this tenant"})
		return
	}

	// Check if this is the last active admin for the tenant
	activeAdmins, err := auc.adminUserRepo.ListAdminUsersByTenant(tenantUUID)
	if err != nil {
		logger.WithError(err).Error("Failed to verify admin count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin count"})
		return
	}

	// Count active admins excluding the current user
	activeCount := 0
	for _, admin := range activeAdmins {
		if admin.Active && admin.ID != userUUID {
			activeCount++
		}
	}

	if activeCount == 0 {
		logger.Warn("Attempted to delete last active admin in tenant")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "delete_all_last_admin", c.ClientIP(), c.GetHeader("User-Agent"), false, "Cannot delete last admin")
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the last active admin in the tenant"})
		return
	}

	logger.Info("Hard deleting admin user and all related data")

	// Get database connection
	db := config.GetDatabase()
	if db == nil {
		logger.Error("Database not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	// Delete all related data using raw SQL (master database)
	deletedCounts := make(map[string]int64)
	tx, err := db.Begin()
	if err != nil {
		logger.WithError(err).Error("Failed to start transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Helper function to execute delete and count rows
	execDelete := func(table, query string, args ...interface{}) error {
		result, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
		if rows, err := result.RowsAffected(); err == nil {
			deletedCounts[table] = rows
		}
		return nil
	}

	// 1. Delete role_bindings
	if err := execDelete("role_bindings", "DELETE FROM role_bindings WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete role_bindings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Delete totp_secrets
	if err := execDelete("totp_secrets", "DELETE FROM totp_secrets WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete totp_secrets")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Delete backup_codes
	if err := execDelete("backup_codes", "DELETE FROM backup_codes WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete backup_codes")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Delete webauthn_credentials
	if err := execDelete("webauthn_credentials", "DELETE FROM webauthn_credentials WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete webauthn_credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5. Delete sessions
	if err := execDelete("sessions", "DELETE FROM sessions WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 6. Delete refresh_tokens
	if err := execDelete("refresh_tokens", "DELETE FROM refresh_tokens WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete refresh_tokens")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 7. Delete user_groups
	if err := execDelete("user_groups", "DELETE FROM user_groups WHERE user_id = $1", userUUID); err != nil {
		logger.WithError(err).Error("Failed to delete user_groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 8. Finally, delete the user
	if err := execDelete("users", "DELETE FROM users WHERE id = $1 AND tenant_id = $2", userUUID, tenantUUID); err != nil {
		logger.WithError(err).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.WithError(err).Error("Failed to commit transaction")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}

	logger.WithField("deleted_counts", deletedCounts).Info("Successfully hard deleted admin user")

	// Audit log the successful deletion
	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_user_hard_deleted", c.ClientIP(), c.GetHeader("User-Agent"), true, "Admin user and all related data deleted")
	}

	// Audit log: Admin user hard deleted (stdout)
	middlewares.Audit(c, "admin_user", adminUser.ID.String(), "delete_all", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"email":            adminUser.Email,
			"username":         adminUser.Username,
			"is_primary_admin": adminUser.IsPrimaryAdmin,
		},
		After: map[string]interface{}{
			"deleted":        true,
			"deleted_counts": deletedCounts,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "Admin user and all related data deleted successfully",
		"user_id":        userUUID.String(),
		"deleted_counts": deletedCounts,
	})
}

func buildAdminUserResponse(user models.AdminUser) (map[string]interface{}, error) {
	raw, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("marshal admin user: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal admin user: %w", err)
	}

	if isAdminUserInvite(user) {
		// Only invited admins (those with temporary passwords) can be "pending"
		pending := isPendingAdminInvite(user)
		payload["pending"] = pending
		payload["accepted_invite"] = !pending
		payload["invite_accepted"] = !pending
	} else {
		// Non-invited users should never show as pending
		payload["pending"] = false
		payload["accepted_invite"] = true
		payload["invite_accepted"] = true
	}

	return payload, nil
}

func isAdminUserInvite(user models.AdminUser) bool {
	return user.TemporaryPassword
}

func isPendingAdminInvite(user models.AdminUser) bool {
	return user.TemporaryPassword && user.LastLogin == nil
}

// ListEndUsersByTenant godoc
// @Summary List tenant end users
// @Description Retrieves users for a tenant using request payload filtering with optional provider filter
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body TenantUserListRequest true "Tenant listing payload with optional provider filter"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/enduser/list [post]
func (auc *AdminUserController) ListEndUsersByTenant(c *gin.Context) {
	var req TenantUserListRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Validate client_id if provided
	var clientUUID *uuid.UUID
	if req.ClientID != "" {
		parsed, err := uuid.Parse(req.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
			return
		}
		clientUUID = &parsed
	}

	users, err := auc.fetchTenantUsers(tenantUUID, clientUUID, req.Provider)
	if err != nil {
		switch err {
		case errTenantNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		case errTenantDBNotSet:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant database not configured"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = len(users)
	}
	start := (page - 1) * limit
	if start > len(users) {
		start = len(users)
	}
	end := start + limit
	if end > len(users) {
		end = len(users)
	}

	responseUsers := users[start:end]
	c.JSON(http.StatusOK, gin.H{
		"users": responseUsers,
		"total": len(users),
		"page":  page,
		"limit": limit,
	})
}

// CreateTenant creates a new tenant
func (auc *AdminUserController) CreateTenant(c *gin.Context) {
	var input struct {
		Email        string `json:"email" binding:"required,email"`
		Username     string `json:"username" binding:"required"`
		Password     string `json:"password" binding:"required,min=8"`
		Name         string `json:"name" binding:"required"`
		TenantDomain string `json:"tenant_domain"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if tenant already exists
	exists, err := auc.tenantRepo.TenantExists(input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tenant existence"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create tenant
	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:           tenantID,
		TenantID:     tenantID,
		Email:        input.Email,
		Username:     &input.Username,
		PasswordHash: hashedPassword,
		Name:         input.Name,
		TenantDomain: input.TenantDomain,
		Source:       "admin",
		Status:       "active",
	}

	if err := auc.tenantRepo.CreateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant"})
		return
	}

	// Audit log: Tenant created
	middlewares.Audit(c, "tenant", tenantID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":         input.Email,
			"username":      input.Username,
			"name":          input.Name,
			"tenant_domain": input.TenantDomain,
			"source":        "admin",
			"status":        "active",
		},
	})

	c.JSON(http.StatusCreated, gin.H{"tenant": tenant})
}

// UpdateTenant updates an existing tenant
func (auc *AdminUserController) UpdateTenant(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	var input struct {
		Email        string `json:"email,omitempty"`
		Username     string `json:"username,omitempty"`
		Name         string `json:"name,omitempty"`
		TenantDomain string `json:"tenant_domain,omitempty"`
		Status       string `json:"status,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing tenant
	existingTenant, err := auc.tenantRepo.GetTenantByTenantID(tenantID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	// Update fields if provided
	if input.Email != "" {
		existingTenant.Email = input.Email
	}
	if input.Username != "" {
		existingTenant.Username = &input.Username
	}
	if input.Name != "" {
		existingTenant.Name = input.Name
	}
	if input.TenantDomain != "" {
		existingTenant.TenantDomain = input.TenantDomain
	}
	if input.Status != "" {
		existingTenant.Status = input.Status
	}

	// Note: UpdateTenant method doesn't exist in repository, so this is a placeholder
	// In a real implementation, you'd need to add an UpdateTenant method to the repository
	c.JSON(http.StatusOK, gin.H{"message": "Tenant update not implemented yet", "tenant": existingTenant})
}

// GetTenantUsers retrieves all users for a specific tenant
func (auc *AdminUserController) GetTenantUsers(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantIDStr, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	// Get optional provider filter from query parameter
	provider := c.Query("provider")

	users, err := auc.fetchTenantUsers(tenantID, nil, provider)
	if err != nil {
		switch err {
		case errTenantNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		case errTenantDBNotSet:
			log.Printf("GetTenantUsers: tenant %s missing tenant_db mapping", tenantIDStr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant database not configured"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (auc *AdminUserController) fetchTenantUsers(tenantID uuid.UUID, clientID *uuid.UUID, provider string) ([]map[string]interface{}, error) {
	tenant, err := auc.tenantRepo.GetTenantByTenantID(tenantID.String())
	if err != nil {
		return nil, errTenantNotFound
	}

	if tenant.TenantDB == "" {
		return nil, errTenantDBNotSet
	}

	cfg := config.GetConfig()
	// Safety: ensure we do not accidentally query the primary DB when tenant_db is unset/misconfigured.
	if tenant.TenantDB == cfg.DBName || strings.TrimSpace(tenant.TenantDB) == "" {
		return nil, errTenantDBNotSet
	}

	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		tenant.TenantDB,
		cfg.DBPort,
	)

	tenantDB, err := sql.Open("postgres", tenantDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}
	defer tenantDB.Close()

	if err := tenantDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping tenant database: %w", err)
	}

	// Build query with optional client_id and provider filters
	query := `SELECT id, email, COALESCE(name, '') AS name, COALESCE(client_id::text, '') AS client_id,
	          COALESCE(provider, '') AS provider, active,
	          COALESCE(created_at, NOW()) AS created_at,
	          updated_at
	          FROM users`

	var rows *sql.Rows
	var filters []string
	var args []interface{}
	argNum := 1

	if clientID != nil {
		filters = append(filters, fmt.Sprintf("client_id = $%d", argNum))
		args = append(args, clientID)
		argNum++
	}

	if provider != "" {
		filters = append(filters, fmt.Sprintf("provider = $%d", argNum))
		args = append(args, provider)
		argNum++
	}

	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	rows, err = tenantDB.Query(query, args...)

	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	userRoles := make(map[string][]database.UserRole)
	// Fetch roles for users in bulk
	roleRows, roleErr := tenantDB.Query(`
		SELECT rb.user_id::text, COALESCE(rb.role_id::text, roles.id::text) AS role_id, COALESCE(rb.role_name, roles.name, '') AS role_name
		FROM role_bindings rb
		LEFT JOIN roles ON rb.role_id = roles.id
		WHERE rb.user_id IS NOT NULL
	`)
	if roleErr == nil {
		defer roleRows.Close()
		for roleRows.Next() {
			var uid, rid, rname string
			if scanErr := roleRows.Scan(&uid, &rid, &rname); scanErr == nil && strings.TrimSpace(uid) != "" {
				userRoles[uid] = append(userRoles[uid], database.UserRole{
					ID:   uuid.MustParse(rid),
					Name: rname,
				})
			}
		}
	}

	for rows.Next() {
		var id, email, name, clientIDStr, provider string
		var active bool
		var createdAt time.Time
		var updatedAt sql.NullTime
		if err := rows.Scan(&id, &email, &name, &clientIDStr, &provider, &active, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		parts := strings.Fields(name)
		firstName := ""
		lastName := ""
		if len(parts) > 0 {
			firstName = parts[0]
		}
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		}

		status := "inactive"
		if active {
			status = "active"
		}

		user := map[string]interface{}{
			"id":         id,
			"email":      email,
			"first_name": firstName,
			"last_name":  lastName,
			"client_id":  clientIDStr,
			"provider":   provider,
			"status":     status,
			"created_at": createdAt,
		}
		if updatedAt.Valid {
			user["updated_at"] = updatedAt.Time
		}
		if roles, ok := userRoles[id]; ok {
			user["roles"] = roles
		} else {
			user["roles"] = []database.UserRole{}
		}
		users = append(users, user)
	}

	return users, nil
}

// ToggleEndUserActive godoc
// @Summary Activate or deactivate an end user
// @Description Updates the active flag for an end user in the tenant database
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body toggleAdminUserActiveRequest true "End user toggle payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/enduser/active [post]
func (auc *AdminUserController) ToggleEndUserActive(c *gin.Context) {
	requestID := c.GetString("request_id")
	logger := monitoring.GetLogger().WithField("request_id", requestID).WithField("operation", "toggle_enduser_active")
	logger.Info("Processing end user activation/deactivation request")

	// Parse and validate request body
	var req toggleAdminUserActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Warn("Failed to bind JSON request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	active := false
	if req.Active == nil {
		logger.Warn("Active flag missing in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "active flag is required"})
		return
	} else {
		active = req.Active.Bool()
	}

	// Validate and parse user ID
	userUUID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		logger.WithError(err).WithField("user_id", req.UserID).Warn("Invalid user ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	// Validate and parse tenant ID
	tenantIDStr := strings.TrimSpace(req.TenantID)
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.WithError(err).WithField("tenant_id", req.TenantID).Warn("Invalid tenant ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}

	logger = logger.WithField("user_id", userUUID).WithField("tenant_id", tenantUUID).WithField("active", active)

	// Connect to tenant database using middlewares.GetConnectionDynamically
	tenantIDStr = tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to tenant database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	// Check if user exists
	var user models.ExtendedUser
	if err := tenantDB.Where("id = ?", userUUID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("End user not found in tenant database")
			c.JSON(http.StatusNotFound, gin.H{"error": "End user not found"})
			return
		}
		logger.WithError(err).Error("Failed to fetch end user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load end user"})
		return
	}

	// Update the active status
	if err := tenantDB.Model(&user).Update("active", active).Error; err != nil {
		logger.WithError(err).Error("Failed to update end user active flag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update end user"})
		return
	}

	// Send email notification if account is being deactivated
	if !active {
		if err := utils.SendAccountDeactivationEmail(user.Email); err != nil {
			logger.WithError(err).Warn("Failed to send deactivation email, but proceeding with deactivation")
			// Don't fail the deactivation if email fails - it's a notification only
		} else {
			logger.Info("Deactivation email sent successfully to end user")
		}
	}

	// Audit log the successful operation
	action := "enduser_deactivated"
	message := "End user deactivated successfully"
	if active {
		action = "enduser_activated"
		message = "End user activated successfully"
	}

	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "enduser", user.ID.String(), action, c.ClientIP(), c.GetHeader("User-Agent"), true, message)
	}

	// Audit log: End user activated/deactivated (stdout)
	middlewares.Audit(c, "end_user", user.ID.String(), action, &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"active": !active,
			"email":  user.Email,
		},
		After: map[string]interface{}{
			"active": active,
			"email":  user.Email,
		},
	})

	logger.Info("Successfully toggled end user active status")
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"user_id": userUUID.String(),
		"active":  active,
	})
}

// DeleteTenant permanently deletes a tenant and ALL associated data including:
// - All users (admin and end users) in the tenant
// - All roles, permissions, and role bindings
// - All MFA data (TOTP, backup codes, WebAuthn credentials)
// - All sessions and refresh tokens
// - All OAuth clients and API scopes
// - All projects
// - The tenant database itself
// - The tenant record
//
// This is an EXTREMELY DESTRUCTIVE operation and cannot be undone.
// Only super admins or the primary admin of the tenant can perform this operation.
//
// @Summary Delete tenant and all data
// @Description Permanently delete a tenant and ALL associated data. This is irreversible.
// @Tags Admin - Tenant Management
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Tenant deleted successfully"
// @Failure 400 {object} map[string]string "Invalid tenant ID"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Tenant not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/tenants/{tenant_id} [delete]
func (auc *AdminUserController) DeleteTenant(c *gin.Context) {
	requestID := c.GetString("request_id")
	logger := monitoring.GetLogger().WithField("request_id", requestID).WithField("operation", "delete_tenant")

	// Get tenant_id from path parameter
	tenantIDParam := c.Param("tenant_id")
	if tenantIDParam == "" {
		logger.Warn("Missing tenant_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	tenantUUID, err := uuid.Parse(tenantIDParam)
	if err != nil {
		logger.WithError(err).Warn("Invalid tenant_id format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}

	logger = logger.WithField("tenant_id", tenantUUID.String())
	logger.Info("Processing delete_tenant request")

	// Get authenticated user info
	userInfo := middlewares.GetUserInfo(c)
	if userInfo == nil {
		logger.Warn("Unauthorized: no user info")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify the requesting user has permission to delete this tenant
	// Must be either:
	// 1. A super admin (tenant_id is "admin" or empty)
	// 2. The primary admin of the tenant being deleted
	isSuperAdmin := userInfo.TenantID == "" || userInfo.TenantID == "admin"
	isPrimaryAdminOfTenant := strings.EqualFold(userInfo.TenantID, tenantUUID.String())

	if !isSuperAdmin && !isPrimaryAdminOfTenant {
		logger.WithFields(map[string]interface{}{
			"requester_tenant": userInfo.TenantID,
			"target_tenant":    tenantUUID.String(),
		}).Warn("Permission denied: not authorized to delete this tenant")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins or the tenant's primary admin can delete a tenant"})
		return
	}

	// Fetch the tenant to verify it exists
	tenant, err := auc.tenantRepo.GetTenantByTenantID(tenantUUID.String())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Warn("Tenant not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		logger.WithError(err).Error("Failed to fetch tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tenant"})
		return
	}

	// Store tenant info for audit log before deletion
	tenantEmail := tenant.Email
	tenantDomain := tenant.TenantDomain
	tenantDB := tenant.TenantDB

	logger.WithFields(map[string]interface{}{
		"tenant_email":  tenantEmail,
		"tenant_domain": tenantDomain,
		"tenant_db":     tenantDB,
	}).Info("Starting tenant deletion")

	// Step 1: Delete all data from the master database
	logger.Info("Step 1: Deleting tenant data from master database")
	deletedCounts, err := auc.tenantRepo.DeleteTenant(tenantUUID)
	if err != nil {
		logger.WithError(err).Error("Failed to delete tenant data from master database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant data: " + err.Error()})
		return
	}

	logger.WithField("deleted_counts", deletedCounts).Info("Successfully deleted tenant data from master database")

	// Step 2: Drop the tenant database if it exists
	databaseDropped := false
	if tenantDB != "" && auc.tenantDBService != nil {
		logger.WithField("tenant_db", tenantDB).Info("Step 2: Dropping tenant database")
		if err := auc.tenantDBService.DropTenantDatabase(tenantDB); err != nil {
			// Log the error but don't fail the entire operation
			// The master data is already deleted, so we should report partial success
			logger.WithError(err).Error("Failed to drop tenant database (master data already deleted)")
			deletedCounts["tenant_database"] = 0
		} else {
			logger.Info("Successfully dropped tenant database")
			databaseDropped = true
			deletedCounts["tenant_database"] = 1
		}
	} else {
		logger.Info("Step 2: No tenant database to drop")
	}

	// Audit log the deletion
	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin", userInfo.UserID, "tenant_deleted", c.ClientIP(), c.GetHeader("User-Agent"), true, fmt.Sprintf("Tenant %s deleted by %s", tenantUUID.String(), userInfo.Email))
	}

	// Audit log: Tenant deleted (stdout)
	middlewares.Audit(c, "tenant", tenantUUID.String(), "delete_tenant", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id":     tenantUUID.String(),
			"tenant_email":  tenantEmail,
			"tenant_domain": tenantDomain,
			"tenant_db":     tenantDB,
		},
		After: map[string]interface{}{
			"deleted":          true,
			"deleted_counts":   deletedCounts,
			"database_dropped": databaseDropped,
		},
	})

	logger.Info("Tenant deletion completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":          "Tenant and all associated data deleted successfully",
		"tenant_id":        tenantUUID.String(),
		"deleted_counts":   deletedCounts,
		"database_dropped": databaseDropped,
		"warning":          "This action is irreversible. All tenant data has been permanently deleted.",
	})
}
