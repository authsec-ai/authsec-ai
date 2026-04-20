// Package controllers — OocmgrController: OIDC Configuration Manager.
// Ported from oath_oidc_configuration_manager microservice.
package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	oocmgrdto "github.com/authsec-ai/authsec/internal/oocmgr/dto"
	oocmgrrepo "github.com/authsec-ai/authsec/internal/oocmgr/repository"
	oocmgrsvc "github.com/authsec-ai/authsec/internal/oocmgr/service"
	"github.com/authsec-ai/authsec/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ===== CONTROLLER =====

// OocmgrController is the OIDC Configuration Manager controller.
type OocmgrController struct {
	authService           *oocmgrsvc.AuthService
	hydraConfig           oocmgrHydraConfig
	tenantHydraClientRepo *oocmgrrepo.TenantHydraClientRepository
}

type oocmgrHydraConfig struct {
	AdminURL  string
	PublicURL string
}

// NewOocmgrController initialises the controller wiring up all dependencies.
func NewOocmgrController() *OocmgrController {
	authRepo := oocmgrrepo.NewAuthRepository()
	authService := oocmgrsvc.NewAuthService(authRepo)
	return &OocmgrController{
		authService: authService,
		hydraConfig: oocmgrHydraConfig{
			AdminURL:  config.AppConfig.HydraAdminURL,
			PublicURL: config.AppConfig.HydraPublicURL,
		},
		tenantHydraClientRepo: oocmgrrepo.NewTenantHydraClientRepository(),
	}
}

// ===== HYDRA CLIENT TYPES =====

// oocmgrHydraClient mirrors the Hydra admin API client object.
type oocmgrHydraClient struct {
	ClientID      string                 `json:"client_id"`
	ClientSecret  string                 `json:"client_secret,omitempty"`
	ClientName    string                 `json:"client_name"`
	GrantTypes    []string               `json:"grant_types"`
	RedirectURIs  []string               `json:"redirect_uris"`
	ResponseTypes []string               `json:"response_types"`
	TokenEndpoint string                 `json:"token_endpoint_auth_method"`
	Scope         string                 `json:"scope"`
	Audience      []string               `json:"audience,omitempty"`
	SubjectType   string                 `json:"subject_type,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ===== SAML PROVIDER MODEL =====

// OocmgrSAMLProvider is the GORM model for SAML providers stored in tenant databases.
type OocmgrSAMLProvider struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_oocmgr_saml_unique" json:"tenant_id"`
	ClientID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_oocmgr_saml_unique" json:"client_id"`
	ProviderName     string         `gorm:"type:varchar(255);not null;index:idx_oocmgr_saml_unique;uniqueIndex:idx_oocmgr_saml_unique" json:"provider_name"`
	DisplayName      string         `gorm:"type:varchar(255);not null" json:"display_name"`
	EntityID         string         `gorm:"type:varchar(500);not null" json:"entity_id"`
	SSOURL           string         `gorm:"type:varchar(500);not null" json:"sso_url"`
	SLOURL           string         `gorm:"type:varchar(500)" json:"slo_url"`
	Certificate      string         `gorm:"type:text;not null" json:"certificate"`
	MetadataURL      string         `gorm:"type:varchar(500)" json:"metadata_url"`
	NameIDFormat     string         `gorm:"type:varchar(255);default:'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress'" json:"name_id_format"`
	AttributeMapping datatypes.JSON `gorm:"type:jsonb" json:"attribute_mapping"`
	IsActive         bool           `gorm:"default:true" json:"is_active"`
	SortOrder        int            `gorm:"default:0" json:"sort_order"`
	CreatedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (OocmgrSAMLProvider) TableName() string { return "saml_providers" }

func (s *OocmgrSAMLProvider) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.ProviderName = strings.ToLower(strings.TrimSpace(s.ProviderName))
	return nil
}

func (s *OocmgrSAMLProvider) BeforeUpdate(_ *gorm.DB) error {
	s.ProviderName = strings.ToLower(strings.TrimSpace(s.ProviderName))
	return nil
}

// ===== REQUEST STRUCTS =====

type oocmgrCompleteOIDCConfigRequest struct {
	TenantID      string                      `json:"tenant_id"`
	OrgID         string                      `json:"org_id"`
	TenantName    string                      `json:"tenant_name"`
	TenantClient  oocmgrTenantClientConfig    `json:"tenant_client"`
	OIDCProviders []oocmgrOIDCProviderConfig  `json:"oidc_providers"`
	CreatedBy     string                      `json:"created_by"`
}

type oocmgrTenantClientConfig struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
}

type oocmgrOIDCProviderConfig struct {
	ProviderName     string                 `json:"provider_name"`
	DisplayName      string                 `json:"display_name"`
	ClientID         string                 `json:"client_id"`
	ClientSecret     string                 `json:"client_secret"`
	AuthURL          string                 `json:"auth_url"`
	TokenURL         string                 `json:"token_url"`
	UserInfoURL      string                 `json:"user_info_url"`
	Scopes           []string               `json:"scopes"`
	IssuerURL        string                 `json:"issuer_url,omitempty"`
	JWKsURL          string                 `json:"jwks_url,omitempty"`
	AdditionalParams map[string]interface{} `json:"additional_params,omitempty"`
	IsActive         bool                   `json:"is_active"`
	SortOrder        int                    `json:"sort_order"`
}

