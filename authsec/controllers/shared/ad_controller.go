package shared

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ADSyncController struct{}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// ADSyncConfig holds configuration for AD connection
type ADSyncConfig struct {
	Server     string `json:"server"`      // AD server address (e.g., "dc.company.com:636")
	Username   string `json:"username"`    // Service account username
	Password   string `json:"password"`    // Service account password
	BaseDN     string `json:"base_dn"`     // Base DN for user search (e.g., "OU=Users,DC=company,DC=com")
	Filter     string `json:"filter"`      // LDAP filter (e.g., "(&(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))")
	UseSSL     bool   `json:"use_ssl"`     // Whether to use SSL/TLS
	SkipVerify bool   `json:"skip_verify"` // Skip SSL certificate verification (for testing)
}

// SyncUsersInput represents the input for syncing users from AD
type SyncUsersInput struct {
	TenantID  string        `json:"tenant_id" binding:"required"`
	ClientID  string        `json:"client_id" binding:"required"`
	ProjectID string        `json:"project_id" binding:"required"`
	ConfigID  *string       `json:"config_id,omitempty"` // ID of stored config to use
	Config    *ADSyncConfig `json:"config,omitempty"`    // Or provide config directly (for backward compatibility)
	DryRun    bool          `json:"dry_run,omitempty"`   // Preview changes without applying
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	UsersFound   int      `json:"users_found"`
	UsersCreated int      `json:"users_created"`
	UsersUpdated int      `json:"users_updated"`
	Errors       []string `json:"errors,omitempty"`
	PreviewUsers []ADUser `json:"preview_users,omitempty"` // Only populated for dry runs
}

// ADUser represents a user from Active Directory
type ADUser struct {
	ObjectGUID        string            `json:"object_guid"`
	UserPrincipalName string            `json:"user_principal_name"`
	DisplayName       string            `json:"display_name"`
	Email             string            `json:"email"`
	Username          string            `json:"username"`
	Department        string            `json:"department"`
	Title             string            `json:"title"`
	Groups            []string          `json:"groups"`
	Attributes        map[string]string `json:"attributes"`
	IsActive          bool              `json:"is_active"`
}

// SyncADUsers godoc
// @Summary Sync users from Active Directory
// @Description Synchronizes users from Active Directory to the tenant database. Supports both stored config (via config_id) or direct config.
// @Tags ADSync
// @Accept json
// @Produce json
// @Param input body object true "AD sync configuration"
// @Success 200 {object} object
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /uflow/ad/sync [post]
func (asc *ADSyncController) SyncADUsers(c *gin.Context) {
	var input models.SyncUsersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Determine which config to use
	var adConfig models.ADSyncConfig
	var err error

	if input.ConfigID != nil && *input.ConfigID != "" {
		// Load config from database
		adConfig, err = asc.loadStoredADConfig(*input.ConfigID, input.TenantID, input.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Failed to load stored configuration",
				Details: err.Error(),
			})
			return
		}
	} else if input.Config != nil {
		// Use provided config directly
		adConfig = *input.Config
	} else {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Either config_id or config must be provided",
			Details: "Please provide config_id to use stored credentials or config for direct credentials",
		})
		return
	}

	// Connect to AD and fetch users
	adUsers, err := asc.FetchADUsers(adConfig)
	if err != nil {
		log.Printf("Failed to fetch AD users: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to connect to AD",
			Details: err.Error(),
		})
		return
	}

	result := models.SyncResult{
		UsersFound: len(adUsers),
		Errors:     []string{},
	}

	// If dry run, return preview without making changes
	if input.DryRun {
		result.PreviewUsers = adUsers
		c.JSON(http.StatusOK, result)
		return
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &input.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Sync users to database
	for _, adUser := range adUsers {
		if err := asc.syncUserToDatabase(tenantDB, adUser, input.TenantID, input.ClientID, input.ProjectID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to sync user %s: %v", adUser.Email, err))
			continue
		}
		result.UsersCreated++
	}

	log.Printf("AD sync completed for tenant %s: %d users processed, %d created, %d errors",
		input.TenantID, result.UsersFound, result.UsersCreated, len(result.Errors))

	// Audit log: AD sync completed
	middlewares.Audit(c, "ad_sync", input.TenantID, "sync_users", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":     input.TenantID,
			"client_id":     input.ClientID,
			"users_found":   result.UsersFound,
			"users_created": result.UsersCreated,
			"errors_count":  len(result.Errors),
		},
	})

	c.JSON(http.StatusOK, result)
}

