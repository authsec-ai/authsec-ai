package enduser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	hydra "github.com/ory/hydra-client-go/v2"
	"gorm.io/gorm"
)

var (
	tenantConnectionProvider = middlewares.GetConnectionDynamically
	timeNow                  = time.Now
)

type EndUserController struct{}

// RegisterEndUser godoc
// @Summary Register a new end user in tenant database
// @Description Registers a new end user in the specified tenant database with all default associations
// @Tags EndUser
// @Accept json
// @Produce json
// @Param register body object true "End user registration data"
// @Success 201 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/clients/register [post]
func (euc *EndUserController) RegisterClient(c *gin.Context) {
	var input models.RegisterClientsRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var tenant models.Tenant
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	if err := config.DB.Where("tenant_id = ?", input.TenantID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	// Parse UUIDs
	tenantID := input.TenantID
	projectID := input.ProjectID
	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}
	// Check if user with email already exists
	// var existingClient models.Client
	// if err := tenantDB.Where("email = ?", input.Email).First(&existingClient).Error; err == nil {
	// 	c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
	// 	return
	// }

	// Generate unique IDs
	clientID := uuid.New()

	// Create new user with all required data
	client := models.Client{
		ID:        clientID,
		ClientID:  clientID,
		TenantID:  uuid.MustParse(tenantID),
		ProjectID: uuid.MustParse(projectID),
		Name:      input.Name,
		Email:     shared.StringPtr(input.Email),
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start transaction for client creation and associations
	tx := tenantDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the user record
	if err := tx.Create(&client).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Assign default associations (scopes, roles, groups, resources)

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	// Save secret to vault (optional - based on your requirements)
	secretID, err := config.SaveSecretToVault(client.TenantID.String(), client.ProjectID.String(), client.ClientID.String())
	if err != nil {
		// Log the error but don't fail the registration
		fmt.Printf("Warning: failed to save secret to vault: %v\n", err)
	}

	// Register client with Hydra (optional - based on your requirements)
	if secretID != "" {
		if err := services.RegisterClientWithHydra(client.ClientID.String(), secretID, *client.Email, client.TenantID.String(), tenant.TenantDomain); err != nil {
			// Log the error but don't fail the registration
			fmt.Printf("Warning: failed to register client with Hydra: %v\n", err)
		}
	}

	// Audit log: Client registered
	middlewares.Audit(c, "client", client.ClientID.String(), "register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"client_id":  client.ClientID.String(),
			"tenant_id":  client.TenantID.String(),
			"project_id": client.ProjectID.String(),
			"name":       client.Name,
			"email":      *client.Email,
		},
	})

	// Return success response
	response := models.RegisterClientsResponse{
		ID:        client.ID.String(),
		ClientID:  client.ClientID.String(),
		TenantID:  client.TenantID.String(),
		ProjectID: client.ProjectID.String(),
		Name:      client.Name,
		SecretID:  secretID,
		Email:     *client.Email, // Dereference pointer
		Active:    client.Active,
		CreatedAt: client.CreatedAt,
		Message:   "Client registered successfully",
	}

	c.JSON(http.StatusCreated, response)
}

// GetEndUser godoc
// @Summary Get end user
// @Description Retrieves an end user by ID or by email (requires client_id) with all associations
// @Tags EndUser
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param user_id path string true "User ID"
// @Param client_id query string false "Client ID (required when using email identifier)"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/enduser/{tenant_id}/{user_id} [get]
type GetEndUsersFilter struct {
	TenantID string `json:"tenant_id" binding:"required" validate:"required"`
	Page     int    `json:"page,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Active   *bool  `json:"active,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func (euc *EndUserController) GetEndUser(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}
	userIdentifier := c.Param("user_id")

	lookupByID, userUUID, clientUUID, emailIdentifier, parseErr := resolveEndUserLookup(userIdentifier, c.Query("client_id"))
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	normalizedTenantID := tenantID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	normalizedTenantID = tenantUUID.String()

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &normalizedTenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Fetch user with all associations
	var user models.User
	if lookupByID {
		if err := tenantDB.Preload("Scopes").Preload("Roles").Preload("Groups").Preload("Resources").
			Where("id = ?", userUUID).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
			return
		}
	} else {
		if err := tenantDB.Preload("Scopes").Preload("Roles").Preload("Groups").Preload("Resources").
			Where("client_id = ? AND LOWER(email) = LOWER(?)", clientUUID, emailIdentifier).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
			return
		}
	}

	if user.TenantID != uuid.Nil && !strings.EqualFold(user.TenantID.String(), tenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "user does not belong to tenant"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func resolveEndUserLookup(identifier, clientIDParam string) (byID bool, userID uuid.UUID, clientID uuid.UUID, email string, err error) {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		err = fmt.Errorf("user identifier is required")
		return
	}

	if parsedID, parseErr := uuid.Parse(trimmedIdentifier); parseErr == nil {
		byID = true
		userID = parsedID
		return
	}

	trimmedClientID := strings.TrimSpace(clientIDParam)
	if trimmedClientID == "" {
		err = fmt.Errorf("client_id is required when using email identifier")
		return
	}

	clientUUID, parseErr := uuid.Parse(trimmedClientID)
	if parseErr != nil {
		err = fmt.Errorf("invalid client_id")
		return
	}

	clientID = clientUUID
	email = trimmedIdentifier
	return
}

