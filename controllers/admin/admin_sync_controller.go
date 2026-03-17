package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminSyncController struct {
	adminUserRepo *database.AdminUserRepository
	tenantRepo    *database.TenantRepository
}

// NewAdminSyncController creates a new admin sync controller
func NewAdminSyncController() (*AdminSyncController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	return &AdminSyncController{
		adminUserRepo: database.NewAdminUserRepository(db),
		tenantRepo:    database.NewTenantRepository(db),
	}, nil
}

// AdminSyncInput represents the input for syncing admin users
type AdminSyncInput struct {
	TenantID    string               `json:"tenant_id" binding:"required"`
	ClientID    string               `json:"client_id,omitempty"`          // Optional client_id
	ProjectID   string               `json:"project_id,omitempty"`         // Optional project_id
	ConfigID    *string              `json:"config_id,omitempty"`          // ID of stored config to use
	ADConfig    *models.ADSyncConfig `json:"ad_config,omitempty"`          // Direct AD config
	EntraConfig *shared.EntraIDConfig       `json:"entra_config,omitempty"`       // Direct Entra config
	SyncType    string               `json:"sync_type" binding:"required"` // "ad" or "entra_id"
	DryRun      bool                 `json:"dry_run,omitempty"`
}

// AdminSyncResult represents the result of an admin sync operation
type AdminSyncResult struct {
	UsersFound   int           `json:"users_found"`
	UsersCreated int           `json:"users_created"`
	UsersUpdated int           `json:"users_updated"`
	Errors       []string      `json:"errors,omitempty"`
	PreviewUsers []interface{} `json:"preview_users,omitempty"` // Can be ADUser or shared.EntraIDUser
}

// SyncADAdminUsers godoc
// @Summary Sync admin users from Active Directory
// @Description Synchronizes admin users from Active Directory to the main database and creates tenant records
// @Tags Admin-Sync
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body AdminSyncInput true "AD admin sync configuration"
// @Success 200 {object} AdminSyncResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/admin-users/ad/sync [post]
func (asc *AdminSyncController) SyncADAdminUsers(c *gin.Context) {
	var input AdminSyncInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate tenant ID
	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Parse client_id and project_id if provided
	var clientUUID, projectUUID *uuid.UUID
	if input.ClientID != "" {
		parsed, err := uuid.Parse(input.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
			return
		}
		clientUUID = &parsed
	}
	if input.ProjectID != "" {
		parsed, err := uuid.Parse(input.ProjectID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
			return
		}
		projectUUID = &parsed
	}

	// Determine which config to use
	var adConfig models.ADSyncConfig
	if input.ConfigID != nil && *input.ConfigID != "" {
		// Load config from database
		adConfig, err = asc.loadStoredADConfig(*input.ConfigID, input.TenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to load stored configuration",
				"details": err.Error(),
			})
			return
		}
	} else if input.ADConfig != nil {
		// Use provided config directly
		adConfig = *input.ADConfig
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Either config_id or ad_config must be provided",
		})
		return
	}

	// Fetch users from AD
	adController := &shared.ADSyncController{}
	adUsers, err := adController.FetchADUsers(adConfig)
	if err != nil {
		log.Printf("Failed to fetch AD users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to AD: %v", err)})
		return
	}

	result := AdminSyncResult{
		UsersFound: len(adUsers),
		Errors:     []string{},
	}

	// If dry run, return preview without making changes
	if input.DryRun {
		previewUsers := make([]interface{}, len(adUsers))
		for i, user := range adUsers {
			previewUsers[i] = user
		}
		result.PreviewUsers = previewUsers
		c.JSON(http.StatusOK, result)
		return
	}

	// Sync users to main database
	for _, adUser := range adUsers {
		created, err := asc.syncADUserToMainDB(adUser, tenantUUID, clientUUID, projectUUID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to sync user %s: %v", adUser.Email, err))
			continue
		}
		if created {
			result.UsersCreated++
		} else {
			result.UsersUpdated++
		}
	}

	log.Printf("AD admin sync completed for tenant %s: %d users processed, %d created, %d updated, %d errors",
		input.TenantID, result.UsersFound, result.UsersCreated, result.UsersUpdated, len(result.Errors))

	// Audit log: AD admin sync completed
	middlewares.Audit(c, "admin_sync", input.TenantID, "ad_sync", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"sync_type":     "active_directory",
			"tenant_id":     input.TenantID,
			"users_found":   result.UsersFound,
			"users_created": result.UsersCreated,
			"users_updated": result.UsersUpdated,
			"errors_count":  len(result.Errors),
		},
	})

	c.JSON(http.StatusOK, result)
}

