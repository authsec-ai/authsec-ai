package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SCIMController handles SCIM 2.0 provisioning endpoints for end-users (tenant DB)
type SCIMController struct{}

// scimContentType sets the proper SCIM content type header
func scimContentType(c *gin.Context) {
	c.Header("Content-Type", "application/scim+json; charset=utf-8")
}

// scimBaseURL returns the base URL for SCIM resource locations
func scimBaseURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	// Include client_id and project_id in base URL if present (end-user routes)
	clientID := c.Param("client_id")
	projectID := c.Param("project_id")
	if clientID != "" && projectID != "" {
		return fmt.Sprintf("%s://%s/uflow/scim/v2/%s/%s", scheme, c.Request.Host, clientID, projectID)
	}
	return fmt.Sprintf("%s://%s/uflow/scim/v2", scheme, c.Request.Host)
}

// getClientAndProjectID extracts and validates client_id and project_id from URL params
func getClientAndProjectID(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	clientIDStr := c.Param("client_id")
	projectIDStr := c.Param("project_id")

	if clientIDStr == "" || projectIDStr == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("client_id and project_id are required")
	}

	clientUUID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid client_id format: %w", err)
	}

	projectUUID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid project_id format: %w", err)
	}

	return clientUUID, projectUUID, nil
}

// getTenantDB resolves the tenant DB from the JWT context
func getTenantDB(c *gin.Context) (*gorm.DB, string, error) {
	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		return nil, "", fmt.Errorf("tenant not found in token")
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	return tenantDB, tenantID, nil
}

// ──────────────────────────────────────────────
// Discovery Endpoints
// ──────────────────────────────────────────────

// GetServiceProviderConfig returns the SCIM ServiceProviderConfig
func (sc *SCIMController) GetServiceProviderConfig(c *gin.Context) {
	shared.SCIMContentType(c)
	c.JSON(http.StatusOK, models.SCIMServiceProviderConfig{
		Schemas:          []string{models.SCIMSchemaServiceProvider},
		DocumentationURI: "",
		Patch:            models.SCIMSupported{Supported: true},
		Bulk:             models.SCIMBulkConfig{Supported: false, MaxOperations: 0, MaxPayloadSize: 0},
		Filter:           models.SCIMFilterConfig{Supported: true, MaxResults: 200},
		ChangePassword:   models.SCIMSupported{Supported: false},
		Sort:             models.SCIMSupported{Supported: false},
		ETag:             models.SCIMSupported{Supported: false},
		AuthenticationSchemes: []models.SCIMAuthScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication scheme using the OAuth Bearer Token Standard (RFC 6750)",
				SpecURI:     "https://www.rfc-editor.org/info/rfc6750",
				Primary:     true,
			},
		},
		Meta: &models.SCIMMeta{
			ResourceType: "ServiceProviderConfig",
			Created:      time.Now(),
			LastModified: time.Now(),
			Location:     scimBaseURL(c) + "/ServiceProviderConfig",
		},
	})
}