// GetEndUsers godoc
// @Summary Get all end users for a tenant
// @Description Retrieves all end users for a specific tenant with pagination and filtering. Supports both GET (query parameters) and POST (JSON body) methods.
// @Tags EndUser
// @Accept json
// @Produce json
// @Param tenant_id query string true "Tenant ID (GET) or in body (POST)"
// @Param page query int false "Page number (default: 1) - GET method"
// @Param limit query int false "Items per page (default: 10, max: 100) - GET method"
// @Param active query bool false "Filter by active status - GET method"
// @Param client_id query string false "Filter by client ID - GET method"
// @Param email query string false "Filter by email - GET method"
// @Param name query string false "Filter by name - GET method"
// @Param provider query string false "Filter by provider - GET method"
// @Param input body GetEndUsersFilter false "End users filter and pagination parameters - POST method"
// @Success 200 {object} sharedmodels.PaginatedEndUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/enduser/list [get]
// @Router /uflow/user/enduser/list [post]
func (euc *EndUserController) GetEndUsers(c *gin.Context) {
	var filter GetEndUsersFilter

	// Handle different HTTP methods
	if c.Request.Method == "POST" {
		// POST: Bind from JSON body
		if err := c.ShouldBindJSON(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if c.Request.Method == "GET" {
		// GET: Bind from query parameters
		if err := c.ShouldBindQuery(&filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Set default values
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Validate client ID format if provided
	if filter.ClientID != "" {
		if _, err := uuid.Parse(filter.ClientID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id format"})
			return
		}
	}

	offset := (filter.Page - 1) * filter.Limit

	// Determine tenant identifier: prefer request filter, fall back to authenticated context
	tenantIdentifier := filter.TenantID
	if tenantIdentifier == "" {
		if tenantVal, exists := c.Get("tenant_id"); exists {
			if tenantStr, ok := tenantVal.(string); ok && tenantStr != "" {
				tenantIdentifier = tenantStr
			}
		}
	}
	if tenantIdentifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	filter.TenantID = tenantIdentifier

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIdentifier)
	if err != nil {
		log.Printf("GetEndUsers: failed to connect to tenant %s database: %v", tenantIdentifier, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Build query - no base tenant filter needed since we're in tenant-specific DB
	query := tenantDB.Model(&models.User{})

	// Apply filters
	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	} else {
		query = query.Where("active = ?", true)
	}

	if filter.ClientID != "" {
		query = query.Where("client_id = ?", filter.ClientID)
	}

	if filter.Email != "" {
		// Use ILIKE for case-insensitive partial matching (PostgreSQL)
		// Use LIKE for case-sensitive partial matching (other databases)
		query = query.Where("LOWER(email) LIKE LOWER(?)", "%"+filter.Email+"%")
	}

	if filter.Name != "" {
		// Case-insensitive partial matching for name
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+filter.Name+"%")
	}

	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}

	// Count total records with filters applied
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count users"})
		return
	}

	// Fetch users with pagination and all associations
	var users []models.User
	if err := query.Preload("Scopes").Preload("Roles").Preload("Groups").Preload("Resources").
		Order("created_at DESC").
		Offset(offset).Limit(filter.Limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	// Calculate total pages
	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	response := models.PaginatedEndUsersResponse{
		Users:      users,
		Total:      int(total),
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// GetTenantDatabases godoc
// @Summary Get tenant database metadata
// @Description Returns the tenant database mapping for the supplied tenant identifier
// @Tags End-User Management
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/enduser/databases [get]
func (euc *EndUserController) GetTenantDatabases(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
		return
	}

	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not available"})
		return
	}

	var tenant models.Tenant
	if err := config.DB.Where("tenant_id = ? OR id::text = ?", tenantID, tenantID).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		log.Printf("GetTenantDatabases: failed to load tenant %s: %v", tenantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query tenant"})
		return
	}

	info := gin.H{
		"tenant_id":           tenant.TenantID.String(),
		"tenant_db":           tenant.TenantDB,
		"status":              tenant.Status,
		"database_configured": tenant.TenantDB != "",
		"database_reachable":  false,
	}

	if tenant.TenantDB != "" {
		if _, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID); err == nil {
			info["database_reachable"] = true
		} else {
			log.Printf("GetTenantDatabases: tenant %s database %s unreachable: %v", tenantID, tenant.TenantDB, err)
		}
	}

	c.JSON(http.StatusOK, info)
}

// UpdateEndUserStatus godoc
// @Summary Update end user status
// @Description Updates the active status of an end user
// @Tags EndUser
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param user_id path string true "User ID"
// @Param input body object true "Status update data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/enduser/{tenant_id}/{user_id}/status [put]
func (euc *EndUserController) UpdateEndUserStatus(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}
	userID := c.Param("user_id")

	var input models.UpdateEndUserStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Update user status
	result := tenantDB.Model(&models.User{}).Where("id = ? AND tenant_id = ?", userUUID, tenantID).
		Updates(map[string]interface{}{
			"active":     input.Active,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user status"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Audit log: End user status updated
	middlewares.Audit(c, "enduser", userID, "update_status", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"active":    input.Active,
		},
	})

	response := models.UpdateEndUserStatusResponse{
		Message:   "User status updated successfully",
		Active:    input.Active,
		UpdatedAt: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateUser godoc
// @Summary Update user profile
// @Description Updates user profile information in tenant database
// @Tags EndUser
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param user_id path string true "User ID"
// @Param input body object true "User update data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/enduser/{tenant_id}/{user_id} [put]
func (euc *EndUserController) UpdateUser(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}
	userID := c.Param("user_id")

	var input models.UpdateUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Prepare update data
	updateData := make(map[string]interface{})
	updateData["updated_at"] = time.Now()

	if input.Name != nil {
		updateData["name"] = *input.Name
	}
	if input.Username != nil {
		updateData["username"] = *input.Username
	}
	if input.Email != nil {
		updateData["email"] = *input.Email
	}
	if input.AvatarURL != nil {
		updateData["avatar_url"] = *input.AvatarURL
	}
	if input.TenantDomain != nil {
		updateData["tenant_domain"] = *input.TenantDomain
	}

	// Update user
	result := tenantDB.Model(&models.User{}).Where("id = ? AND tenant_id = ?", userUUID, tenantID).
		Updates(updateData)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Fetch updated user
	var updatedUser models.User
	if err := tenantDB.Where("id = ? AND tenant_id = ?", userUUID, tenantID).First(&updatedUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated user"})
		return
	}

	// Audit log: End user profile updated
	middlewares.Audit(c, "enduser", userID, "update", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"updates":   updateData,
		},
	})

	response := map[string]interface{}{
		"message": "User updated successfully",
		"user":    updatedUser,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteEndUser godoc

// @Summary Delete end user

// @Description Soft deletes an end user from tenant database

// @Tags EndUser

// @Accept json

// @Produce json

// @Param tenant_id path string true "Tenant ID"

// @Param user_id path string true "User ID"

// @Success 200 {object} map[string]string

// @Failure 400 {object} map[string]string

// @Failure 404 {object} map[string]string

// @Failure 500 {object} map[string]string

// @Router /uflow/user/enduser/{tenant_id}/{user_id} [delete]

// @Router /uflow/user/enduser/delete [post]

func (euc *EndUserController) DeleteEndUser(c *gin.Context) {

	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)

	if !ok {

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})

		return

	}

	userID := c.Param("user_id")

	jsonData := make(map[string]string)

	if userID == "" {

		if err := c.ShouldBindJSON(&jsonData); err != nil {

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format: " + err.Error()})

			return

		}

		if userID == "" {

			userID = jsonData["user_id"]

		}

	}

	if userID == "" {

		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})

		return

	}

	userInfo := middlewares.GetUserInfo(c)

	if userInfo == nil || strings.TrimSpace(userInfo.TenantID) == "" {

		c.JSON(http.StatusForbidden, gin.H{"error": "tenant scope is required"})

		return

	}

	if !strings.EqualFold(strings.TrimSpace(userInfo.TenantID), tenantID) {

		c.JSON(http.StatusForbidden, gin.H{"error": "cross-tenant deletion is not allowed"})

		return

	}

	if config.DB == nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})

		return

	}

	tenantUUID, err := uuid.Parse(tenantID)

	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})

		return

	}

	normalizedTenantID := tenantUUID.String()

	tenantDB, err := tenantConnectionProvider(config.DB, nil, &normalizedTenantID)

	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})

		return

	}

	userUUID, err := uuid.Parse(userID)

	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})

		return

	}

	if others, err := countOtherActiveEndUsers(tenantDB, tenantUUID, userUUID); err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify active users"})

		return

	} else if others == 0 {

		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate the last active user in this tenant"})

		return

	}

	rowsAffected, err := updateUserActiveStatus(tenantDB, tenantUUID, userUUID, false)

	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})

		return

	}

	// Check if a user was actually found and disabled.

	if rowsAffected == 0 {

		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})

		return

	}

	// Audit log: End user deleted (soft delete)

	middlewares.Audit(c, "enduser", userID, "delete", &middlewares.AuditChanges{

		Before: map[string]interface{}{

			"user_id": userID,

			"tenant_id": tenantID,

			"active": true,
		},

		After: map[string]interface{}{

			"active": false,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})

}

// DeleteUserAllRequest is the request body for hard delete

type DeleteUserAllRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`

	UserID string `json:"user_id" binding:"required"`
}

// DeleteUserAll godoc

// @Summary Hard delete end user and all related data

// @Description Permanently deletes an end user and all associated data from the tenant database. This includes role_bindings, totp_secrets, backup_codes, webauthn_credentials, ciba_push_devices, ciba_auth_requests, etc. Cannot delete the last active user.

// @Tags EndUser

// @Accept json

// @Produce json

// @Security BearerAuth

// @Param input body DeleteUserAllRequest true "User delete payload"

// @Success 200 {object} map[string]interface{} "User and all related data deleted successfully"

// @Failure 400 {object} map[string]string "Invalid request or cannot delete last user"

// @Failure 403 {object} map[string]string "Cross-tenant operation not allowed"

// @Failure 404 {object} map[string]string "User not found"

// @Failure 500 {object} map[string]string "Internal server error"

// @Router /uflow/user/enduser/delete_all [post]

func (euc *EndUserController) DeleteUserAll(c *gin.Context) {

	var req DeleteUserAllRequest

	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})

		return

	}

	tenantID := strings.TrimSpace(req.TenantID)

	userID := strings.TrimSpace(req.UserID)

	if tenantID == "" || userID == "" {

		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})

		return

	}

	// Verify caller's tenant matches target tenant

	userInfo := middlewares.GetUserInfo(c)

	if userInfo == nil || strings.TrimSpace(userInfo.TenantID) == "" {

		c.JSON(http.StatusForbidden, gin.H{"error": "tenant scope is required"})

		return

	}

	if !strings.EqualFold(strings.TrimSpace(userInfo.TenantID), tenantID) {

		c.JSON(http.StatusForbidden, gin.H{"error": "cross-tenant deletion is not allowed"})

		return

	}

	tenantUUID, err := uuid.Parse(tenantID)

	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})

		return

	}

	userUUID, err := uuid.Parse(userID)

	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})

		return

	}

	if config.DB == nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})

		return

	}

	tenantIDStr := tenantUUID.String()

	tenantDB, err := tenantConnectionProvider(config.DB, nil, &tenantIDStr)

	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})

		return

	}

	// Use fresh session to avoid stale transaction states

	freshDB := tenantDB.Session(&gorm.Session{NewDB: true})

	// Verify user exists

	var user models.User

	if err := freshDB.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {

			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})

			return

		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user: " + err.Error()})

		return

	}

	// Check if this is the last active user

	others, err := countOtherActiveEndUsers(freshDB, tenantUUID, userUUID)

	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify active users"})

		return

	}

	if others == 0 {

		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last active user in this tenant"})

		return

	}

	log.Printf("INFO: Hard deleting user %s and all related data for tenant %s", userUUID, tenantUUID)

	// Delete all related data in a transaction

	deletedCounts := make(map[string]int64)

	err = freshDB.Transaction(func(tx *gorm.DB) error {

		// 1. Delete role_bindings

		result := tx.Where("user_id = ? AND tenant_id = ?", userUUID, tenantUUID).Delete(&models.RoleBinding{})

		if result.Error != nil {

			return fmt.Errorf("failed to delete role_bindings: %w", result.Error)

		}

		deletedCounts["role_bindings"] = result.RowsAffected

		// 2. Delete totp_secrets (MFA devices)

		result = tx.Exec("DELETE FROM totp_secrets WHERE user_id = ? AND tenant_id = ?", userUUID, tenantUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete totp_secrets: %w", result.Error)

		}

		deletedCounts["totp_secrets"] = result.RowsAffected

		// 3. Delete backup_codes

		result = tx.Exec("DELETE FROM backup_codes WHERE user_id = ? AND tenant_id = ?", userUUID, tenantUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete backup_codes: %w", result.Error)

		}

		deletedCounts["backup_codes"] = result.RowsAffected

		// 4. Delete webauthn_credentials

		result = tx.Exec("DELETE FROM webauthn_credentials WHERE user_id = ? AND tenant_id = ?", userUUID, tenantUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete webauthn_credentials: %w", result.Error)

		}

		deletedCounts["webauthn_credentials"] = result.RowsAffected

		// 5. Delete ciba_push_devices

		result = tx.Exec("DELETE FROM ciba_push_devices WHERE user_id = ? AND tenant_id = ?", userUUID, tenantUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete ciba_push_devices: %w", result.Error)

		}

		deletedCounts["ciba_push_devices"] = result.RowsAffected

		// 6. Delete ciba_auth_requests

		result = tx.Exec("DELETE FROM ciba_auth_requests WHERE user_id = ? AND tenant_id = ?", userUUID, tenantUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete ciba_auth_requests: %w", result.Error)

		}

		deletedCounts["ciba_auth_requests"] = result.RowsAffected

		// 7. Delete voice_identity_links

		result = tx.Exec("DELETE FROM voice_identity_links WHERE user_id = ?", userUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete voice_identity_links: %w", result.Error)

		}

		deletedCounts["voice_identity_links"] = result.RowsAffected

		// 8. Delete user_groups

		result = tx.Exec("DELETE FROM user_groups WHERE user_id = ?", userUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete user_groups: %w", result.Error)

		}

		deletedCounts["user_groups"] = result.RowsAffected

		// 9. Delete refresh_tokens

		result = tx.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", userUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete refresh_tokens: %w", result.Error)

		}

		deletedCounts["refresh_tokens"] = result.RowsAffected

		// 10. Delete sessions

		result = tx.Exec("DELETE FROM sessions WHERE user_id = ?", userUUID)

		if result.Error != nil {

			return fmt.Errorf("failed to delete sessions: %w", result.Error)

		}

		deletedCounts["sessions"] = result.RowsAffected

		// 11. Finally, delete the user

		result = tx.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).Delete(&models.User{})

		if result.Error != nil {

			return fmt.Errorf("failed to delete user: %w", result.Error)

		}

		deletedCounts["users"] = result.RowsAffected

		return nil

	})

	if err != nil {

		log.Printf("ERROR: Failed to hard delete user %s: %v", userUUID, err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user: " + err.Error()})

		return

	}

	log.Printf("INFO: Successfully hard deleted user %s with counts: %+v", userUUID, deletedCounts)

	// Audit log: End user hard deleted

	middlewares.Audit(c, "enduser", userID, "delete_all", &middlewares.AuditChanges{

		Before: map[string]interface{}{

			"user_id": userID,

			"tenant_id": tenantID,

			"email": user.Email,

			"username": user.Username,
		},

		After: map[string]interface{}{

			"deleted": true,

			"deleted_counts": deletedCounts,
		},
	})

	c.JSON(http.StatusOK, gin.H{

		"message": "User and all related data deleted successfully",

		"user_id": userID,

		"deleted_counts": deletedCounts,
	})

}

type toggleEndUserActiveRequest struct {
	TenantID string        `json:"tenant_id" binding:"required"`
	UserID   string        `json:"user_id" binding:"required"`
	Active   *shared.FlexibleBool `json:"active" binding:"required"`
}

func countOtherActiveEndUsers(db *gorm.DB, tenantID uuid.UUID, excludeUser uuid.UUID) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("tenant database connection not available")
	}

	var count int64
	if err := db.Model(&models.User{}).
		Where("tenant_id = ? AND id <> ? AND active = ?", tenantID, excludeUser, true).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (euc *EndUserController) ActiveOrDeactiveEndUser(c *gin.Context) {
	var req toggleEndUserActiveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	tenantID := strings.TrimSpace(req.TenantID)
	userID := strings.TrimSpace(req.UserID)
	if tenantID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})
		return
	}

	if req.Active == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active field is required"})
		return
	}

	active := req.Active.Bool()

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	tenantIDStr := tenantUUID.String()

	tenantDB, err := tenantConnectionProvider(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	if !active {
		others, err := countOtherActiveEndUsers(tenantDB, tenantUUID, userUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify active users"})
			return
		}
		if others == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate the last active user in this tenant"})
			return
		}
	}

	rowsAffected, err := updateUserActiveStatus(tenantDB, tenantUUID, userUUID, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user status"})
		return
	}

	// Check if a user was actually found and deleted.
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Audit log: End user active status toggled
	action := "deactivate"
	if active {
		action = "activate"
	}
	middlewares.Audit(c, "enduser", userID, action, &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"active":    active,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// Private helper methods

func updateUserActiveStatus(db *gorm.DB, tenantID uuid.UUID, userID uuid.UUID, active bool) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("tenant database connection not available")
	}

	result := db.Table("users").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]interface{}{
			"active":     active,
			"updated_at": timeNow(),
		})

	return result.RowsAffected, result.Error
}

func (euc *EndUserController) OIDCLogin(c *gin.Context) {
	var input models.OIDCLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate OIDC token against Ory Hydra
	introspection, err := euc.validateOIDCToken(input.AccessToken)
	if err != nil || !*introspection.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or inactive OIDC token"})
		return
	}

	// Safely extract tenantID, emailID, and clientID with type assertions and checks
	ext := introspection.Ext
	if ext == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing extension fields in OIDC token"})
		return
	}
	tenantID, ok := ext["tenant_id"].(string)
	if !ok || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing tenant_id in OIDC token"})
		return
	}
	emailID, ok := ext["email"].(string)
	if !ok || emailID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing email in OIDC token"})
		return
	}

	// Extract client_id from the token introspection response
	clientID := introspection.ClientID
	if clientID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing client_id in OIDC token"})
		return
	}

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}
	clientID = strings.TrimSuffix(clientID, "-main-client")
	// Find enduser details, prioritizing MFA-enabled check with client_id validation
	var user models.User
	err = tenantDB.Where("email = ? AND client_id = ? AND active = ? AND mfa_enabled = ?", emailID, clientID, true, true).First(&user).Error
	if err != nil {
		// MFA check failed: log and fallback to query without MFA condition
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("MFA not enabled or user not found for email: %s, client_id: %s", emailID, clientID)
		} else {
			log.Printf("Database error during MFA-enabled user query for email: %s, client_id: %s, error: %v", emailID, clientID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Fallback query without MFA condition but still include client_id
		err = tenantDB.Where("email = ? AND client_id = ? AND active = ?", emailID, clientID, true).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("User not found in fallback query for email: %s, client_id: %s", emailID, clientID)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			} else {
				log.Printf("Database error in fallback user query for email: %s, client_id: %s, error: %v", emailID, clientID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
			return
		}
		// At this point, user exists but MFA is not enabled
		log.Printf("Fallback successful: User found with MFA disabled for email: %s, client_id: %s", user.Email, clientID)
	} else {
		// MFA-enabled user found
		log.Printf("MFA-enabled user found for email: %s, client_id: %s", user.Email, clientID)
	}

	// Cross-verify tenant ID from token matches user's tenant
	tenantIDUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	if user.TenantID != tenantIDUUID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant mismatch in credentials"})
		return
	}

	// Check if this is first-time login by examining last_login column
	isFirstLogin := user.LastLogin == nil

	// Prepare base response
	response := models.LoginResponse{
		TenantID:    user.TenantID.String(),
		Email:       user.Email,
		FirstLogin:  isFirstLogin,
		OTPRequired: false,
	}

	// Handle logic based on MFA and login type
	if !user.MFAEnabled {
		log.Printf("MFA not enabled for user: %s - proceeding with login", user.Email)
		// TODO: Generate and include token here if MFA is disabled (e.g., response.Token = generateJWT(user))
		// For now, assuming token issuance is handled elsewhere or not required
	} else if isFirstLogin {
		log.Printf("First-time login for: %s - may require MFA setup", user.Email)
		// TODO: Optionally generate temporary token or redirect to MFA enrollment
	} else {
		// Returning user with MFA enabled: require verification, no token yet
		log.Printf("Returning user login for: %s - requires MFA verification", user.Email)
		c.JSON(http.StatusOK, response)
		return
	}

	// Update last_login for successful partial/full login (before token issuance or MFA prompt)

	// Return response (with token if applicable)
	c.JSON(http.StatusOK, response)
}

// validateOIDCToken validates the OIDC token against Ory Hydra's introspection endpoint
func (tc *EndUserController) validateOIDCToken(token string) (*sharedmodels.Introspection, error) {
	// Initialize Ory Hydra client
	if config.AppConfig == nil {
		return nil, errors.New("application configuration not available")
	}
	hydraAdminURL := config.AppConfig.HydraAdminURL
	if hydraAdminURL == "" {
		return nil, errors.New("hydra Admin URL is not configured")
	}
	//remove initial "http://"
	if hydraAdminURL[:7] == "http://" {
		hydraAdminURL = hydraAdminURL[7:]
	}

	// Create a new Hydra client
	config := hydra.NewConfiguration()
	config.Host = hydraAdminURL
	config.Scheme = "http"
	client := hydra.NewAPIClient(config)

	// Perform token introspection using the correct API method
	resp, httpResp, err := client.OAuth2API.IntrospectOAuth2Token(context.Background()).Token(token).Execute()
	if err != nil {
		return nil, errors.New("failed to introspect token: " + err.Error())
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New("token introspection failed with status: " + httpResp.Status)
	}

	// Convert the response to IntrospectionResponse
	introspection := &sharedmodels.Introspection{
		Active:   &resp.Active,
		Scope:    *resp.Scope, // Dereference the pointer to get the string value
		ClientID: *resp.ClientId,
		Ext:      resp.Ext,
	}

	return introspection, nil
}

func (euc *EndUserController) CustomLogin(c *gin.Context) {
	var input models.CustomLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantID := tenantUUID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Find tenant user
	var user models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"custom", "ad_sync", "entra_id", "scim"}).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if !user.CheckPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if MFA is enabled
	var user2 models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND mfa_enabled = ?", input.Email, input.ClientID, "true").First(&user2).Error; err != nil {
		// MFA is not enabled
		var isFirstLogin bool
		if user.Provider == "ad_sync" || user.Provider == "entra_id" {
			isFirstLogin = true
		} else {
			isFirstLogin = user.LastLogin == nil
		}

		response := models.LoginResponse{
			TenantID:    user.TenantID.String(),
			Email:       user.Email,
			FirstLogin:  isFirstLogin,
			OTPRequired: false,
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// MFA is enabled
	isFirstLogin := user2.LastLogin == nil

	// Prepare base response
	response := models.LoginResponse{
		TenantID:    user2.TenantID.String(),
		Email:       user2.Email,
		FirstLogin:  isFirstLogin,
		OTPRequired: false,
	}

	log.Printf("Returning user login for: %s - requires MFA verification", user.Email)
	c.JSON(http.StatusOK, response)
}

func (euc *EndUserController) CustomLoginStatus(c *gin.Context) {
	var input models.CustomLoginStatus
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantID := tenantUUID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Check if email already exists in main tenant table
	var existingUser models.User
	input.Email = strings.ToLower(input.Email)
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"custom", "ad_sync", "scim", "entra_id"}).First(&existingUser).Error; err == nil {
		// If user exists, check if it's ad_sync with empty password_hash
		if (existingUser.Provider == "ad_sync" || existingUser.Provider == "entra_id" || existingUser.Provider == "scim") && existingUser.PasswordHash == "" {
			c.JSON(http.StatusOK, gin.H{"response": "false", "message": "User does not exist, proceed with registration"})
			return
		}

		// For all other cases (custom users or ad_sync users with password_hash)
		c.JSON(http.StatusOK, gin.H{"response": "true", "message": "User already exists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": "false", "message": "User does not exist, proceed with registration"})
}

// InitiateCustomLoginRegister godoc
// @Summary Initiate custom login registration with OTP
// @Description Initiates custom login registration by sending OTP to email for verification
// @Tags EndUser Auth
// @Accept json
// @Produce json
// @Param input body object true "User registration initiation data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/register/initiate [post]
func (euc *EndUserController) InitiateCustomLoginRegister(c *gin.Context) {
	var input models.CustomLoginRegister
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantID := tenantUUID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"custom", "ad_sync", "entra_id", "scim"}).First(&existingUser).Error; err == nil {
		// If user exists and is not a synced user with empty password, reject registration
		if !((existingUser.Provider == "ad_sync" || existingUser.Provider == "entra_id" || existingUser.Provider == "scim") && existingUser.PasswordHash == "") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
	}

	// Fetch client details
	var client models.Client
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", input.ClientID, tenantID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find client: %v", err)})
		return
	}

	// Hash the password for storage in pending registration
	tempUser := models.ExtendedUser{
		User: sharedmodels.User{
			PasswordHash: input.Password,
		},
	}
	if err := tempUser.HashPassword(); err != nil {
		log.Printf("Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Delete any existing pending registration for this email
	db := config.GetDatabase()
	if _, err := db.Exec("DELETE FROM pending_registrations WHERE email = $1", input.Email); err != nil {
		log.Printf("Error deleting existing pending registration: %v", err)
	}

	// Create pending registration record - we'll store client_id and tenant_id
	clientIDUUID, _ := uuid.Parse(input.ClientID)
	tenantIDUUID, _ := uuid.Parse(tenantID)
	pendingReg := models.PendingRegistration{
		Email:        input.Email,
		PasswordHash: tempUser.PasswordHash,
		FirstName:    input.Email, // Use email as first name for custom login
		LastName:     "",
		TenantID:     tenantIDUUID,
		ProjectID:    client.ProjectID,
		ClientID:     clientIDUUID,
		ExpiresAt:    time.Now().Add(30 * time.Minute),    // Expires in 30 minutes to match OTP
		TenantDomain: config.AppConfig.TenantDomainSuffix, // Use configured domain suffix (authsec.dev)
	}

	insertQuery := `INSERT INTO pending_registrations (email, password_hash, first_name, last_name, tenant_id, project_id, client_id, tenant_domain, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`
	if _, err := db.Exec(insertQuery, pendingReg.Email, pendingReg.PasswordHash, pendingReg.FirstName, pendingReg.LastName,
		pendingReg.TenantID, pendingReg.ProjectID, pendingReg.ClientID, pendingReg.TenantDomain, pendingReg.ExpiresAt); err != nil {
		log.Printf("Failed to create pending registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate registration"})
		return
	}

	// Generate and send OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Printf("Failed to generate OTP: %v", err)
		// Cleanup pending registration
		db.Exec("DELETE FROM pending_registrations WHERE email = $1", input.Email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	// Delete any existing OTP for this email
	if _, err := db.Exec("DELETE FROM otp_entries WHERE email = $1", input.Email); err != nil {
		log.Printf("Warning - failed to delete old OTPs: %v", err)
	}

	// Create new OTP entry
	otpInsert := `INSERT INTO otp_entries (email, otp, expires_at, verified, created_at, updated_at)
		VALUES ($1, $2, $3, false, NOW(), NOW())`
	if _, err := db.Exec(otpInsert, input.Email, otp, time.Now().Add(10*time.Minute)); err != nil {
		log.Printf("Failed to create OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
		return
	}

	// Send OTP via email
	if err := utils.SendOTPEmail(input.Email, otp); err != nil {
		log.Printf("Failed to send OTP email: %v", err)
		// Cleanup
		db.Exec("DELETE FROM otp_entries WHERE email = $1", input.Email)
		db.Exec("DELETE FROM pending_registrations WHERE email = $1", input.Email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	log.Printf("Custom login registration initiated for: %s", input.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Registration initiated. Please check your email for OTP verification.",
		"email":   input.Email,
	})
}

// CompleteCustomLoginRegister godoc
// @Summary Complete custom login registration with OTP verification
// @Description Verifies the OTP and completes custom login user registration
// @Tags EndUser Auth
// @Accept json
// @Produce json
// @Param input body object true "OTP verification data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/register/complete [post]
func (euc *EndUserController) CompleteCustomLoginRegister(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		OTP      string `json:"otp" binding:"required"`
		ClientID string `json:"client_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	db := config.GetDatabase()

	// Verify OTP
	var otpVerified bool
	var otpExpiry time.Time
	err := db.QueryRow("SELECT verified, expires_at FROM otp_entries WHERE email = $1 AND otp = $2", input.Email, input.OTP).Scan(&otpVerified, &otpExpiry)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	if otpExpiry.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OTP has expired"})
		return
	}

	// Mark OTP as verified
	if _, err := db.Exec("UPDATE otp_entries SET verified = true, updated_at = NOW() WHERE email = $1 AND otp = $2", input.Email, input.OTP); err != nil {
		log.Printf("Failed to mark OTP as verified: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	// Get pending registration
	var pendingReg models.PendingRegistration
	err = db.QueryRow(`SELECT email, password_hash, first_name, last_name, tenant_id, project_id, client_id, tenant_domain, expires_at
		FROM pending_registrations WHERE email = $1`, input.Email).Scan(
		&pendingReg.Email, &pendingReg.PasswordHash, &pendingReg.FirstName, &pendingReg.LastName,
		&pendingReg.TenantID, &pendingReg.ProjectID, &pendingReg.ClientID, &pendingReg.TenantDomain, &pendingReg.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration session expired. Please initiate registration again"})
		return
	}

	// Verify client_id matches
	if pendingReg.ClientID.String() != input.ClientID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID mismatch"})
		return
	}

	tenantID := pendingReg.TenantID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Check if there's an existing synced user (AD/Entra/SCIM) that needs password update
	var adSyncUser models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"ad_sync", "entra_id", "scim"}).First(&adSyncUser).Error; err == nil {
		// Update the existing synced user's password_hash
		if err := tenantDB.Model(&adSyncUser).Update("password_hash", pendingReg.PasswordHash).Error; err != nil {
			log.Printf("Failed to update user password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}

		// Cleanup
		db.Exec("DELETE FROM pending_registrations WHERE email = $1", input.Email)
		db.Exec("DELETE FROM otp_entries WHERE email = $1", input.Email)

		c.JSON(http.StatusOK, gin.H{
			"message": "Registration completed successfully",
			"email":   input.Email,
		})
		return
	}

	// Fetch client details
	var client models.Client
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", input.ClientID, tenantID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find client: %v", err)})
		return
	}

	// Create new user
	newUser := models.ExtendedUser{
		User: sharedmodels.User{
			ID:           uuid.New(),
			ClientID:     pendingReg.ClientID,
			TenantID:     pendingReg.TenantID,
			ProjectID:    pendingReg.ProjectID,
			Name:         pendingReg.Email,
			Email:        pendingReg.Email,
			PasswordHash: pendingReg.PasswordHash,
			TenantDomain: pendingReg.TenantDomain,
			Provider:     "custom",
			ProviderID:   pendingReg.Email,
			Active:       true,
			MFAEnabled:   false,
		},
	}

	if err := tenantDB.Create(&newUser).Error; err != nil {
		log.Printf("Failed to create new user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete registration"})
		return
	}

	// Cleanup pending registration and OTP
	db.Exec("DELETE FROM pending_registrations WHERE email = $1", input.Email)
	db.Exec("DELETE FROM otp_entries WHERE email = $1", input.Email)

	log.Printf("Custom login registration completed for: %s", input.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "Registration completed successfully",
		"email":   input.Email,
	})
}

