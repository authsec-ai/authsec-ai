package admin

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminInviteController handles admin user invitation
type AdminInviteController struct {
	adminUserRepo *database.AdminUserRepository
}

// NewAdminInviteController creates a new admin invite controller
func NewAdminInviteController() (*AdminInviteController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, nil
	}

	return &AdminInviteController{
		adminUserRepo: database.NewAdminUserRepository(db),
	}, nil
}

// InviteAdminRequest represents the request body for inviting an admin
type InviteAdminRequest struct {
	Email        string `json:"email" binding:"required,email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username" binding:"required"`
	ClientID     string `json:"client_id"`
	TenantID     string `json:"tenant_id"`
	ProjectID    string `json:"project_id"`
	TenantDomain string `json:"tenant_domain"`
}

// InviteAdminResponse represents the response after inviting an admin
type InviteAdminResponse struct {
	Message           string              `json:"message"`
	UserID            string              `json:"user_id"`
	Email             string              `json:"email"`
	Username          string              `json:"username"`
	TemporaryPassword string              `json:"temporary_password"`
	ExpiresAt         string              `json:"expires_at"`
	EmailSent         bool                `json:"email_sent"`
	User              *InvitedUserPayload `json:"user,omitempty"`
}

// InvitedUserPayload is a sanitized view of the invited admin.
type InvitedUserPayload struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	ClientID     string `json:"client_id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	TenantDomain string `json:"tenant_domain,omitempty"`
}

// generateTemporaryPassword generates a secure random password with proper entropy
func generateTemporaryPassword(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)
	for i := range password {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		password[i] = chars[n.Int64()]
	}
	return string(password), nil
}

// InviteAdmin creates a new admin user with a temporary password
// @Summary Invite a new admin user
// @Description Create a new admin user with a temporary password that must be changed on first login
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body InviteAdminRequest true "Admin invitation details"
// @Success 201 {object} InviteAdminResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /uflow/admin/invite [post]
func (aic *AdminInviteController) InviteAdmin(c *gin.Context) {
	var req InviteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("User-flow: error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get tenant_id from token (set by auth middleware)
	var tenantIDFromToken string
	var tenantUUID uuid.UUID
	if tenantVal, exists := c.Get("tenant_id"); exists {
		if tenantStr, ok := tenantVal.(string); ok {
			tenantIDFromToken = tenantStr
			if parsedUUID, parseErr := uuid.Parse(tenantIDFromToken); parseErr == nil {
				tenantUUID = parsedUUID
			}
		}
	}

	// Check if user with this email already exists IN THIS TENANT (tenant-scoped check)
	// This respects the new composite UNIQUE constraint (email, tenant_id)
	if tenantUUID != uuid.Nil {
		u, err := aic.adminUserRepo.GetAdminUserByEmailAndTenant(req.Email, tenantUUID)
		if err == nil && u != nil {
			// User already exists in THIS tenant
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists in this tenant"})
			return
		} else if err != nil && err != sql.ErrNoRows {
			// Database error
			log.Printf("User-flow:ERROR: Failed to check user existence: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user existence"})
			return
		}
		// If err == sql.ErrNoRows, user doesn't exist in this tenant - proceed with invite
	}

	// Check if username already exists using direct database query
	db := config.GetDatabase()
	var existingUsername string

	// Parse tenant_id from token for database query
	var tenantUUIDForQuery interface{}
	if tenantIDFromToken != "" {
		if parsedUUID, parseErr := uuid.Parse(tenantIDFromToken); parseErr == nil {
			tenantUUIDForQuery = parsedUUID
		}
	}

	err := db.DB.QueryRow("SELECT username FROM users WHERE username = $1 AND tenant_id = $2 LIMIT 1", req.Username, tenantUUIDForQuery).Scan(&existingUsername)
	if err == nil {
		//reactivate the user just in case the user was deleted or deactivated.
		_, err = db.DB.Exec("UPDATE users SET active = true WHERE username = $1 AND tenant_id = $2", req.Username, tenantUUIDForQuery)

		c.JSON(http.StatusConflict, gin.H{"error": "User with this username already exists in this tenant"})
		return
	}

	// Generate temporary password (20 characters)
	temporaryPassword, err := generateTemporaryPassword(20)
	if err != nil {
		log.Printf("User-flow: failed to generate temporary password: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temporary password"})
		return
	}

	// Set expiration to 7 days from now
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	fullName := strings.TrimSpace(strings.TrimSpace(req.FirstName + " " + req.LastName))
	if fullName == "" {
		fullName = req.Username
	}
	if fullName == "" {
		fullName = req.Email
	}

	var clientIDPtr *uuid.UUID
	if strings.TrimSpace(req.ClientID) != "" {
		clientUUID, parseErr := uuid.Parse(req.ClientID)
		if parseErr != nil {
			log.Printf("User-flow: invalid client_id format: %v", parseErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
			return
		}
		clientIDPtr = &clientUUID
	}

	var tenantIDPtr *uuid.UUID
	if strings.TrimSpace(req.TenantID) != "" {
		tenantUUID, parseErr := uuid.Parse(req.TenantID)
		if parseErr != nil {
			log.Printf("User-flow: invalid tenant_id format: %v", parseErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
			return
		}
		tenantIDPtr = &tenantUUID
	}

	var projectIDPtr *uuid.UUID
	if strings.TrimSpace(req.ProjectID) != "" {
		projectUUID, parseErr := uuid.Parse(req.ProjectID)
		if parseErr != nil {
			log.Printf("User-flow: invalid project_id format: %v", parseErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
			return
		}
		projectIDPtr = &projectUUID
	}

	// Create new admin user
	adminUser := models.AdminUser{
		Email:                      req.Email,
		Username:                   req.Username,
		Name:                       fullName,
		Password:                   temporaryPassword,
		Provider:                   "local",
		Active:                     true,
		TemporaryPassword:          true,
		TemporaryPasswordExpiresAt: &expiresAt,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
		ClientID:                   clientIDPtr,
		TenantID:                   tenantIDPtr,
		ProjectID:                  projectIDPtr,
		TenantDomain:               strings.TrimSpace(req.TenantDomain),
	}

	// Hash the password
	if err := adminUser.HashPassword(); err != nil {
		log.Printf("User-flow: failed to hash password: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Save to database using repository
	if err := aic.adminUserRepo.CreateAdminUser(&adminUser); err != nil {
		log.Printf("User-flow: failed to create admin user: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin user: " + err.Error()})
		return
	}

	// Assign admin role and binding for this tenant/user (tenant-wide)
	if tenantUUID != uuid.Nil {
		roleID, err := database.NewAdminSeedRepository(config.GetDatabase()).EnsureAdminRoleAndPermissions(tenantUUID)
		if err != nil {
			log.Printf("User-flow:ERROR: Failed to ensure admin role/perms for invited admin: %v", err)
		} else {
			// Insert into role_bindings (user_roles is deprecated)
			// scope_type and scope_id are NULL for tenant-wide role assignments
			if _, err := config.GetDatabase().Exec(`
				INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
				SELECT $1, $2, $3, $4, NULL, NULL, NOW(), NOW()
				WHERE NOT EXISTS (
					SELECT 1 FROM role_bindings 
					WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 
					AND scope_type IS NULL AND scope_id IS NULL
				)
			`, uuid.New(), tenantUUID, adminUser.ID, roleID); err != nil {
				log.Printf("User-flow:ERROR: Failed to bind admin role to invited user: %v", err)
			}
		}

		// Create corresponding end user account in tenant database
		if err := aic.createEndUserInTenantDBForInvite(&adminUser, tenantUUID, clientIDPtr, projectIDPtr); err != nil {
			log.Printf("User-flow:WARNING: Failed to create end user account in tenant database: %v", err)
			// Don't fail the invitation - admin can still use global admin account
		} else {
			log.Printf("User-flow:INFO: Successfully created end user account in tenant database for invited admin: %s", adminUser.Email)
		}
	}

	emailSent := false
	if err := utils.SendAdminInviteEmail(adminUser.Email, adminUser.Username, adminUser.TenantDomain, temporaryPassword); err != nil {
		log.Printf("User-flow: failed to send admin invite email to %s: %v", adminUser.Email, err)
	} else {
		emailSent = true
	}

	responseMessage := "Admin user invited successfully. Please send the temporary password securely."
	if emailSent {
		responseMessage = "Admin user invited successfully. Temporary password emailed to the recipient."
	}

	// Audit log: Admin user invited
	middlewares.Audit(c, "admin_user", adminUser.ID.String(), "invite", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":      adminUser.Email,
			"username":   adminUser.Username,
			"name":       adminUser.Name,
			"tenant_id":  tenantIDFromToken,
			"email_sent": emailSent,
		},
	})

	c.JSON(http.StatusCreated, InviteAdminResponse{
		Message:           responseMessage,
		UserID:            adminUser.ID.String(),
		Email:             adminUser.Email,
		Username:          adminUser.Username,
		TemporaryPassword: temporaryPassword,
		ExpiresAt:         expiresAt.Format(time.RFC3339),
		EmailSent:         emailSent,
		User: &InvitedUserPayload{
			ID:           adminUser.ID.String(),
			Email:        adminUser.Email,
			Username:     adminUser.Username,
			ClientID:     uuidOrEmpty(adminUser.ClientID),
			TenantID:     uuidOrEmpty(adminUser.TenantID),
			ProjectID:    uuidOrEmpty(adminUser.ProjectID),
			TenantDomain: adminUser.TenantDomain,
		},
	})
}

func uuidOrEmpty(id *uuid.UUID) string {
	if id == nil || *id == uuid.Nil {
		return ""
	}
	return id.String()
}

// CancelInviteRequest represents the request body for canceling an invite
type CancelInviteRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// CancelInviteResponse represents the response after canceling an invite
type CancelInviteResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
}

// CancelInvite cancels a pending admin invitation by deleting the user
// @Summary Cancel a pending admin invitation
// @Description Cancel a pending admin invitation. Only works for users who have not yet logged in (temporary_password=true and last_login is null)
// @Tags Admin - Invitations
// @Accept json
// @Produce json
// @Param request body CancelInviteRequest true "Cancel invitation request"
// @Success 200 {object} CancelInviteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /uflow/admin/invite/cancel [post]
func (aic *AdminInviteController) CancelInvite(c *gin.Context) {
	var req CancelInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("User-flow: error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	// Get tenant_id from token
	var tenantUUID uuid.UUID
	if tenantVal, exists := c.Get("tenant_id"); exists {
		if tenantStr, ok := tenantVal.(string); ok {
			if parsedUUID, parseErr := uuid.Parse(tenantStr); parseErr == nil {
				tenantUUID = parsedUUID
			}
		}
	}

	// Get the user to verify they are a pending invite
	// Try to get from main users table first
	user, err := aic.adminUserRepo.GetAdminUserByID(userUUID)
	if err != nil {
		// User not found in users table, may not exist or invite already deleted
		log.Printf("User-flow: failed to get admin user by ID %s: %v", userUUID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
			"hint":  "User may have already been deleted or invite was not created successfully",
		})
		return
	}

	// Verify the user belongs to the same tenant
	if user.TenantID != nil && tenantUUID != uuid.Nil && *user.TenantID != tenantUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot cancel invite for user in different tenant"})
		return
	}

	// Verify this is a pending invite (temporary_password=true and never logged in)
	if !user.TemporaryPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User has already accepted the invitation and set their password"})
		return
	}

	if user.LastLogin != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User has already logged in. Use deactivate instead."})
		return
	}

	// Delete the user (hard delete since they never used the account)
	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not available"})
		return
	}

	// Delete role bindings first
	if _, err := db.Exec("DELETE FROM role_bindings WHERE user_id = $1", userUUID); err != nil {
		log.Printf("User-flow: failed to delete role bindings: %v", err)
	}

	// Delete the user
	if _, err := db.Exec("DELETE FROM users WHERE id = $1", userUUID); err != nil {
		log.Printf("User-flow: failed to delete user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel invitation"})
		return
	}

	// Audit log
	middlewares.Audit(c, "admin_user", userUUID.String(), "invite_cancelled", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"email":    user.Email,
			"username": user.Username,
		},
		After: map[string]interface{}{
			"deleted": true,
		},
	})

	c.JSON(http.StatusOK, CancelInviteResponse{
		Message: "Invitation cancelled successfully",
		UserID:  userUUID.String(),
		Email:   user.Email,
	})
}

// ResendInviteRequest represents the request body for resending an invite
type ResendInviteRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// ResendInviteResponse represents the response after resending an invite
type ResendInviteResponse struct {
	Message           string `json:"message"`
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	TemporaryPassword string `json:"temporary_password"`
	ExpiresAt         string `json:"expires_at"`
	EmailSent         bool   `json:"email_sent"`
}

// ResendInvite resends the invitation email with a new temporary password
// @Summary Resend admin invitation email
// @Description Resend the invitation email to a pending admin with a new temporary password. Only works for users who haven't logged in yet.
// @Tags Admin - Invitations
// @Accept json
// @Produce json
// @Param request body ResendInviteRequest true "Resend invitation request"
// @Success 200 {object} ResendInviteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /uflow/admin/invite/resend [post]
func (aic *AdminInviteController) ResendInvite(c *gin.Context) {
	var req ResendInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("User-flow: error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
		return
	}

	// Get tenant_id from token
	var tenantUUID uuid.UUID
	if tenantVal, exists := c.Get("tenant_id"); exists {
		if tenantStr, ok := tenantVal.(string); ok {
			if parsedUUID, parseErr := uuid.Parse(tenantStr); parseErr == nil {
				tenantUUID = parsedUUID
			}
		}
	}

	// Get the user
	user, err := aic.adminUserRepo.GetAdminUserByID(userUUID)
	if err != nil {
		log.Printf("User-flow: failed to get admin user by ID %s: %v", userUUID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
			"hint":  "User may have already been deleted or invite was not created successfully",
		})
		return
	}

	// Verify the user belongs to the same tenant
	if user.TenantID != nil && tenantUUID != uuid.Nil && *user.TenantID != tenantUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot resend invite for user in different tenant"})
		return
	}

	// Verify this is a pending invite
	if !user.TemporaryPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User has already accepted the invitation and set their password"})
		return
	}

	if user.LastLogin != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User has already logged in. They should use 'forgot password' instead."})
		return
	}

	// Generate new temporary password
	newTempPassword, err := generateTemporaryPassword(20)
	if err != nil {
		log.Printf("User-flow: failed to generate temporary password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temporary password"})
		return
	}

	// Set new expiration to 7 days from now
	newExpiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Hash the new password
	user.Password = newTempPassword
	if err := user.HashPassword(); err != nil {
		log.Printf("User-flow: failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update the user in database
	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not available"})
		return
	}

	_, err = db.Exec(`
		UPDATE users 
		SET password_hash = $1, temporary_password_expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`, user.PasswordHash, newExpiresAt, userUUID)
	if err != nil {
		log.Printf("User-flow: failed to update user password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update invitation"})
		return
	}

	// Send the invitation email
	emailSent := false
	if err := utils.SendAdminInviteEmail(user.Email, user.Username, user.TenantDomain, newTempPassword); err != nil {
		log.Printf("User-flow: failed to send admin invite email to %s: %v", user.Email, err)
	} else {
		emailSent = true
	}

	// Audit log
	middlewares.Audit(c, "admin_user", userUUID.String(), "invite_resent", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"email": user.Email,
		},
		After: map[string]interface{}{
			"email_sent":  emailSent,
			"new_expires": newExpiresAt.Format(time.RFC3339),
		},
	})

	responseMessage := "Invitation resent. Please send the new temporary password securely."
	if emailSent {
		responseMessage = "Invitation resent successfully. New temporary password emailed to the recipient."
	}

	c.JSON(http.StatusOK, ResendInviteResponse{
		Message:           responseMessage,
		UserID:            userUUID.String(),
		Email:             user.Email,
		TemporaryPassword: newTempPassword,
		ExpiresAt:         newExpiresAt.Format(time.RFC3339),
		EmailSent:         emailSent,
	})
}

// PendingInvite represents a pending admin invitation
type PendingInvite struct {
	UserID       string  `json:"user_id"`
	Email        string  `json:"email"`
	Username     string  `json:"username"`
	Name         string  `json:"name"`
	TenantDomain string  `json:"tenant_domain,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
	IsExpired    bool    `json:"is_expired"`
	CreatedAt    string  `json:"created_at"`
}

