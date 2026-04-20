package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/authsec-ai/authsec/config"
	repositories "github.com/authsec-ai/authsec/repository"
	"github.com/authsec-ai/authsec/services"
	"github.com/authsec-ai/authsec/vault"
	"github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExternalServiceController handles HTTP requests for the external-service API.
type ExternalServiceController struct {
	globalDB *gorm.DB
	jwtKey   []byte

	vaultOnce        sync.Once
	vaultClient      vault.VaultClient
	vaultErr         error
	tenantMigrations sync.Map
	adminBindings    sync.Map
}

// NewExternalServiceController constructs an ExternalServiceController backed by the global DB.
func NewExternalServiceController(master *gorm.DB) *ExternalServiceController {
	return &ExternalServiceController{
		globalDB: master,
		jwtKey:   []byte(os.Getenv("JWT_SECRET")),
	}
}

/* -------------------------------------------------------------------------- */
/*                              Request types                                 */
/* -------------------------------------------------------------------------- */

// ExternalServiceCreateRequest is the JSON body for POST /authsec/services.
type ExternalServiceCreateRequest struct {
	Name            string            `json:"name" validate:"required"`
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Description     string            `json:"description"`
	Tags            []string          `json:"tags"`
	ResourceID      ExternalServiceResourceID `json:"resource_id" validate:"required,uuid"`
	AuthType        string            `json:"auth_type" validate:"required"`
	AgentAccessible bool              `json:"agent_accessible"`
	SecretData      map[string]string `json:"secret_data,omitempty"`
}