// CustomLoginRegister - Legacy endpoint (kept for backward compatibility)
// Deprecated: Use InitiateCustomLoginRegister and CompleteCustomLoginRegister instead
func (euc *EndUserController) CustomLoginRegister(c *gin.Context) {
	var input models.CustomLoginRegister
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantID := tenantUUID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Check if email already exists in main tenant table
	var existingUser models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"custom", "ad_sync", "entra_id", "scim"}).First(&existingUser).Error; err == nil {
		// If user exists, check if it's a synced user with empty password_hash
		if (existingUser.Provider == "ad_sync" || existingUser.Provider == "entra_id" || existingUser.Provider == "scim") && existingUser.PasswordHash == "" {
			// Allow registration to proceed (will be handled below)
		} else {
			c.JSON(http.StatusOK, gin.H{"response": "true", "message": "User already exists"})
			return
		}
	}

	var user models.User
	if err := tenantDB.Where("client_id = ? AND email = ?", input.ClientID, input.Email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find user: %v", err)})
			return
		}
	}
	// Fetch client details
	var client models.Client
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", input.ClientID, tenantID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find client: %v", err)})
		return
	}

	// Check if there's an existing synced user (AD/Entra/SCIM) that needs password update
	var adSyncUser models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, input.ClientID, []string{"ad_sync", "entra_id", "scim"}).First(&adSyncUser).Error; err == nil {
		// Hash the new password
		tempUser := models.ExtendedUser{
			User: sharedmodels.User{
				PasswordHash: input.Password,
			},
		}
		if err := tempUser.HashPassword(); err != nil {
			log.Printf("Failed to hash password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}

		// Update the existing ad_sync user's password_hash
		if err := tenantDB.Model(&adSyncUser).Update("password_hash", tempUser.PasswordHash).Error; err != nil {
			log.Printf("Failed to update user password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Registration completed successfully",
			"email":   input.Email,
		})
		return
	}

	tempUser := models.User{
		PasswordHash: input.Password,
	}
	if err := tempUser.HashPassword(); err != nil {
		log.Printf("Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create new user with all required data
	clientIDUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}
	tenantIDUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	newUser := models.ExtendedUser{
		User: sharedmodels.User{
			ID:           uuid.New(),
			ClientID:     clientIDUUID,
			TenantID:     tenantIDUUID,
			ProjectID:    client.ProjectID, // Already UUID
			Name:         input.Email,      // Use email as name if Name field doesn't exist
			Email:        input.Email,
			PasswordHash: tempUser.PasswordHash,
			TenantDomain: config.AppConfig.TenantDomainSuffix, // Use configured domain suffix (authsec.dev)
			Provider:     "custom",
			ProviderID:   input.Email, // Ensure ProviderID is not null
			Active:       true,
			MFAEnabled:   false, // Explicitly set MFAEnabled as required by shared-models v0.5.0
		},
	}
	// Prepare base response
	if err := tenantDB.Create(&newUser).Error; err != nil {
		log.Printf("Failed to create new user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate registration"})
		return
	}

	// For returning users, check if they need MFA verification
	// Since this is not first login, they should go through WebAuthn/MFA flow
	// Return response without token - client should redirect to WebAuthn verification
	c.JSON(http.StatusOK, gin.H{
		"message": "Registration completed successfully",
		"email":   input.Email,
	})
}

func (tc *EndUserController) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
	if config.DB == nil {
		return uuid.UUID{}, fmt.Errorf("database connection not available")
	}
	var tenantMapping models.TenantMapping
	if err := config.DB.Where("client_id = ?", clientID).First(&tenantMapping).Error; err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to find tenant: %w", err)
	}
	return tenantMapping.TenantID, nil
}

