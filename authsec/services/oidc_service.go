package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// OIDCService handles OIDC provider interactions
type OIDCService struct {
	providerRepo  *database.OIDCProviderRepository
	stateRepo     *database.OIDCStateRepository
	identityRepo  *database.OIDCUserIdentityRepository
	httpClient    *http.Client
	requestHost   string // Store current request host for dynamic callbacks
	requestOrigin string // Store origin domain for post-auth redirect
}

// NewOIDCService creates a new OIDC service
func NewOIDCService(db *database.DBConnection) *OIDCService {
	return &OIDCService{
		providerRepo: database.NewOIDCProviderRepository(db),
		stateRepo:    database.NewOIDCStateRepository(db),
		identityRepo: database.NewOIDCUserIdentityRepository(db),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetActiveProviders returns list of active OIDC providers for login UI
func (s *OIDCService) GetActiveProviders() ([]models.OIDCProviderPublic, error) {
	providers, err := s.providerRepo.GetActiveProviders()
	if err != nil {
		return nil, err
	}

	var publicProviders []models.OIDCProviderPublic
	for _, p := range providers {
		publicProviders = append(publicProviders, models.OIDCProviderPublic{
			ProviderName: p.ProviderName,
			DisplayName:  p.DisplayName,
			IconURL:      p.IconURL,
		})
	}

	return publicProviders, nil
}

// InitiateOIDCFlow starts the OIDC authentication flow
// Returns the authorization URL to redirect the user to
func (s *OIDCService) InitiateOIDCFlow(input *models.OIDCInitiateInput, action string, tenantID *uuid.UUID) (*models.OIDCInitiateResponse, error) {
	// Get provider configuration
	provider, err := s.providerRepo.GetProviderByName(input.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %s", input.Provider)
	}

	if !provider.IsActive {
		return nil, fmt.Errorf("provider %s is not active", input.Provider)
	}

	// Generate state token (for CSRF protection and tenant context)
	stateToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}

	// Generate PKCE code verifier and challenge
	codeVerifier, err := generateSecureToken(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Store state in database (expires in 10 minutes)
	state := &models.OIDCState{
		StateToken:    stateToken,
		TenantID:      tenantID,
		TenantDomain:  input.TenantDomain,
		OriginDomain:  s.requestOrigin, // Store origin domain for post-auth redirect
		ProviderName:  input.Provider,
		Action:        action, // "login" or "register"
		CodeVerifier:  codeVerifier,
		RedirectAfter: input.RedirectAfter,
		ExpiresAt:     time.Now().Add(30 * time.Minute),
	}
	log.Printf("DEBUG InitiateOIDCFlow: Creating state with origin_domain='%s' (request_host column)", s.requestOrigin)

	if err := s.stateRepo.CreateState(state); err != nil {
		log.Printf("ERROR: Failed to store OIDC state: %v", err)
		return nil, fmt.Errorf("failed to store OIDC state: %w", err)
	}
	log.Printf("DEBUG InitiateOIDCFlow: Successfully created state with token='%s', tenant_domain='%s', origin_domain='%s', action='%s'", stateToken, input.TenantDomain, s.requestOrigin, action)

	// Build authorization URL
	callbackURL := s.getCallbackURL()
	log.Printf("DEBUG InitiateOIDCFlow: Using callbackURL='%s' for provider '%s'", callbackURL, provider.ProviderName)
	authURL, err := s.buildAuthorizationURL(provider, stateToken, codeChallenge, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build authorization URL: %w", err)
	}
	log.Printf("DEBUG InitiateOIDCFlow: Built authURL with callbackURL='%s'", callbackURL)

	return &models.OIDCInitiateResponse{
		RedirectURL: authURL,
		State:       stateToken,
	}, nil
}

// HandleCallback processes the OIDC callback and returns user info
func (s *OIDCService) HandleCallback(input *models.OIDCCallbackInput) (*models.OIDCState, *models.OIDCUserInfo, error) {
	// Check for error from provider
	if input.Error != "" {
		return nil, nil, fmt.Errorf("OIDC provider error: %s", input.Error)
	}

	// Validate and retrieve state
	log.Printf("DEBUG HandleCallback: Looking up state with token='%s'", input.State)
	state, err := s.stateRepo.GetStateByToken(input.State)
	if err != nil {
		log.Printf("ERROR HandleCallback: Failed to get state for token='%s': %v", input.State, err)
		return nil, nil, fmt.Errorf("invalid or expired state: %w", err)
	}
	log.Printf("DEBUG HandleCallback: Found state: tenant_domain='%s', action='%s', provider='%s'", state.TenantDomain, state.Action, state.ProviderName)

	// Get provider configuration
	provider, err := s.providerRepo.GetProviderByName(state.ProviderName)
	if err != nil {
		return nil, nil, fmt.Errorf("provider not found: %s", state.ProviderName)
	}

	// Get client secret from Vault
	clientSecret, err := s.getClientSecret(provider.ClientSecretVaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get client secret: %w", err)
	}

	// Exchange authorization code for tokens
	tokens, err := s.exchangeCodeForTokens(provider, input.Code, state.CodeVerifier, clientSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.getUserInfo(provider, tokens.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Delete used state
	if err := s.stateRepo.DeleteState(input.State); err != nil {
		log.Printf("Warning: failed to delete OIDC state: %v", err)
	}

	return state, userInfo, nil
}

// GetIdentityByProviderUser looks up if a provider user exists in any tenant
func (s *OIDCService) GetIdentityByProviderUser(providerName, providerUserID string) (*models.OIDCUserIdentity, error) {
	return s.identityRepo.GetIdentityByProviderUser(providerName, providerUserID)
}

// GetIdentityByTenantAndProviderUser looks up if a provider user exists in a specific tenant
func (s *OIDCService) GetIdentityByTenantAndProviderUser(tenantID uuid.UUID, providerName, providerUserID string) (*models.OIDCUserIdentity, error) {
	return s.identityRepo.GetIdentityByTenantAndProviderUser(tenantID, providerName, providerUserID)
}

// CreateIdentity creates a new OIDC user identity link
func (s *OIDCService) CreateIdentity(identity *models.OIDCUserIdentity) error {
	return s.identityRepo.CreateIdentity(identity)
}

// UpdateLastLogin updates the last login timestamp for an identity
func (s *OIDCService) UpdateLastLogin(identityID uuid.UUID) error {
	return s.identityRepo.UpdateLastLogin(identityID)
}

// GetIdentitiesByUser retrieves all OIDC identities for a user
func (s *OIDCService) GetIdentitiesByUser(tenantID, userID uuid.UUID) ([]models.OIDCUserIdentity, error) {
	return s.identityRepo.GetIdentitiesByUserID(tenantID, userID)
}

// UnlinkIdentity removes an OIDC identity from a user
func (s *OIDCService) UnlinkIdentity(tenantID, userID uuid.UUID, providerName string) error {
	return s.identityRepo.DeleteIdentity(tenantID, userID, providerName)
}

// GetTenantsByEmail finds all tenants where a user with this email has OIDC identity
func (s *OIDCService) GetTenantsByEmail(email string) ([]uuid.UUID, error) {
	return s.identityRepo.GetTenantsByProviderEmail(email)
}

// GetStateByToken retrieves OIDC state by token
func (s *OIDCService) GetStateByToken(token string) (*models.OIDCState, error) {
	return s.stateRepo.GetStateByToken(token)
}

// CleanupExpiredStates removes expired OIDC states (should be called periodically)
func (s *OIDCService) CleanupExpiredStates() error {
	return s.stateRepo.DeleteExpiredStates()
}

// ========================================
// Admin methods for managing providers
// ========================================

// GetAllProviders returns all OIDC providers (for admin)
func (s *OIDCService) GetAllProviders() ([]models.OIDCProvider, error) {
	return s.providerRepo.GetAllProviders()
}

// GetProviderByName returns a specific OIDC provider
func (s *OIDCService) GetProviderByName(name string) (*models.OIDCProvider, error) {
	return s.providerRepo.GetProviderByName(name)
}

// UpdateProvider updates an OIDC provider configuration
func (s *OIDCService) UpdateProvider(providerName string, input *models.OIDCProviderUpdateInput) error {
	return s.providerRepo.UpdateProvider(providerName, input)
}

// ========================================
// Helper methods
// ========================================

// getCallbackURL returns the OIDC callback URL
// Always uses BASE_URL from config to ensure the redirect_uri sent to OAuth providers
// (Google, GitHub, Microsoft) matches exactly what is registered in their consoles.
// Do NOT use requestHost here — that would send the API backend host (e.g., prod.api.authsec.ai)
// instead of the registered redirect URI, causing redirect_uri_mismatch errors.
func (s *OIDCService) getCallbackURL() string {
	baseURL := config.AppConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://app.authsec.dev"
	}
	callbackURL := fmt.Sprintf("%s/authsec/uflow/oidc/callback", baseURL)
	log.Printf("DEBUG getCallbackURL: Using BASE_URL='%s', callbackURL='%s'", baseURL, callbackURL)
	return callbackURL
}

// SetRequestHost sets the current request host for dynamic callback URLs
func (s *OIDCService) SetRequestHost(host string) {
	s.requestHost = host
}

// SetRequestOrigin sets the origin domain for post-auth redirect
func (s *OIDCService) SetRequestOrigin(origin string) {
	s.requestOrigin = origin
}

// GetRequestOrigin returns the stored request origin
func (s *OIDCService) GetRequestOrigin() string {
	return s.requestOrigin
}

// buildAuthorizationURL constructs the OAuth2 authorization URL
func (s *OIDCService) buildAuthorizationURL(provider *models.OIDCProvider, state, codeChallenge, callbackURL string) (string, error) {
	log.Printf("DEBUG buildAuthorizationURL: Building auth URL for provider '%s' with redirect_uri='%s'", provider.ProviderName, callbackURL)

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", callbackURL)
	params.Set("response_type", "code")
	params.Set("scope", provider.Scopes)
	params.Set("state", state)

	// Add PKCE parameters
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	// Provider-specific parameters
	switch provider.ProviderName {
	case "google":
		params.Set("access_type", "offline")
		params.Set("prompt", "select_account")
	case "github":
		// GitHub doesn't support PKCE, but we include state for CSRF
		params.Del("code_challenge")
		params.Del("code_challenge_method")
	case "microsoft":
		params.Set("response_mode", "query")
	}

	return fmt.Sprintf("%s?%s", provider.AuthorizationURL, params.Encode()), nil
}

// exchangeCodeForTokens exchanges the authorization code for access tokens
func (s *OIDCService) exchangeCodeForTokens(provider *models.OIDCProvider, code, codeVerifier, clientSecret string) (*models.OIDCTokenResponse, error) {
	callbackURL := s.getCallbackURL()

	// URL decode the code before setting it, to prevent double encoding issues
	decodedCode, err := url.QueryUnescape(code)
	if err != nil {
		log.Printf("Warning: failed to URL unescape code: %v", err)
		decodedCode = code // Use original code if unescaping fails
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", decodedCode)
	data.Set("redirect_uri", callbackURL)

	// Add PKCE verifier (except for GitHub)
	if provider.ProviderName != "github" && codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequest("POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Token exchange failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokens models.OIDCTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokens, nil
}

// getUserInfo retrieves user info from the OIDC provider
func (s *OIDCService) getUserInfo(provider *models.OIDCProvider, accessToken string) (*models.OIDCUserInfo, error) {
	req, err := http.NewRequest("GET", provider.UserinfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("UserInfo request failed: %s", string(body))
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	// Parse response based on provider
	var userInfo models.OIDCUserInfo
	switch provider.ProviderName {
	case "github":
		userInfo, err = parseGitHubUserInfo(body, accessToken, s.httpClient)
	default:
		err = json.Unmarshal(body, &userInfo)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return &userInfo, nil
}

// getClientSecret retrieves the client secret from Vault
func (s *OIDCService) getClientSecret(vaultPath string) (string, error) {
	// For now, try to get from environment as fallback
	// In production, this should read from HashiCorp Vault

	// Try Vault first
	secret, err := GetSecretFromVault(vaultPath)
	if err == nil && secret != "" {
		return secret, nil
	}

	// Fallback to environment variables
	switch {
	case strings.Contains(strings.ToLower(vaultPath), "google"):
		sec := config.AppConfig.GoogleClientSecret
		log.Printf("DEBUG getClientSecret: vault_path=%q matched 'google', secret_empty=%v", vaultPath, sec == "")
		return sec, nil
	case strings.Contains(strings.ToLower(vaultPath), "github"):
		sec := config.AppConfig.GitHubClientSecret
		log.Printf("DEBUG getClientSecret: vault_path=%q matched 'github', secret_empty=%v", vaultPath, sec == "")
		return sec, nil
	case strings.Contains(strings.ToLower(vaultPath), "microsoft"):
		sec := config.AppConfig.MicrosoftClientSecret
		log.Printf("DEBUG getClientSecret: vault_path=%q matched 'microsoft', secret_empty=%v", vaultPath, sec == "")
		return sec, nil
	}

	log.Printf("DEBUG getClientSecret: vault_path=%q did not match any known provider pattern", vaultPath)
	return "", fmt.Errorf("client secret not found for path: %s", vaultPath)
}

// ========================================
// Utility functions
// ========================================

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge generates a PKCE code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// parseGitHubUserInfo parses GitHub's userinfo response
// GitHub's response format is different from standard OIDC
func parseGitHubUserInfo(body []byte, accessToken string, client *http.Client) (models.OIDCUserInfo, error) {
	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &ghUser); err != nil {
		return models.OIDCUserInfo{}, err
	}

	userInfo := models.OIDCUserInfo{
		Sub:     fmt.Sprintf("%d", ghUser.ID),
		Name:    ghUser.Name,
		Picture: ghUser.AvatarURL,
	}

	// GitHub might not return email in main response, need to fetch from /user/emails
	if ghUser.Email != "" {
		userInfo.Email = ghUser.Email
		userInfo.EmailVerified = true
	} else {
		// Fetch primary email from GitHub
		email, err := fetchGitHubPrimaryEmail(accessToken, client)
		if err == nil && email != "" {
			userInfo.Email = email
			userInfo.EmailVerified = true
		}
	}

	return userInfo, nil
}

// fetchGitHubPrimaryEmail fetches the primary email from GitHub API
func fetchGitHubPrimaryEmail(accessToken string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch GitHub emails")
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no primary verified email found")
}

// GetSecretFromVault retrieves a secret from HashiCorp Vault
// This is a placeholder - implement based on your Vault setup
func GetSecretFromVault(path string) (string, error) {
	// TODO: Implement Vault integration
	// For now, return empty to fall back to environment variables
	return "", fmt.Errorf("vault not configured")
}
