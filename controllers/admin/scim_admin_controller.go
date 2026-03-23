package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SCIMAdminController handles SCIM 2.0 provisioning endpoints for admin users (master DB)
type SCIMAdminController struct {
	adminUserRepo *database.AdminUserRepository
	tenantRepo    *database.TenantRepository
}

// NewSCIMAdminController creates a new SCIM admin controller
func NewSCIMAdminController() (*SCIMAdminController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	return &SCIMAdminController{
		adminUserRepo: database.NewAdminUserRepository(db),
		tenantRepo:    database.NewTenantRepository(db),
	}, nil
}

// scimAdminBaseURL returns the base URL for admin SCIM resource locations
func scimAdminBaseURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/uflow/scim/v2/admin", scheme, c.Request.Host)
}

// ──────────────────────────────────────────────
// Admin User Endpoints (Master DB)
// ──────────────────────────────────────────────

// ListAdminUsers handles GET /scim/v2/admin/Users
func (sac *SCIMAdminController) ListAdminUsers(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	startIndex, _ := strconv.Atoi(c.DefaultQuery("startIndex", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "100"))
	if startIndex < 1 {
		startIndex = 1
	}
	if count < 1 || count > 200 {
		count = 100
	}

	filter := c.Query("filter")
	baseURL := scimAdminBaseURL(c)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid tenant ID", "invalidValue"))
		return
	}

	// Build query with filter
	query := "SELECT COUNT(*) FROM users WHERE tenant_id = $1"
	filterClause, filterArgs := buildAdminUserFilterClause(filter)
	args := []interface{}{tenantUUID}
	if filterClause != "" {
		query += " AND " + filterClause
		args = append(args, filterArgs...)
	}

	var totalResults int
	if err := db.DB.QueryRow(query, args...).Scan(&totalResults); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to count users", ""))
		return
	}

	// Fetch page
	selectQuery := `SELECT id, email, COALESCE(username, ''), COALESCE(name, ''),
		active, COALESCE(external_id, ''), COALESCE(sync_source, ''),
		created_at, updated_at
		FROM users WHERE tenant_id = $1`
	if filterClause != "" {
		selectQuery += " AND " + filterClause
	}
	selectQuery += " ORDER BY created_at ASC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, count, startIndex-1)

	rows, err := db.DB.Query(selectQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to fetch users", ""))
		return
	}
	defer rows.Close()

	resources := make([]interface{}, 0)
	for rows.Next() {
		var user models.AdminUser
		var username, name, externalID, syncSource sql.NullString

		if err := rows.Scan(&user.ID, &user.Email, &username, &name,
			&user.Active, &externalID, &syncSource,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			continue
		}

		if username.Valid {
			user.Username = username.String
		}
		if name.Valid {
			user.Name = name.String
		}
		if externalID.Valid {
			user.ExternalID = externalID.String
		}

		resources = append(resources, models.AdminUserToSCIMUser(user, baseURL))
	}

	c.JSON(http.StatusOK, models.NewSCIMListResponse(resources, totalResults, startIndex, len(resources)))
}

// GetAdminUser handles GET /scim/v2/admin/Users/:id
func (sac *SCIMAdminController) GetAdminUser(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	user, err := sac.fetchAdminUser(db, userUUID, tenantUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
		return
	}

	c.JSON(http.StatusOK, models.AdminUserToSCIMUser(*user, scimAdminBaseURL(c)))
}

