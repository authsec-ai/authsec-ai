package platform

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/internal/clients/authmethods"
	"github.com/authsec-ai/authsec/internal/clients/library"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/services"
	sharedmodels "github.com/authsec-ai/sharedmodels"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientsTenant is the tenant model used by clients controller for DB queries.
type ClientsTenant struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	TenantDB     string    `json:"tenant_db"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"password_hash,omitempty"`
	Provider     string    `gorm:"default:'local';index:idx_users_provider" json:"provider"`
	Name         string    `json:"name,omitempty"`
	Source       string    `json:"source,omitempty"`
	Status       string    `json:"status,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	TenantDomain string    `json:"tenant_domain,omitempty" gorm:"uniqueIndex;not null"`
}

// ClientsTenantMapping maps tenants to clients.
type ClientsTenantMapping struct {
	ID       uuid.UUID     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID string        `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClientID string        `json:"client_id" gorm:"uniqueIndex;not null"`
	Tenant   ClientsTenant `json:"tenant"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (ClientsTenantMapping) TableName() string { return "tenant_mappings" }

type clientsErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      int       `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	Details   []string  `json:"details,omitempty"`
}

type clientsMessageResponse struct {
	Message   string      `json:"message"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ClientsPagination summarizes paging information for list endpoints.
type ClientsPagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

// ClientsWithMethods extends the client model with authentication methods and user count
type ClientsWithMethods struct {
	sharedmodels.Client
	AuthenticationMethods []string `json:"authentication_methods"`
	UserCount             int64    `json:"user_count"`
	Deleted               bool     `json:"deleted"`
}

// ClientsListResponse is the top-level payload returned by GetClients.
type ClientsListResponse struct {
	Clients        []ClientsWithMethods `json:"clients"`
	Pagination     ClientsPagination    `json:"pagination"`
	HydraPublicURL string               `json:"hydra_public_url"`
}

func isClientsClientNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "client not found")
}

type clientsCreateClientPayload struct {
	OwnerID       string   `json:"owner_id"`
	OrgID         string   `json:"org_id"`
	Name          string   `json:"name" binding:"required"`
	Email         *string  `json:"email,omitempty"`
	Active        *bool    `json:"active,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	HydraClientID *string  `json:"hydra_client_id,omitempty"`
	OIDCEnabled   *bool    `json:"oidc_enabled,omitempty"`
}

type clientsUpdateClientPayload struct {
	Name          *string   `json:"name,omitempty"`
	Email         *string   `json:"email,omitempty"`
	Active        *bool     `json:"active,omitempty"`
	Status        *string   `json:"status,omitempty"`
	Tags          *[]string `json:"tags,omitempty"`
	HydraClientID *string   `json:"hydra_client_id,omitempty"`
	OIDCEnabled   *bool     `json:"oidc_enabled,omitempty"`
}

func getClientsUUIDFromContext(c *gin.Context, key string) (uuid.UUID, bool) {
	value, exists := c.Get(key)
	if !exists {
		return uuid.UUID{}, false
	}
	switch v := value.(type) {
	case uuid.UUID:
		return v, true
	case *uuid.UUID:
		if v == nil {
			return uuid.UUID{}, false
		}
		return *v, true
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.UUID{}, false
		}
		return parsed, true
	default:
		return uuid.UUID{}, false
	}
}

