package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	oocmgrdto "github.com/authsec-ai/authsec/internal/oocmgr/dto"
	oocmgrrepo "github.com/authsec-ai/authsec/internal/oocmgr/repository"
	"github.com/golang-jwt/jwt/v5"
)

// hydraClient mirrors the Hydra admin API client object used for direct calls.
type hydraClient struct {
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

func hydraAdminURL() string {
	return config.AppConfig.HydraAdminURL
}

func hydraAdminGetClient(clientID string) (*hydraClient, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/admin/clients/%s", hydraAdminURL(), clientID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := CircuitDoHydra(req)
	if err != nil {
		return nil, fmt.Errorf("hydra get client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hydra get client status %d", resp.StatusCode)
	}
	var c hydraClient
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, fmt.Errorf("hydra get client decode: %w", err)
	}
	return &c, nil
}

func hydraAdminCreateClient(c hydraClient) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/admin/clients", hydraAdminURL()), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := CircuitDoHydra(req)
	if err != nil {
		return fmt.Errorf("hydra create client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hydra create client status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func hydraAdminUpdateClient(clientID string, c hydraClient) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/admin/clients/%s", hydraAdminURL(), clientID), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := CircuitDoHydra(req)
	if err != nil {
		return fmt.Errorf("hydra update client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hydra update client status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func hydraAdminDeleteClient(clientID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/admin/clients/%s", hydraAdminURL(), clientID), nil)
	if err != nil {
		return err
	}
	resp, err := CircuitDoHydra(req)
	if err != nil {
		return fmt.Errorf("hydra delete client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hydra delete client status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func hydraAdminGetAllClients() ([]hydraClient, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/admin/clients", hydraAdminURL()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := CircuitDoHydra(req)
	if err != nil {
		return nil, fmt.Errorf("hydra get all clients: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hydra get all clients status %d", resp.StatusCode)
	}
	var clients []hydraClient
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return nil, fmt.Errorf("hydra get all clients decode: %w", err)
	}
	return clients, nil
}

func oocmgrNormalizeProviderName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}

// OOCManager represents the request structure for OOC Manager API
type OOCManager struct {
	TenantID     string   `json:"tenant_id" validate:"required"`
	TenantName   string   `json:"tenant_name" validate:"required"`
	ClientID     string   `json:"client_id" validate:"required"`
	ClientSecret string   `json:"client_secret" validate:"required"`
	RedirectURIs []string `json:"redirect_uris" validate:"required"`
	Scopes       []string `json:"scopes,omitempty"`
	CreatedBy    string   `json:"created_by"`
}