// CreateAdminUser handles POST /scim/v2/admin/Users
func (sac *SCIMAdminController) CreateAdminUser(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	var input models.SCIMCreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid request: "+err.Error(), "invalidValue"))
		return
	}

	if input.UserName == "" {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "userName is required", "invalidValue"))
		return
	}

	email := input.GetPrimaryEmail()
	tenantUUID, _ := uuid.Parse(tenantID)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	// Check if user already exists
	var existingCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) AND tenant_id = $2", email, tenantUUID).Scan(&existingCount)
	if existingCount > 0 {
		c.JSON(http.StatusConflict, models.NewSCIMError("409", "User with this email already exists", "uniqueness"))
		return
	}

	now := time.Now()
	providerData, _ := json.Marshal(map[string]interface{}{
		"scim_external_id": input.ExternalID,
		"scim_user_name":   input.UserName,
		"title":            input.Title,
		"department":       input.Department,
		"sync_timestamp":   now.Unix(),
	})

	newUser := &models.AdminUser{
		ID:           uuid.New(),
		Email:        strings.ToLower(email),
		Username:     input.UserName,
		Name:         input.GetDisplayName(),
		TenantID:     &tenantUUID,
		Provider:     "scim",
		ProviderID:   input.UserName,
		ProviderData: providerData,
		Active:       input.GetActive(),
		ExternalID:   input.ExternalID,
		SyncSource:   "scim",
		LastSyncAt:   &now,
		IsSyncedUser: true,
		CreatedAt:    now,
		UpdatedAt:    now,
		PasswordHash: "",
	}

	if err := sac.adminUserRepo.CreateAdminUser(newUser); err != nil {
		log.Printf("SCIM Admin: Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to create user", ""))
		return
	}

	// Create tenant record for this admin user (same pattern as AdminSyncController)
	existingTenant, err := sac.tenantRepo.GetTenantByTenantID(tenantID)
	if err == nil && existingTenant != nil {
		adminSyncCtrl := &AdminSyncController{
			adminUserRepo: sac.adminUserRepo,
			tenantRepo:    sac.tenantRepo,
		}
		if err := adminSyncCtrl.createTenantForAdminUser(newUser, existingTenant); err != nil {
			log.Printf("SCIM Admin: Warning - failed to create tenant record for %s: %v", email, err)
		}
	}

	log.Printf("SCIM Admin: Created user %s (tenant: %s)", email, tenantID)

	middlewares.Audit(c, "scim_admin", tenantID, "create_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":     newUser.ID.String(),
			"email":       email,
			"external_id": input.ExternalID,
		},
	})

	c.JSON(http.StatusCreated, models.AdminUserToSCIMUser(*newUser, scimAdminBaseURL(c)))
}

// ReplaceAdminUser handles PUT /scim/v2/admin/Users/:id
func (sac *SCIMAdminController) ReplaceAdminUser(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	// Verify user exists
	user, err := sac.fetchAdminUser(db, userUUID, tenantUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
		return
	}

	var input models.SCIMCreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid request: "+err.Error(), "invalidValue"))
		return
	}

	email := input.GetPrimaryEmail()
	now := time.Now()

	providerData, _ := json.Marshal(map[string]interface{}{
		"scim_external_id": input.ExternalID,
		"scim_user_name":   input.UserName,
		"title":            input.Title,
		"department":       input.Department,
		"sync_timestamp":   now.Unix(),
	})

	updates := map[string]interface{}{
		"name":          input.GetDisplayName(),
		"username":      input.UserName,
		"email":         strings.ToLower(email),
		"active":        input.GetActive(),
		"external_id":   input.ExternalID,
		"provider_data": providerData,
		"last_sync_at":  &now,
	}

	if err := sac.adminUserRepo.UpdateAdminUser(user.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to update user", ""))
		return
	}

	// Re-fetch
	updatedUser, _ := sac.fetchAdminUser(db, userUUID, tenantUUID)

	middlewares.Audit(c, "scim_admin", tenantID, "replace_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id": userID,
			"email":   email,
		},
	})

	c.JSON(http.StatusOK, models.AdminUserToSCIMUser(*updatedUser, scimAdminBaseURL(c)))
}

// PatchAdminUser handles PATCH /scim/v2/admin/Users/:id
func (sac *SCIMAdminController) PatchAdminUser(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	user, err := sac.fetchAdminUser(db, userUUID, tenantUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
		return
	}

	var patchReq models.SCIMPatchRequest
	if err := c.ShouldBindJSON(&patchReq); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid PATCH request: "+err.Error(), "invalidValue"))
		return
	}

	updates := map[string]interface{}{
		"last_sync_at": time.Now(),
	}

	for _, op := range patchReq.Operations {
		switch strings.ToLower(op.Op) {
		case "replace", "add":
			applyAdminUserPatchReplace(op, updates)
		case "remove":
			if strings.EqualFold(op.Path, "active") {
				updates["active"] = false
			}
		}
	}

	if err := sac.adminUserRepo.UpdateAdminUser(user.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to patch user", ""))
		return
	}

	// Re-fetch
	updatedUser, _ := sac.fetchAdminUser(db, userUUID, tenantUUID)

	middlewares.Audit(c, "scim_admin", tenantID, "patch_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":    userID,
			"operations": len(patchReq.Operations),
		},
	})

	c.JSON(http.StatusOK, models.AdminUserToSCIMUser(*updatedUser, scimAdminBaseURL(c)))
}