// clientsGetAndValidateTenantID extracts the tenant ID from the JWT context and
// cross-validates it against the URL path parameter ":tenantId" when both are present.
// Returns 401 if the JWT has no tenant, 403 if URL and JWT tenants do not match.
func clientsGetAndValidateTenantID(c *gin.Context) (uuid.UUID, bool) {
	jwtTenantID, hasJWT := getClientsUUIDFromContext(c, "validated_tenant_id")

	urlTenantRaw := c.Param("tenantId")
	if hasJWT && urlTenantRaw != "" {
		urlTenantID, err := uuid.Parse(urlTenantRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID in URL"})
			return uuid.UUID{}, false
		}
		if jwtTenantID != urlTenantID {
			log.Printf("[SECURITY] Tenant mismatch: JWT tenant=%s, URL tenant=%s, IP=%s",
				jwtTenantID.String(), urlTenantID.String(), c.ClientIP())
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant ID mismatch between token and URL"})
			return uuid.UUID{}, false
		}
		return jwtTenantID, true
	}
	if hasJWT {
		return jwtTenantID, true
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
	return uuid.UUID{}, false
}

// RegisterClientsResponse is the response payload for client registration.
type RegisterClientsResponse struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	TenantID  string    `json:"tenant_id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	SecretID  string    `json:"secret_id,omitempty"`
	Email     string    `json:"email"`
	Active    bool      `json:"active"`
	SpiffeID  string    `json:"spiffe_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// RegisterClientsRequest is the request payload for client registration.
type RegisterClientsRequest struct {
	Name         string            `json:"name" binding:"required,min=1,max=256"`
	Email        string            `json:"email" binding:"required,email,max=512"`
	TenantID     string            `json:"tenant_id" binding:"max=64"`
	ProjectID    string            `json:"project_id" binding:"max=64"`
	TenantDomain string            `json:"react_app_url" binding:"max=2048"`
	ClientType   string            `json:"client_type,omitempty" binding:"max=100"`
	AgentType    *string           `json:"agent_type,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	Selectors    map[string]string `json:"selectors,omitempty"`
}

// platformSelectorKeys defines the pre-filled selector keys per platform.
// The UI shows these as key fields; the user only fills in values.
var platformSelectorKeys = map[string][]string{
	"kubernetes": {
		"k8s:ns",
		"k8s:sa",
		"k8s:pod-label:app",
		"k8s:container-name",
	},
	"docker": {
		"docker:label:app",
		"docker:image-id",
		"docker:container-id",
	},
	"unix": {
		"unix:uid",
		"unix:gid",
	},
}

// GetPlatformSelectorKeys returns the allowed selector keys for a given platform.
// The UI calls this to render pre-filled key fields with empty value inputs.
func GetPlatformSelectorKeys(c *gin.Context) {
	platform := strings.ToLower(c.Query("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform query param is required"})
		return
	}
	keys, ok := platformSelectorKeys[platform]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":               "unsupported platform",
			"supported_platforms": []string{"kubernetes", "docker", "unix"},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"platform": platform, "selector_keys": keys})
}

// validatePlatformSelectors checks that the supplied selector keys are valid for the platform.
func validatePlatformSelectors(platform string, selectors map[string]string) error {
	allowed, ok := platformSelectorKeys[strings.ToLower(platform)]
	if !ok {
		return fmt.Errorf("unsupported platform %q, must be one of: kubernetes, docker, unix", platform)
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		allowedSet[k] = true
	}
	for key := range selectors {
		if !allowedSet[key] {
			return fmt.Errorf("selector key %q is not valid for platform %q, allowed keys: %v", key, platform, allowed)
		}
	}
	return nil
}

// buildPlatformSelectors validates user-supplied key-value selectors and returns the filtered map.
func buildPlatformSelectors(platform string, selectors map[string]string) (map[string]string, error) {
	if len(selectors) == 0 {
		return nil, nil
	}
	if err := validatePlatformSelectors(platform, selectors); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, v := range selectors {
		if v != "" {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// ClientsDeleteCompleteRequest is the request body for hard-delete operations.
type ClientsDeleteCompleteRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
}

// ClientsStatusRequest is the request body for set-status operations.
type ClientsStatusRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
	Active   bool   `json:"active"`
}


func clientsRegisterWithHydra(clientID, clientSecret, clientName, tenantID string, tenantDomain string) error {
	return services.RegisterClientWithHydra(clientID, clientSecret, clientName, tenantID, tenantDomain)
}

func clientsDeleteFromOOCManager(tenantID, clientID string) error {
	log.Printf("[OOC-DELETE-START] Deleting Hydra clients for tenant=%s client=%s", tenantID, clientID)
	if err := services.DeleteClientFromHydra(clientID); err != nil {
		return err
	}
	log.Printf("[OOC-DELETE-SUCCESS] Deleted Hydra clients for client=%s", clientID)
	return nil
}

func clientsAddProvider(tenantID, clientID, reactAppURL, createdBy string) error {
	return services.AddProviderToClient(tenantID, clientID, reactAppURL, createdBy)
}

// GetClientsByTenant handles the legacy POST route for getting clients by tenant.
func GetClientsByTenant(c *gin.Context) {
	var req struct {
		TenantID   string `json:"tenant_id" validate:"required"`
		ActiveOnly bool   `json:"active_only"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, clientsErrorResponse{
			Error:     "Invalid request",
			Message:   err.Error(),
			Code:      http.StatusBadRequest,
			Timestamp: time.Now(),
		})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &req.TenantID)
	if err != nil {
		log.Printf("Failed to get tenant DB connection: %v", err)
		c.JSON(http.StatusInternalServerError, clientsErrorResponse{
			Error:     "Database connection failed",
			Message:   err.Error(),
			Code:      http.StatusInternalServerError,
			Timestamp: time.Now(),
		})
		return
	}

	var clients []sharedmodels.Client
	query := tenantDB.Where("tenant_id = ?", req.TenantID)

	if req.ActiveOnly {
		query = query.Where("active = ?", true)
	}

	if err := query.Order("created_at ASC").Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, clientsErrorResponse{
			Error:     "Failed to query clients",
			Message:   err.Error(),
			Code:      http.StatusInternalServerError,
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Clients retrieved successfully",
		"success": true,
		"data": map[string]interface{}{
			"tenant_id":   req.TenantID,
			"clients":     clients,
			"count":       len(clients),
			"active_only": req.ActiveOnly,
		},
		"timestamp": time.Now(),
	})
}