type oocmgrUpdateOIDCConfigRequest struct {
	TenantID     string   `json:"tenant_id"`
	OrgID        string   `json:"org_id"`
	ProviderName string   `json:"provider_name"`
	DisplayName  *string  `json:"display_name,omitempty"`
	ClientID     *string  `json:"client_id,omitempty"`
	ClientSecret *string  `json:"client_secret,omitempty"`
	AuthURL      *string  `json:"auth_url,omitempty"`
	TokenURL     *string  `json:"token_url,omitempty"`
	UserInfoURL  *string  `json:"user_info_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
	UpdatedBy    string   `json:"updated_by"`
}

type oocmgrGetTenantConfigRequest struct {
	TenantID string `json:"client_id"`
}

type oocmgrGetProviderConfigRequest struct {
	TenantID     string `json:"tenant_id"`
	OrgID        string `json:"org_id"`
	ProviderName string `json:"provider_name"`
}

type oocmgrDeleteOIDCConfigRequest struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ProviderName string `json:"provider_name"`
}

type oocmgrGetProviderSecretRequest struct {
	TenantID     string `json:"tenant_id"`
	ProviderName string `json:"provider_name"`
}

type oocmgrTestOIDCFlowRequest struct {
	TenantID     string `json:"tenant_id"`
	OrgID        string `json:"org_id"`
	ProviderName string `json:"provider_name,omitempty"`
}

// oocmgrOIDCProvider is used internally when listing providers for a tenant.
type oocmgrOIDCProvider struct {
	ProviderName string                 `json:"provider_name"`
	DisplayName  string                 `json:"display_name"`
	IsActive     bool                   `json:"is_active"`
	SortOrder    int                    `json:"sort_order"`
	CallbackURL  string                 `json:"callback_url"`
	Config       map[string]interface{} `json:"config"`
}

// editAuthProviderReq is the named type for EditAuthProvider / helper calls.
type editAuthProviderReq struct {
	TenantID       string                 `json:"tenant_id"`
	ClientID       string                 `json:"client_id"`
	ProviderName   string                 `json:"provider_name"`
	DisplayName    string                 `json:"display_name"`
	IsActive       *bool                  `json:"is_active"`
	SortOrder      *int                   `json:"sort_order"`
	CallbackURL    string                 `json:"callback_url"`
	ProviderConfig map[string]interface{} `json:"provider_config"`
	UpdatedBy      string                 `json:"updated_by"`
}

// ===== MAIN CONFIGURATION ENDPOINT =====

func (ac *OocmgrController) CompleteOIDCConfiguration(c *gin.Context) {
	var req oocmgrCompleteOIDCConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	log.Printf("[oocmgr] CompleteOIDCConfiguration tenant=%s", req.TenantID)

	tenantClientID := fmt.Sprintf("%s-main-client", req.TenantID)
	clientSecret := oocmgrGenerateSecureSecret()

	grantTypes := req.TenantClient.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	scopes := req.TenantClient.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email", "offline_access"}
	}

	tenantClient := oocmgrHydraClient{
		ClientID:      tenantClientID,
		ClientSecret:  clientSecret,
		GrantTypes:    grantTypes,
		RedirectURIs:  req.TenantClient.RedirectURIs,
		ResponseTypes: []string{"code"},
		TokenEndpoint: "client_secret_post",
		Scope:         strings.Join(scopes, " "),
		ClientName:    req.TenantClient.ClientName,
		Metadata: map[string]interface{}{
			"type":        "tenant_main_client",
			"tenant_id":   req.TenantID,
			"org_id":      req.OrgID,
			"tenant_name": req.TenantName,
			"created_at":  time.Now().Format(time.RFC3339),
			"created_by":  req.CreatedBy,
		},
	}

	if err := ac.createHydraClient(tenantClient); err != nil {
		log.Printf("[oocmgr] audit failure: oidc_config create tenant=%s err=%v", req.TenantID, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to create tenant OAuth client", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	tenantHydraClient := &oocmgrdto.TenantHydraClient{
		OrgID: req.OrgID, TenantID: req.TenantID, TenantName: req.TenantName,
		HydraClientID: tenantClientID, HydraClientSecret: clientSecret,
		ClientName: req.TenantClient.ClientName, RedirectURIs: req.TenantClient.RedirectURIs,
		Scopes: scopes, ClientType: "main", IsActive: true, CreatedBy: req.CreatedBy, UpdatedBy: req.CreatedBy,
	}
	if err := ac.tenantHydraClientRepo.Create(tenantHydraClient); err != nil {
		log.Printf("[oocmgr] Warning: Failed to store tenant-client mapping: %v", err)
	}

	var createdProviders []map[string]interface{}
	var failedProviders []map[string]interface{}

	for _, provider := range req.OIDCProviders {
		// Auto-resolve Microsoft authority URLs if authority_type is specified
		oocmgrResolveMicrosoftAuthorityURLs(&provider)

		oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, oocmgrNormalizeProviderName(provider.ProviderName))
		oidcClient := oocmgrHydraClient{
			ClientID: oidcClientID, ClientSecret: "not-used-for-oidc-config",
			GrantTypes: []string{"client_credentials"},
			ClientName: fmt.Sprintf("%s %s OIDC Config", req.TenantName, provider.DisplayName),
			Metadata: map[string]interface{}{
				"type": "oidc_provider", "tenant_id": req.TenantID, "org_id": req.OrgID,
				"provider_name": provider.ProviderName, "display_name": provider.DisplayName,
				"provider_config": map[string]interface{}{
					"client_id": provider.ClientID, "client_secret": provider.ClientSecret,
					"auth_url": provider.AuthURL, "token_url": provider.TokenURL,
					"user_info_url": provider.UserInfoURL, "scopes": provider.Scopes,
					"issuer_url": provider.IssuerURL, "jwks_url": provider.JWKsURL,
					"additional_params": provider.AdditionalParams,
				},
				"is_active":  provider.IsActive,
				"sort_order": provider.SortOrder,
				"created_at": time.Now().Format(time.RFC3339),
				"created_by": req.CreatedBy,
				"callback_url": fmt.Sprintf("%s/callback/%s",
					config.AppConfig.IdentityProviderURL, oocmgrNormalizeProviderName(provider.ProviderName)),
			},
		}

		if err := ac.createHydraClient(oidcClient); err != nil {
			failedProviders = append(failedProviders, map[string]interface{}{"provider_name": provider.ProviderName, "error": err.Error()})
			continue
		}

		if provider.ClientSecret != "" {
			if err := ac.oocmgrSaveProviderSecret(req.TenantID, provider.ProviderName, provider.ClientSecret); err != nil {
				log.Printf("[oocmgr] Warning: Failed to store provider secret for %s: %v", provider.ProviderName, err)
			}
		}

		providerHydraClient := &oocmgrdto.TenantHydraClient{
			OrgID: req.OrgID, TenantID: req.TenantID, TenantName: req.TenantName,
			HydraClientID: oidcClientID, HydraClientSecret: "not-used-for-oidc-config",
			ClientName:   fmt.Sprintf("%s %s OIDC Config", req.TenantName, provider.DisplayName),
			ClientType:   "oidc_provider", ProviderName: provider.ProviderName,
			IsActive: provider.IsActive, CreatedBy: req.CreatedBy, UpdatedBy: req.CreatedBy,
		}
		if err := ac.tenantHydraClientRepo.Create(providerHydraClient); err != nil {
			log.Printf("[oocmgr] Warning: Failed to store OIDC provider mapping for %s: %v", provider.ProviderName, err)
		}

		createdProviders = append(createdProviders, map[string]interface{}{
			"provider_name": provider.ProviderName, "display_name": provider.DisplayName,
			"client_id": oidcClientID, "is_active": provider.IsActive,
			"callback_url": fmt.Sprintf("%s/callback/%s",
				config.AppConfig.IdentityProviderURL, oocmgrNormalizeProviderName(provider.ProviderName)),
		})
	}

	response := map[string]interface{}{
		"success": true, "tenant_id": req.TenantID, "org_id": req.OrgID, "tenant_name": req.TenantName,
		"tenant_client": map[string]interface{}{
			"client_id": tenantClientID, "client_secret": clientSecret,
			"redirect_uris": req.TenantClient.RedirectURIs, "scopes": scopes,
		},
		"oidc_providers": map[string]interface{}{
			"created": createdProviders, "failed": failedProviders,
			"total": len(req.OIDCProviders), "success_count": len(createdProviders), "failed_count": len(failedProviders),
		},
		"login_url": fmt.Sprintf("%s/oauth2/auth?client_id=%s&response_type=code&scope=%s&redirect_uri=",
			ac.hydraConfig.PublicURL, tenantClientID, strings.Join(scopes, "+")),
		"callback_urls": ac.oocmgrGenerateCallbackURLs(req.OIDCProviders),
	}

	statusCode := http.StatusCreated
	message := "OIDC configuration completed successfully"
	if len(failedProviders) > 0 {
		if len(createdProviders) == 0 {
			statusCode = http.StatusInternalServerError
			message = "Failed to create OIDC configuration"
		} else {
			statusCode = http.StatusPartialContent
			message = "OIDC configuration completed with some failures"
		}
	}

	log.Printf("[oocmgr] CompleteOIDCConfiguration tenant=%s providers_created=%d providers_failed=%d", req.TenantID, len(createdProviders), len(failedProviders))

	c.JSON(statusCode, oocmgrdto.MessageResponse{
		Message: message, Success: len(createdProviders) > 0, Data: response, Timestamp: time.Now(),
	})
}

// ===== MANAGEMENT ENDPOINTS =====

func (ac *OocmgrController) UpdateOIDCProvider(c *gin.Context) {
	var req oocmgrUpdateOIDCConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	headerClientID := oocmgrGetClientIDFromHeaders(c)
	providerKey := oocmgrNormalizeProviderName(req.ProviderName)
	oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, providerKey)
	if mapping, err := ac.tenantHydraClientRepo.GetByTenantAndProvider(req.TenantID, req.ProviderName); err == nil && mapping != nil {
		oidcClientID = mapping.HydraClientID
	}

	existingClient, err := ac.getHydraClient(oidcClientID)
	if err != nil {
		existingClient, oidcClientID, err = ac.oocmgrTryResolveHydraClientForUpdate(req, providerKey, headerClientID, err)
		if err != nil {
			c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "OIDC provider not found", Message: err.Error(), Code: http.StatusNotFound, Timestamp: time.Now()})
			return
		}
	}

	metadata := existingClient.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	if providerConfig, ok := metadata["provider_config"].(map[string]interface{}); ok {
		if req.ClientID != nil {
			providerConfig["client_id"] = *req.ClientID
		}
		if req.ClientSecret != nil {
			providerConfig["client_secret"] = *req.ClientSecret
		}
		if req.AuthURL != nil {
			providerConfig["auth_url"] = *req.AuthURL
		}
		if req.TokenURL != nil {
			providerConfig["token_url"] = *req.TokenURL
		}
		if req.UserInfoURL != nil {
			providerConfig["user_info_url"] = *req.UserInfoURL
		}
		if len(req.Scopes) > 0 {
			providerConfig["scopes"] = req.Scopes
		}
		metadata["provider_config"] = providerConfig
	}
	if req.DisplayName != nil {
		metadata["display_name"] = *req.DisplayName
	}
	if req.IsActive != nil {
		metadata["is_active"] = *req.IsActive
	}
	metadata["updated_at"] = time.Now().Format(time.RFC3339)
	metadata["updated_by"] = req.UpdatedBy

	updatedClient := oocmgrHydraClient{
		ClientID: oidcClientID, ClientName: existingClient.ClientName,
		GrantTypes: existingClient.GrantTypes, Metadata: metadata,
	}
	if req.DisplayName != nil {
		tenantName, _ := metadata["tenant_name"].(string)
		if tenantName != "" {
			updatedClient.ClientName = fmt.Sprintf("%s %s OIDC Config", tenantName, *req.DisplayName)
		} else {
			updatedClient.ClientName = fmt.Sprintf("%s OIDC Config", *req.DisplayName)
		}
	}

	if err := ac.updateHydraClient(oidcClientID, updatedClient); err != nil {
		log.Printf("[oocmgr] audit failure: oidc_provider update %s err=%v", req.ProviderName, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to update OIDC provider", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	if req.ClientSecret != nil && *req.ClientSecret != "" {
		if err := ac.oocmgrSaveProviderSecret(req.TenantID, req.ProviderName, *req.ClientSecret); err != nil {
			log.Printf("[oocmgr] Warning: Failed to update provider secret for %s: %v", req.ProviderName, err)
		}
	}

	updateData := map[string]interface{}{"client_name": updatedClient.ClientName, "updated_at": time.Now()}
	if req.IsActive != nil {
		updateData["is_active"] = *req.IsActive
	}
	if req.UpdatedBy != "" {
		updateData["updated_by"] = req.UpdatedBy
	}
	if err := ac.tenantHydraClientRepo.UpdateByHydraClientID(oidcClientID, updateData); err != nil {
		log.Printf("[oocmgr] Warning: failed to update tenant hydra client mapping for %s: %v", oidcClientID, err)
	}

	log.Printf("[oocmgr] UpdateOIDCProvider provider=%s tenant=%s", req.ProviderName, req.TenantID)
	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "OIDC provider updated successfully", Success: true,
		Data: map[string]interface{}{
			"provider_name": req.ProviderName, "client_id": oidcClientID, "updated_at": time.Now(),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetTenantOIDCConfig(c *gin.Context) {
	var req oocmgrGetTenantConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var tenantClient map[string]interface{}
	var oidcProviders []map[string]interface{}

	for _, client := range clients {
		if meta, ok := client.Metadata["tenant_id"].(string); ok && meta == req.TenantID {
			if clientType, ok := client.Metadata["type"].(string); ok {
				switch clientType {
				case "tenant_main_client":
					tenantClient = map[string]interface{}{
						"client_id": client.ClientID, "client_name": client.ClientName,
						"redirect_uris": client.RedirectURIs,
						"scopes":        strings.Split(client.Scope, " "),
						"created_at":    client.Metadata["created_at"],
					}
				case "oidc_provider":
					provider := map[string]interface{}{
						"provider_name": client.Metadata["provider_name"],
						"display_name":  client.Metadata["display_name"],
						"client_id":     client.ClientID,
						"is_active":     client.Metadata["is_active"],
						"sort_order":    client.Metadata["sort_order"],
						"callback_url":  client.Metadata["callback_url"],
						"created_at":    client.Metadata["created_at"],
					}
					if pc, ok := client.Metadata["provider_config"].(map[string]interface{}); ok {
						provider["provider_config"] = pc
					}
					oidcProviders = append(oidcProviders, provider)
				}
			}
		}
	}

	if tenantClient == nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "Tenant configuration not found", Message: "No configuration found for the specified tenant", Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Tenant OIDC configuration retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id":      req.TenantID,
			"tenant_client":  tenantClient,
			"oidc_providers": oidcProviders,
			"provider_count": len(oidcProviders),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetOIDCProvider(c *gin.Context) {
	var req oocmgrGetProviderConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, oocmgrNormalizeProviderName(req.ProviderName))
	client, err := ac.getHydraClient(oidcClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "OIDC provider not found", Message: err.Error(), Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	response := map[string]interface{}{
		"provider_name": client.Metadata["provider_name"],
		"display_name":  client.Metadata["display_name"],
		"client_id":     client.ClientID,
		"is_active":     client.Metadata["is_active"],
		"sort_order":    client.Metadata["sort_order"],
		"callback_url":  client.Metadata["callback_url"],
		"created_at":    client.Metadata["created_at"],
	}
	if pc, ok := client.Metadata["provider_config"].(map[string]interface{}); ok {
		sanitized := make(map[string]interface{})
		for k, v := range pc {
			if k == "client_secret" {
				sanitized[k] = "***hidden***"
			} else {
				sanitized[k] = v
			}
		}
		response["provider_config"] = sanitized
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "OIDC provider retrieved successfully", Success: true, Data: response, Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) DeleteOIDCProvider(c *gin.Context) {
	var req oocmgrDeleteOIDCConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	providerKey := oocmgrNormalizeProviderName(req.ProviderName)
	oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, providerKey)
	if mapping, err := ac.tenantHydraClientRepo.GetByTenantAndProvider(req.TenantID, req.ProviderName); err == nil && mapping != nil {
		oidcClientID = mapping.HydraClientID
	} else if req.ClientID != "" && strings.Contains(req.ClientID, "-"+providerKey+"-oidc") {
		oidcClientID = req.ClientID
	}

	if err := ac.deleteHydraClient(oidcClientID); err != nil {
		log.Printf("[oocmgr] audit failure: oidc_provider delete %s err=%v", req.ProviderName, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to delete OIDC provider", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	if err := ac.oocmgrDeleteProviderSecret(req.TenantID, req.ProviderName); err != nil {
		log.Printf("[oocmgr] Warning: Failed to delete provider secret for %s: %v", req.ProviderName, err)
	}

	if err := ac.tenantHydraClientRepo.DeleteByHydraClientID(oidcClientID); err != nil {
		log.Printf("[oocmgr] Warning: failed to remove tenant hydra client mapping for %s: %v", oidcClientID, err)
	}

	log.Printf("[oocmgr] DeleteOIDCProvider provider=%s tenant=%s", req.ProviderName, req.TenantID)
	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "OIDC provider deleted successfully", Success: true,
		Data: map[string]interface{}{"provider_name": req.ProviderName, "deleted_at": time.Now()},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetProviderSecret(c *gin.Context) {
	var req oocmgrGetProviderSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	secretData, err := config.GetProviderSecretFromVault(req.TenantID, oocmgrNormalizeProviderName(req.ProviderName))
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "Provider secret not found", Message: err.Error(), Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Provider secret retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "provider_name": req.ProviderName, "secret": secretData,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) EditConfig(c *gin.Context) {
	var req oocmgrdto.EditConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantID := req.TenantID
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to connect to tenant database", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}
	c.Set("tenant_db", tenantDB)

	updatedConfig, err := ac.authService.EditConfig(c, &req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case strings.Contains(err.Error(), "validation"):
			status = http.StatusBadRequest
		case strings.Contains(err.Error(), "not found"):
			status = http.StatusNotFound
		case strings.Contains(err.Error(), "mismatch"):
			status = http.StatusBadRequest
		}
		log.Printf("[oocmgr] audit failure: config update %s err=%v", req.ID.String(), err)
		c.JSON(status, oocmgrdto.ErrorResponse{Error: "Failed to edit configuration", Message: err.Error(), Code: status, Timestamp: time.Now()})
		return
	}

	log.Printf("[oocmgr] EditConfig config_id=%s tenant=%s", req.ID.String(), req.TenantID)
	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Configuration updated successfully", Success: true, Data: updatedConfig, Timestamp: time.Now(),
	})
}

// ===== TESTING ENDPOINTS =====

func (ac *OocmgrController) TestOIDCFlow(c *gin.Context) {
	var req oocmgrTestOIDCFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantClientID := fmt.Sprintf("%s-main-client", req.TenantID)
	tenantClient, err := ac.getHydraClient(tenantClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "Tenant client not found", Message: err.Error(), Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	response := map[string]interface{}{
		"tenant_id": req.TenantID, "org_id": req.OrgID, "tenant_client_id": tenantClientID,
		"login_url_template": fmt.Sprintf(
			"%s/oauth2/auth?client_id=%s&response_type=code&scope=%s&redirect_uri={{redirect_uri}}&state={{state}}",
			ac.hydraConfig.PublicURL, tenantClientID, tenantClient.Scope),
	}

	if req.ProviderName != "" {
		callbackURL := fmt.Sprintf("%s/oauth2/callback/%s", ac.hydraConfig.PublicURL, oocmgrNormalizeProviderName(req.ProviderName))
		response["provider_callback_url"] = callbackURL
		oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, oocmgrNormalizeProviderName(req.ProviderName))
		if providerClient, err := ac.getHydraClient(oidcClientID); err == nil {
			response["provider_config"] = map[string]interface{}{
				"provider_name": providerClient.Metadata["provider_name"],
				"display_name":  providerClient.Metadata["display_name"],
				"is_active":     providerClient.Metadata["is_active"],
			}
		}
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "OIDC flow test information retrieved successfully", Success: true, Data: response, Timestamp: time.Now(),
	})
}

// ===== ADDITIONAL HELPER ENDPOINTS =====

func (ac *OocmgrController) GetProviderTemplates(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	templates := map[string]map[string]interface{}{
		"github": {
			"provider_name": "github", "display_name": "GitHub",
			"auth_url": "https://github.com/login/oauth/authorize",
			"token_url": "https://github.com/login/oauth/access_token",
			"user_info_url": "https://api.github.com/user",
			"scopes": []string{"user:email"}, "description": "GitHub OAuth integration",
		},
		"google": {
			"provider_name": "google", "display_name": "Google",
			"auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
			"token_url": "https://oauth2.googleapis.com/token",
			"user_info_url": "https://www.googleapis.com/oauth2/v2/userinfo",
			"scopes": []string{"openid", "profile", "email"}, "description": "Google OAuth 2.0 integration",
		},
		"linkedin": {
			"provider_name": "linkedin", "display_name": "LinkedIn",
			"auth_url": "https://www.linkedin.com/oauth/v2/authorization",
			"token_url": "https://www.linkedin.com/oauth/v2/accessToken",
			"user_info_url": "https://api.linkedin.com/v2/me",
			"scopes": []string{"r_liteprofile", "r_emailaddress"}, "description": "LinkedIn OAuth 2.0 integration",
		},
		"microsoft": {
			"provider_name": "microsoft", "display_name": "Microsoft",
			"auth_url":      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			"token_url":     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			"user_info_url": "https://graph.microsoft.com/v1.0/me",
			"issuer_url":    "https://login.microsoftonline.com/common/v2.0",
			"jwks_url":      "https://login.microsoftonline.com/common/discovery/v2.0/keys",
			"scopes":        []string{"openid", "profile", "email"},
			"description":   "Microsoft Azure AD OAuth 2.0 integration (multi-tenant + personal accounts)",
			"authority_types": map[string]map[string]string{
				"common": {
					"label": "Accounts in any organizational directory and personal Microsoft accounts",
					"auth_url": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
					"token_url": "https://login.microsoftonline.com/common/oauth2/v2.0/token",
					"issuer_url": "https://login.microsoftonline.com/common/v2.0",
					"jwks_url": "https://login.microsoftonline.com/common/discovery/v2.0/keys",
				},
				"organizations": {
					"label": "Accounts in any organizational directory",
					"auth_url": "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize",
					"token_url": "https://login.microsoftonline.com/organizations/oauth2/v2.0/token",
					"issuer_url": "https://login.microsoftonline.com/organizations/v2.0",
					"jwks_url": "https://login.microsoftonline.com/organizations/discovery/v2.0/keys",
				},
				"consumers": {
					"label": "Personal Microsoft accounts only",
					"auth_url": "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize",
					"token_url": "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
					"issuer_url": "https://login.microsoftonline.com/consumers/v2.0",
					"jwks_url": "https://login.microsoftonline.com/consumers/discovery/v2.0/keys",
				},
				"tenant_specific": {
					"label": "Accounts in this organizational directory only (single tenant)",
					"auth_url": "https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/authorize",
					"token_url": "https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token",
					"issuer_url": "https://login.microsoftonline.com/{tenant_id}/v2.0",
					"jwks_url": "https://login.microsoftonline.com/{tenant_id}/discovery/v2.0/keys",
				},
			},
		},
		"okta": {
			"provider_name": "okta", "display_name": "Okta",
			"auth_url": "https://{your-okta-domain}/oauth2/v1/authorize",
			"token_url": "https://{your-okta-domain}/oauth2/v1/token",
			"user_info_url": "https://{your-okta-domain}/oauth2/v1/userinfo",
			"scopes": []string{"openid", "profile", "email"}, "description": "Okta OAuth 2.0 integration",
		},
		"auth0": {
			"provider_name": "auth0", "display_name": "Auth0",
			"auth_url": "https://{your-auth0-domain}/authorize",
			"token_url": "https://{your-auth0-domain}/oauth/token",
			"user_info_url": "https://{your-auth0-domain}/userinfo",
			"scopes": []string{"openid", "profile", "email"}, "description": "Auth0 OAuth 2.0 integration",
		},
		"custom": {
			"provider_name": "custom-provider", "display_name": "Custom Provider",
			"auth_url": "https://your-provider.com/oauth/authorize",
			"token_url": "https://your-provider.com/oauth/token",
			"user_info_url": "https://your-provider.com/oauth/userinfo",
			"scopes": []string{"openid", "profile", "email"}, "description": "Template for custom OAuth 2.0 provider",
		},
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Provider templates retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"templates": templates, "count": len(templates),
			"instructions": "Use these templates as starting points for configuring OIDC providers.",
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) ValidateOIDCConfig(c *gin.Context) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		OrgID        string `json:"org_id"`
		ProviderName string `json:"provider_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, oocmgrNormalizeProviderName(req.ProviderName))
	client, err := ac.getHydraClient(oidcClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "OIDC provider not found", Message: err.Error(), Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	var errs []string
	var warnings []string

	pc, ok := client.Metadata["provider_config"].(map[string]interface{})
	if !ok {
		errs = append(errs, "Provider configuration not found in metadata")
	} else {
		if v, _ := pc["client_id"].(string); v == "" {
			errs = append(errs, "Client ID is required")
		}
		if v, _ := pc["client_secret"].(string); v == "" {
			errs = append(errs, "Client Secret is required")
		}
		if v, _ := pc["auth_url"].(string); v == "" {
			errs = append(errs, "Auth URL is required")
		}
		if v, _ := pc["token_url"].(string); v == "" {
			errs = append(errs, "Token URL is required")
		}
		if v, _ := pc["user_info_url"].(string); v == "" {
			errs = append(errs, "User Info URL is required")
		}
		if sc, _ := pc["scopes"].([]interface{}); len(sc) == 0 {
			errs = append(errs, "At least one scope is required")
		}
		if isActive, ok := client.Metadata["is_active"].(bool); !ok || !isActive {
			warnings = append(warnings, "Provider is not currently active")
		}
	}

	message := "Validation completed successfully"
	if len(errs) > 0 {
		message = "Validation failed with errors"
	} else if len(warnings) > 0 {
		message = "Validation passed with warnings"
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: message, Success: len(errs) == 0,
		Data: map[string]interface{}{
			"provider_name": req.ProviderName, "client_id": oidcClientID,
			"is_valid": len(errs) == 0, "errors": errs, "warnings": warnings,
			"validated_at": time.Now(),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetClientsByTenant(c *gin.Context) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		ActiveOnly bool   `json:"active_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to connect to tenant database", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	type clientRow struct {
		ClientID string `gorm:"column:client_id"`
	}
	var rows []clientRow
	query := tenantDB.Table("clients").Where("tenant_id = ?", req.TenantID)
	if req.ActiveOnly {
		query = query.Where("active = ?", true)
	}
	if err := query.Select("client_id").Order("created_at ASC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to query client IDs", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	clientIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		clientIDs = append(clientIDs, r.ClientID)
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Client IDs retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "client_ids": clientIDs,
			"count": len(clientIDs), "active_only": req.ActiveOnly,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetTenantStats(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get client information", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var tenantClientCount, oidcProviderCount, activeProviders, inactiveProviders int
	providersByType := make(map[string]int)

	for _, client := range clients {
		if tenantID, ok := client.Metadata["tenant_id"].(string); ok && tenantID == req.TenantID {
			if orgID, ok := client.Metadata["org_id"].(string); ok && orgID == req.OrgID {
				if clientType, ok := client.Metadata["type"].(string); ok {
					switch clientType {
					case "tenant_main_client":
						tenantClientCount++
					case "oidc_provider":
						oidcProviderCount++
						if isActive, ok := client.Metadata["is_active"].(bool); ok && isActive {
							activeProviders++
						} else {
							inactiveProviders++
						}
						if pn, ok := client.Metadata["provider_name"].(string); ok {
							providersByType[pn]++
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Tenant statistics retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "org_id": req.OrgID,
			"tenant_clients": tenantClientCount, "total_providers": oidcProviderCount,
			"active_providers": activeProviders, "inactive_providers": inactiveProviders,
			"providers_by_type": providersByType, "last_updated": time.Now(),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) DeleteCompleteTenantConfig(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		ClientID string `json:"client_id"`
		Force    bool   `json:"force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var deletedClients []string
	var failedDeletions []map[string]interface{}

	for _, client := range clients {
		if tenantID, ok := client.Metadata["c_id"].(string); ok && tenantID == req.TenantID {
			if orgID, ok := client.Metadata["tenant_id"].(string); ok && orgID == req.ClientID {
				if err := ac.deleteHydraClient(client.ClientID); err != nil {
					if !req.Force {
						c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{
							Error: "Failed to delete client",
							Message: fmt.Sprintf("Failed to delete client %s: %v", client.ClientID, err),
							Code: http.StatusInternalServerError, Timestamp: time.Now(),
						})
						return
					}
					failedDeletions = append(failedDeletions, map[string]interface{}{"client_id": client.ClientID, "error": err.Error()})
				} else {
					deletedClients = append(deletedClients, client.ClientID)
				}
			}
		}
	}

	if len(deletedClients) == 0 && len(failedDeletions) == 0 {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "Tenant configuration not found", Message: "No configuration found for the specified tenant", Code: http.StatusNotFound, Timestamp: time.Now()})
		return
	}

	statusCode := http.StatusOK
	message := "Tenant configuration deleted successfully"
	if len(failedDeletions) > 0 {
		if len(deletedClients) == 0 {
			statusCode = http.StatusInternalServerError
			message = "Failed to delete tenant configuration"
		} else {
			statusCode = http.StatusPartialContent
			message = "Tenant configuration partially deleted"
		}
	}

	log.Printf("[oocmgr] DeleteCompleteTenantConfig tenant=%s deleted=%d failed=%d", req.TenantID, len(deletedClients), len(failedDeletions))
	c.JSON(statusCode, oocmgrdto.MessageResponse{
		Message: message, Success: len(deletedClients) > 0,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "org_id": req.ClientID,
			"deleted_clients": deletedClients, "failed_deletions": failedDeletions,
			"deleted_count": len(deletedClients), "failed_count": len(failedDeletions),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) ListAllTenants(c *gin.Context) {
	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	tenants := make(map[string]map[string]interface{})
	for _, client := range clients {
		tenantID, ok1 := client.Metadata["tenant_id"].(string)
		orgID, ok2 := client.Metadata["org_id"].(string)
		if !ok1 || !ok2 {
			continue
		}
		tenantKey := fmt.Sprintf("%s:%s", tenantID, orgID)
		if _, exists := tenants[tenantKey]; !exists {
			tenants[tenantKey] = map[string]interface{}{
				"tenant_id": tenantID, "org_id": orgID,
				"tenant_name": client.Metadata["tenant_name"],
				"main_client": nil, "oidc_providers": []map[string]interface{}{},
				"total_clients": 0, "active_providers": 0,
			}
		}
		tenant := tenants[tenantKey]
		tenant["total_clients"] = tenant["total_clients"].(int) + 1

		if clientType, ok := client.Metadata["type"].(string); ok {
			switch clientType {
			case "tenant_main_client":
				tenant["main_client"] = map[string]interface{}{
					"client_id": client.ClientID, "client_name": client.ClientName,
					"created_at": client.Metadata["created_at"],
				}
			case "oidc_provider":
				provider := map[string]interface{}{
					"provider_name": client.Metadata["provider_name"],
					"display_name":  client.Metadata["display_name"],
					"client_id":     client.ClientID,
					"is_active":     client.Metadata["is_active"],
					"sort_order":    client.Metadata["sort_order"],
				}
				providers := tenant["oidc_providers"].([]map[string]interface{})
				tenant["oidc_providers"] = append(providers, provider)
				if isActive, ok := client.Metadata["is_active"].(bool); ok && isActive {
					tenant["active_providers"] = tenant["active_providers"].(int) + 1
				}
			}
		}
	}

	var tenantList []map[string]interface{}
	for _, tenant := range tenants {
		tenantList = append(tenantList, tenant)
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Tenants listed successfully", Success: true,
		Data: map[string]interface{}{"tenants": tenantList, "count": len(tenantList)},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) CheckTenantExists(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	mainClientID := fmt.Sprintf("%s-main-client", req.TenantID)
	client, err := ac.getHydraClient(mainClientID)
	exists := err == nil && client != nil

	var tenantInfo map[string]interface{}
	if exists {
		tenantInfo = map[string]interface{}{
			"tenant_id": req.TenantID, "org_id": req.OrgID,
			"tenant_name": client.Metadata["tenant_name"],
			"client_id":   client.ClientID, "client_name": client.ClientName,
			"created_at": client.Metadata["created_at"],
		}
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Tenant existence check completed", Success: true,
		Data: map[string]interface{}{
			"exists": exists, "tenant_id": req.TenantID, "org_id": req.OrgID, "tenant_info": tenantInfo,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) UpdateCompleteTenantConfig(c *gin.Context) {
	var req struct {
		TenantID      string                      `json:"tenant_id"`
		OrgID         string                      `json:"org_id"`
		TenantName    *string                     `json:"tenant_name,omitempty"`
		TenantClient  *oocmgrTenantClientConfig   `json:"tenant_client,omitempty"`
		OIDCProviders []oocmgrOIDCProviderConfig  `json:"oidc_providers,omitempty"`
		UpdatedBy     string                      `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	mainClientID := fmt.Sprintf("%s-main-client", req.TenantID)
	existingMainClient, err := ac.getHydraClient(mainClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{
			Error: "Tenant not found", Message: fmt.Sprintf("Tenant with ID %s does not exist", req.TenantID),
			Code: http.StatusNotFound, Timestamp: time.Now(),
		})
		return
	}

	var updatedMainClient *oocmgrHydraClient
	var updatedProviders []map[string]interface{}
	var failedProviders []map[string]interface{}

	if req.TenantClient != nil {
		grantTypes := req.TenantClient.GrantTypes
		if len(grantTypes) == 0 {
			grantTypes = []string{"authorization_code", "refresh_token"}
		}
		scopes := req.TenantClient.Scopes
		if len(scopes) == 0 {
			scopes = []string{"openid", "profile", "email", "offline_access"}
		}
		tenantName, _ := existingMainClient.Metadata["tenant_name"].(string)
		if req.TenantName != nil {
			tenantName = *req.TenantName
		}
		updatedMainClient = &oocmgrHydraClient{
			ClientID: mainClientID, ClientName: req.TenantClient.ClientName,
			GrantTypes: grantTypes, RedirectURIs: req.TenantClient.RedirectURIs,
			ResponseTypes: []string{"code"}, Scope: strings.Join(scopes, " "),
			Metadata: map[string]interface{}{
				"type": "tenant_main_client", "tenant_id": req.TenantID, "org_id": req.OrgID,
				"tenant_name": tenantName, "created_at": existingMainClient.Metadata["created_at"],
				"updated_at": time.Now().Format(time.RFC3339), "updated_by": req.UpdatedBy,
			},
		}
		if err := ac.updateHydraClient(mainClientID, *updatedMainClient); err != nil {
			c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to update tenant client", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
			return
		}
	}

	if len(req.OIDCProviders) > 0 {
		existingProviders, _ := ac.getOIDCProvidersForTenant(req.TenantID)
		existingProviderMap := make(map[string]bool)
		for _, p := range existingProviders {
			existingProviderMap[p.ProviderName] = true
		}

		tenantName, _ := existingMainClient.Metadata["tenant_name"].(string)
		if req.TenantName != nil {
			tenantName = *req.TenantName
		}

		for _, provider := range req.OIDCProviders {
			oidcClientID := fmt.Sprintf("%s-%s-oidc", req.TenantID, oocmgrNormalizeProviderName(provider.ProviderName))
			oidcClient := oocmgrHydraClient{
				ClientID: oidcClientID,
				ClientName: fmt.Sprintf("%s %s OIDC Config", tenantName, provider.DisplayName),
				GrantTypes: []string{"client_credentials"},
				Metadata: map[string]interface{}{
					"type": "oidc_provider", "tenant_id": req.TenantID, "org_id": req.OrgID,
					"provider_name": provider.ProviderName, "display_name": provider.DisplayName,
					"provider_config": map[string]interface{}{
						"client_id": provider.ClientID, "client_secret": provider.ClientSecret,
						"auth_url": provider.AuthURL, "token_url": provider.TokenURL,
						"user_info_url": provider.UserInfoURL, "scopes": provider.Scopes,
						"issuer_url": provider.IssuerURL, "jwks_url": provider.JWKsURL,
						"additional_params": provider.AdditionalParams,
					},
					"is_active": provider.IsActive, "sort_order": provider.SortOrder,
					"callback_url": fmt.Sprintf("%s/oauth2/callback/%s",
						ac.hydraConfig.PublicURL, oocmgrNormalizeProviderName(provider.ProviderName)),
					"updated_at": time.Now().Format(time.RFC3339), "updated_by": req.UpdatedBy,
				},
			}

			var opErr error
			if existingProviderMap[provider.ProviderName] {
				opErr = ac.updateHydraClient(oidcClientID, oidcClient)
			} else {
				opErr = ac.createHydraClient(oidcClient)
			}

			if opErr != nil {
				action := "update"
				if !existingProviderMap[provider.ProviderName] {
					action = "create"
				}
				failedProviders = append(failedProviders, map[string]interface{}{
					"provider_name": provider.ProviderName, "action": action, "error": opErr.Error(),
				})
				continue
			}

			action := "created"
			if existingProviderMap[provider.ProviderName] {
				action = "updated"
			}
			updatedProviders = append(updatedProviders, map[string]interface{}{
				"provider_name": provider.ProviderName, "display_name": provider.DisplayName,
				"client_id": oidcClientID, "is_active": provider.IsActive, "action": action,
			})
		}
	}

	response := map[string]interface{}{
		"success": true, "tenant_id": req.TenantID, "org_id": req.OrgID,
		"updated_at": time.Now(), "updated_by": req.UpdatedBy,
	}
	if updatedMainClient != nil {
		response["tenant_client"] = map[string]interface{}{
			"client_id": updatedMainClient.ClientID, "client_name": updatedMainClient.ClientName,
			"redirect_uris": updatedMainClient.RedirectURIs,
			"scopes": strings.Split(updatedMainClient.Scope, " "), "updated": true,
		}
	}
	if len(req.OIDCProviders) > 0 {
		response["oidc_providers"] = map[string]interface{}{
			"updated": updatedProviders, "failed": failedProviders,
			"success_count": len(updatedProviders), "failed_count": len(failedProviders),
		}
	}

	statusCode := http.StatusOK
	message := "Tenant configuration updated successfully"
	if len(failedProviders) > 0 {
		if len(updatedProviders) == 0 {
			statusCode = http.StatusInternalServerError
			message = "Failed to update tenant configuration"
		} else {
			statusCode = http.StatusPartialContent
			message = "Tenant configuration partially updated"
		}
	}

	log.Printf("[oocmgr] UpdateCompleteTenantConfig tenant=%s providers_updated=%d failed=%d", req.TenantID, len(updatedProviders), len(failedProviders))
	c.JSON(statusCode, oocmgrdto.MessageResponse{
		Message: message, Success: len(updatedProviders) > 0 || updatedMainClient != nil,
		Data: response, Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetTenantLoginPageData(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	mainClientID := fmt.Sprintf("%s-main-client", req.TenantID)
	tenantClient, err := ac.getHydraClient(mainClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{
			Error: "Tenant not found", Message: fmt.Sprintf("No client found for tenant %s", req.TenantID),
			Code: http.StatusNotFound, Timestamp: time.Now(),
		})
		return
	}

	providers, err := ac.getOIDCProvidersForTenant(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get providers", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var activeProviders []map[string]interface{}
	for _, provider := range providers {
		if provider.IsActive {
			activeProviders = append(activeProviders, map[string]interface{}{
				"provider_name": provider.ProviderName, "display_name": provider.DisplayName,
				"sort_order": provider.SortOrder,
			})
		}
	}

	sort.Slice(activeProviders, func(i, j int) bool {
		return activeProviders[i]["sort_order"].(int) < activeProviders[j]["sort_order"].(int)
	})

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Login page data retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "org_id": req.OrgID,
			"tenant_name": tenantClient.Metadata["tenant_name"],
			"client_name": tenantClient.ClientName, "main_client_id": tenantClient.ClientID,
			"providers": activeProviders, "provider_count": len(activeProviders),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) CreateBaseTenantClient(c *gin.Context) {
	var req struct {
		TenantID     string   `json:"tenant_id"`
		TenantName   string   `json:"tenant_name"`
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes,omitempty"`
		CreatedBy    string   `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	existingClient, _ := ac.getHydraClient(req.ClientID)
	if existingClient != nil {
		c.JSON(http.StatusConflict, oocmgrdto.ErrorResponse{
			Error: "Tenant client already exists",
			Message: fmt.Sprintf("Client with ID %s already exists in Hydra", req.ClientID),
			Code: http.StatusConflict, Timestamp: time.Now(),
		})
		return
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email", "offline_access"}
	}
	for i, scope := range scopes {
		if scope == "offline" {
			scopes[i] = "offline_access"
		}
	}

	tenantClient := oocmgrHydraClient{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret,
		ClientName:    fmt.Sprintf("%s Main OAuth Client", req.TenantName),
		GrantTypes:    []string{"authorization_code", "refresh_token"},
		RedirectURIs:  req.RedirectURIs, TokenEndpoint: "client_secret_post",
		ResponseTypes: []string{"code"}, Scope: strings.Join(scopes, " "),
		Audience: []string{}, SubjectType: "public",
		Metadata: map[string]interface{}{
			"type": "tenant_main_client",
			"tenant_id": strings.TrimSuffix(req.ClientID, "-main-client"),
			"c_id":      strings.TrimSuffix(req.TenantID, "-main-client"),
			"tenant_name": req.TenantName,
			"created_at":  time.Now().Format(time.RFC3339), "created_by": req.CreatedBy,
		},
	}

	if err := ac.createHydraClient(tenantClient); err != nil {
		log.Printf("[oocmgr] audit failure: tenant_client create %s err=%v", req.TenantID, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to create tenant client", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	thc := &oocmgrdto.TenantHydraClient{
		TenantID: req.TenantID, TenantName: req.TenantName,
		HydraClientID: req.ClientID, HydraClientSecret: req.ClientSecret,
		ClientName: fmt.Sprintf("%s Main OAuth Client", req.TenantName),
		RedirectURIs: req.RedirectURIs, Scopes: scopes,
		ClientType: "main", IsActive: true, CreatedBy: req.CreatedBy, UpdatedBy: req.CreatedBy,
	}
	if err := ac.tenantHydraClientRepo.Create(thc); err != nil {
		log.Printf("[oocmgr] Warning: Failed to store tenant-client mapping: %v", err)
	}

	log.Printf("[oocmgr] CreateBaseTenantClient client_id=%s tenant=%s", req.ClientID, req.TenantID)
	c.JSON(http.StatusCreated, oocmgrdto.MessageResponse{
		Message: "Tenant base client created successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_name": req.TenantName, "client_id": req.ClientID,
			"client_secret": req.ClientSecret, "redirect_uris": req.RedirectURIs,
			"scopes": scopes, "created_at": time.Now(),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) AddOIDCProviderToTenant(c *gin.Context) {
	var req struct {
		TenantID    string                   `json:"tenant_id"`
		ClientID    string                   `json:"client_id"`
		Provider    oocmgrOIDCProviderConfig `json:"provider"`
		ReactAppURL string                   `json:"react_app_url"`
		CreatedBy   string                   `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	baseClientID := strings.TrimSuffix(req.ClientID, "-main-client")
	if baseClientID == "" {
		baseClientID = req.ClientID
	}
	mainClientID := fmt.Sprintf("%s-main-client", baseClientID)
	tenantClient, err := ac.getHydraClient(mainClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{
			Error: "Tenant not found",
			Message: fmt.Sprintf("No base client found for tenant %s. Create base client first.", req.ClientID),
			Code: http.StatusNotFound, Timestamp: time.Now(),
		})
		return
	}

	oidcClientID := fmt.Sprintf("%s-%s-oidc", baseClientID, oocmgrNormalizeProviderName(req.Provider.ProviderName))

	existingProvider, _ := ac.getHydraClient(oidcClientID)
	if existingProvider != nil {
		isDuplicate := false
		if pc, ok := existingProvider.Metadata["provider_config"].(map[string]interface{}); ok {
			if existingCID, ok := pc["client_id"].(string); ok && existingCID == req.Provider.ClientID {
				isDuplicate = true
			}
		}
		if isDuplicate {
			c.JSON(http.StatusConflict, oocmgrdto.ErrorResponse{
				Error: "Provider already exists",
				Message: fmt.Sprintf("Provider %s with client_id %s already exists for tenant %s",
					req.Provider.ProviderName, req.Provider.ClientID, req.TenantID),
				Code: http.StatusConflict, Timestamp: time.Now(),
			})
			return
		}
	}

	tenantName, _ := tenantClient.Metadata["tenant_name"].(string)

	// Auto-resolve Microsoft authority URLs if authority_type is specified
	oocmgrResolveMicrosoftAuthorityURLs(&req.Provider)

	oidcClient := oocmgrHydraClient{
		ClientID: oidcClientID,
		ClientName: fmt.Sprintf("%s %s OIDC Config", tenantName, req.Provider.DisplayName),
		GrantTypes: []string{"client_credentials"},
		Metadata: map[string]interface{}{
			"type": "oidc_provider", "tenant_id": baseClientID, "c_id": req.TenantID,
			"provider_name": req.Provider.ProviderName, "display_name": req.Provider.DisplayName,
			"provider_config": map[string]interface{}{
				"client_id": req.Provider.ClientID, "client_secret": req.Provider.ClientSecret,
				"auth_url": req.Provider.AuthURL, "token_url": req.Provider.TokenURL,
				"user_info_url": req.Provider.UserInfoURL, "scopes": req.Provider.Scopes,
				"issuer_url": req.Provider.IssuerURL, "jwks_url": req.Provider.JWKsURL,
				"additional_params": req.Provider.AdditionalParams,
			},
			"is_active":    req.Provider.IsActive,
			"sort_order":   req.Provider.SortOrder,
			"callback_url": fmt.Sprintf("%s/oidc/auth/callback/%s", req.ReactAppURL, oocmgrNormalizeProviderName(req.Provider.ProviderName)),
			"created_at":   time.Now().Format(time.RFC3339), "created_by": req.CreatedBy,
		},
	}

	if err := ac.createHydraClient(oidcClient); err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to add OIDC provider", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	if req.Provider.ClientSecret != "" {
		if err := ac.oocmgrSaveProviderSecret(req.TenantID, req.Provider.ProviderName, req.Provider.ClientSecret); err != nil {
			log.Printf("[oocmgr] Warning: Failed to store provider secret for %s: %v", req.Provider.ProviderName, err)
		}
	}

	thc := &oocmgrdto.TenantHydraClient{
		TenantID: req.TenantID, TenantName: tenantName,
		HydraClientID: oidcClientID, HydraClientSecret: "not-used-for-oidc-config",
		ClientName:   fmt.Sprintf("%s %s OIDC Config", tenantName, req.Provider.DisplayName),
		ClientType:   "oidc_provider", ProviderName: req.Provider.ProviderName,
		IsActive: req.Provider.IsActive, CreatedBy: req.CreatedBy, UpdatedBy: req.CreatedBy,
	}
	if err := ac.tenantHydraClientRepo.Create(thc); err != nil {
		log.Printf("[oocmgr] Warning: Failed to store OIDC provider mapping for %s: %v", req.Provider.ProviderName, err)
	}

	log.Printf("[oocmgr] AddOIDCProviderToTenant provider=%s tenant=%s", req.Provider.ProviderName, req.TenantID)
	c.JSON(http.StatusCreated, oocmgrdto.MessageResponse{
		Message: "OIDC provider added successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "provider_name": req.Provider.ProviderName,
			"display_name": req.Provider.DisplayName, "client_id": oidcClientID,
			"callback_url": fmt.Sprintf("%s/oidc/auth/callback/%s", req.ReactAppURL, oocmgrNormalizeProviderName(req.Provider.ProviderName)),
			"is_active": req.Provider.IsActive, "created_at": time.Now(),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) GetTenantHydraClients(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.tenantHydraClientRepo.GetByTenantID(req.TenantID, req.OrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get tenant clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var mainClient *oocmgrdto.TenantHydraClientResponse
	var providerClients []oocmgrdto.TenantHydraClientResponse

	for _, client := range clients {
		resp := oocmgrdto.TenantHydraClientResponse{
			ID: client.ID, OrgID: client.OrgID, TenantID: client.TenantID,
			TenantName: client.TenantName, HydraClientID: client.HydraClientID,
			HydraClientSecret: client.HydraClientSecret, ClientName: client.ClientName,
			RedirectURIs: client.RedirectURIs, Scopes: client.Scopes,
			ClientType: client.ClientType, ProviderName: client.ProviderName,
			IsActive: client.IsActive, CreatedAt: client.CreatedAt, UpdatedAt: client.UpdatedAt,
			CreatedBy: client.CreatedBy, UpdatedBy: client.UpdatedBy,
		}
		if client.ClientType == "main" {
			mainClient = &resp
		} else {
			providerClients = append(providerClients, resp)
		}
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Tenant Hydra clients retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"main_client": mainClient, "provider_clients": providerClients, "total_clients": len(clients),
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) SyncHydraClients(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id,omitempty"`
		OrgID    string `json:"org_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	hydraClients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	syncedCount, missingCount := 0, 0
	for _, hClient := range hydraClients {
		if _, err := ac.tenantHydraClientRepo.GetByHydraClientID(hClient.ClientID); err != nil {
			log.Printf("[oocmgr] Hydra client %s not found in database mappings", hClient.ClientID)
			missingCount++
		} else {
			syncedCount++
		}
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Hydra clients sync completed", Success: true,
		Data: map[string]interface{}{
			"total_hydra_clients": len(hydraClients),
			"synced_clients": syncedCount, "missing_mappings": missingCount,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) ListTenantHydraClients(c *gin.Context) {
	var req oocmgrdto.GetTenantHydraClientsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.tenantHydraClientRepo.ListAll(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to list clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var responses []oocmgrdto.TenantHydraClientResponse
	for _, client := range clients {
		resp := oocmgrdto.TenantHydraClientResponse{
			ID: client.ID, OrgID: client.OrgID, TenantID: client.TenantID,
			TenantName: client.TenantName, HydraClientID: client.HydraClientID,
			ClientName: client.ClientName, RedirectURIs: client.RedirectURIs,
			Scopes: client.Scopes, ClientType: client.ClientType,
			ProviderName: client.ProviderName, IsActive: client.IsActive,
			CreatedAt: client.CreatedAt, UpdatedAt: client.UpdatedAt,
			CreatedBy: client.CreatedBy, UpdatedBy: client.UpdatedBy,
		}
		if req.TenantID != "" && req.OrgID != "" {
			resp.HydraClientSecret = client.HydraClientSecret
		}
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Hydra clients retrieved successfully", Success: true, Data: responses, Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) DumpHydraRawData(c *gin.Context) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		ClientType string `json:"client_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "tenant_id is required", Message: "Provide a tenant_id to dump Hydra data.", Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to query Hydra", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	clientTypeFilter := strings.TrimSpace(req.ClientType)
	var dump []map[string]interface{}

	for _, client := range clients {
		matchesTenant := oocmgrBelongsToTenant(client.Metadata, tenantID)
		if !matchesTenant && strings.HasPrefix(strings.ToLower(client.ClientID), strings.ToLower(tenantID)) {
			matchesTenant = true
		}
		if !matchesTenant {
			continue
		}
		if clientTypeFilter != "" {
			if ct, _ := client.Metadata["type"].(string); ct != clientTypeFilter {
				continue
			}
		}
		dump = append(dump, oocmgrSanitizeHydraClientForDump(client))
	}

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Hydra client dump retrieved successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": tenantID, "client_type": clientTypeFilter,
			"count": len(dump), "clients": dump,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) ShowAuthProviders(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id"`
		ClientID string `json:"client_id"`
	}
	if c.Request.Method == http.MethodGet {
		req.TenantID = c.Query("tenant_id")
		req.ClientID = c.Query("client_id")
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
			return
		}
	}

	if req.ClientID == "" {
		for _, key := range []string{"Client-Id", "client-id", "X-Client-Id"} {
			if v := strings.TrimSpace(c.GetHeader(key)); v != "" {
				req.ClientID = v
				break
			}
		}
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	var filteredClients []oocmgrHydraClient
	for _, client := range clients {
		if meta, ok := client.Metadata["tenant_id"].(string); ok {
			if meta == req.TenantID || client.Metadata["c_id"] == req.TenantID {
				if clientType, ok := client.Metadata["type"].(string); ok && clientType == "oidc_provider" {
					filteredClients = append(filteredClients, client)
				}
			}
		}
	}

	if req.ClientID == "" {
		type providerEntry struct {
			ProviderName string
			DisplayName  string
			IsActive     bool
			ClientIDs    []string
			SortOrder    int
		}
		providerMap := make(map[string]*providerEntry)

		for _, client := range filteredClients {
			providerName, _ := client.Metadata["provider_name"].(string)
			displayName, _ := client.Metadata["display_name"].(string)
			isActive, _ := client.Metadata["is_active"].(bool)
			sortOrder := 99
			if sof, ok := client.Metadata["sort_order"].(float64); ok {
				sortOrder = int(sof)
			}
			var serviceClientID string
			if pc, ok := client.Metadata["provider_config"].(map[string]interface{}); ok {
				if cid, ok := pc["client_id"].(string); ok {
					serviceClientID = cid
				}
			}

			if providerName != "" {
				if _, exists := providerMap[providerName]; !exists {
					providerMap[providerName] = &providerEntry{
						ProviderName: providerName, DisplayName: displayName,
						IsActive: isActive, SortOrder: sortOrder, ClientIDs: []string{},
					}
				}
				if serviceClientID != "" {
					found := false
					for _, eid := range providerMap[providerName].ClientIDs {
						if eid == serviceClientID {
							found = true
							break
						}
					}
					if !found {
						providerMap[providerName].ClientIDs = append(providerMap[providerName].ClientIDs, serviceClientID)
					}
				}
			}
		}

		var providers []map[string]interface{}
		if _, exists := providerMap["authsec"]; !exists {
			providers = append(providers, map[string]interface{}{
				"provider_name": "authsec", "display_name": "AuthSec", "is_active": true,
				"sort_order": 0, "provider_type": "internal", "client_ids": "",
			})
		}
		for _, p := range providerMap {
			pt := "oidc"
			if p.ProviderName == "authsec" {
				pt = "internal"
			}
			providers = append(providers, map[string]interface{}{
				"provider_name": p.ProviderName, "display_name": p.DisplayName,
				"is_active": p.IsActive, "sort_order": p.SortOrder,
				"client_ids": strings.Join(p.ClientIDs, ","), "provider_type": pt,
			})
		}
		sort.Slice(providers, func(i, j int) bool {
			return providers[i]["sort_order"].(int) < providers[j]["sort_order"].(int)
		})

		c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
			Message: "Auth providers retrieved successfully (aggregated view)", Success: true,
			Data: map[string]interface{}{
				"tenant_id": req.TenantID, "count": len(providers), "providers": providers,
			},
			Timestamp: time.Now(),
		})
		return
	}

	// Per-client view
	baseClientID := strings.TrimSuffix(req.TenantID, "-main-client")
	if baseClientID == "" {
		baseClientID = req.TenantID
	}

	tenantConfigs := make(map[string]oocmgrHydraClient)
	clientConfigs := make(map[string]bool)

	for _, client := range filteredClients {
		cID, _ := client.Metadata["c_id"].(string)
		if cID != req.TenantID {
			continue
		}
		providerName, _ := client.Metadata["provider_name"].(string)
		if providerName == "" {
			continue
		}
		if tenantID, ok := client.Metadata["tenant_id"].(string); ok {
			if tenantID == baseClientID {
				if _, exists := tenantConfigs[providerName]; !exists {
					tenantConfigs[providerName] = client
				}
				if req.ClientID == baseClientID {
					clientConfigs[providerName] = true
				}
			} else if tenantID == req.ClientID {
				clientConfigs[providerName] = true
			}
		}
	}

	var matchedProviders []map[string]interface{}
	for providerName, tenantConfig := range tenantConfigs {
		isActive := clientConfigs[providerName]
		var sanitizedConfig map[string]interface{}
		if pc, ok := tenantConfig.Metadata["provider_config"].(map[string]interface{}); ok {
			sanitizedConfig = make(map[string]interface{})
			for k, v := range pc {
				if k == "client_secret" {
					sanitizedConfig[k] = "***hidden***"
				} else {
					sanitizedConfig[k] = v
				}
			}
		}
		hydraClientID := ""
		if isActive {
			hydraClientID = fmt.Sprintf("%s-%s-oidc", req.ClientID, oocmgrNormalizeProviderName(providerName))
		}
		matchedProviders = append(matchedProviders, map[string]interface{}{
			"hydra_client_id": hydraClientID,
			"provider_name":   tenantConfig.Metadata["provider_name"],
			"display_name":    tenantConfig.Metadata["display_name"],
			"client_id":       req.ClientID, "is_active": isActive,
			"sort_order":      tenantConfig.Metadata["sort_order"],
			"callback_url":    tenantConfig.Metadata["callback_url"],
			"provider_config": sanitizedConfig,
		})
	}

	sort.Slice(matchedProviders, func(i, j int) bool {
		iOrder, iOk := matchedProviders[i]["sort_order"].(int)
		jOrder, jOk := matchedProviders[j]["sort_order"].(int)
		if !iOk {
			iOrder = 99
		}
		if !jOk {
			jOrder = 99
		}
		return iOrder < jOrder
	})

	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Auth providers retrieved successfully (filtered by client_id)", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "client_id": req.ClientID,
			"count": len(matchedProviders), "providers": matchedProviders,
		},
		Timestamp: time.Now(),
	})
}

func (ac *OocmgrController) EditAuthProvider(c *gin.Context) {
	var req editAuthProviderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "Invalid request", Message: err.Error(), Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "client_id is required", Message: "client_id parameter must be provided", Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}
	if req.ProviderName == "" {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "provider_name is required", Message: "provider_name parameter must be provided", Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	clients, err := ac.getAllHydraClients()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to get Hydra clients", Message: err.Error(), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	hydraClientID := fmt.Sprintf("%s-%s-oidc", req.ClientID, oocmgrNormalizeProviderName(req.ProviderName))
	var matchedClient *oocmgrHydraClient
	for i, client := range clients {
		if client.ClientID == hydraClientID {
			if meta, ok := client.Metadata["tenant_id"].(string); ok && meta == req.ClientID {
				if ct, ok := client.Metadata["type"].(string); ok && ct == "oidc_provider" {
					matchedClient = &clients[i]
					break
				}
			}
		}
	}

	// is_active = false → DELETE
	if req.IsActive != nil && !*req.IsActive {
		if matchedClient == nil {
			c.JSON(http.StatusNotFound, oocmgrdto.ErrorResponse{Error: "Provider not found", Message: "No provider found to deactivate", Code: http.StatusNotFound, Timestamp: time.Now()})
			return
		}
		if strings.ToLower(req.ProviderName) == "authsec" {
			c.JSON(http.StatusForbidden, oocmgrdto.ErrorResponse{Error: "Cannot delete AuthSec provider", Message: "AuthSec is the internal authentication provider and cannot be deleted", Code: http.StatusForbidden, Timestamp: time.Now()})
			return
		}
		baseClientID := strings.TrimSuffix(req.TenantID, "-main-client")
		if baseClientID == "" {
			baseClientID = req.TenantID
		}
		if req.ClientID == baseClientID {
			c.JSON(http.StatusForbidden, oocmgrdto.ErrorResponse{Error: "Cannot delete base client config", Message: "This is the base client configuration and cannot be deactivated", Code: http.StatusForbidden, Timestamp: time.Now()})
			return
		}
		if err := ac.deleteHydraClient(matchedClient.ClientID); err != nil {
			log.Printf("[oocmgr] audit failure: auth_provider delete %s err=%v", req.ProviderName, err)
			c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Failed to delete provider", Message: fmt.Sprintf("Failed to delete from Hydra: %v", err), Code: http.StatusInternalServerError, Timestamp: time.Now()})
			return
		}
		if err := ac.tenantHydraClientRepo.DeleteByHydraClientID(matchedClient.ClientID); err != nil {
			log.Printf("[oocmgr] Warning: Failed to delete database mapping for %s: %v", matchedClient.ClientID, err)
		}
		log.Printf("[oocmgr] EditAuthProvider DELETE provider=%s tenant=%s", req.ProviderName, req.TenantID)
		c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
			Message: "Provider deleted successfully", Success: true,
			Data: map[string]interface{}{
				"tenant_id": req.TenantID, "client_id": req.ClientID,
				"provider_name": req.ProviderName, "deleted_id": matchedClient.ClientID, "operation": "deleted",
			},
			Timestamp: time.Now(),
		})
		return
	}

	// UPSERT: CREATE or UPDATE
	if matchedClient == nil {
		// Try to find tenant-level config to copy from
		baseClientID := strings.TrimSuffix(req.TenantID, "-main-client")
		if baseClientID == "" {
			baseClientID = req.TenantID
		}
		tenantConfigID := fmt.Sprintf("%s-%s-oidc", baseClientID, oocmgrNormalizeProviderName(req.ProviderName))
		var tenantConfig *oocmgrHydraClient
		for i, client := range clients {
			if client.ClientID == tenantConfigID {
				tenantConfig = &clients[i]
				break
			}
		}

		if tenantConfig != nil {
			if req.ProviderConfig == nil {
				req.ProviderConfig = make(map[string]interface{})
			}
			if tcp, ok := tenantConfig.Metadata["provider_config"].(map[string]interface{}); ok {
				for k, v := range tcp {
					if ev, exists := req.ProviderConfig[k]; !exists {
						req.ProviderConfig[k] = v
					} else if str, isStr := ev.(string); isStr && str == "" {
						req.ProviderConfig[k] = v
					} else if ev == nil {
						req.ProviderConfig[k] = v
					}
				}
			}
			if req.DisplayName == "" {
				if dn, ok := tenantConfig.Metadata["display_name"].(string); ok {
					req.DisplayName = dn
				}
			}
			if req.SortOrder == nil {
				if sof, ok := tenantConfig.Metadata["sort_order"].(float64); ok {
					order := int(sof)
					req.SortOrder = &order
				}
			}
			if req.CallbackURL == "" {
				if cu, ok := tenantConfig.Metadata["callback_url"].(string); ok {
					req.CallbackURL = cu
				}
			}
		}

		ac.oocmgrCreateNewProviderForEdit(c, req)
		return
	}

	// UPDATE existing
	modified := false
	updatedFields := make(map[string]interface{})

	if req.DisplayName != "" {
		matchedClient.Metadata["display_name"] = req.DisplayName
		updatedFields["display_name"] = req.DisplayName
		modified = true
	}
	if req.SortOrder != nil {
		matchedClient.Metadata["sort_order"] = *req.SortOrder
		updatedFields["sort_order"] = *req.SortOrder
		modified = true
	}
	if req.CallbackURL != "" {
		matchedClient.Metadata["callback_url"] = req.CallbackURL
		updatedFields["callback_url"] = req.CallbackURL
		modified = true
	}
	if req.ProviderConfig != nil && len(req.ProviderConfig) > 0 {
		existingConfig, ok := matchedClient.Metadata["provider_config"].(map[string]interface{})
		if !ok {
			existingConfig = make(map[string]interface{})
		}
		for k, v := range req.ProviderConfig {
			existingConfig[k] = v
		}
		existingConfig["client_id"] = req.ClientID
		matchedClient.Metadata["provider_config"] = existingConfig
		updatedFields["provider_config"] = req.ProviderConfig
		modified = true
	}

	if !modified {
		c.JSON(http.StatusBadRequest, oocmgrdto.ErrorResponse{Error: "No fields to update", Message: "No updatable fields provided in request", Code: http.StatusBadRequest, Timestamp: time.Now()})
		return
	}

	if err := ac.updateHydraClient(matchedClient.ClientID, *matchedClient); err != nil {
		log.Printf("[oocmgr] audit failure: auth_provider update %s err=%v", req.ProviderName, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{Error: "Update failed", Message: fmt.Sprintf("Failed to update provider: %v", err), Code: http.StatusInternalServerError, Timestamp: time.Now()})
		return
	}

	dbUpdates := map[string]interface{}{"updated_at": time.Now()}
	if req.UpdatedBy != "" {
		dbUpdates["updated_by"] = req.UpdatedBy
	}
	if err := ac.tenantHydraClientRepo.UpdateByHydraClientID(matchedClient.ClientID, dbUpdates); err != nil {
		log.Printf("[oocmgr] Warning: Failed to update database mapping for %s: %v", matchedClient.ClientID, err)
	}

	log.Printf("[oocmgr] EditAuthProvider UPDATE provider=%s tenant=%s", req.ProviderName, req.TenantID)
	c.JSON(http.StatusOK, oocmgrdto.MessageResponse{
		Message: "Provider updated successfully", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "client_id": req.ClientID,
			"provider_name": req.ProviderName, "hydra_id": matchedClient.ClientID,
			"updated_fields": updatedFields, "operation": "updated",
		},
		Timestamp: time.Now(),
	})
}