// DeleteAdminUser handles DELETE /scim/v2/admin/Users/:id
func (sac *SCIMAdminController) DeleteAdminUser(c *gin.Context) {
	shared.SCIMContentType(c)

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", "Tenant not found in token", ""))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	db := config.GetDatabase()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Database not initialized", ""))
		return
	}

	// Verify user exists and belongs to tenant
	_, err = sac.fetchAdminUser(db, userUUID, tenantUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
		return
	}

	// Delete the user
	_, err = db.DB.Exec("DELETE FROM users WHERE id = $1 AND tenant_id = $2", userUUID, tenantUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to delete user", ""))
		return
	}

	log.Printf("SCIM Admin: Deleted user %s (tenant: %s)", userID, tenantID)

	middlewares.Audit(c, "scim_admin", tenantID, "delete_user", &middlewares.AuditChanges{
		Before: map[string]interface{}{"user_id": userID},
	})

	c.Status(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────

// fetchAdminUser retrieves an admin user by ID and tenant
func (sac *SCIMAdminController) fetchAdminUser(db *database.DBConnection, userID, tenantID uuid.UUID) (*models.AdminUser, error) {
	query := `SELECT id, email, COALESCE(username, ''), COALESCE(name, ''),
		active, COALESCE(external_id, ''), COALESCE(sync_source, ''),
		COALESCE(provider, ''), COALESCE(provider_id, ''),
		created_at, updated_at
		FROM users WHERE id = $1 AND tenant_id = $2`

	var user models.AdminUser
	var username, name, externalID, syncSource, provider, providerID sql.NullString

	err := db.DB.QueryRow(query, userID, tenantID).Scan(
		&user.ID, &user.Email, &username, &name,
		&user.Active, &externalID, &syncSource,
		&provider, &providerID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if username.Valid {
		user.Username = username.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if syncSource.Valid {
		user.SyncSource = syncSource.String
	}
	if provider.Valid {
		user.Provider = provider.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	user.TenantID = &tenantID

	return &user, nil
}

// buildAdminUserFilterClause converts SCIM filter to SQL WHERE clause
func buildAdminUserFilterClause(filter string) (string, []interface{}) {
	if filter == "" {
		return "", nil
	}

	filter = strings.TrimSpace(filter)
	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 {
		return "", nil
	}

	attr := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	switch strings.ToLower(attr) {
	case "username":
		return "(LOWER(email) = LOWER($2) OR LOWER(username) = LOWER($2))", []interface{}{value}
	case "externalid":
		return "external_id = $2", []interface{}{value}
	case "emails.value":
		return "LOWER(email) = LOWER($2)", []interface{}{value}
	case "displayname":
		return "LOWER(name) = LOWER($2)", []interface{}{value}
	case "active":
		boolVal := strings.ToLower(value) == "true"
		return "active = $2", []interface{}{boolVal}
	default:
		return "", nil
	}
}

// applyAdminUserPatchReplace processes a PATCH replace/add operation for an admin user
func applyAdminUserPatchReplace(op models.SCIMPatchOp, updates map[string]interface{}) {
	path := strings.ToLower(op.Path)

	switch path {
	case "active":
		if boolVal, ok := op.Value.(bool); ok {
			updates["active"] = boolVal
		}
		if strVal, ok := op.Value.(string); ok {
			updates["active"] = strings.ToLower(strVal) == "true"
		}
	case "username":
		if strVal, ok := op.Value.(string); ok {
			updates["username"] = strVal
		}
	case "displayname":
		if strVal, ok := op.Value.(string); ok {
			updates["name"] = strVal
		}
	case "name.givenname":
		if strVal, ok := op.Value.(string); ok {
			updates["name"] = strVal
		}
	case "name.familyname":
		if strVal, ok := op.Value.(string); ok {
			if existing, ok := updates["name"].(string); ok && existing != "" {
				updates["name"] = existing + " " + strVal
			} else {
				updates["name"] = strVal
			}
		}
	case "externalid":
		if strVal, ok := op.Value.(string); ok {
			updates["external_id"] = strVal
		}
	case "emails":
		if emails, ok := op.Value.([]interface{}); ok && len(emails) > 0 {
			if emailObj, ok := emails[0].(map[string]interface{}); ok {
				if val, ok := emailObj["value"].(string); ok {
					updates["email"] = strings.ToLower(val)
				}
			}
		}
	case "":
		if valueMap, ok := op.Value.(map[string]interface{}); ok {
			for k, v := range valueMap {
				innerOp := models.SCIMPatchOp{Path: k, Value: v}
				applyAdminUserPatchReplace(innerOp, updates)
			}
		}
	}
}