// DeleteCompleteClient hard-deletes a client from both tenant DB and main DB.
func DeleteCompleteClient(c *gin.Context) {
	log.Printf("[HARD-DELETE-START] Delete-complete request initiated")

	var input ClientsDeleteCompleteRequest
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported Content-Type",
			"details": "Content-Type must be application/json",
		})
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.TenantID == "" || input.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required fields",
			"details": "Both tenant_id and client_id are required",
		})
		return
	}

	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format", "details": err.Error()})
		return
	}
	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id format", "details": err.Error()})
		return
	}

	tenantIDStr := tenantUUID.String()
	clientIDStr := clientUUID.String()
	input.TenantID = tenantIDStr
	input.ClientID = clientIDStr

	var existingClient sharedmodels.Client
	if err := config.DB.Where("client_id = ? AND tenant_id = ?", clientIDStr, tenantIDStr).First(&existingClient).Error; err == nil {
		if existingClient.Name == "Default client" || existingClient.Name == "default" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Cannot delete default client",
				"message": "Default clients cannot be deleted",
				"details": "This is a system-protected client",
			})
			return
		}
	}

	var tenantDB *gorm.DB
	if config.DB != nil && config.DB.Dialector.Name() == "sqlite" {
		tenantDB = config.DB
	} else {
		tenantDB, err = middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
		if err != nil {
			log.Printf("[HARD-DELETE-ERROR] Failed to get tenant DB connection - Tenant: %s, Error: %v", input.TenantID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
			return
		}
	}

	tenantTx := tenantDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tenantTx.Rollback()
		}
	}()

	result := tenantTx.Where("client_id = ? AND tenant_id = ?", input.ClientID, input.TenantID).Delete(&sharedmodels.Client{})
	if result.Error != nil {
		tenantTx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client from tenant database"})
		return
	}

	if result.RowsAffected == 0 {
		tenantTx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Client not found",
			"message":   "No client found with the specified tenant_id and client_id",
			"tenant_id": input.TenantID,
			"client_id": input.ClientID,
		})
		return
	}

	if err := tenantTx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit tenant database transaction"})
		return
	}

	mainTx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			mainTx.Rollback()
		}
	}()

	if err := mainTx.Where("client_id = ? AND tenant_id = ?", input.ClientID, input.TenantID).Delete(&ClientsTenantMapping{}).Error; err != nil {
		mainTx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client from tenant_mapping table"})
		return
	}

	if err := mainTx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit main database transaction"})
		return
	}

	if err := clientsDeleteFromOOCManager(input.TenantID, input.ClientID); err != nil {
		log.Printf("[HARD-DELETE-OOC] OOC Manager call failed (non-fatal) - Tenant: %s, Client: %s, Error: %v", input.TenantID, input.ClientID, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Client deleted completely",
		"tenant_id": input.TenantID,
		"client_id": input.ClientID,
	})
}