// GetSchemas returns the SCIM Schema definitions
func (sc *SCIMController) GetSchemas(c *gin.Context) {
	shared.SCIMContentType(c)
	baseURL := scimBaseURL(c)

	userSchema := models.SCIMSchemaDefinition{
		Schemas:     []string{models.SCIMSchemaSchema},
		ID:          models.SCIMSchemaUser,
		Name:        "User",
		Description: "User Account",
		Attributes: []models.SCIMSchemaAttribute{
			{Name: "userName", Type: "string", MultiValued: false, Description: "Unique identifier for the User", Required: true, Mutability: "readWrite", Returned: "default", Uniqueness: "server"},
			{Name: "name", Type: "complex", MultiValued: false, Description: "The components of the user's real name", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "displayName", Type: "string", MultiValued: false, Description: "The name of the User", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "emails", Type: "complex", MultiValued: true, Description: "Email addresses for the user", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "active", Type: "boolean", MultiValued: false, Description: "A Boolean value indicating the User's administrative status", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "externalId", Type: "string", MultiValued: false, Description: "An identifier for the resource as defined by the provisioning client", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "groups", Type: "complex", MultiValued: true, Description: "A list of groups to which the user belongs", Required: false, Mutability: "readOnly", Returned: "default", Uniqueness: "none"},
			{Name: "title", Type: "string", MultiValued: false, Description: "The user's title", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
		},
		Meta: &models.SCIMMeta{ResourceType: "Schema", Created: time.Now(), LastModified: time.Now(), Location: baseURL + "/Schemas/" + models.SCIMSchemaUser},
	}

	groupSchema := models.SCIMSchemaDefinition{
		Schemas:     []string{models.SCIMSchemaSchema},
		ID:          models.SCIMSchemaGroup,
		Name:        "Group",
		Description: "Group",
		Attributes: []models.SCIMSchemaAttribute{
			{Name: "displayName", Type: "string", MultiValued: false, Description: "A human-readable name for the Group", Required: true, Mutability: "readWrite", Returned: "default", Uniqueness: "server"},
			{Name: "members", Type: "complex", MultiValued: true, Description: "A list of members of the Group", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
			{Name: "externalId", Type: "string", MultiValued: false, Description: "An identifier for the resource", Required: false, Mutability: "readWrite", Returned: "default", Uniqueness: "none"},
		},
		Meta: &models.SCIMMeta{ResourceType: "Schema", Created: time.Now(), LastModified: time.Now(), Location: baseURL + "/Schemas/" + models.SCIMSchemaGroup},
	}

	resources := []interface{}{userSchema, groupSchema}
	c.JSON(http.StatusOK, models.NewSCIMListResponse(resources, len(resources), 1, len(resources)))
}

// GetResourceTypes returns the SCIM ResourceType definitions
func (sc *SCIMController) GetResourceTypes(c *gin.Context) {
	shared.SCIMContentType(c)
	baseURL := scimBaseURL(c)

	resourceTypes := []interface{}{
		models.SCIMResourceTypeDefinition{
			Schemas:     []string{models.SCIMSchemaResourceType},
			ID:          "User",
			Name:        "User",
			Description: "User Account",
			Endpoint:    "/Users",
			Schema:      models.SCIMSchemaUser,
			Meta:        &models.SCIMMeta{ResourceType: "ResourceType", Created: time.Now(), LastModified: time.Now(), Location: baseURL + "/ResourceTypes/User"},
		},
		models.SCIMResourceTypeDefinition{
			Schemas:     []string{models.SCIMSchemaResourceType},
			ID:          "Group",
			Name:        "Group",
			Description: "Group",
			Endpoint:    "/Groups",
			Schema:      models.SCIMSchemaGroup,
			Meta:        &models.SCIMMeta{ResourceType: "ResourceType", Created: time.Now(), LastModified: time.Now(), Location: baseURL + "/ResourceTypes/Group"},
		},
	}

	c.JSON(http.StatusOK, models.NewSCIMListResponse(resourceTypes, len(resourceTypes), 1, len(resourceTypes)))
}

// ──────────────────────────────────────────────
// User Endpoints (End-User / Tenant DB)
// ──────────────────────────────────────────────

// ListUsers handles GET /scim/v2/:client_id/:project_id/Users
func (sc *SCIMController) ListUsers(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, _, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
		return
	}

	// Parse pagination
	startIndex, _ := strconv.Atoi(c.DefaultQuery("startIndex", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "100"))
	if startIndex < 1 {
		startIndex = 1
	}
	if count < 1 || count > 200 {
		count = 100
	}

	filter := c.Query("filter")
	baseURL := scimBaseURL(c)

	// Build query — scoped by tenant_id AND client_id
	query := tenantDB.Model(&models.ExtendedUser{}).Where("tenant_id = ? AND client_id = ?", tenantID, clientUUID)

	// When no filter: return only SCIM-provisioned users
	// When filter is provided (e.g., Okta searching by userName): search ALL users for duplicate detection
	if filter == "" {
		query = query.Where("sync_source = ?", "scim")
	}
	query = applyUserFilter(query, filter)

	// Count total
	var totalResults int64
	query.Count(&totalResults)

	// Fetch page
	var users []models.ExtendedUser
	query.Offset(startIndex - 1).Limit(count).Order("created_at ASC").Find(&users)

	resources := make([]interface{}, len(users))
	for i, user := range users {
		resources[i] = models.UserToSCIMUser(user, baseURL)
	}

	c.JSON(http.StatusOK, models.NewSCIMListResponse(resources, int(totalResults), startIndex, len(resources)))
}

// GetUser handles GET /scim/v2/:client_id/:project_id/Users/:id
func (sc *SCIMController) GetUser(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, _, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	var user models.ExtendedUser
	if err := tenantDB.Where("id = ? AND tenant_id = ? AND client_id = ?", userUUID, tenantID, clientUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	c.JSON(http.StatusOK, models.UserToSCIMUser(user, scimBaseURL(c)))
}

// CreateUser handles POST /scim/v2/:client_id/:project_id/Users
func (sc *SCIMController) CreateUser(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, projectUUID, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
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

	// Check if user already exists — scoped by client_id (matching AD/Entra pattern)
	var existing models.ExtendedUser
	if err := tenantDB.Where("(email = ? OR external_id = ?) AND client_id = ?", strings.ToLower(email), input.ExternalID, clientUUID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, models.NewSCIMError("409", "User with this email already exists", "uniqueness"))
		return
	}

	domainSuffix := "app.authsec.ai"
	if config.AppConfig != nil && config.AppConfig.TenantDomainSuffix != "" {
		domainSuffix = config.AppConfig.TenantDomainSuffix
	}

	userName := input.UserName
	now := time.Now()

	providerData, _ := json.Marshal(map[string]interface{}{
		"scim_external_id": input.ExternalID,
		"scim_user_name":   input.UserName,
		"title":            input.Title,
		"department":       input.Department,
		"sync_timestamp":   now.Unix(),
	})

	// Generate a temporary password for the SCIM-provisioned user
	tempPassword, err := utils.GenerateTemporaryPassword()
	if err != nil {
		log.Printf("SCIM: Failed to generate temporary password: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to generate password", ""))
		return
	}

	hashedPassword, err := utils.HashPassword(tempPassword)
	if err != nil {
		log.Printf("SCIM: Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to hash password", ""))
		return
	}

	newUser := models.ExtendedUser{
		User: sharedmodels.User{
			ID:           uuid.New(),
			ClientID:     clientUUID,
			TenantID:     tenantUUID,
			ProjectID:    projectUUID,
			Name:         input.GetDisplayName(),
			Username:     &userName,
			Email:        strings.ToLower(email),
			Provider:     "scim",
			ProviderID:   input.UserName,
			Active:       input.GetActive(),
			ProviderData: datatypes.JSON(providerData),
			TenantDomain: domainSuffix,
			MFAEnabled:   false,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		ExternalID:   strPtrOrNil(input.ExternalID),
		SyncSource:   shared.StringPtr("scim"),
		LastSyncAt:   &now,
		IsSyncedUser: true,
	}
	newUser.PasswordHash = hashedPassword

	if err := tenantDB.Create(&newUser).Error; err != nil {
		log.Printf("SCIM: Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to create user", ""))
		return
	}

	log.Printf("SCIM: Created user %s (tenant: %s)", email, tenantID)

	// Send temporary password email to the user (async — don't block SCIM response)
	go func() {
		if err := utils.SendTemporaryPasswordEmail(strings.ToLower(email), tempPassword); err != nil {
			log.Printf("SCIM: Failed to send temporary password email to %s: %v", email, err)
		} else {
			log.Printf("SCIM: Sent temporary password email to %s", email)
		}
	}()

	middlewares.Audit(c, "scim", tenantID, "create_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":     newUser.ID.String(),
			"email":       email,
			"external_id": input.ExternalID,
		},
	})

	c.JSON(http.StatusCreated, models.UserToSCIMUser(newUser, scimBaseURL(c)))
}