// TestNetworkConnection godoc
// @Summary Test network connectivity to AD server
// @Description Tests basic TCP connectivity to AD server before LDAP
// @Tags ADSync
// @Accept json
// @Produce json
// @Param config body map[string]string true "Server address"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /uflow/ad/test-network [post]
func (asc *ADSyncController) TestNetworkConnection(c *gin.Context) {
	var input map[string]string
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	server := input["server"]
	if server == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Missing required parameter",
			Details: "server address required",
		})
		return
	}

	// Test basic TCP connection
	conn, err := ldap.Dial("tcp", server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "TCP connection failed",
			Details: err.Error(),
		})
		return
	}
	defer conn.Close()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "TCP connection successful",
		"server":  server,
	})
}

// TestADConnection godoc
// @Summary Test Active Directory connection
// @Description Tests connection to Active Directory with provided configuration
// @Tags ADSync
// @Accept json
// @Produce json
// @Param input body object true "AD connection configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /uflow/ad/test-connection [post]
func (asc *ADSyncController) TestADConnection(c *gin.Context) {
	var config models.ADSyncConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Test connection to AD
	conn, err := asc.connectToAD(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to connect to AD: %v", err),
		})
		return
	}
	defer conn.Close()
	filter := config.Filter
	if filter == "" {
		filter = "(objectClass=user)"
	}

	// Test a simple search to verify permissions (limit to 5 results)
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 5, 30, false,
		filter, // <-- FIX: Use the filter from the input config
		[]string{"cn", "mail", "userPrincipalName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to search AD: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Successfully connected to Active Directory",
		"users_found": len(sr.Entries),
		"base_dn":     config.BaseDN,
		"sample_users": func() []map[string]string {
			var samples []map[string]string
			for i, entry := range sr.Entries {
				if i >= 3 { // Limit to first 3 users
					break
				}
				user := map[string]string{
					"name":  entry.GetAttributeValue("cn"),
					"email": entry.GetAttributeValue("mail"),
					"upn":   entry.GetAttributeValue("userPrincipalName"),
				}
				samples = append(samples, user)
			}
			return samples
		}(),
	})
}

// Private helper methods

func (asc *ADSyncController) connectToAD(config models.ADSyncConfig) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if config.UseSSL {
		// Connect with SSL/TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.SkipVerify,
		}
		conn, err = ldap.DialTLS("tcp", config.Server, tlsConfig)
	} else {
		// Connect without SSL (not recommended for production)
		conn, err = ldap.Dial("tcp", config.Server)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to AD server: %w", err)
	}

	// Bind with service account
	if err := conn.Bind(config.Username, config.Password); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to AD: %w", err)
	}

	return conn, nil
}