// SyncEntraAdminUsers godoc
// @Summary Sync admin users from Entra ID
// @Description Synchronizes admin users from Entra ID to the main database and creates tenant records
// @Tags Admin-Sync
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body AdminSyncInput true "Entra ID admin sync configuration"
// @Success 200 {object} AdminSyncResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/admin-users/entra/sync [post]
func (asc *AdminSyncController) SyncEntraAdminUsers(c *gin.Context) {
	var input AdminSyncInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate tenant ID
	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Parse client_id and project_id if provided
	var clientUUID, projectUUID *uuid.UUID
	if input.ClientID != "" {
		parsed, err := uuid.Parse(input.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
			return
		}
		clientUUID = &parsed
	}
	if input.ProjectID != "" {
		parsed, err := uuid.Parse(input.ProjectID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
			return
		}
		projectUUID = &parsed
	}

	// Determine which config to use
	var entraConfig shared.EntraIDConfig
	if input.ConfigID != nil && *input.ConfigID != "" {
		// Load config from database
		entraConfig, err = asc.loadStoredEntraConfig(*input.ConfigID, input.TenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to load stored configuration",
				"details": err.Error(),
			})
			return
		}
	} else if input.EntraConfig != nil {
		// Use provided config directly
		entraConfig = *input.EntraConfig
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Either config_id or entra_config must be provided",
		})
		return
	}

	// Fetch users from Entra ID
	entraController := &shared.EntraIDController{}
	service := entraController.NewEntraIDService(&entraConfig)
	entraUsers, err := service.FetchEntraIDUsers()
	if err != nil {
		log.Printf("Failed to fetch Entra ID users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to Entra ID: %v", err)})
		return
	}

	result := AdminSyncResult{
		UsersFound: len(entraUsers),
		Errors:     []string{},
	}

	// If dry run, return preview without making changes
	if input.DryRun {
		previewUsers := make([]interface{}, len(entraUsers))
		for i, user := range entraUsers {
			previewUsers[i] = user
		}
		result.PreviewUsers = previewUsers
		c.JSON(http.StatusOK, result)
		return
	}

	// Sync users to main database
	for _, entraUser := range entraUsers {
		created, err := asc.syncEntraUserToMainDB(entraUser, tenantUUID, clientUUID, projectUUID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to sync user %s: %v", entraUser.Mail, err))
			continue
		}
		if created {
			result.UsersCreated++
		} else {
			result.UsersUpdated++
		}
	}

	log.Printf("Entra ID admin sync completed for tenant %s: %d users processed, %d created, %d updated, %d errors",
		input.TenantID, result.UsersFound, result.UsersCreated, result.UsersUpdated, len(result.Errors))

	// Audit log: Entra ID admin sync completed
	middlewares.Audit(c, "admin_sync", input.TenantID, "entra_sync", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"sync_type":     "entra_id",
			"tenant_id":     input.TenantID,
			"users_found":   result.UsersFound,
			"users_created": result.UsersCreated,
			"users_updated": result.UsersUpdated,
			"errors_count":  len(result.Errors),
		},
	})

	c.JSON(http.StatusOK, result)
}