// ReplaceUser handles PUT /scim/v2/:client_id/:project_id/Users/:id
func (sc *SCIMController) ReplaceUser(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, _, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	var input models.SCIMCreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid request: "+err.Error(), "invalidValue"))
		return
	}

	var user models.ExtendedUser
	if err := tenantDB.Where("id = ? AND tenant_id = ? AND client_id = ?", userUUID, tenantID, clientUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	email := input.GetPrimaryEmail()
	now := time.Now()
	userName := input.UserName

	providerData, _ := json.Marshal(map[string]interface{}{
		"scim_external_id": input.ExternalID,
		"scim_user_name":   input.UserName,
		"title":            input.Title,
		"department":       input.Department,
		"sync_timestamp":   now.Unix(),
	})

	updates := map[string]interface{}{
		"name":          input.GetDisplayName(),
		"username":      userName,
		"email":         strings.ToLower(email),
		"active":        input.GetActive(),
		"external_id":   input.ExternalID,
		"provider_data": datatypes.JSON(providerData),
		"last_sync_at":  &now,
		"updated_at":    now,
	}

	if err := tenantDB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to update user", ""))
		return
	}

	// Re-fetch updated user
	tenantDB.Where("id = ?", userUUID).First(&user)

	middlewares.Audit(c, "scim", tenantID, "replace_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id": userID,
			"email":   email,
		},
	})

	c.JSON(http.StatusOK, models.UserToSCIMUser(user, scimBaseURL(c)))
}