func (asc *ADSyncController) FetchADUsers(config models.ADSyncConfig) ([]models.ADUser, error) {
	conn, err := asc.connectToAD(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Default filter if none provided
	filter := config.Filter
	if filter == "" {
		// Default filter: active user accounts only
		filter = "(&(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))"
	}

	// Define attributes to retrieve
	attributes := []string{
		"objectGUID",
		"userPrincipalName",
		"displayName",
		"mail",
		"sAMAccountName",
		"department",
		"title",
		"memberOf",
		"userAccountControl",
		"cn",
	}

	// Use pagination to handle large result sets
	var allUsers []models.ADUser
	pageSize := 100
	pagingControl := ldap.NewControlPaging(uint32(pageSize))

	for {
		searchRequest := ldap.NewSearchRequest(
			config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0, 30, false, // size limit 0, time limit 30 seconds
			filter,
			attributes,
			[]ldap.Control{pagingControl},
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to search AD: %w", err)
		}

		// Process this page of results
		for _, entry := range sr.Entries {
			user := asc.mapLDAPEntryToUser(entry)
			if user.Email != "" { // Only include users with email addresses
				allUsers = append(allUsers, user)
			}
		}

		// Check if there are more pages
		pagingResult := ldap.FindControl(sr.Controls, ldap.ControlTypePaging)
		if pagingResult == nil {
			break
		}

		pagingControl = pagingResult.(*ldap.ControlPaging)
		if len(pagingControl.Cookie) == 0 {
			break
		}
	}

	return allUsers, nil
}

func (asc *ADSyncController) mapLDAPEntryToUser(entry *ldap.Entry) models.ADUser {
	// Helper function to safely get attribute value
	getAttr := func(name string) string {
		values := entry.GetAttributeValues(name)
		if len(values) > 0 {
			return values[0]
		}
		return ""
	}

	var objectGUID string
	rawGUID := entry.GetRawAttributeValue("objectGUID")
	if len(rawGUID) == 16 {
		// Parse the 16-byte slice into a UUID object
		parsedGUID, err := uuid.FromBytes(rawGUID)
		if err == nil {
			// Convert the UUID object to its standard string format
			objectGUID = parsedGUID.String()
		}
	}

	// Get email (try mail first, then userPrincipalName)
	email := getAttr("mail")
	if email == "" {
		email = getAttr("userPrincipalName")
	}

	// Parse user account control to determine if account is active
	uacStr := getAttr("userAccountControl")
	isActive := true
	if uacStr != "" {
		// Bit 2 (0x2) indicates disabled account
		// This is a simplified check - you might want more robust parsing
		isActive = !strings.Contains(uacStr, "2")
	}

	// Get group memberships
	groups := entry.GetAttributeValues("memberOf")

	// Clean up group names (extract CN from DN)
	var cleanGroups []string
	for _, group := range groups {
		if strings.HasPrefix(group, "CN=") {
			parts := strings.Split(group, ",")
			if len(parts) > 0 {
				cn := strings.TrimPrefix(parts[0], "CN=")
				cleanGroups = append(cleanGroups, cn)
			}
		}
	}

	return models.ADUser{
		ObjectGUID:        objectGUID,
		UserPrincipalName: getAttr("userPrincipalName"),
		DisplayName:       getAttr("displayName"),
		Email:             strings.ToLower(email),
		Username:          getAttr("sAMAccountName"),
		Department:        getAttr("department"),
		Title:             getAttr("title"),
		Groups:            cleanGroups,
		IsActive:          isActive,
		Attributes: map[string]string{
			"cn":                 getAttr("cn"),
			"userPrincipalName":  getAttr("userPrincipalName"),
			"sAMAccountName":     getAttr("sAMAccountName"),
			"userAccountControl": getAttr("userAccountControl"),
		},
	}
}

func (asc *ADSyncController) syncUserToDatabase(tenantDB *gorm.DB, adUser models.ADUser, tenantID string, clientID string, projectID string) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID format: %w", err)
	}

	domainSuffix := "app.authsec.ai"
	if config.AppConfig != nil && config.AppConfig.TenantDomainSuffix != "" {
		domainSuffix = config.AppConfig.TenantDomainSuffix
	}

	var existingUser models.User
	err = tenantDB.Where("(email = ? OR external_id = ?) AND client_id = ?", adUser.Email, adUser.ObjectGUID, clientUUID).First(&existingUser).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		newUser := models.ExtendedUser{
			User: sharedmodels.User{
				ID:         uuid.New(),
				ClientID:   clientUUID,
				TenantID:   tenantUUID,
				ProjectID:  projectUUID,
				Name:       adUser.DisplayName,
				Username:   stringPtr(adUser.Username),
				Email:      adUser.Email,
				Provider:   "ad_sync",
				ProviderID: adUser.UserPrincipalName,
				Active:     adUser.IsActive,
				ProviderData: func() datatypes.JSON {
					data, _ := json.Marshal(map[string]interface{}{
						"objectGUID":        adUser.ObjectGUID,
						"userPrincipalName": adUser.UserPrincipalName,
						"sAMAccountName":    adUser.Username,
						"department":        adUser.Department,
						"title":             adUser.Title,
						"groups":            adUser.Groups,
						"attributes":        adUser.Attributes,
					})
					return datatypes.JSON(data)
				}(),
				TenantDomain: domainSuffix,
				MFAEnabled:   false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			ExternalID:   stringPtr(adUser.ObjectGUID),
			SyncSource:   stringPtr("active_directory"),
			LastSyncAt:   &now,
			IsSyncedUser: true,
		}

		if err := tenantDB.Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		log.Printf("Created new AD user: %s (%s)", adUser.Email, adUser.ObjectGUID)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	updates := map[string]interface{}{
		"name":         adUser.DisplayName,
		"username":     adUser.Username,
		"active":       adUser.IsActive,
		"last_sync_at": &now,
		"updated_at":   now,
		"provider_data": map[string]interface{}{
			"objectGUID":        adUser.ObjectGUID,
			"userPrincipalName": adUser.UserPrincipalName,
			"sAMAccountName":    adUser.Username,
			"department":        adUser.Department,
			"title":             adUser.Title,
			"groups":            adUser.Groups,
			"attributes":        adUser.Attributes,
		},
	}

	if err := tenantDB.Model(&existingUser).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("Updated existing AD user: %s (%s)", adUser.Email, adUser.ObjectGUID)
	return nil
}