// Add these methods to your EndUserController struct in enduser_controller.go

// CustomForgotPassword godoc
// @Summary Initiate forgot password for custom login
// @Description Sends OTP to user's email for password reset verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body object true "Forgot password data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/forgot-password [post]
func (euc *EndUserController) CustomForgotPassword(c *gin.Context) {
	var input models.CustomForgotPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// Get tenant ID from client mapping
	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		// For security, don't reveal if client ID is invalid
		c.JSON(http.StatusOK, models.CustomForgotPasswordResponse{
			Message: "If your email is registered, you will receive a password reset OTP",
			Email:   input.Email,
		})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		// For security, don't reveal if client ID is invalid
		c.JSON(http.StatusOK, models.CustomForgotPasswordResponse{
			Message: "If your email is registered, you will receive a password reset OTP",
			Email:   input.Email,
		})
		return
	}

	tenantID := tenantUUID.String()
	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		log.Printf("CustomForgotPassword: failed to connect to tenant database for tenant %s: %v", tenantUUID.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Check if user exists with custom provider
	var user models.User
	if err := tenantDB.Where("email = ? AND provider IN (?) AND client_id = ? AND active = ?", input.Email, []string{"custom", "ad_sync", "entra_id", "scim"}, input.ClientID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("User not found for forgot password request: %s", input.Email)
		} else {
			log.Printf("Database error during forgot password user lookup: %v", err)
		}
		// For security, always return success message regardless of whether user exists
		c.JSON(http.StatusOK, models.CustomForgotPasswordResponse{
			Message: "If your email is registered, you will receive a password reset OTP",
			Email:   input.Email,
		})
		return
	}

	// Generate and send OTP using existing utility
	if err := euc.generateAndSendCustomPasswordResetOTP(input.Email); err != nil {
		log.Printf("Failed to send password reset OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	log.Printf("Password reset OTP sent for custom login user: %s", input.Email)

	c.JSON(http.StatusOK, models.CustomForgotPasswordResponse{
		Message: "If your email is registered, you will receive a password reset OTP",
		Email:   input.Email,
	})
}

// CustomVerifyPasswordResetOTP godoc
// @Summary Verify OTP for custom login password reset
// @Description Verifies the OTP sent for custom login password reset
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body object true "OTP verification data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/forgot-password/verify-otp [post]
func (euc *EndUserController) CustomVerifyPasswordResetOTP(c *gin.Context) {
	var input models.CustomVerifyPasswordResetOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// Verify OTP using the same pattern as tenant controller
	var otpEntry models.OTPEntry
	if err := config.DB.Where("email = ? AND otp = ? AND expires_at > ? AND verified = ?",
		input.Email, input.OTP, time.Now(), false).First(&otpEntry).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Mark OTP as verified
	if err := config.DB.Model(&otpEntry).Update("verified", true).Error; err != nil {
		log.Printf("Failed to mark OTP as verified: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	log.Printf("Password reset OTP verified for custom login user: %s", input.Email)

	c.JSON(http.StatusOK, models.CustomVerifyPasswordResetOTPResponse{
		Message: "OTP verified successfully. You can now reset your password",
		Email:   input.Email,
	})
}

// CustomResetPassword godoc
// @Summary Reset password for custom login user
// @Description Resets password for custom login user after OTP verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body object true "Password reset data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/forgot-password/reset [post]
func (euc *EndUserController) CustomResetPassword(c *gin.Context) {
	var input models.CustomResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// Validate password strength
	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters long"})
		return
	}

	// Check if OTP was verified (following tenant controller pattern)
	var otpEntry models.OTPEntry
	if err := config.DB.Where("email = ? AND verified = ? AND expires_at > ?",
		input.Email, true, time.Now()).First(&otpEntry).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP not verified or expired. Please request a new OTP"})
		return
	}

	// Get tenant ID from client mapping
	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantUUID, err := euc.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantID := tenantUUID.String()

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Find the user in tenant database
	var user models.User
	if err := tenantDB.Where("email = ? AND provider IN (?) AND client_id = ? AND active = ?", input.Email, []string{"custom", "ad_sync", "entra_id", "scim"}, input.ClientID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Hash the new password using the same method as your existing code
	tempUser := models.ExtendedUser{
		User: sharedmodels.User{
			PasswordHash: input.NewPassword,
		},
	}
	if err := tempUser.HashPassword(); err != nil {
		log.Printf("Failed to hash new password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	// Begin transaction for password update (following tenant controller pattern)
	tx := config.DB.Begin()
	tenantTx := tenantDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			tenantTx.Rollback()
		}
	}()

	// Update user password in tenant database
	if err := tenantTx.Model(&user).Updates(map[string]interface{}{
		"password_hash": tempUser.PasswordHash,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		tenantTx.Rollback()
		log.Printf("Failed to update user password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Clean up OTP entries (following tenant controller cleanup pattern)
	tx.Where("email = ?", input.Email).Delete(&models.OTPEntry{})

	// Commit both transactions
	if err := tenantTx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to commit tenant transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete password reset"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit main transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete password reset"})
		return
	}

	log.Printf("Password reset completed successfully for custom login user: %s", input.Email)

	c.JSON(http.StatusOK, models.CustomResetPasswordResponse{
		Message: "Password reset successfully",
		Email:   input.Email,
	})
}

// Helper function to generate and send password reset OTP (reusing existing OTP utilities)
func (euc *EndUserController) generateAndSendCustomPasswordResetOTP(email string) error {
	log.Printf("generateAndSendCustomPasswordResetOTP: starting for %s", email)
	// Check if config.DB is available
	if config.DB == nil {
		log.Printf("generateAndSendCustomPasswordResetOTP: database connection unavailable for %s", email)
		return fmt.Errorf("database connection not available")
	}

	// Generate OTP using existing utility
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Printf("generateAndSendCustomPasswordResetOTP: failed to generate OTP for %s: %v", email, err)
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	log.Printf("generateAndSendCustomPasswordResetOTP: generated OTP for %s", email)

	// Delete any existing OTP for this email (following tenant controller pattern)
	if err := config.DB.Where("email = ?", email).Delete(&models.OTPEntry{}).Error; err != nil {
		log.Printf("generateAndSendCustomPasswordResetOTP: warning - failed to delete old OTPs for %s: %v", email, err)
	} else {
		log.Printf("generateAndSendCustomPasswordResetOTP: cleared existing OTPs for %s", email)
	}

	// Create new OTP entry using existing structure
	otpEntry := models.OTPEntry{
		Email:     email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(30 * time.Minute), // OTP expires in 30 minutes
		Verified:  false,
	}

	if err := config.DB.Create(&otpEntry).Error; err != nil {
		log.Printf("generateAndSendCustomPasswordResetOTP: failed to persist OTP for %s: %v", email, err)
		return fmt.Errorf("failed to save password reset OTP: %w", err)
	}

	log.Printf("generateAndSendCustomPasswordResetOTP: stored OTP entry (%s) for %s", otpEntry.ID.String(), email)

	// Send password reset OTP email using modified version of existing function
	if err := utils.SendPasswordResetOTPEmail(email, otp); err != nil {
		// FIX: Don't delete OTP on email failure - the OTP is still valid
		// and the email might still be delivered despite the error
		log.Printf("generateAndSendCustomPasswordResetOTP: failed to send email to %s, but OTP remains valid: %v", email, err)
		return fmt.Errorf("failed to send password reset OTP email: %w", err)
	}

	log.Printf("generateAndSendCustomPasswordResetOTP: password reset OTP email sent successfully to %s", email)

	return nil
}
func (euc *EndUserController) AdminChangeUserPassword(c *gin.Context) {
	var input models.AdminChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// Check if config.DB is available
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not available"})
		return
	}

	// Validate password strength
	if len(input.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters long"})
		return
	}

	// Parse tenant ID
	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	tenantID := tenantUUID.String()

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Find the user in tenant database
	var user models.User
	query := tenantDB.Where("tenant_id = ? AND active = ?", tenantID, true)

	// Search by email or user ID based on what's provided
	if input.Email != "" {
		query = query.Where("email = ? AND provider IN (?)", input.Email, []string{"custom", "ad_sync"})
	} else if input.UserID != "" {
		userUUID, err := uuid.Parse(input.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		query = query.Where("id = ?", userUUID.String())
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either email or user_id must be provided"})
		return
	}

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Hash the new password using the same method as your existing code
	tempUser := models.ExtendedUser{
		User: sharedmodels.User{
			PasswordHash: input.NewPassword,
		},
	}
	if err := tempUser.HashPassword(); err != nil {
		log.Printf("Failed to hash new password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	// Update user password in tenant database
	if err := tenantDB.Model(&user).Updates(map[string]interface{}{
		"password_hash": tempUser.PasswordHash,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		log.Printf("Failed to update user password via admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	log.Printf("Admin changed password for user: %s (ID: %s) in tenant: %s", user.Email, user.ID, tenantID)

	// Audit log: Admin changed user password
	middlewares.Audit(c, "enduser", user.ID.String(), "admin_change_password", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":   user.ID.String(),
			"email":     user.Email,
			"tenant_id": tenantID,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "User password changed successfully",
		"user_id":   user.ID.String(),
		"email":     user.Email,
		"tenant_id": user.TenantID.String(),
	})
}

// AdminResetUserPassword godoc
// @Summary Admin reset user password to temporary password
// @Description Allows admin to reset user password to a temporary password and optionally send it via email
// @Tags Admin
// @Accept json
// @Produce json
// @Param input body object true "Admin password reset data"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/admin/reset-password [post]
func (euc *EndUserController) AdminResetUserPassword(c *gin.Context) {
	var input models.AdminResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// Check if config.DB is available
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not available"})
		return
	}

	// Parse tenant ID
	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	tenantID := tenantUUID.String()

	// Get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Find the user in tenant database
	var user models.User
	query := tenantDB.Where("tenant_id = ? AND active = ?", tenantID, true)

	if input.Email != "" {
		query = query.Where("email = ? AND provider IN (?)", input.Email, []string{"custom", "ad_sync"})
	} else if input.UserID != "" {
		userUUID, err := uuid.Parse(input.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		query = query.Where("id = ?", userUUID.String())
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either email or user_id must be provided"})
		return
	}

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Generate temporary password
	tempPassword, err := utils.GenerateTemporaryPassword()
	if err != nil {
		log.Printf("Failed to generate temporary password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate temporary password"})
		return
	}

	// Hash the temporary password
	tempUser := models.ExtendedUser{
		User: sharedmodels.User{
			PasswordHash: tempPassword,
		},
	}
	if err := tempUser.HashPassword(); err != nil {
		log.Printf("Failed to hash temporary password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process temporary password"})
		return
	}

	// Update user password in tenant database
	if err := tenantDB.Model(&user).Updates(map[string]interface{}{
		"password_hash": tempUser.PasswordHash,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		log.Printf("Failed to reset user password via admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Send temporary password via email if requested
	var emailSent bool
	if input.SendEmail {
		if err := utils.SendTemporaryPasswordEmail(user.Email, tempPassword); err != nil {
			log.Printf("Failed to send temporary password email: %v", err)
			// Don't fail the request, just note that email wasn't sent
			emailSent = false
		} else {
			emailSent = true
		}
	}

	log.Printf("Admin reset password for user: %s (ID: %s) in tenant: %s", user.Email, user.ID, tenantID)

	// Audit log: Admin reset user password
	middlewares.Audit(c, "enduser", user.ID.String(), "admin_reset_password", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":    user.ID.String(),
			"email":      user.Email,
			"tenant_id":  tenantID,
			"email_sent": emailSent,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"message":            "User password reset successfully",
		"user_id":            user.ID.String(),
		"email":              user.Email,
		"tenant_id":          user.TenantID.String(),
		"temporary_password": tempPassword,
		"email_sent":         emailSent,
	})

	// If email wasn't requested or failed, include temp password in response
	if !input.SendEmail || !emailSent {
		c.JSON(http.StatusOK, gin.H{
			"success":            true,
			"message":            "User password reset successfully. Temporary password included in response",
			"user_id":            user.ID.String(),
			"email":              user.Email,
			"tenant_id":          user.TenantID.String(),
			"temporary_password": tempPassword,
			"email_sent":         emailSent,
		})
	} else {
		// For security, don't include password in response if email was sent successfully
		c.JSON(http.StatusOK, gin.H{
			"success":            true,
			"message":            "User password reset successfully",
			"user_id":            user.ID.String(),
			"email":              user.Email,
			"tenant_id":          user.TenantID.String(),
			"temporary_password": "Sent via email",
			"email_sent":         emailSent,
		})
	}
}

// GetClients godoc
// @Summary Get all clients for a tenant
// @Description Retrieves all clients for a specific tenant with pagination and filtering
// @Tags Clients
// @Accept json
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Param active query bool false "Filter by active status"
// @Param client_id query string false "Filter by specific client ID"
// @Success 200 {object} sharedmodels.PaginatedEndUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/clients [get]
func (euc *EndUserController) GetClients(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
		return
	}

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	activeStr := c.Query("active")
	clientIDFilter := c.Query("client_id")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Parse active filter
	var activeFilter *bool
	if activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err == nil {
			activeFilter = &active
		}
	}

	// Validate client ID format if provided
	if clientIDFilter != "" {
		if _, err := uuid.Parse(clientIDFilter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id format"})
			return
		}
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Build query with base tenant filter
	query := tenantDB.Model(&models.Client{}).Where("tenant_id = ?", tenantID)

	// Apply filters
	if activeFilter != nil {
		query = query.Where("active = ?", *activeFilter)
	}
	if clientIDFilter != "" {
		query = query.Where("client_id = ?", clientIDFilter)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count clients"})
		return
	}

	// Get paginated results
	var clients []models.Client
	if err := query.Offset(offset).Limit(limit).Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve clients"})
		return
	}

	// Convert to response format
	var clientResponses []models.RegisterClientsResponse
	for _, client := range clients {
		email := ""
		if client.Email != nil {
			email = *client.Email
		}
		response := models.RegisterClientsResponse{
			ID:        client.ID.String(),
			ClientID:  client.ClientID.String(),
			TenantID:  client.TenantID.String(),
			ProjectID: client.ProjectID.String(),
			Name:      client.Name,
			Email:     email,
			Active:    client.Active,
			CreatedAt: client.CreatedAt,
		}
		clientResponses = append(clientResponses, response)
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + limit - 1) / limit

	response := gin.H{
		"clients":    clientResponses,
		"total":      int(totalCount),
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// GetClientsPost godoc
// @Summary Get all clients for a tenant (POST version)
// @Description Retrieves all clients for a specific tenant with pagination and filtering via POST request
// @Tags Clients
// @Accept json
// @Produce json
// @Param input body object true "Client list request payload"
// @Success 200 {object} sharedmodels.PaginatedEndUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/clients/get [post]
func (euc *EndUserController) GetClientsPost(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id" binding:"required"`
		Page     int    `json:"page,omitempty"`
		Limit    int    `json:"limit,omitempty"`
		Active   *bool  `json:"active,omitempty"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	// Validate client ID format if provided
	if req.ClientID != "" {
		if _, err := uuid.Parse(req.ClientID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id format"})
			return
		}
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Build query with base tenant filter
	query := tenantDB.Model(&models.Client{}).Where("tenant_id = ?", req.TenantID)

	// Apply filters
	if req.Active != nil {
		query = query.Where("active = ?", *req.Active)
	}
	if req.ClientID != "" {
		query = query.Where("client_id = ?", req.ClientID)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count clients"})
		return
	}

	// Get paginated results
	var clients []models.Client
	if err := query.Offset(offset).Limit(req.Limit).Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve clients"})
		return
	}

	// Convert to response format
	var clientResponses []models.RegisterClientsResponse
	for _, client := range clients {
		email := ""
		if client.Email != nil {
			email = *client.Email
		}
		response := models.RegisterClientsResponse{
			ID:        client.ID.String(),
			ClientID:  client.ClientID.String(),
			TenantID:  client.TenantID.String(),
			ProjectID: client.ProjectID.String(),
			Name:      client.Name,
			Email:     email,
			Active:    client.Active,
			CreatedAt: client.CreatedAt,
		}
		clientResponses = append(clientResponses, response)
	}

	// Calculate pagination info
	totalPages := (int(totalCount) + req.Limit - 1) / req.Limit

	response := gin.H{
		"clients":    clientResponses,
		"total":      int(totalCount),
		"page":       req.Page,
		"limit":      req.Limit,
		"totalPages": totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// GetClientsByTenantID godoc
// @Summary Get clients for a specific tenant (URL path version)
// @Description Retrieves clients for a tenant specified in URL path with filtering via request body
// @Tags Clients
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param input body object true "Client filter options"
// @Success 200 {object} sharedmodels.PaginatedEndUsersResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/clientms/tenants/{tenant_id}/clients/getClients [post]
func (euc *EndUserController) GetClientsByTenantID(c *gin.Context) {
	// Get tenant_id from token
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	// Parse request body for filters
	var req struct {
		Page     int    `json:"page,omitempty"`
		Limit    int    `json:"limit,omitempty"`
		Active   *bool  `json:"active_only,omitempty"` // Map active_only to active
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Set default values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	// Validate client ID format if provided
	if req.ClientID != "" {
		if _, err := uuid.Parse(req.ClientID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id format"})
			return
		}
	}

	// Connect to tenant database
	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Build query with base tenant filter
	query := tenantDB.Model(&models.Client{}).Where("deleted_at IS NULL")

	// Add active filter if provided
	if req.Active != nil {
		query = query.Where("active = ?", *req.Active)
	}

	// Add client ID filter if provided
	if req.ClientID != "" {
		query = query.Where("id = ?", req.ClientID)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count clients"})
		return
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(req.Limit) - 1) / int64(req.Limit))

	// Get paginated results
	var clients []models.Client
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.Limit).Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch clients"})
		return
	}

	// Build response
	response := gin.H{
		"clients": clients,
		"pagination": gin.H{
			"total":      int(totalCount),
			"page":       req.Page,
			"limit":      req.Limit,
			"totalPages": totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// NotifyOwnerNewRegistration godoc
// @Summary Notify tenant owner about a new user registration
// @Description Sends a notification email to the specified owner email with details of the newly registered user
// @Tags EndUser
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body object true "Notification request with owner_email and optional user details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/auth/notify/new-user-registration [post]
func (euc *EndUserController) NotifyOwnerNewRegistration(c *gin.Context) {
	const ownerEmail = "a@authnull.com"

	var input struct {
		UserName     string `json:"user_name,omitempty"`
		TenantDomain string `json:"tenant_domain,omitempty"`
	}
	// Body is optional — ignore bind errors for empty body
	_ = c.ShouldBindJSON(&input)

	// Extract user email from JWT context
	userEmail := c.GetString("email_id")
	if userEmail == "" {
		userEmail = c.GetString("email")
	}
	if userEmail == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user email not found in authentication token"})
		return
	}

	// Extract tenant ID from JWT context
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
		return
	}

	// Use provided user_name or fall back to email
	userName := input.UserName
	if userName == "" {
		userName = userEmail
	}

	// Use provided tenant_domain or fall back to tenant ID
	tenantDomain := input.TenantDomain
	if tenantDomain == "" {
		tenantDomain = tenantID
	}

	if err := utils.SendNewUserRegistrationNotificationEmail(ownerEmail, userName, userEmail, tenantDomain); err != nil {
		log.Printf("NotifyOwnerNewRegistration: failed to send notification email to %s: %v", ownerEmail, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send notification email"})
		return
	}

	log.Printf("NotifyOwnerNewRegistration: notification sent to %s for new user %s in tenant %s", ownerEmail, userEmail, tenantID)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Owner notification email sent successfully",
		"owner_email": ownerEmail,
		"user_email":  userEmail,
	})
}
