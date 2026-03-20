package models

import (
	"time"

	"github.com/google/uuid"
)

// OIDCProvider represents a platform-level OIDC provider configuration
// These are YOUR app's credentials for Google, GitHub, Microsoft
type OIDCProvider struct {
	ID                    uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProviderName          string    `json:"provider_name" gorm:"uniqueIndex;not null"`    // 'google', 'github', 'microsoft'
	DisplayName           string    `json:"display_name" gorm:"not null"`                 // 'Google', 'GitHub', 'Microsoft'
	ClientID              string    `json:"client_id" gorm:"not null"`                    // OAuth client ID
	ClientSecretVaultPath string    `json:"client_secret_vault_path" gorm:"not null"`     // Vault path for secret
	AuthorizationURL      string    `json:"authorization_url" gorm:"not null"`            // OAuth authorize endpoint
	TokenURL              string    `json:"token_url" gorm:"not null"`                    // OAuth token endpoint
	UserinfoURL           string    `json:"userinfo_url" gorm:"not null"`                 // OAuth userinfo endpoint
	Scopes                string    `json:"scopes" gorm:"default:'openid email profile'"` // Space-separated scopes
	IconURL               string    `json:"icon_url,omitempty"`                           // Provider icon for UI
	IsActive              bool      `json:"is_active" gorm:"default:false"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TableName specifies the table name for OIDCProvider
func (OIDCProvider) TableName() string {
	return "oidc_providers"
}

// OIDCState represents a short-lived state for OIDC flow security
// Used to pass tenant context through OAuth redirects
type OIDCState struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	StateToken    string     `json:"state_token" gorm:"uniqueIndex;not null"` // Random state for CSRF
	TenantID      *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`    // NULL for new registration
	TenantDomain  string     `json:"tenant_domain" gorm:"not null"`           // e.g., 'ritam'
	OriginDomain  string     `json:"origin_domain,omitempty" gorm:"column:request_host"` // The actual domain user came from (maps to request_host column)
	ProviderName  string     `json:"provider_name" gorm:"not null"`           // 'google', 'github', etc.
	Action        string     `json:"action" gorm:"not null"`                  // 'login' or 'register'
	CodeVerifier  string     `json:"code_verifier,omitempty"`                 // For PKCE
	RedirectAfter string     `json:"redirect_after,omitempty"`                // Where to redirect after success
	ExpiresAt     time.Time  `json:"expires_at" gorm:"not null"`              // State expiry
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName specifies the table name for OIDCState
func (OIDCState) TableName() string {
	return "oidc_states"
}

// OIDCUserIdentity links OIDC provider identities to users
// Allows lookup: "Does this Google user exist in this tenant?"
type OIDCUserIdentity struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID       uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID         uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	ProviderName   string     `json:"provider_name" gorm:"not null"`            // 'google', 'github', 'microsoft'
	ProviderUserID string     `json:"provider_user_id" gorm:"not null"`         // Provider's unique user ID (sub claim)
	Email          string     `json:"email,omitempty"`                          // Email from provider
	ProfileData    string     `json:"profile_data,omitempty" gorm:"type:jsonb"` // Additional profile info
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName specifies the table name for OIDCUserIdentity
func (OIDCUserIdentity) TableName() string {
	return "oidc_user_identities"
}

// ========================================
// Input/Output DTOs for OIDC operations
// ========================================

// OIDCInitiateInput represents the input for initiating OIDC flow
type OIDCInitiateInput struct {
	TenantDomain  string `json:"tenant_domain"`               // e.g., 'ritam' (optional - empty = discover mode)
	Provider      string `json:"provider" binding:"required"` // 'google', 'github', 'microsoft'
	RedirectAfter string `json:"redirect_after,omitempty"`    // Optional: where to go after
}

// OIDCInitiateResponse represents the response for OIDC initiation
type OIDCInitiateResponse struct {
	RedirectURL string `json:"redirect_url"` // URL to redirect user to
	State       string `json:"state"`        // State token (for reference)
}

// OIDCCallbackInput represents the callback from OIDC provider
type OIDCCallbackInput struct {
	Code  string `json:"code" form:"code" binding:"required"`   // Authorization code
	State string `json:"state" form:"state" binding:"required"` // State token for verification
	Error string `json:"error,omitempty" form:"error"`          // Error from provider (if any)
}

// OIDCCallbackResponse represents the response after successful callback
type OIDCCallbackResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	TenantDomain string `json:"tenant_domain,omitempty"`
	RedirectURL  string `json:"redirect_url,omitempty"` // Where to redirect user
	Token        string `json:"token,omitempty"`        // JWT token if login
	FirstLogin   bool   `json:"first_login,omitempty"`  // True if first OIDC login
}

// OIDCUserInfo represents user info received from OIDC provider
type OIDCUserInfo struct {
	Sub           string `json:"sub"` // Unique user ID from provider
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

// OIDCTokenResponse represents the token response from OIDC provider
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OIDCProviderUpdateInput represents input for updating OIDC provider config
type OIDCProviderUpdateInput struct {
	ClientID              string `json:"client_id,omitempty"`
	ClientSecretVaultPath string `json:"client_secret_vault_path,omitempty"`
	IsActive              *bool  `json:"is_active,omitempty"`
	IconURL               string `json:"icon_url,omitempty"`
}

// OIDCProviderListResponse represents list of available providers for UI
type OIDCProviderListResponse struct {
	Providers []OIDCProviderPublic `json:"providers"`
}

// OIDCProviderPublic represents public info about OIDC provider (for login UI)
type OIDCProviderPublic struct {
	ProviderName string `json:"provider_name"`
	DisplayName  string `json:"display_name"`
	IconURL      string `json:"icon_url,omitempty"`
}

// LinkOIDCIdentityInput represents input for linking OIDC to existing user
type LinkOIDCIdentityInput struct {
	Provider string `json:"provider" binding:"required"` // 'google', 'github', etc.
}