// ListPendingInvitesResponse represents the response for listing pending invites
type ListPendingInvitesResponse struct {
	Invites []PendingInvite `json:"invites"`
	Total   int             `json:"total"`
}

// ListPendingInvites returns all pending admin invitations for the tenant
// @Summary List pending admin invitations
// @Description Get all pending admin invitations (users with temporary_password=true who haven't logged in)
// @Tags Admin - Invitations
// @Accept json
// @Produce json
// @Success 200 {object} ListPendingInvitesResponse
// @Failure 500 {object} map[string]interface{}
// @Router /uflow/admin/invite/pending [get]
func (aic *AdminInviteController) ListPendingInvites(c *gin.Context) {
	// Get tenant_id from token
	var tenantUUID uuid.UUID
	if tenantVal, exists := c.Get("tenant_id"); exists {
		if tenantStr, ok := tenantVal.(string); ok {
			if parsedUUID, parseErr := uuid.Parse(tenantStr); parseErr == nil {
				tenantUUID = parsedUUID
			}
		}
	}

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not available"})
		return
	}

	// Query for pending invites
	query := `
		SELECT id, email, username, name, tenant_domain, temporary_password_expires_at, created_at
		FROM users
		WHERE temporary_password = true 
		  AND last_login IS NULL
		  AND ($1::uuid IS NULL OR tenant_id = $1)
		ORDER BY created_at DESC
	`

	var tenantIDParam interface{}
	if tenantUUID != uuid.Nil {
		tenantIDParam = tenantUUID
	}

	rows, err := db.Query(query, tenantIDParam)
	if err != nil {
		log.Printf("User-flow: failed to query pending invites: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pending invitations"})
		return
	}
	defer rows.Close()

	invites := []PendingInvite{}
	now := time.Now()

	for rows.Next() {
		var (
			id           uuid.UUID
			email        string
			username     string
			name         sql.NullString
			tenantDomain sql.NullString
			expiresAt    sql.NullTime
			createdAt    time.Time
		)

		if err := rows.Scan(&id, &email, &username, &name, &tenantDomain, &expiresAt, &createdAt); err != nil {
			log.Printf("User-flow: failed to scan pending invite row: %v", err)
			continue
		}

		invite := PendingInvite{
			UserID:    id.String(),
			Email:     email,
			Username:  username,
			CreatedAt: createdAt.Format(time.RFC3339),
		}

		if name.Valid {
			invite.Name = name.String
		}
		if tenantDomain.Valid {
			invite.TenantDomain = tenantDomain.String
		}
		if expiresAt.Valid {
			expStr := expiresAt.Time.Format(time.RFC3339)
			invite.ExpiresAt = &expStr
			invite.IsExpired = expiresAt.Time.Before(now)
		}

		invites = append(invites, invite)
	}

	c.JSON(http.StatusOK, ListPendingInvitesResponse{
		Invites: invites,
		Total:   len(invites),
	})
}