// PatchUser handles PATCH /scim/v2/:client_id/:project_id/Users/:id
func (sc *SCIMController) PatchUser(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, _, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	var patchReq models.SCIMPatchRequest
	if err := c.ShouldBindJSON(&patchReq); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid PATCH request: "+err.Error(), "invalidValue"))
		return
	}

	var user models.ExtendedUser
	if err := tenantDB.Where("id = ? AND tenant_id = ? AND client_id = ?", userUUID, tenantID, clientUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	updates := map[string]interface{}{
		"updated_at":   time.Now(),
		"last_sync_at": time.Now(),
	}

	for _, op := range patchReq.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			applyUserPatchReplace(op, updates)
		case "add":
			applyUserPatchReplace(op, updates) // add and replace behave the same for single-valued attrs
		case "remove":
			if strings.EqualFold(op.Path, "active") {
				updates["active"] = false
			}
		}
	}

	if err := tenantDB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to patch user", ""))
		return
	}

	// Re-fetch
	tenantDB.Where("id = ?", userUUID).First(&user)

	middlewares.Audit(c, "scim", tenantID, "patch_user", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":    userID,
			"operations": len(patchReq.Operations),
		},
	})

	c.JSON(http.StatusOK, models.UserToSCIMUser(user, scimBaseURL(c)))
}

