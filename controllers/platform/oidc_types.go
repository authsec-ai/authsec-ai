package platform

// OIDCCompleteRegistrationInput documents the payload for completing OIDC registration.
// Used only for Swagger documentation.
type OIDCCompleteRegistrationInput struct {
	TenantDomain   string `json:"tenant_domain" example:"myworkspace" binding:"required"`
	Provider       string `json:"provider" example:"google" binding:"required"`
	Email          string `json:"email" example:"user@example.com" binding:"required"`
	Name           string `json:"name" example:"Jane Doe"`
	Picture        string `json:"picture" example:"https://example.com/avatar.png"`
	ProviderUserID string `json:"provider_user_id" example:"google-oauth2|12345" binding:"required"`
}