// createEndUserInTenantDBForInvite creates a corresponding end user account in the tenant database
// This allows invited admins to also authenticate as end users within their tenant
func (aic *AdminInviteController) createEndUserInTenantDBForInvite(adminUser *models.AdminUser, tenantID uuid.UUID, clientID, projectID *uuid.UUID) error {
	// Get tenant information
	var tenant models.Tenant
	if err := config.DB.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		return fmt.Errorf("failed to get tenant info: %w", err)
	}

	// Generate tenant database name
	tenantDBName := fmt.Sprintf("tenant_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))

	// Connect to tenant database
	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, tenantDBName, config.AppConfig.DBPort)

	tenantDB, err := sql.Open("postgres", tenantDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}
	defer tenantDB.Close()

	// Determine client_id and project_id
	var effectiveClientID, effectiveProjectID uuid.UUID
	if clientID != nil {
		effectiveClientID = *clientID
	} else {
		effectiveClientID = tenantID // Use tenant ID as default
	}

	if projectID != nil {
		effectiveProjectID = *projectID
	} else {
		// Try to get default project for this tenant
		var defaultProject models.Project
		if err := config.DB.Where("tenant_id = ? AND active = true", tenantID).First(&defaultProject).Error; err == nil {
			effectiveProjectID = defaultProject.ID
		} else {
			effectiveProjectID = tenantID // Use tenant ID as fallback
		}
	}

	// Create end user with same credentials as admin
	endUserInsert := `
		INSERT INTO users (id, client_id, tenant_id, project_id, email, name, username, 
			password_hash, tenant_domain, provider, provider_id, active, 
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true, NOW(), NOW())
		ON CONFLICT (email, client_id) DO NOTHING
	`

	_, err = tenantDB.Exec(endUserInsert,
		adminUser.ID,           // Use same ID as admin user for consistency
		effectiveClientID,      // client_id
		tenantID,               // tenant_id
		effectiveProjectID,     // project_id
		adminUser.Email,        // email
		adminUser.Name,         // name
		adminUser.Username,     // username
		adminUser.PasswordHash, // password_hash (same as admin)
		adminUser.TenantDomain, // tenant_domain
		adminUser.Provider,     // provider
		adminUser.Email,        // provider_id
	)

	if err != nil {
		return fmt.Errorf("failed to insert end user in tenant database: %w", err)
	}

	log.Printf("Created end user account in tenant database for invited admin: email=%s, user_id=%s", adminUser.Email, adminUser.ID.String())
	return nil
}