// ExternalServiceUpdateRequest is the JSON body for PUT /authsec/services/:id.
type ExternalServiceUpdateRequest struct {
	Name            *string           `json:"name,omitempty"`
	Type            *string           `json:"type,omitempty"`
	Description     *string           `json:"description,omitempty"`
	URL             *string           `json:"url,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	ResourceID      *ExternalServiceResourceID `json:"resource_id,omitempty"`
	AuthType        *string           `json:"auth_type,omitempty"`
	AgentAccessible *bool             `json:"agent_accessible,omitempty"`
	SecretData      map[string]string `json:"secret_data,omitempty"`
}

// ExternalServiceResourceID is a string UUID that also accepts positive integers over JSON.
type ExternalServiceResourceID string

func (r *ExternalServiceResourceID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			return fmt.Errorf("resource_id cannot be empty")
		}
		if _, err := uuid.Parse(str); err != nil {
			return fmt.Errorf("resource_id must be a valid UUID: %v", err)
		}
		*r = ExternalServiceResourceID(str)
		return nil
	}
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		if num <= 0 {
			return fmt.Errorf("resource_id number must be positive")
		}
		uuidFromNum := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			num>>32, (num>>16)&0xFFFF, num&0xFFFF, 0x4000|((num>>48)&0xFFF), num&0xFFFFFFFFFFFF)
		*r = ExternalServiceResourceID(uuidFromNum)
		return nil
	}
	return fmt.Errorf("resource_id must be a valid UUID string or positive number")
}

func (r ExternalServiceResourceID) String() string { return string(r) }

/* -------------------------------------------------------------------------- */
/*                            Internal helpers                                */
/* -------------------------------------------------------------------------- */

func (ctl *ExternalServiceController) resolveTenant(c *gin.Context) (*gorm.DB, string, string, error) {
	claimsInterface, exists := c.Get("claims")
	if !exists {
		return nil, "", "", fmt.Errorf("claims not found in context")
	}

	var claims map[string]interface{}
	switch v := claimsInterface.(type) {
	case map[string]interface{}:
		claims = v
	case jwt.MapClaims:
		claims = map[string]interface{}(v)
	default:
		return nil, "", "", fmt.Errorf("invalid claims format: %T", claimsInterface)
	}

	tenantIDStr, ok := claims["tenant_id"].(string)
	if !ok || tenantIDStr == "" {
		return nil, "", "", fmt.Errorf("tenant_id not found in claims")
	}
	clientIDStr, ok := claims["client_id"].(string)
	if !ok || clientIDStr == "" {
		return nil, "", "", fmt.Errorf("client_id not found in claims")
	}

	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	var tenant sharedmodels.Tenant
	if err := ctl.globalDB.Where("tenant_id = ?", tenantUUID).First(&tenant).Error; err != nil {
		return nil, "", "", fmt.Errorf("tenant not found: %w", err)
	}

	tenantDB, err := config.ConnectTenantDB(tenant.TenantDB)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to connect to tenant DB: %w", err)
	}

	if err := ctl.ensureTenantSchema(tenantIDStr, tenantDB); err != nil {
		return nil, "", "", fmt.Errorf("failed to prepare tenant schema: %w", err)
	}

	if err := ctl.ensureAdminBinding(c, tenantDB, tenantIDStr); err != nil {
		return nil, "", "", fmt.Errorf("failed to ensure admin role binding: %w", err)
	}

	return tenantDB, clientIDStr, tenantIDStr, nil
}

func (ctl *ExternalServiceController) ensureTenantSchema(tenantID string, tenantDB *gorm.DB) error {
	if _, ok := ctl.tenantMigrations.Load(tenantID); ok {
		return nil
	}
	// Schema and permissions are provisioned by the migration system
	// (migrations/tenant/ and migrations/permissions/master/).
	ctl.tenantMigrations.Store(tenantID, struct{}{})
	log.Printf("EXTSVC: schema ready for tenant %s", tenantID)
	return nil
}

func (ctl *ExternalServiceController) ensureAdminBinding(c *gin.Context, tenantDB *gorm.DB, tenantID string) error {
	claimsAny, exists := c.Get("claims")
	if !exists {
		return nil
	}
	var claims jwt.MapClaims
	switch v := claimsAny.(type) {
	case jwt.MapClaims:
		claims = v
	case map[string]interface{}:
		claims = jwt.MapClaims(v)
	default:
		return nil
	}
	if !extsvcHasAdminRole(claims) {
		return nil
	}
	userID := extsvcClaimString(claims, "user_id")
	if userID == "" {
		userID = extsvcClaimString(claims, "sub")
	}
	if userID == "" {
		return fmt.Errorf("admin token missing user identifier")
	}
	cacheKey := fmt.Sprintf("%s:%s", tenantID, userID)
	if _, ok := ctl.adminBindings.Load(cacheKey); ok {
		return nil
	}
	ctl.adminBindings.Store(cacheKey, struct{}{})
	return nil
}

func (ctl *ExternalServiceController) getVaultClient() (vault.VaultClient, error) {
	ctl.vaultOnce.Do(func() {
		addr := os.Getenv("VAULT_ADDR")
		token := os.Getenv("VAULT_TOKEN")
		if addr == "" || token == "" {
			ctl.vaultErr = fmt.Errorf("VAULT_ADDR or VAULT_TOKEN not set")
			return
		}
		ctl.vaultClient, ctl.vaultErr = vault.NewClient(addr, token)
	})
	return ctl.vaultClient, ctl.vaultErr
}

/* -------------------------------------------------------------------------- */
/*                                Handlers                                    */
/* -------------------------------------------------------------------------- */

// CreateExternalService handles POST /authsec/services.
func (ctl *ExternalServiceController) CreateExternalService(c *gin.Context) {
	tenantDB, clientID, tenantID, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var input ExternalServiceCreateRequest
	if err = c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vClient, vErr := ctl.getVaultClient()
	if vErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": vErr.Error()})
		return
	}

	svc := &repositories.ExternalService{
		Name:            input.Name,
		Type:            input.Type,
		URL:             input.URL,
		Description:     input.Description,
		Tags:            input.Tags,
		ResourceID:      input.ResourceID.String(),
		AuthType:        input.AuthType,
		AgentAccessible: input.AgentAccessible,
	}

	secretData := make(map[string]interface{}, len(input.SecretData))
	for k, v := range input.SecretData {
		secretData[k] = v
	}

	manager := services.NewExternalServiceManager(repositories.NewExternalServiceRepository(tenantDB), vClient)
	out, err := manager.Create(svc, clientID, tenantID, secretData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

// GetExternalService handles GET /authsec/services/:id.
func (ctl *ExternalServiceController) GetExternalService(c *gin.Context) {
	tenantDB, clientID, _, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	manager := services.NewExternalServiceManager(repositories.NewExternalServiceRepository(tenantDB), nil)
	out, err := manager.Get(c.Param("id"), clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// ListExternalServices handles GET /authsec/services.
func (ctl *ExternalServiceController) ListExternalServices(c *gin.Context) {
	tenantDB, clientID, _, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	manager := services.NewExternalServiceManager(repositories.NewExternalServiceRepository(tenantDB), nil)
	list, err := manager.List(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"services": list})
}

// UpdateExternalService handles PUT /authsec/services/:id.
func (ctl *ExternalServiceController) UpdateExternalService(c *gin.Context) {
	tenantDB, clientID, _, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var req ExternalServiceUpdateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateInput := services.ExternalServiceUpdateInput{
		Name:        req.Name,
		Type:        req.Type,
		URL:         req.URL,
		Description: req.Description,
		Tags:        req.Tags,
		AuthType:    req.AuthType,
		AgentAccessible: req.AgentAccessible,
	}
	if req.ResourceID != nil {
		rid := req.ResourceID.String()
		updateInput.ResourceID = &rid
	}
	if len(req.SecretData) > 0 {
		secretData := make(map[string]interface{}, len(req.SecretData))
		for k, v := range req.SecretData {
			secretData[k] = v
		}
		updateInput.SecretData = secretData
	}

	var vaultClient vault.VaultClient
	if len(updateInput.SecretData) > 0 {
		if vaultClient, err = ctl.getVaultClient(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	manager := services.NewExternalServiceManager(repositories.NewExternalServiceRepository(tenantDB), vaultClient)
	out, err := manager.Update(c.Param("id"), clientID, updateInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// DeleteExternalService handles DELETE /authsec/services/:id.
func (ctl *ExternalServiceController) DeleteExternalService(c *gin.Context) {
	tenantDB, clientID, _, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	vClient, _ := ctl.getVaultClient()
	manager := services.NewExternalServiceManager(repositories.NewExternalServiceRepository(tenantDB), vClient)
	if err := manager.Delete(c.Param("id"), clientID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetExternalServiceCredentials handles GET /authsec/services/:id/credentials.
func (ctl *ExternalServiceController) GetExternalServiceCredentials(c *gin.Context) {
	tenantDB, clientID, _, err := ctl.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	serviceID := c.Param("id")

	repo := repositories.NewExternalServiceRepository(tenantDB)
	var svc *repositories.ExternalService

	// SPIFFE JWT-SVID agents may access agent-accessible services directly,
	// bypassing client ownership checks.
	authMethod, _ := c.Get("auth_method")
	if authMethod == "spiffe-jwt-svid" {
		svc, err = repo.GetByID(serviceID)
		if err != nil || !svc.AgentAccessible {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
	} else {
		manager := services.NewExternalServiceManager(repo, nil)
		svc, err = manager.Get(serviceID, clientID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
	}

	vaultClient, err := ctl.getVaultClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	secrets, err := vaultClient.ReadSecret(svc.VaultPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve secrets", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id":   serviceID,
		"service_name": svc.Name,
		"service_type": svc.Type,
		"auth_type":    svc.AuthType,
		"url":          svc.URL,
		"credentials":  secrets,
		"metadata":     gin.H{},
		"retrieved_at": time.Now().Format(time.RFC3339),
	})
}

// DebugExternalServiceAuth dumps JWT claims — useful for troubleshooting.
func DebugExternalServiceAuth(c *gin.Context) {
	claimsInterface, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No claims found"})
		return
	}

	var claims map[string]interface{}
	switch v := claimsInterface.(type) {
	case map[string]interface{}:
		claims = v
	case jwt.MapClaims:
		claims = map[string]interface{}(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims format", "type": fmt.Sprintf("%T", claimsInterface)})
		return
	}

	roles, scopes, resources := []string{}, []string{}, []string{}
	if rv, ok := claims["roles"].([]interface{}); ok {
		for _, r := range rv {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	if sv, ok := claims["scopes"].([]interface{}); ok {
		for _, s := range sv {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
	}
	if perms, ok := claims["perms"].(map[string]interface{}); ok {
		if allow, ok := perms["allow"].([]interface{}); ok {
			for _, p := range allow {
				if s, ok := p.(string); ok {
					scopes = append(scopes, s)
				}
			}
		}
	}
	if rv, ok := claims["resources"].([]interface{}); ok {
		for _, r := range rv {
			if s, ok := r.(string); ok {
				resources = append(resources, s)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"claims":              claims,
		"extracted_roles":     roles,
		"extracted_scopes":    scopes,
		"extracted_resources": resources,
		"client_id":           claims["client_id"],
		"tenant_id":           claims["tenant_id"],
	})
}

/* -------------------------------------------------------------------------- */
/*                              Local helpers                                 */
/* -------------------------------------------------------------------------- */

func extsvcHasAdminRole(claims jwt.MapClaims) bool {
	if rv, ok := claims["roles"]; ok {
		switch roles := rv.(type) {
		case []string:
			for _, r := range roles {
				if r == "admin" || r == "super_admin" {
					return true
				}
			}
		case []interface{}:
			for _, r := range roles {
				if s, ok := r.(string); ok && (s == "admin" || s == "super_admin") {
					return true
				}
			}
		}
	}
	if r, ok := claims["role"].(string); ok && (r == "admin" || r == "super_admin") {
		return true
	}
	return false
}

func extsvcClaimString(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