// DeleteUser handles DELETE /scim/v2/:client_id/:project_id/Users/:id
func (sc *SCIMController) DeleteUser(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	clientUUID, _, err := getClientAndProjectID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", err.Error(), "invalidValue"))
		return
	}

	userID := c.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid user ID format", "invalidValue"))
		return
	}

	result := tenantDB.Where("id = ? AND tenant_id = ? AND client_id = ?", userUUID, tenantID, clientUUID).Delete(&models.ExtendedUser{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to delete user", ""))
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "User not found", ""))
		return
	}

	log.Printf("SCIM: Deleted user %s (tenant: %s)", userID, tenantID)

	middlewares.Audit(c, "scim", tenantID, "delete_user", &middlewares.AuditChanges{
		Before: map[string]interface{}{"user_id": userID},
	})

	c.Status(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Group Endpoints (End-User / Tenant DB)
// ──────────────────────────────────────────────

// ListGroups handles GET /scim/v2/:client_id/:project_id/Groups
func (sc *SCIMController) ListGroups(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
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
	baseURL := scimBaseURL(c)
	tenantUUID, _ := uuid.Parse(tenantID)

	query := tenantDB.Model(&models.TenantGroup{}).Where("tenant_id = ?", tenantUUID)
	query = applyGroupFilter(query, filter)

	var totalResults int64
	query.Count(&totalResults)

	var groups []models.TenantGroup
	query.Offset(startIndex - 1).Limit(count).Order("created_at ASC").Find(&groups)

	resources := make([]interface{}, len(groups))
	for i, group := range groups {
		members := sc.getGroupMembers(tenantDB, group.ID, tenantUUID)
		resources[i] = models.TenantGroupToSCIMGroup(group, members, baseURL)
	}

	c.JSON(http.StatusOK, models.NewSCIMListResponse(resources, int(totalResults), startIndex, len(resources)))
}

// GetGroup handles GET /scim/v2/:client_id/:project_id/Groups/:id
func (sc *SCIMController) GetGroup(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	groupID := c.Param("id")
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid group ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	var group models.TenantGroup
	if err := tenantDB.Where("id = ? AND tenant_id = ?", groupUUID, tenantUUID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "Group not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	members := sc.getGroupMembers(tenantDB, group.ID, tenantUUID)
	c.JSON(http.StatusOK, models.TenantGroupToSCIMGroup(group, members, scimBaseURL(c)))
}

// CreateGroup handles POST /scim/v2/:client_id/:project_id/Groups
func (sc *SCIMController) CreateGroup(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	var input models.SCIMCreateGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid request: "+err.Error(), "invalidValue"))
		return
	}

	if input.DisplayName == "" {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "displayName is required", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	// Check for existing group with same name
	var existing models.TenantGroup
	if err := tenantDB.Where("name = ? AND tenant_id = ?", input.DisplayName, tenantUUID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, models.NewSCIMError("409", "Group with this name already exists", "uniqueness"))
		return
	}

	newGroup := models.TenantGroup{
		ID:        uuid.New(),
		Name:      input.DisplayName,
		TenantID:  tenantUUID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tenantDB.Create(&newGroup).Error; err != nil {
		log.Printf("SCIM: Failed to create group: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to create group", ""))
		return
	}

	// Add members if provided
	for _, member := range input.Members {
		memberUUID, err := uuid.Parse(member.Value)
		if err != nil {
			continue
		}
		tenantDB.Exec(
			"INSERT INTO user_groups (user_id, group_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			memberUUID, newGroup.ID, tenantUUID,
		)
	}

	log.Printf("SCIM: Created group %s (tenant: %s)", input.DisplayName, tenantID)

	middlewares.Audit(c, "scim", tenantID, "create_group", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"group_id":     newGroup.ID.String(),
			"display_name": input.DisplayName,
			"member_count": len(input.Members),
		},
	})

	members := sc.getGroupMembers(tenantDB, newGroup.ID, tenantUUID)
	c.JSON(http.StatusCreated, models.TenantGroupToSCIMGroup(newGroup, members, scimBaseURL(c)))
}

// ReplaceGroup handles PUT /scim/v2/:client_id/:project_id/Groups/:id
func (sc *SCIMController) ReplaceGroup(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	groupID := c.Param("id")
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid group ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	var group models.TenantGroup
	if err := tenantDB.Where("id = ? AND tenant_id = ?", groupUUID, tenantUUID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "Group not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	var input models.SCIMCreateGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid request: "+err.Error(), "invalidValue"))
		return
	}

	// Update group name
	tenantDB.Model(&group).Updates(map[string]interface{}{
		"name":       input.DisplayName,
		"updated_at": time.Now(),
	})

	// Replace members: remove all, then add new
	tenantDB.Exec("DELETE FROM user_groups WHERE group_id = $1 AND tenant_id = $2", groupUUID, tenantUUID)
	for _, member := range input.Members {
		memberUUID, err := uuid.Parse(member.Value)
		if err != nil {
			continue
		}
		tenantDB.Exec(
			"INSERT INTO user_groups (user_id, group_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			memberUUID, groupUUID, tenantUUID,
		)
	}

	middlewares.Audit(c, "scim", tenantID, "replace_group", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"group_id":     groupID,
			"display_name": input.DisplayName,
		},
	})

	// Re-fetch
	tenantDB.Where("id = ?", groupUUID).First(&group)
	members := sc.getGroupMembers(tenantDB, groupUUID, tenantUUID)
	c.JSON(http.StatusOK, models.TenantGroupToSCIMGroup(group, members, scimBaseURL(c)))
}

