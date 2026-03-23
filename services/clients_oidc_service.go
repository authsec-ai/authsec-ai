package services

import (
	"fmt"
	"time"
)

// ClientsOIDCService handles OIDC client operations in-process.
type ClientsOIDCService struct{}

// NewClientsOIDCService creates a new OIDC service instance
func NewClientsOIDCService() *ClientsOIDCService {
	return &ClientsOIDCService{}
}

// ClientsOIDCClientResponse represents the response from creating an OIDC client
type ClientsOIDCClientResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
}

// ClientsCreateTenantClientRequest represents the request structure for creating a tenant client
type ClientsCreateTenantClientRequest struct {
	TenantID     string   `json:"tenant_id"`
	TenantName   string   `json:"tenant_name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	CreatedBy    string   `json:"created_by"`
}

// CreateTenantClient creates a new OIDC client directly in Hydra.
func (o *ClientsOIDCService) CreateTenantClient(tenantID, clientName string) (*ClientsOIDCClientResponse, error) {
	clientID := fmt.Sprintf("client_%s_%s_%d", tenantID, clientName, time.Now().Unix())
	clientSecret := generateClientsClientSecret()

	redirectURIs := []string{"http://localhost:3000/callback"}

	if err := RegisterClientWithHydra(clientID, clientSecret, clientName, tenantID, "localhost:3000"); err != nil {
		// RegisterClientWithHydra builds a main-client suffix; here we want the raw clientID so create directly
		c := hydraClient{
			ClientID:      clientID,
			ClientSecret:  clientSecret,
			ClientName:    clientName,
			GrantTypes:    []string{"authorization_code", "refresh_token"},
			RedirectURIs:  redirectURIs,
			ResponseTypes: []string{"code"},
			TokenEndpoint: "client_secret_post",
			Scope:         "openid profile email",
			Audience:      []string{},
			SubjectType:   "public",
			Metadata: map[string]interface{}{
				"type":       "tenant_main_client",
				"tenant_id":  clientID,
				"c_id":       tenantID,
				"created_by": "clients-microservice",
			},
		}
		if err2 := hydraAdminCreateClient(c); err2 != nil {
			return nil, fmt.Errorf("failed to create Hydra client: %w", err2)
		}
	}

	return &ClientsOIDCClientResponse{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Success:      true,
		Message:      "Tenant base client created successfully",
	}, nil
}

// CheckOIDCManagerHealth is a no-op now that oocmgr is in-process.
func (o *ClientsOIDCService) CheckOIDCManagerHealth() error {
	return nil
}

func generateClientsClientSecret() string {
	return fmt.Sprintf("secret_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
