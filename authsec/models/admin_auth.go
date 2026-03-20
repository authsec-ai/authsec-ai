package models

import "time"

// AdminLoginInput captures admin login credentials while allowing optional tenant context.
type AdminLoginInput struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=10"`
	TenantDomain string `json:"tenant_domain,omitempty"`

	// Anti-replay attack protection fields
	Nonce     string `json:"nonce,omitempty"`     // Unique request identifier
	Timestamp int64  `json:"timestamp,omitempty"` // Unix timestamp of request
	Challenge string `json:"challenge,omitempty"` // Challenge token (if using challenge-response)
	Signature string `json:"signature,omitempty"` // HMAC signature of request
}

// AuthChallenge represents a server-issued challenge for authentication
type AuthChallenge struct {
	Challenge string    `json:"challenge"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SecureLoginRequest extends login with full anti-replay protection
type SecureLoginRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	TenantDomain string `json:"tenant_domain,omitempty"`
	Nonce        string `json:"nonce"`
	Timestamp    int64  `json:"timestamp"`
	Challenge    string `json:"challenge,omitempty"`
	Signature    string `json:"signature,omitempty"`
}

// AdminPrecheckInput captures email for pre-login validation
type AdminPrecheckInput struct {
	Email         string `json:"email" binding:"required,email"`
	CurrentDomain string `json:"current_domain,omitempty"` // Domain the user is currently accessing from
}

// AdminPrecheckResponse returns user existence and tenant context
type AdminPrecheckResponse struct {
	Exists             bool     `json:"exists"`
	DisplayName        string   `json:"display_name,omitempty"`
	TenantDomain       string   `json:"tenant_domain,omitempty"`
	TenantID           string   `json:"tenant_id,omitempty"`
	NextStep           string   `json:"next_step"` // "login", "bootstrap", "register"
	RequiresPassword   bool     `json:"requires_password"`
	AvailableProviders []string `json:"available_providers,omitempty"`
}

// AdminBootstrapInput captures details for creating a new tenant with admin user
type AdminBootstrapInput struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=10"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
	TenantDomain    string `json:"tenant_domain" binding:"required"`
	Name            string `json:"name,omitempty"`
}

// AdminBootstrapResponse returns the status of bootstrap operation
type AdminBootstrapResponse struct {
	Message      string `json:"message"`
	Status       string `json:"status"` // "pending_verification", "success"
	TenantID     string `json:"tenant_id,omitempty"`
	TenantDomain string `json:"tenant_domain,omitempty"`
	UserID       string `json:"user_id,omitempty"`
}