// syncADUserToMainDB syncs an AD user to the main database as an admin user and creates/updates tenant record
// Returns (created bool, error) where created=true means new user, created=false means updated existing user
func (asc *AdminSyncController) syncADUserToMainDB(adUser models.ADUser, tenantID uuid.UUID, clientID, projectID *uuid.UUID) (bool, error) {
	db := config.GetDatabase()
	if db == nil {
		return false, fmt.Errorf("database not initialized")
	}

	// Get the existing tenant to copy its configuration
	existingTenant, err := asc.tenantRepo.GetTenantByTenantID(tenantID.String())
	if err != nil {
		return false, fmt.Errorf("failed to get tenant configuration: %w", err)
	}

	// Check if user already exists (by email or external ID scoped to tenant)
	var existingUser models.AdminUser
	query := `SELECT id, email, COALESCE(username, ''), COALESCE(password_hash, ''), COALESCE(name, ''),
	          client_id, tenant_id, project_id, COALESCE(tenant_domain, ''), COALESCE(provider, ''),
	          COALESCE(provider_id, ''), COALESCE(provider_data::text, '{}'), COALESCE(avatar_url, ''),
	          active, mfa_enabled, COALESCE(mfa_method, ARRAY[]::text[]), COALESCE(mfa_default_method, ''),
	          mfa_enrolled_at, mfa_verified, COALESCE(external_id, ''), COALESCE(sync_source, ''),
	          last_sync_at, is_synced_user, last_login, created_at, updated_at
	          FROM users
	          WHERE (LOWER(email) = LOWER($1) OR external_id = $2) AND tenant_id = $3`

	var username, passwordHash, name, tenantDomain, provider, providerID, providerData, avatarURL, mfaDefaultMethod, externalID, syncSource sql.NullString
	var clientIDStr, tenantIDVal, projectIDStr sql.NullString
	var mfaEnrolledAt, lastSyncAt, lastLogin sql.NullTime
	var mfaMethodBytes []byte

	err = db.DB.QueryRow(query, adUser.Email, adUser.ObjectGUID, tenantID).Scan(
		&existingUser.ID, &existingUser.Email, &username, &passwordHash,
		&name, &clientIDStr, &tenantIDVal, &projectIDStr,
		&tenantDomain, &provider, &providerID, &providerData,
		&avatarURL, &existingUser.Active, &existingUser.MFAEnabled, &mfaMethodBytes,
		&mfaDefaultMethod, &mfaEnrolledAt, &existingUser.MFAVerified,
		&externalID, &syncSource, &lastSyncAt, &existingUser.IsSyncedUser,
		&lastLogin, &existingUser.CreatedAt, &existingUser.UpdatedAt,
	)

	// Parse mfa_method array from bytes
	if len(mfaMethodBytes) > 0 {
		var mfaArray []string
		if err := json.Unmarshal(mfaMethodBytes, &mfaArray); err == nil {
			existingUser.MFAMethod = mfaArray
		}
	}

	// Assign nullable fields
	if username.Valid {
		existingUser.Username = username.String
	}
	if passwordHash.Valid {
		existingUser.PasswordHash = passwordHash.String
	}
	if name.Valid {
		existingUser.Name = name.String
	}
	if tenantDomain.Valid {
		existingUser.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		existingUser.Provider = provider.String
	}
	if providerID.Valid {
		existingUser.ProviderID = providerID.String
	}
	if providerData.Valid {
		existingUser.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		existingUser.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		existingUser.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		existingUser.ExternalID = externalID.String
	}
	if syncSource.Valid {
		existingUser.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		existingUser.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastSyncAt.Valid {
		existingUser.LastSyncAt = &lastSyncAt.Time
	}
	if lastLogin.Valid {
		existingUser.LastLogin = &lastLogin.Time
	}
	if clientIDStr.Valid && clientIDStr.String != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			existingUser.ClientID = &parsed
		}
	}
	if tenantIDVal.Valid && tenantIDVal.String != "" {
		if parsed, err := uuid.Parse(tenantIDVal.String); err == nil {
			existingUser.TenantID = &parsed
		}
	}
	if projectIDStr.Valid && projectIDStr.String != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			existingUser.ProjectID = &parsed
		}
	}

	now := time.Now()

	if err == sql.ErrNoRows || existingUser.ID == uuid.Nil {
		// Create new admin user
		providerDataBytes, _ := json.Marshal(map[string]interface{}{
			"objectGUID":        adUser.ObjectGUID,
			"userPrincipalName": adUser.UserPrincipalName,
			"sAMAccountName":    adUser.Username,
			"department":        adUser.Department,
			"title":             adUser.Title,
			"groups":            adUser.Groups,
			"attributes":        adUser.Attributes,
		})

		newUser := &models.AdminUser{
			ID:           uuid.New(),
			Email:        adUser.Email,
			Username:     adUser.Username,
			Name:         adUser.DisplayName,
			ClientID:     clientID,
			TenantID:     &tenantID,
			ProjectID:    projectID,
			Provider:     "ad_sync",
			ProviderID:   adUser.UserPrincipalName,
			ProviderData: providerDataBytes,
			Active:       true, // Always set to true for synced admin users
			ExternalID:   adUser.ObjectGUID,
			SyncSource:   "active_directory",
			LastSyncAt:   &now,
			IsSyncedUser: true,
			CreatedAt:    now,
			UpdatedAt:    now,
			PasswordHash: "", // No password for synced users
		}

		if err := asc.adminUserRepo.CreateAdminUser(newUser); err != nil {
			return false, fmt.Errorf("failed to create admin user: %w", err)
		}

		// Create tenant record for this admin user with same tenant configuration
		if err := asc.createTenantForAdminUser(newUser, existingTenant); err != nil {
			log.Printf("Warning: Failed to create tenant record for admin user %s: %v", adUser.Email, err)
			// Don't fail the sync if tenant creation fails, just log it
		}

		log.Printf("Created new AD admin user: %s (%s)", adUser.Email, adUser.ObjectGUID)
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Update existing user
	providerDataBytes, _ := json.Marshal(map[string]interface{}{
		"objectGUID":        adUser.ObjectGUID,
		"userPrincipalName": adUser.UserPrincipalName,
		"sAMAccountName":    adUser.Username,
		"department":        adUser.Department,
		"title":             adUser.Title,
		"groups":            adUser.Groups,
		"attributes":        adUser.Attributes,
		"sync_timestamp":    now.Unix(),
	})

	updates := map[string]interface{}{
		"name":          adUser.DisplayName,
		"username":      adUser.Username,
		"active":        true, // Always set to true for synced admin users
		"last_sync_at":  &now,
		"provider_data": providerDataBytes,
	}

	if err := asc.adminUserRepo.UpdateAdminUser(existingUser.ID, updates); err != nil {
		return false, fmt.Errorf("failed to update admin user: %w", err)
	}

	// Update tenant record for this admin user (allows multiple admin emails for same tenant)
	log.Printf("Attempting to create/update tenant entry for admin user: %s", adUser.Email)
	if err := asc.createTenantForAdminUser(&existingUser, existingTenant); err != nil {
		log.Printf("ERROR: Failed to update tenant record for admin user %s: %v", adUser.Email, err)
	} else {
		log.Printf("SUCCESS: Tenant record created/updated for admin user: %s", adUser.Email)
	}

	log.Printf("Updated existing AD admin user: %s (%s)", adUser.Email, adUser.ObjectGUID)
	return false, nil
}