func (asc *ADSyncController) AgentSyncUsers(c *gin.Context) {
	var input models.AgentSyncRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	result := models.AgentSyncResponse{
		UsersProcessed: len(input.Users),
		Errors:         []models.ErrorResponse{},
	}

	// If dry run, just return preview
	if input.DryRun {
		result.Message = "Dry run completed - no users were actually synced"
		c.JSON(http.StatusOK, result)
		return
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &input.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Process each user
	for _, user := range input.Users {
		if err := asc.syncAgentUserToDatabase(tenantDB, user, input.TenantID, input.ProjectID, input.ClientID); err != nil {
			result.Errors = append(result.Errors, models.ErrorResponse{
				Error:   "User sync failed",
				Details: fmt.Sprintf("Failed to sync user %s: %v", user.Email, err),
			})
			continue
		}
		// If we reach here, there was no error, so increment the counter
		result.UsersCreated++
	}

	result.Message = fmt.Sprintf("Sync completed: %d users processed, %d created/updated, %d errors",
		result.UsersProcessed, result.UsersCreated, len(result.Errors))

	log.Printf("Agent sync completed for tenant %s: %s", input.TenantID, result.Message)

	// Audit log: Agent sync completed
	middlewares.Audit(c, "ad_sync", input.TenantID, "agent_sync_users", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":       input.TenantID,
			"client_id":       input.ClientID,
			"users_processed": result.UsersProcessed,
			"users_created":   result.UsersCreated,
			"errors_count":    len(result.Errors),
		},
	})

	c.JSON(http.StatusOK, result)
}

// AgentSyncRequest represents the request from AD Agent
type AgentSyncRequest struct {
	TenantID  string          `json:"tenant_id" binding:"required"`
	ProjectID string          `json:"project_id" binding:"required"`
	ClientID  string          `json:"client_id" binding:"required"`
	Users     []AgentUserData `json:"users" binding:"required"`
	DryRun    bool            `json:"dry_run,omitempty"`
}

type AgentUserData struct {
	ExternalID   string                 `json:"external_id"`
	Email        string                 `json:"email"`
	Username     string                 `json:"username"`
	Provider     string                 `json:"provider"`
	ProviderID   string                 `json:"provider_id"`
	ProviderData map[string]interface{} `json:"provider_data"`
	IsActive     bool                   `json:"is_active"`
	IsSyncedUser bool                   `json:"is_synced_user"`
	SyncSource   string                 `json:"sync_source"`
}

type AgentSyncResponse struct {
	Message        string   `json:"message"`
	UsersProcessed int      `json:"users_processed"`
	UsersCreated   int      `json:"users_created"`
	Errors         []string `json:"errors,omitempty"`
}

func (asc *ADSyncController) syncAgentUserToDatabase(tenantDB *gorm.DB, agentUser models.AgentUserData, tenantID, projectID string, clientID string) error {
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID format: %w", err)
	}

	domainSuffix := "app.authsec.ai"
	if config.AppConfig != nil && config.AppConfig.TenantDomainSuffix != "" {
		domainSuffix = config.AppConfig.TenantDomainSuffix
	}

	var existingUser models.User
	err = tenantDB.Where("(email = ? OR external_id = ?) AND client_id = ?", agentUser.Email, agentUser.ExternalID, clientUUID).First(&existingUser).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		providerData, _ := json.Marshal(agentUser.ProviderData)

		newUser := models.ExtendedUser{
			User: sharedmodels.User{
				ID:           uuid.New(),
				ClientID:     clientUUID,
				TenantID:     tenantUUID,
				ProjectID:    projectUUID,
				Name:         agentUser.Name,
				Username:     stringPtr(agentUser.Username),
				Email:        agentUser.Email,
				Provider:     agentUser.Provider,
				ProviderID:   agentUser.ProviderID,
				Active:       agentUser.IsActive,
				ProviderData: datatypes.JSON(providerData),
				TenantDomain: domainSuffix,
				MFAEnabled:   false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			ExternalID:   stringPtr(agentUser.ExternalID),
			SyncSource:   stringPtr(agentUser.SyncSource),
			LastSyncAt:   &now,
			IsSyncedUser: agentUser.IsSyncedUser,
		}

		if err := tenantDB.Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		log.Printf("Created new AD user via agent: %s (%s)", agentUser.Email, agentUser.ExternalID)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	providerData, _ := json.Marshal(agentUser.ProviderData)
	updates := map[string]interface{}{
		"name":          agentUser.Name,
		"username":      agentUser.Username,
		"active":        agentUser.IsActive,
		"last_sync_at":  &now,
		"updated_at":    now,
		"provider_data": datatypes.JSON(providerData),
	}

	if err := tenantDB.Model(&existingUser).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("Updated existing AD user via agent: %s (%s)", agentUser.Email, agentUser.ExternalID)
	return nil
}

// loadStoredADConfig loads AD configuration from database and decrypts credentials
func (asc *ADSyncController) loadStoredADConfig(configID, tenantID, clientID string) (models.ADSyncConfig, error) {
	// Import utils package for decryption
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
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return models.ADSyncConfig{}, fmt.Errorf("invalid client_id format")
	}

	// Fetch configuration from database
	if err := config.DB.Where("id = ? AND tenant_id = ? AND client_id = ? AND sync_type = ?",
		configUUID, tenantUUID, clientUUID, "active_directory").First(&syncConfig).Error; err != nil {
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