// ===== SAML ENDPOINTS =====

func (ac *OocmgrController) AddSAMLProvider(c *gin.Context) {
	var req oocmgrdto.AddSAMLProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	log.Printf("[oocmgr] AddSAMLProvider %s for tenant %s", req.ProviderName, req.TenantID)

	tenantDB, err := oocmgrGetTenantDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database: " + err.Error()})
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid tenant ID format"})
		return
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Client ID is required"})
		return
	}
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid client ID format"})
		return
	}

	normalizedProviderName := strings.ToLower(strings.TrimSpace(req.ProviderName))
	var existingProvider OocmgrSAMLProvider
	err = tenantDB.Where("tenant_id = ? AND client_id = ? AND provider_name = ?",
		tenantUUID, clientUUID, normalizedProviderName).First(&existingProvider).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": "SAML provider with this name already exists for this tenant and client"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Database error: " + err.Error()})
		return
	}

	nameIDFormat := req.NameIDFormat
	if nameIDFormat == "" {
		nameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	attributeMapping := req.AttributeMapping
	if attributeMapping == nil {
		attributeMapping = map[string]interface{}{
			"email": "email", "first_name": "givenName", "last_name": "surname", "name": "displayName",
		}
	}

	attributeMappingJSON, err := datatypes.NewJSONType(attributeMapping).Value()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to serialize attribute mapping: " + err.Error()})
		return
	}

	provider := OocmgrSAMLProvider{
		TenantID: tenantUUID, ClientID: clientUUID,
		ProviderName: req.ProviderName, DisplayName: req.DisplayName,
		EntityID: req.EntityID, SSOURL: req.SSOURL, SLOURL: req.SLOURL,
		Certificate: req.Certificate, MetadataURL: req.MetadataURL,
		NameIDFormat:     nameIDFormat,
		AttributeMapping: datatypes.JSON(attributeMappingJSON.([]byte)),
		IsActive:         isActive, SortOrder: req.SortOrder,
	}

	if err := tenantDB.Create(&provider).Error; err != nil {
		log.Printf("[oocmgr] audit failure: saml_provider create tenant=%s err=%v", req.TenantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create SAML provider: " + err.Error()})
		return
	}

	log.Printf("[oocmgr] SAML provider %s created with ID %s", req.ProviderName, provider.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true, "message": "SAML provider added successfully",
		"provider": oocmgrdto.SAMLProviderResponse{
			ID: provider.ID, TenantID: provider.TenantID,
			ProviderName: provider.ProviderName, DisplayName: provider.DisplayName,
			EntityID: provider.EntityID, SSOURL: provider.SSOURL, SLOURL: provider.SLOURL,
			MetadataURL: provider.MetadataURL, NameIDFormat: provider.NameIDFormat,
			AttributeMapping: oocmgrJSONToMap(provider.AttributeMapping),
			IsActive: provider.IsActive, SortOrder: provider.SortOrder,
			CreatedAt: provider.CreatedAt.Format(time.RFC3339),
			UpdatedAt: provider.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (ac *OocmgrController) ListSAMLProviders(c *gin.Context) {
	var req oocmgrdto.ListSAMLProvidersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	tenantDB, err := oocmgrGetTenantDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database: " + err.Error()})
		return
	}

	query := tenantDB.Where("tenant_id = ?", req.TenantID)
	if req.ClientID != "" {
		clientID := strings.TrimSuffix(req.ClientID, "-main-client")
		query = query.Where("client_id = ?", clientID)
	}

	var providers []OocmgrSAMLProvider
	if err := query.Order("sort_order ASC, display_name ASC").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch SAML providers: " + err.Error()})
		return
	}

	providerResponses := make([]oocmgrdto.SAMLProviderResponse, 0, len(providers))
	for _, p := range providers {
		providerResponses = append(providerResponses, oocmgrdto.SAMLProviderResponse{
			ID: p.ID, TenantID: p.TenantID, ProviderName: p.ProviderName,
			DisplayName: p.DisplayName, EntityID: p.EntityID, SSOURL: p.SSOURL,
			SLOURL: p.SLOURL, MetadataURL: p.MetadataURL, NameIDFormat: p.NameIDFormat,
			AttributeMapping: oocmgrJSONToMap(p.AttributeMapping),
			IsActive: p.IsActive, SortOrder: p.SortOrder,
			CreatedAt: p.CreatedAt.Format(time.RFC3339), UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, oocmgrdto.SAMLConfigResponse{
		Success: true, TenantID: req.TenantID, Providers: providerResponses,
	})
}

func (ac *OocmgrController) GetSAMLProvider(c *gin.Context) {
	var req oocmgrdto.GetSAMLProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid provider_id format"})
		return
	}

	tenantDB, err := oocmgrGetTenantDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database: " + err.Error()})
		return
	}

	var provider OocmgrSAMLProvider
	if err := tenantDB.Where("id = ? AND tenant_id = ?", providerID, req.TenantID).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "SAML provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Database error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"provider": oocmgrdto.SAMLProviderResponse{
			ID: provider.ID, TenantID: provider.TenantID, ProviderName: provider.ProviderName,
			DisplayName: provider.DisplayName, EntityID: provider.EntityID, SSOURL: provider.SSOURL,
			SLOURL: provider.SLOURL, MetadataURL: provider.MetadataURL, NameIDFormat: provider.NameIDFormat,
			AttributeMapping: oocmgrJSONToMap(provider.AttributeMapping),
			IsActive: provider.IsActive, SortOrder: provider.SortOrder,
			CreatedAt: provider.CreatedAt.Format(time.RFC3339), UpdatedAt: provider.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (ac *OocmgrController) UpdateSAMLProvider(c *gin.Context) {
	var req oocmgrdto.UpdateSAMLProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid provider_id format"})
		return
	}

	tenantDB, err := oocmgrGetTenantDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database: " + err.Error()})
		return
	}

	query := tenantDB.Where("id = ? AND tenant_id = ?", providerID, req.TenantID)
	if req.ClientID != "" {
		clientID := strings.TrimSuffix(req.ClientID, "-main-client")
		query = query.Where("client_id = ?", clientID)
	}

	var provider OocmgrSAMLProvider
	if err := query.First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "SAML provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Database error: " + err.Error()})
		return
	}

	if req.ProviderName != "" {
		provider.ProviderName = req.ProviderName
	}
	if req.DisplayName != "" {
		provider.DisplayName = req.DisplayName
	}
	if req.EntityID != "" {
		provider.EntityID = req.EntityID
	}
	if req.SSOURL != "" {
		provider.SSOURL = req.SSOURL
	}
	if req.SLOURL != "" {
		provider.SLOURL = req.SLOURL
	}
	if req.Certificate != "" {
		provider.Certificate = req.Certificate
	}
	if req.MetadataURL != "" {
		provider.MetadataURL = req.MetadataURL
	}
	if req.NameIDFormat != "" {
		provider.NameIDFormat = req.NameIDFormat
	}
	if req.AttributeMapping != nil {
		amJSON, err := datatypes.NewJSONType(req.AttributeMapping).Value()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to serialize attribute mapping: " + err.Error()})
			return
		}
		provider.AttributeMapping = datatypes.JSON(amJSON.([]byte))
	}
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		provider.SortOrder = *req.SortOrder
	}
	provider.UpdatedAt = time.Now()

	if err := tenantDB.Save(&provider).Error; err != nil {
		log.Printf("[oocmgr] audit failure: saml_provider update %s err=%v", req.ProviderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update SAML provider: " + err.Error()})
		return
	}

	log.Printf("[oocmgr] UpdateSAMLProvider %s tenant=%s", req.ProviderID, req.TenantID)
	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": "SAML provider updated successfully",
		"provider": oocmgrdto.SAMLProviderResponse{
			ID: provider.ID, TenantID: provider.TenantID, ProviderName: provider.ProviderName,
			DisplayName: provider.DisplayName, EntityID: provider.EntityID, SSOURL: provider.SSOURL,
			SLOURL: provider.SLOURL, MetadataURL: provider.MetadataURL, NameIDFormat: provider.NameIDFormat,
			AttributeMapping: oocmgrJSONToMap(provider.AttributeMapping),
			IsActive: provider.IsActive, SortOrder: provider.SortOrder,
			CreatedAt: provider.CreatedAt.Format(time.RFC3339), UpdatedAt: provider.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (ac *OocmgrController) DeleteSAMLProvider(c *gin.Context) {
	var req oocmgrdto.DeleteSAMLProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid provider_id format"})
		return
	}

	tenantDB, err := oocmgrGetTenantDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database: " + err.Error()})
		return
	}

	query := tenantDB.Where("id = ? AND tenant_id = ?", providerID, req.TenantID)
	if req.ClientID != "" {
		clientID := strings.TrimSuffix(req.ClientID, "-main-client")
		query = query.Where("client_id = ?", clientID)
	}

	result := query.Delete(&OocmgrSAMLProvider{})
	if result.Error != nil {
		log.Printf("[oocmgr] audit failure: saml_provider delete %s err=%v", req.ProviderID, result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete SAML provider: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "SAML provider not found"})
		return
	}

	log.Printf("[oocmgr] DeleteSAMLProvider %s tenant=%s", req.ProviderID, req.TenantID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "SAML provider deleted successfully"})
}