// ProviderConfig represents the provider configuration for OIDC
type ProviderConfig struct {
	ProviderName string   `json:"provider_name"`
	DisplayName  string   `json:"display_name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
	Scopes       []string `json:"scopes"`
	IsActive     bool     `json:"is_active"`
}

// AddProviderRequest represents the request structure for adding a provider
type AddProviderRequest struct {
	TenantID    string         `json:"tenant_id"`
	ClientID    string         `json:"client_id"`
	ReactAppURL string         `json:"react_app_url"`
	Provider    ProviderConfig `json:"provider"`
	CreatedBy   string         `json:"created_by"`
}

// generateServiceToken generates a JWT token for service-to-service authentication
func generateServiceToken() (string, error) {
	secret := config.AppConfig.JWTSdkSecret
	if secret == "" {
		return "", fmt.Errorf("JWT_SDK_SECRET not configured")
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"service": "user-flow",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(5 * time.Minute).Unix(), // Short-lived service token
	})

	// Sign the token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign service token: %w", err)
	}

	return tokenString, nil
}

// RegisterClientWithHydra creates the tenant's main OAuth2 client directly in Hydra.
func RegisterClientWithHydra(clientID, clientSecret, clientName, tenantID, tenantDomain string) error {
	mainClientID := fmt.Sprintf("%s-main-client", clientID)

	// Skip if already exists
	if existing, _ := hydraAdminGetClient(mainClientID); existing != nil {
		log.Printf("Hydra client %s already exists, skipping creation", mainClientID)
		return nil
	}

	scopes := []string{"openid", "offline_access", "email", "profile"}
	c := hydraClient{
		ClientID:      mainClientID,
		ClientSecret:  clientSecret,
		ClientName:    fmt.Sprintf("%s Main OAuth Client", clientName),
		GrantTypes:    []string{"authorization_code", "refresh_token"},
		RedirectURIs:  []string{fmt.Sprintf("https://%s/oidc/auth/callback", tenantDomain)},
		ResponseTypes: []string{"code"},
		TokenEndpoint: "client_secret_post",
		Scope:         strings.Join(scopes, " "),
		Audience:      []string{},
		SubjectType:   "public",
		Metadata: map[string]interface{}{
			"type":        "tenant_main_client",
			"tenant_id":   clientID,
			"c_id":        tenantID,
			"tenant_name": clientName,
			"created_at":  time.Now().Format(time.RFC3339),
			"created_by":  "system",
		},
	}

	if err := hydraAdminCreateClient(c); err != nil {
		return fmt.Errorf("failed to create Hydra client: %w", err)
	}

	// Best-effort: store mapping in master DB
	thc := &oocmgrdto.TenantHydraClient{
		TenantID: tenantID, TenantName: clientName,
		HydraClientID: mainClientID, HydraClientSecret: clientSecret,
		ClientName:   fmt.Sprintf("%s Main OAuth Client", clientName),
		RedirectURIs: []string{fmt.Sprintf("https://%s/oidc/auth/callback", tenantDomain)},
		Scopes:       scopes, ClientType: "main", IsActive: true,
		CreatedBy: "system", UpdatedBy: "system",
	}
	if err := oocmgrrepo.NewTenantHydraClientRepository().Create(thc); err != nil {
		log.Printf("Warning: failed to store tenant-client mapping for %s: %v", mainClientID, err)
	}

	log.Printf("Successfully registered Hydra client %s", mainClientID)
	return nil
}

// DeleteClientFromHydra removes all Hydra clients belonging to the tenant directly.
func DeleteClientFromHydra(clientID string) error {
	clients, err := hydraAdminGetAllClients()
	if err != nil {
		return fmt.Errorf("failed to list Hydra clients: %w", err)
	}

	deleted := 0
	for _, c := range clients {
		cID, _ := c.Metadata["c_id"].(string)
		tID, _ := c.Metadata["tenant_id"].(string)
		if cID != clientID && tID != clientID {
			continue
		}
		if err := hydraAdminDeleteClient(c.ClientID); err != nil {
			log.Printf("Warning: failed to delete Hydra client %s: %v", c.ClientID, err)
		} else {
			deleted++
		}
	}

	log.Printf("Deleted %d Hydra client(s) for clientID=%s", deleted, clientID)
	return nil
}

// UpdateClientInHydra updates the main Hydra client's secret directly.
func UpdateClientInHydra(clientID, secret, email, tenantID string) error {
	mainClientID := fmt.Sprintf("%s-main-client", clientID)
	existing, err := hydraAdminGetClient(mainClientID)
	if err != nil {
		return fmt.Errorf("client %s not found in Hydra: %w", mainClientID, err)
	}
	existing.ClientSecret = secret
	if err := hydraAdminUpdateClient(mainClientID, *existing); err != nil {
		return fmt.Errorf("failed to update Hydra client %s: %w", mainClientID, err)
	}
	log.Printf("Successfully updated Hydra client %s", mainClientID)
	return nil
}

// AddProviderToClient adds a dummy AuthSec OIDC provider client directly in Hydra.
func AddProviderToClient(tenantID, clientID, reactAppURL, createdBy string) error {
	baseClientID := strings.TrimSuffix(clientID, "-main-client")
	mainClientID := fmt.Sprintf("%s-main-client", baseClientID)

	tenantClient, err := hydraAdminGetClient(mainClientID)
	if err != nil {
		return fmt.Errorf("base client %s not found in Hydra: %w", mainClientID, err)
	}

	providerName := "authsec"
	oidcClientID := fmt.Sprintf("%s-%s-oidc", baseClientID, oocmgrNormalizeProviderName(providerName))
	tenantName, _ := tenantClient.Metadata["tenant_name"].(string)

	oidcClient := hydraClient{
		ClientID:   oidcClientID,
		ClientName: fmt.Sprintf("%s AuthSec OIDC Config", tenantName),
		GrantTypes: []string{"client_credentials"},
		Metadata: map[string]interface{}{
			"type":          "oidc_provider",
			"tenant_id":     baseClientID,
			"c_id":          tenantID,
			"provider_name": providerName,
			"display_name":  "AuthSec",
			"provider_config": map[string]interface{}{
				"client_id":     clientID,
				"client_secret": "dummy-secret-" + clientID,
				"auth_url":      fmt.Sprintf("https://%s/oauth2/auth", reactAppURL),
				"token_url":     fmt.Sprintf("https://%s/oauth2/token", reactAppURL),
				"user_info_url": fmt.Sprintf("https://%s/userinfo", reactAppURL),
				"scopes":        []string{"openid", "profile", "email"},
			},
			"is_active":    true,
			"callback_url": fmt.Sprintf("%s/oidc/auth/callback/%s", reactAppURL, oocmgrNormalizeProviderName(providerName)),
			"created_at":   time.Now().Format(time.RFC3339),
			"created_by":   createdBy,
		},
	}

	if err := hydraAdminCreateClient(oidcClient); err != nil {
		return fmt.Errorf("failed to create OIDC provider client: %w", err)
	}

	// Best-effort: store mapping in master DB
	thc := &oocmgrdto.TenantHydraClient{
		TenantID: tenantID, TenantName: tenantName,
		HydraClientID:     oidcClientID,
		HydraClientSecret: "not-used-for-oidc-config",
		ClientName:        fmt.Sprintf("%s AuthSec OIDC Config", tenantName),
		ClientType:        "oidc_provider", ProviderName: providerName,
		IsActive: true, CreatedBy: createdBy, UpdatedBy: createdBy,
	}
	if err := oocmgrrepo.NewTenantHydraClientRepository().Create(thc); err != nil {
		log.Printf("Warning: failed to store OIDC provider mapping for %s: %v", oidcClientID, err)
	}

	log.Printf("Successfully added AuthSec provider client %s", oidcClientID)
	return nil
}