// SetClientStatus activates or deactivates a client by tenant_id and client_id from request body.
func SetClientStatus(c *gin.Context) {
	var req ClientsStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, clientsErrorResponse{
			Error:     "Invalid request",
			Message:   err.Error(),
			Code:      http.StatusBadRequest,
			Timestamp: time.Now(),
		})
		return
	}

	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, clientsErrorResponse{Error: "Invalid client_id", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, clientsErrorResponse{Error: "Invalid tenant_id", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, clientsErrorResponse{Error: "Database connection failed", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var client sharedmodels.Client
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).First(&client).Error; err != nil {
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, clientsErrorResponse{Error: "Client not found", Message: "Client with the specified ID does not exist", Code: http.StatusNotFound, Timestamp: time.Now()})
		} else {
			c.JSON(http.StatusInternalServerError, clientsErrorResponse{Error: "Database error", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		}
		return
	}

	statusValue := sharedmodels.StatusInactive
	if req.Active {
		statusValue = sharedmodels.StatusActive
	}

	updates := map[string]interface{}{
		"active":     req.Active,
		"status":     statusValue,
		"updated_at": time.Now(),
	}

	if err := tenantDB.Model(&sharedmodels.Client{}).
		Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, clientsErrorResponse{Error: "Failed to update client status", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, clientsErrorResponse{Error: "Failed to retrieve updated client", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	status := "deactivated"
	if req.Active {
		status = "activated"
	}

	c.JSON(http.StatusOK, clientsMessageResponse{
		Message: fmt.Sprintf("Client %s successfully", status),
		Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID,
			"client_id": req.ClientID,
			"active":    req.Active,
			"client":    client,
		},
		Timestamp: time.Now(),
	})
}

// RegisterClient registers a new client (full registration with Hydra and Vault).
// For AI agents (client_type=ai_agent), registers a SPIFFE workload identity instead.
// @Security Bearer
func RegisterClient(c *gin.Context) {
	var input RegisterClientsRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := input.TenantID
	if tenantID == "" {
		if ctxTenantID, ok := getClientsUUIDFromContext(c, "validated_tenant_id"); ok {
			tenantID = ctxTenantID.String()
		}
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required (in body or token)"})
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	projectID := input.ProjectID
	if projectID == "" {
		projectID = tenantID
	}
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	clientType := sharedmodels.ClientTypeApplication
	if input.ClientType != "" {
		clientType = input.ClientType
	}

	// AI agents don't get Hydra client IDs — their identity comes from SPIRE
	clientID := uuid.New()
	hydraClientID := fmt.Sprintf("%s-main-client", clientID.String())
	if clientType == sharedmodels.ClientTypeAIAgent {
		hydraClientID = ""
	}

	client := sharedmodels.Client{
		ID:            clientID,
		ClientID:      clientID,
		TenantID:      tenantUUID,
		ProjectID:     projectUUID,
		OwnerID:       tenantUUID,
		OrgID:         tenantUUID,
		Name:          input.Name,
		Email:         &input.Email,
		Status:        sharedmodels.StatusActive,
		Active:        true,
		MFAEnabled:    false,
		MFAVerified:   false,
		HydraClientID: hydraClientID,
		OIDCEnabled:   false,
		ClientType:    clientType,
		AgentType:     input.AgentType,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		log.Printf("Failed to get tenant DB connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	tx := tenantDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&client).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	// AI agents get SPIFFE identity from SPIRE; others get Hydra/Vault identity
	secretID := ""
	spiffeIDStr := ""
	if clientType == sharedmodels.ClientTypeAIAgent {
		// Validate and build platform selectors
		platformSelectors, selectorErr := buildPlatformSelectors(input.Platform, input.Selectors)
		if selectorErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": selectorErr.Error()})
			return
		}

		agentTypeStr := ""
		if client.AgentType != nil {
			agentTypeStr = *client.AgentType
		}
		if agentTypeStr == "" {
			agentTypeStr = "ai_agent"
		}

		spiffeID, spireErr := RegisterAgentWorkload(
			client.TenantID.String(),
			client.ClientID.String(),
			agentTypeStr,
			input.Platform,
			platformSelectors,
		)
		if spireErr != nil {
			log.Printf("Warning: failed to register AI agent with SPIRE: %v", spireErr)
			// Don't fail client creation — SPIFFE identity can be retried
		} else {
			spiffeIDStr = spiffeID
			if updateErr := tenantDB.Model(&sharedmodels.Client{}).
				Where("client_id = ?", client.ClientID).
				Update("spiffe_id", spiffeID).Error; updateErr != nil {
				log.Printf("Warning: failed to save spiffe_id to client: %v", updateErr)
			}
		}
	} else {
		// Standard path: Vault + Hydra + OIDC provider
		var vaultErr error
		secretID, vaultErr = config.SaveSecretToVault(client.TenantID.String(), client.ProjectID.String(), client.ClientID.String())
		if vaultErr != nil {
			log.Printf("Warning: failed to save secret to vault: %v", vaultErr)
		}

		if secretID != "" {
			email := ""
			if client.Email != nil {
				email = *client.Email
			}
			if err := clientsRegisterWithHydra(client.ClientID.String(), secretID, email, client.TenantID.String(), input.TenantDomain); err != nil {
				log.Printf("Warning: failed to register client with Hydra: %v", err)
			}

			createdBy := email
			if createdBy == "" {
				createdBy = "system"
			}
			if err := clientsAddProvider(client.TenantID.String(), client.ClientID.String(), input.TenantDomain, createdBy); err != nil {
				log.Printf("Warning: failed to add provider to client: %v", err)
			}
		}
	}

	tenantMapping := ClientsTenantMapping{
		TenantID: client.TenantID.String(),
		ClientID: client.ClientID.String(),
	}
	tp := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tp.Rollback()
		}
	}()
	if err := tp.Create(&tenantMapping).Error; err != nil {
		tp.Rollback()
		log.Printf("Failed to create tenant mapping: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete registration"})
		return
	}
	if err := tp.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	if err := services.SeedClientAdminRBAC(c.Request.Context(), config.DB, tenantUUID); err != nil {
		log.Printf("Warning: failed to seed clients RBAC for tenant %s: %v", tenantUUID.String(), err)
	}

	email := ""
	if client.Email != nil {
		email = *client.Email
	}
	c.JSON(http.StatusCreated, RegisterClientsResponse{
		ID:        client.ID.String(),
		ClientID:  client.ClientID.String(),
		TenantID:  client.TenantID.String(),
		ProjectID: client.ProjectID.String(),
		Name:      client.Name,
		SecretID:  secretID,
		Email:     email,
		Active:    client.Active,
		SpiffeID:  spiffeIDStr,
		CreatedAt: client.CreatedAt,
		Message:   "Client registered successfully",
	})
}

func clientsUUIDFromValueOrContext(c *gin.Context, value, contextKey, field string) (uuid.UUID, error) {
	if value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("invalid %s: %w", field, err)
		}
		return parsed, nil
	}

	if ctxVal, ok := getClientsUUIDFromContext(c, contextKey); ok {
		return ctxVal, nil
	}

	return uuid.UUID{}, fmt.Errorf("%s is required", field)
}