// syncEntraUserToMainDB syncs an Entra ID user to the main database as an admin user and creates/updates tenant record
// Returns (created bool, error) where created=true means new user, created=false means updated existing user
func (asc *AdminSyncController) syncEntraUserToMainDB(entraUser shared.EntraIDUser, tenantID uuid.UUID, clientID, projectID *uuid.UUID) (bool, error) {
	db := config.GetDatabase()
	if db == nil {
		return false, fmt.Errorf("database not initialized")
	}

	// Get the existing tenant to copy its configuration
	existingTenant, err := asc.tenantRepo.GetTenantByTenantID(tenantID.String())
	if err != nil {
		return false, fmt.Errorf("failed to get tenant configuration: %w", err)
	}

	// Check if user already exists (by email or external ID scoped to tenant)
	var existingUser models.AdminUser
	query := `SELECT id, email, COALESCE(username, ''), COALESCE(password_hash, ''), COALESCE(name, ''),
	          client_id, tenant_id, project_id, COALESCE(tenant_domain, ''), COALESCE(provider, ''),
	          COALESCE(provider_id, ''), COALESCE(provider_data::text, '{}'), COALESCE(avatar_url, ''),
	          active, mfa_enabled, COALESCE(mfa_method, ARRAY[]::text[]), COALESCE(mfa_default_method, ''),
	          mfa_enrolled_at, mfa_verified, COALESCE(external_id, ''), COALESCE(sync_source, ''),
	          last_sync_at, is_synced_user, last_login, created_at, updated_at
	          FROM users
	          WHERE (LOWER(email) = LOWER($1) OR external_id = $2) AND tenant_id = $3`

	var username, passwordHash, name, tenantDomain, provider, providerID, providerData, avatarURL, mfaDefaultMethod, externalID, syncSource sql.NullString
	var clientIDStr, tenantIDVal, projectIDStr sql.NullString
	var mfaEnrolledAt, lastSyncAt, lastLogin sql.NullTime
	var mfaMethodBytes []byte

	err = db.DB.QueryRow(query, entraUser.Mail, entraUser.ID, tenantID).Scan(
		&existingUser.ID, &existingUser.Email, &username, &passwordHash,
		&name, &clientIDStr, &tenantIDVal, &projectIDStr,
		&tenantDomain, &provider, &providerID, &providerData,
		&avatarURL, &existingUser.Active, &existingUser.MFAEnabled, &mfaMethodBytes,
		&mfaDefaultMethod, &mfaEnrolledAt, &existingUser.MFAVerified,
		&externalID, &syncSource, &lastSyncAt, &existingUser.IsSyncedUser,
		&lastLogin, &existingUser.CreatedAt, &existingUser.UpdatedAt,
	)

	// Parse mfa_method array from bytes
	if len(mfaMethodBytes) > 0 {
		var mfaArray []string
		if err := json.Unmarshal(mfaMethodBytes, &mfaArray); err == nil {
			existingUser.MFAMethod = mfaArray
		}
	}

	// Assign nullable fields
	if username.Valid {
		existingUser.Username = username.String
	}
	if passwordHash.Valid {
		existingUser.PasswordHash = passwordHash.String
	}
	if name.Valid {
		existingUser.Name = name.String
	}
	if tenantDomain.Valid {
		existingUser.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		existingUser.Provider = provider.String
	}
	if providerID.Valid {
		existingUser.ProviderID = providerID.String
	}
	if providerData.Valid {
		existingUser.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		existingUser.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		existingUser.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		existingUser.ExternalID = externalID.String
	}
	if syncSource.Valid {
		existingUser.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		existingUser.MFAEnrolledAt = &mfaEnrolledAt.Time
	}
	if lastSyncAt.Valid {
		existingUser.LastSyncAt = &lastSyncAt.Time
	}
	if lastLogin.Valid {
		existingUser.LastLogin = &lastLogin.Time
	}
	if clientIDStr.Valid && clientIDStr.String != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			existingUser.ClientID = &parsed
		}
	}
	if tenantIDVal.Valid && tenantIDVal.String != "" {
		if parsed, err := uuid.Parse(tenantIDVal.String); err == nil {
			existingUser.TenantID = &parsed
		}
	}
	if projectIDStr.Valid && projectIDStr.String != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			existingUser.ProjectID = &parsed
		}
	}

	now := time.Now()

	if err == sql.ErrNoRows || existingUser.ID == uuid.Nil {
		// Create new admin user
		providerDataBytes, _ := json.Marshal(map[string]interface{}{
			"id":                entraUser.ID,
			"userPrincipalName": entraUser.UserPrincipalName,
			"displayName":       entraUser.DisplayName,
			"mailNickname":      entraUser.MailNickname,
			"givenName":         entraUser.GivenName,
			"surname":           entraUser.Surname,
			"jobTitle":          entraUser.JobTitle,
			"department":        entraUser.Department,
			"accountEnabled":    entraUser.AccountEnabled,
			"groups":            entraUser.Groups,
			"attributes":        entraUser.Attributes,
		})

		newUser := &models.AdminUser{
			ID:           uuid.New(),
			Email:        entraUser.Mail,
			Username:     entraUser.MailNickname,
			Name:         entraUser.DisplayName,
			ClientID:     clientID,
			TenantID:     &tenantID,
			ProjectID:    projectID,
			Provider:     "entra_id",
			ProviderID:   entraUser.UserPrincipalName,
			ProviderData: providerDataBytes,
			Active:       true, // Always set to true for synced admin users
			ExternalID:   entraUser.ID,
			SyncSource:   "entra_id",
			LastSyncAt:   &now,
			IsSyncedUser: true,
			CreatedAt:    now,
			UpdatedAt:    now,
			PasswordHash: "", // No password for synced users
		}

		if err := asc.adminUserRepo.CreateAdminUser(newUser); err != nil {
			return false, fmt.Errorf("failed to create admin user: %w", err)
		}

		// Create tenant record for this admin user with same tenant configuration
		if err := asc.createTenantForAdminUser(newUser, existingTenant); err != nil {
			log.Printf("Warning: Failed to create tenant record for admin user %s: %v", entraUser.Mail, err)
			// Don't fail the sync if tenant creation fails, just log it
		}

		log.Printf("Created new Entra ID admin user: %s (%s)", entraUser.Mail, entraUser.ID)
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Update existing user
	providerDataBytes, _ := json.Marshal(map[string]interface{}{
		"id":                entraUser.ID,
		"userPrincipalName": entraUser.UserPrincipalName,
		"displayName":       entraUser.DisplayName,
		"mailNickname":      entraUser.MailNickname,
		"givenName":         entraUser.GivenName,
		"surname":           entraUser.Surname,
		"jobTitle":          entraUser.JobTitle,
		"department":        entraUser.Department,
		"accountEnabled":    entraUser.AccountEnabled,
		"groups":            entraUser.Groups,
		"attributes":        entraUser.Attributes,
		"sync_timestamp":    now.Unix(),
	})

	updates := map[string]interface{}{
		"name":          entraUser.DisplayName,
		"username":      entraUser.MailNickname,
		"active":        true, // Always set to true for synced admin users
		"last_sync_at":  &now,
		"provider_data": providerDataBytes,
	}

	if err := asc.adminUserRepo.UpdateAdminUser(existingUser.ID, updates); err != nil {
		return false, fmt.Errorf("failed to update admin user: %w", err)
	}

	// Update tenant record for this admin user (allows multiple admin emails for same tenant)
	log.Printf("Attempting to create/update tenant entry for admin user: %s", entraUser.Mail)
	if err := asc.createTenantForAdminUser(&existingUser, existingTenant); err != nil {
		log.Printf("ERROR: Failed to update tenant record for admin user %s: %v", entraUser.Mail, err)
	} else {
		log.Printf("SUCCESS: Tenant record created/updated for admin user: %s", entraUser.Mail)
	}

	log.Printf("Updated existing Entra ID admin user: %s (%s)", entraUser.Mail, entraUser.ID)
	return false, nil
}