// PatchGroup handles PATCH /scim/v2/:client_id/:project_id/Groups/:id
func (sc *SCIMController) PatchGroup(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	groupID := c.Param("id")
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid group ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	var group models.TenantGroup
	if err := tenantDB.Where("id = ? AND tenant_id = ?", groupUUID, tenantUUID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.NewSCIMError("404", "Group not found", ""))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Internal server error", ""))
		return
	}

	var patchReq models.SCIMPatchRequest
	if err := c.ShouldBindJSON(&patchReq); err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid PATCH request: "+err.Error(), "invalidValue"))
		return
	}

	for _, op := range patchReq.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			if strings.EqualFold(op.Path, "displayName") {
				if name, ok := op.Value.(string); ok {
					tenantDB.Model(&group).Update("name", name)
				}
			}
			if op.Path == "" || strings.EqualFold(op.Path, "members") {
				// Full member replacement
				sc.replaceGroupMembers(tenantDB, groupUUID, tenantUUID, op.Value)
			}
		case "add":
			if strings.EqualFold(op.Path, "members") {
				sc.addGroupMembers(tenantDB, groupUUID, tenantUUID, op.Value)
			}
		case "remove":
			if strings.EqualFold(op.Path, "members") {
				sc.removeGroupMembers(tenantDB, groupUUID, tenantUUID, op.Value)
			} else if strings.HasPrefix(strings.ToLower(op.Path), "members[value eq") {
				// Handle format: members[value eq "user-id"]
				memberID := extractFilterValue(op.Path)
				if memberUUID, err := uuid.Parse(memberID); err == nil {
					tenantDB.Exec("DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2 AND tenant_id = $3",
						memberUUID, groupUUID, tenantUUID)
				}
			}
		}
	}

	tenantDB.Model(&group).Update("updated_at", time.Now())

	middlewares.Audit(c, "scim", tenantID, "patch_group", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"group_id":   groupID,
			"operations": len(patchReq.Operations),
		},
	})

	// Re-fetch
	tenantDB.Where("id = ?", groupUUID).First(&group)
	members := sc.getGroupMembers(tenantDB, groupUUID, tenantUUID)
	c.JSON(http.StatusOK, models.TenantGroupToSCIMGroup(group, members, scimBaseURL(c)))
}

// DeleteGroup handles DELETE /scim/v2/:client_id/:project_id/Groups/:id
func (sc *SCIMController) DeleteGroup(c *gin.Context) {
	shared.SCIMContentType(c)
	tenantDB, tenantID, err := getTenantDB(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewSCIMError("401", err.Error(), ""))
		return
	}

	groupID := c.Param("id")
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewSCIMError("400", "Invalid group ID format", "invalidValue"))
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	// Remove member associations first
	tenantDB.Exec("DELETE FROM user_groups WHERE group_id = $1 AND tenant_id = $2", groupUUID, tenantUUID)

	result := tenantDB.Where("id = ? AND tenant_id = ?", groupUUID, tenantUUID).Delete(&models.TenantGroup{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.NewSCIMError("500", "Failed to delete group", ""))
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, models.NewSCIMError("404", "Group not found", ""))
		return
	}

	log.Printf("SCIM: Deleted group %s (tenant: %s)", groupID, tenantID)

	middlewares.Audit(c, "scim", tenantID, "delete_group", &middlewares.AuditChanges{
		Before: map[string]interface{}{"group_id": groupID},
	})

	c.Status(http.StatusNoContent)
}

// ──────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────