// CreateClient creates a new client.
// @Summary Create a new client
// @Tags clientms
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param client body library.ClientCreateRequest true "Client object"
// @Success 201 {object} sharedmodels.Client
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /clientms/tenants/{tenantId}/clients/create [post]
// @Security Bearer
func CreateClient(c *gin.Context) {
	tenantID, ok := getClientsUUIDFromContext(c, "validated_tenant_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id not found in context"})
		return
	}

	var payload clientsCreateClientPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID, err := clientsUUIDFromValueOrContext(c, payload.OwnerID, "validated_owner_id", "owner_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, err := clientsUUIDFromValueOrContext(c, payload.OrgID, "validated_org_id", "org_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	active := true
	if payload.Active != nil {
		active = *payload.Active
	}

	status := sharedmodels.StatusActive
	if payload.Status != nil && *payload.Status != "" {
		status = *payload.Status
	}

	oidcEnabled := false
	if payload.OIDCEnabled != nil {
		oidcEnabled = *payload.OIDCEnabled
	}

	req := &library.ClientCreateRequest{
		TenantID:      tenantID,
		OwnerID:       ownerID,
		OrgID:         orgID,
		Name:          payload.Name,
		Email:         payload.Email,
		Active:        active,
		Status:        status,
		Tags:          payload.Tags,
		HydraClientID: payload.HydraClientID,
		OIDCEnabled:   oidcEnabled,
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		log.Printf("Failed to get tenant DB connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)
	createdClient, err := clientLib.CreateClient(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client", "details": err.Error()})
		return
	}

	if err := services.SeedClientAdminRBAC(c.Request.Context(), config.DB, tenantID); err != nil {
		log.Printf("Warning: failed to seed clients RBAC for tenant %s: %v", tenantID.String(), err)
	}

	c.JSON(http.StatusCreated, createdClient)
}

// GetClients retrieves a list of clients with filtering.
// @Summary List clients
// @Tags clientms
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param status query string false "Filter by status"
// @Param active query bool false "Filter by active flag"
// @Param name query string false "Filter by name (partial match)"
// @Param tags query string false "Filter by tags (CSV)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} ClientsListResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /clientms/tenants/{tenantId}/clients/getClients [get]
// @Security Bearer
func GetClients(c *gin.Context) {
	tenantID, ok := clientsGetAndValidateTenantID(c)
	if !ok {
		return
	}

	status := c.Query("status")
	name := c.Query("name")
	email := c.Query("email")
	tagsParam := c.Query("tags")
	tags := make([]string, 0)
	if tagsParam != "" {
		for _, tag := range strings.Split(tagsParam, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	var activeFilter *bool
	if raw := c.Query("active_only"); raw != "" {
		activeOnly, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active_only value"})
			return
		}
		if activeOnly {
			value := true
			activeFilter = &value
		}
	}
	if raw := c.Query("active"); raw != "" {
		activeValue, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active value"})
			return
		}
		activeFilter = &activeValue
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	var includeDeleted *bool
	if raw := c.Query("deleted"); raw != "" {
		deletedValue, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deleted value, use true or false"})
			return
		}
		includeDeleted = &deletedValue
	}

	filters := &library.ClientListFilters{
		TenantID:       tenantID,
		Status:         status,
		Tags:           tags,
		Name:           name,
		Email:          email,
		Active:         activeFilter,
		Page:           page,
		Limit:          limit,
		IncludeDeleted: includeDeleted,
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		log.Printf("Failed to get tenant DB connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)
	clients, total, err := clientLib.ListClients(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clients", "details": err.Error()})
		return
	}

	if err := services.PromoteExternalServicePermissions(c.Request.Context(), config.DB, tenantID); err != nil {
		log.Printf("Warning: failed to promote external-service permissions for tenant %s: %v", tenantID.String(), err)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	authSvc := authmethods.NewService(config.DB)
	methodsByClient, err := authSvc.MethodsForClients(tenantID, clients)
	if err != nil {
		log.Printf("Failed to fetch authentication methods for tenant %s: %v", tenantID.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch authentication methods"})
		return
	}

	clientIDs := make([]uuid.UUID, 0, len(clients))
	for _, cl := range clients {
		clientIDs = append(clientIDs, cl.ClientID)
	}

	userCounts, err := clientsCountUsersByClient(tenantDB, clientIDs)
	if err != nil {
		log.Printf("Failed to fetch user counts for tenant %s: %v", tenantID.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user counts"})
		return
	}

	deletedFlags, err := clientsDeletedFlagsByClient(tenantDB, clientIDs)
	if err != nil {
		log.Printf("Failed to fetch deleted flags for tenant %s: %v", tenantID.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch client deletion state"})
		return
	}

	items := make([]ClientsWithMethods, 0, len(clients))
	for _, cl := range clients {
		methods := methodsByClient[cl.ClientID]
		if len(methods) == 0 {
			methods = []string{"password"}
		}
		items = append(items, ClientsWithMethods{
			Client:                cl,
			AuthenticationMethods: methods,
			UserCount:             userCounts[cl.ClientID],
			Deleted:               deletedFlags[cl.ClientID],
		})
	}

	c.JSON(http.StatusOK, ClientsListResponse{
		Clients: items,
		Pagination: ClientsPagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
		HydraPublicURL: config.AppConfig.HydraPublicURL,
	})
}

// GetClient retrieves a specific client by ID.
// @Summary Get client by ID
// @Tags clientms
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} sharedmodels.Client
// @Failure 404 {object} map[string]string
// @Router /clientms/tenants/{tenantId}/clients/{id} [get]
// @Security Bearer
func GetClient(c *gin.Context) {
	tenantID, ok := clientsGetAndValidateTenantID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}

	clientID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)
	cl, err := clientLib.GetClientByClientID(clientID, tenantID)
	if err != nil {
		if isClientsClientNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch client", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, cl)
}

// UpdateClient fully updates an existing client.
// @Summary Update client
// @Tags clientms
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} sharedmodels.Client
// @Router /clientms/tenants/{tenantId}/clients/{id} [put]
// @Security Bearer
func UpdateClient(c *gin.Context) {
	clientsHandleClientUpdate(c)
}

// EditClient partially updates an existing client.
// @Summary Edit client
// @Tags clientms
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} sharedmodels.Client
// @Router /clientms/tenants/{tenantId}/clients/{id} [patch]
// @Security Bearer
func EditClient(c *gin.Context) {
	clientsHandleClientUpdate(c)
}

func clientsHandleClientUpdate(c *gin.Context) {
	tenantID, ok := getClientsUUIDFromContext(c, "validated_tenant_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id not found in context"})
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}

	clientID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)
	if _, err := clientLib.GetClientByClientID(clientID, tenantID); err != nil {
		if isClientsClientNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch client", "details": err.Error()})
		return
	}

	var payload clientsUpdateClientPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateReq := &library.ClientUpdateRequest{
		Name:          payload.Name,
		Email:         payload.Email,
		Active:        payload.Active,
		Status:        payload.Status,
		Tags:          payload.Tags,
		HydraClientID: payload.HydraClientID,
		OIDCEnabled:   payload.OIDCEnabled,
	}

	updatedClient, err := clientLib.UpdateClientByClientID(clientID, tenantID, updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedClient)
}

// SoftDeleteClient soft deletes a client via PATCH route.
// @Summary Soft delete client
// @Tags clientms
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Router /clientms/tenants/{tenantId}/clients/{id}/soft-delete [patch]
// @Security Bearer
func SoftDeleteClient(c *gin.Context) {
	clientsHandleSoftDelete(c, "[SOFT-DELETE-PATCH]")
}

// DeleteClient soft deletes a client via DELETE route.
// @Summary Delete client (soft delete)
// @Tags clientms
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Router /clientms/tenants/{tenantId}/clients/{id} [delete]
// @Security Bearer
func DeleteClient(c *gin.Context) {
	clientsHandleSoftDelete(c, "[SOFT-DELETE-DELETE]")
}

func clientsHandleSoftDelete(c *gin.Context, logPrefix string) {
	tenantID, ok := clientsGetAndValidateTenantID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}

	clientID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		log.Printf("%s Failed to get tenant DB connection - Tenant: %s, Error: %v", logPrefix, tenantIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)

	if err := clientLib.DeleteClientByClientID(clientID, tenantID); err != nil {
		if isClientsClientNotFoundError(err) {
			if fallbackErr := clientLib.DeleteClient(clientID, tenantID); fallbackErr != nil {
				if isClientsClientNotFoundError(fallbackErr) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client", "details": fallbackErr.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Client soft deleted successfully", "id": clientID, "deleted": true})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client soft deleted successfully", "id": clientID, "deleted": true})
}

// ActivateClient activates a client.
// @Summary Activate client
// @Tags clientms
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Router /clientms/tenants/{tenantId}/clients/{id}/activate [patch]
// @Security Bearer
func ActivateClient(c *gin.Context) {
	clientsUpdateClientStatus(c, sharedmodels.StatusActive)
}

// DeactivateClient deactivates a client.
// @Summary Deactivate client
// @Tags clientms
// @Param tenantId path string true "Tenant ID"
// @Param id path string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Router /clientms/tenants/{tenantId}/clients/{id}/deactivate [patch]
// @Security Bearer
func DeactivateClient(c *gin.Context) {
	clientsUpdateClientStatus(c, sharedmodels.StatusInactive)
}

func clientsUpdateClientStatus(c *gin.Context, status string) {
	tenantID, ok := getClientsUUIDFromContext(c, "validated_tenant_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id not found in context"})
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
		return
	}

	clientID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	clientLib := library.NewClientLibrary(tenantDB)
	active := status == sharedmodels.StatusActive
	updateReq := &library.ClientUpdateRequest{
		Active: &active,
		Status: &status,
	}

	updatedClient, err := clientLib.UpdateClientByClientID(clientID, tenantID, updateReq)
	if err != nil {
		if isClientsClientNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client status", "details": err.Error()})
		}
		return
	}

	message := "Client updated successfully"
	if status == sharedmodels.StatusActive {
		message = "Client activated successfully"
	} else if status == sharedmodels.StatusInactive {
		message = "Client deactivated successfully"
	}

	c.JSON(http.StatusOK, gin.H{"message": message, "client": updatedClient})
}