func (ac *OocmgrController) GetSAMLProviderTemplates(c *gin.Context) {
	templates := map[string]oocmgrdto.SAMLTemplateDTO{
		"okta": {
			ProviderName: "Okta", DisplayName: "Okta SAML",
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AttributeMapping: map[string]interface{}{
				"email": "email", "first_name": "firstName", "last_name": "lastName", "name": "displayName",
			},
			Instructions:     "1. Create SAML 2.0 application in Okta\n2. Set Single sign-on URL to your ACS URL\n3. Set Audience URI to your SP Entity ID\n4. Copy the IdP metadata URL or certificate",
			DocumentationURL: "https://developer.okta.com/docs/guides/build-sso-integration/saml2/main/",
			ConfigFields:     []string{"entity_id", "sso_url", "certificate", "metadata_url"},
		},
		"azure": {
			ProviderName: "Azure-SAML", DisplayName: "Azure AD SAML",
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AttributeMapping: map[string]interface{}{
				"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
				"first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
				"last_name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
				"name":       "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			},
			Instructions:     "1. Create Enterprise Application in Azure AD\n2. Select 'Set up single sign on' and choose SAML\n3. Configure Basic SAML settings\n4. Download the Certificate (Base64) and Federation Metadata XML",
			DocumentationURL: "https://learn.microsoft.com/en-us/azure/active-directory/manage-apps/add-application-portal-setup-sso",
			ConfigFields:     []string{"entity_id", "sso_url", "certificate", "metadata_url"},
		},
		"onelogin": {
			ProviderName: "OneLogin", DisplayName: "OneLogin SAML",
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AttributeMapping: map[string]interface{}{
				"email": "email", "first_name": "firstname", "last_name": "lastname", "name": "name",
			},
			Instructions:     "1. Add a new SAML Test Connector app in OneLogin\n2. Configure the ACS URL and Audience\n3. Go to SSO tab to get the Issuer URL and SAML 2.0 Endpoint\n4. Download the X.509 Certificate",
			DocumentationURL: "https://developers.onelogin.com/saml",
			ConfigFields:     []string{"entity_id", "sso_url", "certificate"},
		},
		"google": {
			ProviderName: "Google-SAML", DisplayName: "Google Workspace SAML",
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AttributeMapping: map[string]interface{}{
				"email": "email", "first_name": "firstName", "last_name": "lastName", "name": "fullName",
			},
			Instructions:     "1. Go to Google Admin Console > Apps > Web and mobile apps\n2. Add custom SAML app\n3. Download IdP metadata or copy certificate\n4. Configure ACS URL and Entity ID",
			DocumentationURL: "https://support.google.com/a/answer/6087519",
			ConfigFields:     []string{"entity_id", "sso_url", "certificate"},
		},
	}

	c.JSON(http.StatusOK, oocmgrdto.SAMLProviderTemplatesResponse{Success: true, Templates: templates})
}

// ===== HYDRA API HELPERS =====

func (ac *OocmgrController) createHydraClient(client oocmgrHydraClient) error {
	jsonData, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client data: %w", err)
	}
	log.Printf("[oocmgr] Sending to Hydra: %s", string(jsonData))

	url := fmt.Sprintf("%s/admin/clients", ac.hydraConfig.AdminURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Hydra API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[oocmgr] Hydra error response: %s", string(body))
		return fmt.Errorf("Hydra API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (ac *OocmgrController) getHydraClient(clientID string) (*oocmgrHydraClient, error) {
	url := fmt.Sprintf("%s/admin/clients/%s", ac.hydraConfig.AdminURL, clientID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client not found or API error: %d", resp.StatusCode)
	}

	var client oocmgrHydraClient
	if err := json.NewDecoder(resp.Body).Decode(&client); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &client, nil
}

func (ac *OocmgrController) getAllHydraClients() ([]oocmgrHydraClient, error) {
	url := fmt.Sprintf("%s/admin/clients", ac.hydraConfig.AdminURL)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var clients []oocmgrHydraClient
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return clients, nil
}

func (ac *OocmgrController) updateHydraClient(clientID string, client oocmgrHydraClient) error {
	jsonData, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client data: %w", err)
	}

	url := fmt.Sprintf("%s/admin/clients/%s", ac.hydraConfig.AdminURL, clientID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Hydra API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Hydra API returned status %d", resp.StatusCode)
	}
	return nil
}

func (ac *OocmgrController) deleteHydraClient(clientID string) error {
	url := fmt.Sprintf("%s/admin/clients/%s", ac.hydraConfig.AdminURL, clientID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Hydra API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[oocmgr] deleteHydraClient: %s already removed (404)", clientID)
		return nil
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Hydra API returned status %d", resp.StatusCode)
	}
	return nil
}

// ===== UTILITY HELPERS =====

func (ac *OocmgrController) oocmgrSaveProviderSecret(tenantID, providerName, clientSecret string) error {
	return config.SaveProviderSecretToVault(tenantID, oocmgrNormalizeProviderName(providerName), map[string]interface{}{
		"client_secret": clientSecret,
	})
}

func (ac *OocmgrController) oocmgrDeleteProviderSecret(tenantID, providerName string) error {
	return config.DeleteProviderSecretFromVault(tenantID, oocmgrNormalizeProviderName(providerName))
}

func (ac *OocmgrController) oocmgrGenerateCallbackURLs(providers []oocmgrOIDCProviderConfig) map[string]string {
	callbackURLs := make(map[string]string)
	for _, provider := range providers {
		callbackURLs[provider.ProviderName] = fmt.Sprintf("%s/oauth2/callback/%s",
			ac.hydraConfig.PublicURL, oocmgrNormalizeProviderName(provider.ProviderName))
	}
	return callbackURLs
}

func (ac *OocmgrController) getOIDCProvidersForTenant(tenantID string) ([]oocmgrOIDCProvider, error) {
	clients, err := ac.getAllHydraClients()
	if err != nil {
		return nil, err
	}

	var providers []oocmgrOIDCProvider
	for _, client := range clients {
		if clientTenantID, ok := client.Metadata["tenant_id"].(string); ok && clientTenantID == tenantID {
			if clientType, ok := client.Metadata["type"].(string); ok && clientType == "oidc_provider" {
				providerName, _ := client.Metadata["provider_name"].(string)
				displayName, _ := client.Metadata["display_name"].(string)
				isActive, _ := client.Metadata["is_active"].(bool)
				sortOrder, _ := client.Metadata["sort_order"].(float64)
				callbackURL, _ := client.Metadata["callback_url"].(string)
				providerConfig, _ := client.Metadata["provider_config"].(map[string]interface{})

				if providerName != "" {
					providers = append(providers, oocmgrOIDCProvider{
						ProviderName: providerName, DisplayName: displayName,
						IsActive: isActive, SortOrder: int(sortOrder),
						CallbackURL: callbackURL, Config: providerConfig,
					})
				}
			}
		}
	}
	return providers, nil
}

func (ac *OocmgrController) oocmgrTryResolveHydraClientForUpdate(
	req oocmgrUpdateOIDCConfigRequest, providerKey, headerClientID string, originalErr error,
) (*oocmgrHydraClient, string, error) {
	clients, err := ac.getAllHydraClients()
	if err != nil {
		return nil, "", fmt.Errorf("%s; fallback Hydra lookup failed: %w", originalErr.Error(), err)
	}

	headerAlias := strings.TrimSpace(headerClientID)
	var payloadClientID string
	if req.ClientID != nil {
		payloadClientID = strings.TrimSpace(*req.ClientID)
	}

	var headerMatches, payloadMatches, providerMatches []oocmgrHydraClient

	for _, client := range clients {
		if client.Metadata == nil {
			continue
		}
		if !oocmgrBelongsToTenant(client.Metadata, req.TenantID) {
			continue
		}
		if clientType, _ := client.Metadata["type"].(string); clientType != "oidc_provider" {
			continue
		}
		if headerAlias != "" {
			if alias, _ := client.Metadata["tenant_id"].(string); alias == headerAlias {
				headerMatches = append(headerMatches, client)
			} else if alias, _ := client.Metadata["client_id"].(string); alias == headerAlias {
				headerMatches = append(headerMatches, client)
			}
		}
		if payloadClientID != "" && oocmgrExtractServiceClientID(client.Metadata) == payloadClientID {
			payloadMatches = append(payloadMatches, client)
		}
		if oocmgrProviderNameMatches(client.Metadata, providerKey) {
			providerMatches = append(providerMatches, client)
		}
	}

	if headerAlias != "" {
		switch len(headerMatches) {
		case 1:
			return &headerMatches[0], headerMatches[0].ClientID, nil
		case 0:
		default:
			return nil, "", fmt.Errorf("%s; client-id header matches %d Hydra configs", originalErr.Error(), len(headerMatches))
		}
	}
	if payloadClientID != "" {
		switch len(payloadMatches) {
		case 1:
			return &payloadMatches[0], payloadMatches[0].ClientID, nil
		case 0:
		default:
			return nil, "", fmt.Errorf("%s; multiple Hydra configs share client_id=%s", originalErr.Error(), payloadClientID)
		}
	}
	switch len(providerMatches) {
	case 1:
		return &providerMatches[0], providerMatches[0].ClientID, nil
	case 0:
		return nil, "", fmt.Errorf("%s; no Hydra configs exist for tenant=%s provider=%s", originalErr.Error(), req.TenantID, req.ProviderName)
	default:
		return nil, "", fmt.Errorf("%s; multiple Hydra configs exist for provider=%s", originalErr.Error(), req.ProviderName)
	}
}

func (ac *OocmgrController) oocmgrCreateNewProviderForEdit(c *gin.Context, req editAuthProviderReq) {
	oidcClientID := fmt.Sprintf("%s-%s-oidc", req.ClientID, oocmgrNormalizeProviderName(req.ProviderName))
	if existing, _ := ac.getHydraClient(oidcClientID); existing != nil {
		oidcClientID = fmt.Sprintf("%s-%s-oidc-%d", req.ClientID, oocmgrNormalizeProviderName(req.ProviderName), time.Now().Unix())
		log.Printf("[oocmgr] Hydra client ID collision, using: %s", oidcClientID)
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.ProviderName
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	sortOrder := 99
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	providerConfig := make(map[string]interface{})
	for k, v := range req.ProviderConfig {
		providerConfig[k] = v
	}

	callbackURL := req.CallbackURL
	if callbackURL == "" {
		reactAppURL := os.Getenv("REACT_APP_URL")
		if reactAppURL == "" {
			reactAppURL = "https://app.authsec.dev"
		}
		callbackURL = fmt.Sprintf("%s/oidc/auth/callback/%s", reactAppURL, oocmgrNormalizeProviderName(req.ProviderName))
	}

	newClient := oocmgrHydraClient{
		ClientID: oidcClientID, ClientName: fmt.Sprintf("OIDC Provider: %s", displayName),
		GrantTypes: []string{"client_credentials"},
		Metadata: map[string]interface{}{
			"type": "oidc_provider", "tenant_id": req.ClientID, "c_id": req.TenantID,
			"provider_name": req.ProviderName, "display_name": displayName,
			"provider_config": providerConfig, "is_active": isActive,
			"sort_order": sortOrder, "callback_url": callbackURL,
			"created_at": time.Now().Format(time.RFC3339), "created_by": req.UpdatedBy,
		},
	}

	if err := ac.createHydraClient(newClient); err != nil {
		log.Printf("[oocmgr] audit failure: auth_provider create %s err=%v", req.ProviderName, err)
		c.JSON(http.StatusInternalServerError, oocmgrdto.ErrorResponse{
			Error: "Failed to create provider", Message: fmt.Sprintf("Failed to create new provider in Hydra: %v", err),
			Code: http.StatusInternalServerError, Timestamp: time.Now(),
		})
		return
	}

	if clientSecret, ok := providerConfig["client_secret"].(string); ok && clientSecret != "" {
		if err := ac.oocmgrSaveProviderSecret(req.TenantID, req.ProviderName, clientSecret); err != nil {
			log.Printf("[oocmgr] Warning: Failed to store provider secret for %s: %v", req.ProviderName, err)
		}
	}

	thc := &oocmgrdto.TenantHydraClient{
		TenantID: req.TenantID, HydraClientID: oidcClientID, HydraClientSecret: "",
		ClientType: "oidc_provider", ProviderName: req.ProviderName,
		IsActive: isActive, CreatedBy: req.UpdatedBy, UpdatedBy: req.UpdatedBy,
	}
	if err := ac.tenantHydraClientRepo.Create(thc); err != nil {
		log.Printf("[oocmgr] Warning: Failed to store tenant-client mapping: %v", err)
	}

	log.Printf("[oocmgr] oocmgrCreateNewProviderForEdit provider=%s tenant=%s hydra_id=%s", req.ProviderName, req.TenantID, oidcClientID)
	c.JSON(http.StatusCreated, oocmgrdto.MessageResponse{
		Message: "Provider created successfully (upsert: insert)", Success: true,
		Data: map[string]interface{}{
			"tenant_id": req.TenantID, "client_id": req.ClientID,
			"provider_name": req.ProviderName, "hydra_client_id": oidcClientID,
			"display_name": displayName, "is_active": isActive,
			"sort_order": sortOrder, "callback_url": callbackURL, "operation": "created",
		},
		Timestamp: time.Now(),
	})
}

// ===== PACKAGE-LEVEL HELPERS =====

func oocmgrGetTenantDB(orgID string) (*gorm.DB, error) {
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant connection: %w", err)
	}
	return tenantDB, nil
}

func oocmgrJSONToMap(jsonData datatypes.JSON) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil
	}
	return result
}