// createTenantForAdminUser creates a new tenant entry with the same tenant_id as existing tenant
// This allows multiple admin emails to share the same tenant configuration
func (asc *AdminSyncController) createTenantForAdminUser(adminUser *models.AdminUser, existingTenant *sharedmodels.Tenant) error {
	// Check if tenant entry already exists for this email
	existingTenantRecord, err := asc.tenantRepo.GetTenantByEmail(adminUser.Email)
	if err == nil && existingTenantRecord != nil {
		// Tenant entry exists for this email - update it
		log.Printf("Tenant entry already exists for email %s, updating it", adminUser.Email)

		// Update tenant record with latest sync data
		now := time.Now()
		updateQuery := `UPDATE tenants
			SET username = $1, name = $2, tenant_domain = $3, tenant_db = $4,
			    source = $5, status = $6, updated_at = $7
			WHERE email = $8 AND tenant_id = $9`

		db := config.GetDatabase()
		if db == nil {
			return fmt.Errorf("database not initialized")
		}

		_, err := db.DB.Exec(updateQuery,
			adminUser.Username,
			adminUser.Name,
			existingTenant.TenantDomain,
			existingTenant.TenantDB,
			existingTenant.Source,
			existingTenant.Status,
			now,
			adminUser.Email,
			*adminUser.TenantID,
		)

		if err != nil {
			return fmt.Errorf("failed to update tenant entry: %w", err)
		}

		log.Printf("Updated tenant entry for admin user %s with tenant_id %s", adminUser.Email, adminUser.TenantID)
		return nil
	}

	// Tenant entry doesn't exist for this email - create a new entry with same tenant_id
	// This creates a new row in tenants table with the new email but same tenant configuration
	tenant := &sharedmodels.Tenant{
		ID:           uuid.New(),
		TenantID:     *adminUser.TenantID, // Same tenant_id as existing tenant
		Email:        adminUser.Email,     // New email from synced user
		Username:     &adminUser.Username,
		Name:         adminUser.Name,
		TenantDomain: existingTenant.TenantDomain, // Copy from existing tenant
		TenantDB:     existingTenant.TenantDB,     // Copy from existing tenant (same DB)
		Source:       existingTenant.Source,       // Copy from existing tenant
		Status:       existingTenant.Status,       // Copy from existing tenant
		PasswordHash: adminUser.PasswordHash,      // Empty for synced users
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := asc.tenantRepo.CreateTenant(tenant); err != nil {
		return fmt.Errorf("failed to create tenant entry: %w", err)
	}

	log.Printf("Created new tenant entry for admin user %s with tenant_id %s (shares same tenant_db: %s)",
		adminUser.Email, adminUser.TenantID, existingTenant.TenantDB)
	return nil
}

// loadStoredADConfig loads AD configuration from database and decrypts credentials
func (asc *AdminSyncController) loadStoredADConfig(configID, tenantID string) (models.ADSyncConfig, error) {
	var syncConfig models.SyncConfiguration

	// Parse UUIDs
	configUUID, err := uuid.Parse(configID)
	if err != nil {
		return models.ADSyncConfig{}, fmt.Errorf("invalid config_id format")
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return models.ADSyncConfig{}, fmt.Errorf("invalid tenant_id format")
	}

	// Fetch configuration from database
	if err := config.DB.Where("id = ? AND tenant_id = ? AND sync_type = ?",
		configUUID, tenantUUID, "active_directory").First(&syncConfig).Error; err != nil {
		return models.ADSyncConfig{}, fmt.Errorf("sync configuration not found or not authorized")
	}

	// Check if config is active
	if !syncConfig.IsActive {
		return models.ADSyncConfig{}, fmt.Errorf("sync configuration is disabled")
	}

	// Decrypt password
	decryptedPassword, err := utils.Decrypt(syncConfig.ADPassword)
	if err != nil {
		return models.ADSyncConfig{}, fmt.Errorf("failed to decrypt credentials")
	}

	// Build ADSyncConfig
	adConfig := models.ADSyncConfig{
		Server:     syncConfig.ADServer,
		Username:   syncConfig.ADUsername,
		Password:   decryptedPassword,
		BaseDN:     syncConfig.ADBaseDN,
		Filter:     syncConfig.ADFilter,
		UseSSL:     syncConfig.ADUseSSL,
		SkipVerify: syncConfig.ADSkipVerify,
	}

	return adConfig, nil
}

// loadStoredEntraConfig loads Entra ID configuration from database and decrypts credentials
func (asc *AdminSyncController) loadStoredEntraConfig(configID, tenantID string) (shared.EntraIDConfig, error) {
	var syncConfig models.SyncConfiguration

	// Parse UUIDs
	configUUID, err := uuid.Parse(configID)
	if err != nil {
		return shared.EntraIDConfig{}, fmt.Errorf("invalid config_id format")
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return shared.EntraIDConfig{}, fmt.Errorf("invalid tenant_id format")
	}

	// Fetch configuration from database
	if err := config.DB.Where("id = ? AND tenant_id = ? AND sync_type = ?",
		configUUID, tenantUUID, "entra_id").First(&syncConfig).Error; err != nil {
		return shared.EntraIDConfig{}, fmt.Errorf("sync configuration not found or not authorized")
	}

	// Check if config is active
	if !syncConfig.IsActive {
		return shared.EntraIDConfig{}, fmt.Errorf("sync configuration is disabled")
	}

	// Decrypt client secret
	decryptedSecret, err := utils.Decrypt(syncConfig.EntraClientSecret)
	if err != nil {
		return shared.EntraIDConfig{}, fmt.Errorf("failed to decrypt credentials")
	}

	// Parse scopes from JSON
	var scopes []string
	if syncConfig.EntraScopes != "" {
		if err := json.Unmarshal([]byte(syncConfig.EntraScopes), &scopes); err != nil {
			log.Printf("Warning: failed to parse scopes JSON: %v", err)
			scopes = []string{"https://graph.microsoft.com/.default"}
		}
	}

	// Build shared.EntraIDConfig
	entraConfig := shared.EntraIDConfig{
		TenantID:     syncConfig.EntraTenantID,
		ClientID:     syncConfig.EntraClientID,
		ClientSecret: decryptedSecret,
		Scopes:       scopes,
		SkipVerify:   syncConfig.EntraSkipVerify,
	}

	return entraConfig, nil
}