type clientsUserCountRow struct {
	ClientID uuid.UUID
	Count    int64
}

func clientsCountUsersByClient(db *gorm.DB, clientIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	counts := make(map[uuid.UUID]int64, len(clientIDs))
	if len(clientIDs) == 0 {
		return counts, nil
	}

	var rows []clientsUserCountRow
	if err := db.Model(&sharedmodels.User{}).
		Select("client_id, COUNT(*) AS count").
		Where("client_id IN ?", clientIDs).
		Group("client_id").
		Scan(&rows).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return counts, nil
		}
		return nil, err
	}

	for _, row := range rows {
		counts[row.ClientID] = row.Count
	}

	return counts, nil
}

type clientsDeletedRow struct {
	ClientID uuid.UUID
	Deleted  bool
}

func clientsDeletedFlagsByClient(db *gorm.DB, clientIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	flags := make(map[uuid.UUID]bool, len(clientIDs))
	if len(clientIDs) == 0 {
		return flags, nil
	}

	var rows []clientsDeletedRow
	// Use Unscoped so soft-deleted rows are visible, then derive the flag from deleted_at.
	if err := db.Unscoped().Model(&sharedmodels.Client{}).
		Select("client_id, (deleted_at IS NOT NULL) AS deleted").
		Where("client_id IN ?", clientIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		flags[row.ClientID] = row.Deleted
	}

	return flags, nil
}