func oocmgrNormalizeProviderName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, " ", "-"), "_", "-"))
}

// oocmgrResolveMicrosoftAuthorityURLs auto-populates Microsoft auth/token/issuer/jwks URLs
// based on the "authority_type" field in additional_params.
// Supported values: "common" (default), "organizations", "consumers", or a specific tenant ID.
func oocmgrResolveMicrosoftAuthorityURLs(provider *oocmgrOIDCProviderConfig) {
	if !strings.EqualFold(provider.ProviderName, "microsoft") {
		return
	}
	authorityType := "common"
	if provider.AdditionalParams != nil {
		if at, ok := provider.AdditionalParams["authority_type"]; ok {
			if atStr, ok := at.(string); ok && atStr != "" {
				authorityType = atStr
			}
		}
	}
	base := fmt.Sprintf("https://login.microsoftonline.com/%s", authorityType)
	if provider.AuthURL == "" {
		provider.AuthURL = base + "/oauth2/v2.0/authorize"
	}
	if provider.TokenURL == "" {
		provider.TokenURL = base + "/oauth2/v2.0/token"
	}
	if provider.IssuerURL == "" {
		provider.IssuerURL = base + "/v2.0"
	}
	if provider.JWKsURL == "" {
		provider.JWKsURL = base + "/discovery/v2.0/keys"
	}
	if provider.UserInfoURL == "" {
		provider.UserInfoURL = "https://graph.microsoft.com/v1.0/me"
	}
}