// strPtrOrNil returns a pointer to s if non-empty, else nil
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// applyUserFilter applies a SCIM filter to a GORM query for users
func applyUserFilter(query *gorm.DB, filter string) *gorm.DB {
	if filter == "" {
		return query
	}

	// Parse simple "attribute eq value" filters
	// Examples: userName eq "john@example.com", externalId eq "abc123", emails.value eq "john@example.com"
	filter = strings.TrimSpace(filter)

	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 {
		return query
	}

	attr := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	switch strings.ToLower(attr) {
	case "username":
		return query.Where("LOWER(email) = LOWER(?) OR LOWER(username) = LOWER(?)", value, value)
	case "externalid":
		return query.Where("external_id = ?", value)
	case "emails.value":
		return query.Where("LOWER(email) = LOWER(?)", value)
	case "displayname":
		return query.Where("LOWER(name) = LOWER(?)", value)
	case "active":
		boolVal := strings.ToLower(value) == "true"
		return query.Where("active = ?", boolVal)
	default:
		return query
	}
}

// applyGroupFilter applies a SCIM filter to a GORM query for groups
func applyGroupFilter(query *gorm.DB, filter string) *gorm.DB {
	if filter == "" {
		return query
	}

	filter = strings.TrimSpace(filter)
	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 {
		return query
	}

	attr := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	switch strings.ToLower(attr) {
	case "displayname":
		return query.Where("LOWER(name) = LOWER(?)", value)
	case "externalid":
		// groups don't have external_id in current schema, but handle gracefully
		return query.Where("1=0")
	default:
		return query
	}
}

// applyUserPatchReplace processes a PATCH replace/add operation for a user
func applyUserPatchReplace(op models.SCIMPatchOp, updates map[string]interface{}) {
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
		// We only store a single "name" field; partial name updates are best-effort
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
		// Handle email replacement
		if emails, ok := op.Value.([]interface{}); ok && len(emails) > 0 {
			if emailObj, ok := emails[0].(map[string]interface{}); ok {
				if val, ok := emailObj["value"].(string); ok {
					updates["email"] = strings.ToLower(val)
				}
			}
		}
	case "":
		// No path means replace the whole object attributes
		if valueMap, ok := op.Value.(map[string]interface{}); ok {
			for k, v := range valueMap {
				innerOp := models.SCIMPatchOp{Path: k, Value: v}
				applyUserPatchReplace(innerOp, updates)
			}
		}
	}
}

// getGroupMembers retrieves member references for a group
func (sc *SCIMController) getGroupMembers(tenantDB *gorm.DB, groupID, tenantID uuid.UUID) []models.SCIMMemberRef {
	var userGroups []models.UserGroup
	tenantDB.Where("group_id = ? AND tenant_id = ?", groupID, tenantID).Find(&userGroups)

	members := make([]models.SCIMMemberRef, 0, len(userGroups))
	for _, ug := range userGroups {
		var user models.ExtendedUser
		if err := tenantDB.Select("id, name, email").Where("id = ?", ug.UserID).First(&user).Error; err == nil {
			display := user.Name
			if display == "" {
				display = user.Email
			}
			members = append(members, models.SCIMMemberRef{
				Value:   ug.UserID.String(),
				Display: display,
			})
		}
	}
	return members
}

// replaceGroupMembers replaces all members of a group
func (sc *SCIMController) replaceGroupMembers(tenantDB *gorm.DB, groupID, tenantID uuid.UUID, value interface{}) {
	tenantDB.Exec("DELETE FROM user_groups WHERE group_id = $1 AND tenant_id = $2", groupID, tenantID)
	sc.addGroupMembers(tenantDB, groupID, tenantID, value)
}

// addGroupMembers adds members to a group from a SCIM PATCH value
func (sc *SCIMController) addGroupMembers(tenantDB *gorm.DB, groupID, tenantID uuid.UUID, value interface{}) {
	members := parseMemberRefs(value)
	for _, member := range members {
		memberUUID, err := uuid.Parse(member.Value)
		if err != nil {
			continue
		}
		tenantDB.Exec(
			"INSERT INTO user_groups (user_id, group_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			memberUUID, groupID, tenantID,
		)
	}
}