func oocmgrGenerateSecureSecret() string {
	return fmt.Sprintf("secret-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}

func oocmgrGetClientIDFromHeaders(c *gin.Context) string {
	for _, key := range []string{"Client-Id", "client-id", "X-Client-Id"} {
		if v := strings.TrimSpace(c.GetHeader(key)); v != "" {
			return v
		}
	}
	return ""
}

func oocmgrBelongsToTenant(metadata map[string]interface{}, tenantID string) bool {
	if metadata == nil || tenantID == "" {
		return false
	}
	if metaTenantID, ok := metadata["tenant_id"].(string); ok && metaTenantID == tenantID {
		return true
	}
	if cid, ok := metadata["c_id"].(string); ok && cid == tenantID {
		return true
	}
	return false
}

func oocmgrExtractServiceClientID(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	if pc, ok := metadata["provider_config"].(map[string]interface{}); ok {
		if clientID, ok := pc["client_id"].(string); ok {
			return strings.TrimSpace(clientID)
		}
	}
	if clientID, ok := metadata["client_id"].(string); ok {
		return strings.TrimSpace(clientID)
	}
	return ""
}

func oocmgrProviderNameMatches(metadata map[string]interface{}, normalizedTarget string) bool {
	if normalizedTarget == "" || metadata == nil {
		return false
	}
	if pn, ok := metadata["provider_name"].(string); ok {
		return oocmgrNormalizeProviderName(pn) == normalizedTarget
	}
	return false
}

func oocmgrSanitizeHydraClientForDump(client oocmgrHydraClient) map[string]interface{} {
	output := map[string]interface{}{
		"client_id": client.ClientID, "client_name": client.ClientName,
		"grant_types": client.GrantTypes, "redirect_uris": client.RedirectURIs,
		"response_types": client.ResponseTypes,
		"token_endpoint_auth_method": client.TokenEndpoint,
		"scope": client.Scope, "subject_type": client.SubjectType, "audience": client.Audience,
	}
	if client.ClientSecret != "" {
		output["client_secret"] = "***hidden***"
	}
	if client.Metadata != nil {
		output["metadata"] = oocmgrSanitizeMetadata(client.Metadata)
	}
	return output
}

func oocmgrSanitizeMetadata(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	sanitized := make(map[string]interface{}, len(data))
	for key, value := range data {
		sanitized[key] = oocmgrSanitizeEntry(key, value)
	}
	return sanitized
}

func oocmgrSanitizeEntry(key string, value interface{}) interface{} {
	if strings.Contains(strings.ToLower(key), "secret") {
		return "***hidden***"
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		return oocmgrSanitizeMetadata(typed)
	case []interface{}:
		result := make([]interface{}, len(typed))
		for i, item := range typed {
			switch nested := item.(type) {
			case map[string]interface{}:
				result[i] = oocmgrSanitizeMetadata(nested)
			default:
				result[i] = nested
			}
		}
		return result
	default:
		return value
	}
}

// ensure rand is used (generateRandomString equivalent, kept for completeness)
var _ = rand.Intn