// removeGroupMembers removes members from a group
func (sc *SCIMController) removeGroupMembers(tenantDB *gorm.DB, groupID, tenantID uuid.UUID, value interface{}) {
	members := parseMemberRefs(value)
	for _, member := range members {
		memberUUID, err := uuid.Parse(member.Value)
		if err != nil {
			continue
		}
		tenantDB.Exec(
			"DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2 AND tenant_id = $3",
			memberUUID, groupID, tenantID,
		)
	}
}

// parseMemberRefs parses SCIM member references from various input formats
func parseMemberRefs(value interface{}) []models.SCIMMemberRef {
	var members []models.SCIMMemberRef

	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				ref := models.SCIMMemberRef{}
				if val, ok := m["value"].(string); ok {
					ref.Value = val
				}
				if display, ok := m["display"].(string); ok {
					ref.Display = display
				}
				if ref.Value != "" {
					members = append(members, ref)
				}
			}
		}
	case map[string]interface{}:
		ref := models.SCIMMemberRef{}
		if val, ok := v["value"].(string); ok {
			ref.Value = val
		}
		if ref.Value != "" {
			members = append(members, ref)
		}
	}

	return members
}

// extractFilterValue extracts the value from a SCIM filter expression like: members[value eq "uuid"]
func extractFilterValue(path string) string {
	// Expected format: members[value eq "some-id"]
	start := strings.Index(path, "\"")
	end := strings.LastIndex(path, "\"")
	if start >= 0 && end > start {
		return path[start+1 : end]
	}
	return ""
}

// ──────────────────────────────────────────────
// SCIM Token Generation (Admin endpoint)
// ──────────────────────────────────────────────

// SCIMTokenRequest is the input for generating a SCIM Bearer token
type SCIMTokenRequest struct {
	ClientID  string `json:"client_id" binding:"required"`
	ProjectID string `json:"project_id" binding:"required"`
}

// SCIMTokenResponse is the response containing the SCIM Bearer token and base URL
type SCIMTokenResponse struct {
	Token    string `json:"token"`
	BaseURL  string `json:"base_url"`
	ExpireAt string `json:"expire_at"`
}

// GenerateSCIMToken generates a long-lived (365-day) Bearer token for SCIM connector setup.
// The admin copies this token + base URL into their IdP (Okta, Azure AD, OneLogin, etc.).
// POST /uflow/admin/scim/generate-token
func (sc *SCIMController) GenerateSCIMToken(c *gin.Context) {
	// Get admin's tenant_id from JWT (set by AuthMiddleware + ValidateTenantFromToken)
	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in token"})
		return
	}

	// Get admin's user_id and email from JWT context
	userIDStr := shared.ContextStringValue(c, "user_id")
	emailID := shared.ContextStringValue(c, "email_id")
	if emailID == "" {
		emailID = shared.ContextStringValue(c, "email")
	}

	var input SCIMTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id and project_id are required"})
		return
	}

	// Validate UUIDs
	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id format"})
		return
	}
	_, err = uuid.Parse(input.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id format"})
		return
	}

	// Parse or generate user ID for the token
	var userID uuid.UUID
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}
	if userID == uuid.Nil {
		userID = uuid.New()
	}

	// Generate 365-day token using the same token service (passes AuthMiddleware validation)
	scimToken, err := config.TokenService.GenerateEndUserToken(
		userID,
		tenantID,
		clientUUID.String(),
		emailID,
		[]string{"scim:read", "scim:write"},
		365*24*time.Hour,
	)
	if err != nil {
		log.Printf("SCIM: Failed to generate SCIM token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate SCIM token"})
		return
	}

	// Build SCIM base URL
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s/uflow/scim/v2/%s/%s", scheme, c.Request.Host, input.ClientID, input.ProjectID)
	expireAt := time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	log.Printf("SCIM: Generated SCIM token for tenant %s, client %s", tenantID, input.ClientID)

	middlewares.Audit(c, "scim", tenantID, "generate_scim_token", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"client_id":  input.ClientID,
			"project_id": input.ProjectID,
			"expire_at":  expireAt,
		},
	})

	c.JSON(http.StatusOK, SCIMTokenResponse{
		Token:    scimToken,
		BaseURL:  baseURL,
		ExpireAt: expireAt,
	})
}
