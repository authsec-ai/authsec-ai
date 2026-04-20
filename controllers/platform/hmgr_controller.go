package platform

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/authsec-ai/authsec/config"
	hydramodels "github.com/authsec-ai/authsec/internal/hydra/models"
	hydrautils "github.com/authsec-ai/authsec/internal/hydra/utils"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// pkceEntry holds a code verifier with its expiry time.
type pkceEntry struct {
	verifier  string
	expiresAt time.Time
}

// pkceStore maps state/login_challenge → pkceEntry (TTL: 8 minutes).
var pkceStore sync.Map

// storePKCEVerifier saves a code verifier associated with a state or login challenge.
func storePKCEVerifier(key, codeVerifier string) {
	pkceStore.Store(key, pkceEntry{
		verifier:  codeVerifier,
		expiresAt: time.Now().Add(8 * time.Minute),
	})
}

// consumePKCEVerifier retrieves and deletes the stored code verifier.
// Returns an empty string if not found or expired.
func consumePKCEVerifier(key string) string {
	val, ok := pkceStore.LoadAndDelete(key)
	if !ok {
		return ""
	}
	entry := val.(pkceEntry)
	if time.Now().After(entry.expiresAt) {
		return ""
	}
	return entry.verifier
}

// HmgrController handles hydra manager authentication requests
type HmgrController struct {
	service *hydramodels.OAuthLoginService
}

// NewHmgrController creates a new HmgrController
func NewHmgrController(cfg config.Config) *HmgrController {
	return &HmgrController{
		service: hydramodels.NewOAuthLoginService(cfg),
	}
}

// StorePKCEVerifierHandler pre-registers a PKCE code_verifier from an external client
// so it can be retrieved at token-exchange time.
// POST /hmgr/pkce/store  { "state": "...", "code_verifier": "..." }
func (ctrl *HmgrController) StorePKCEVerifierHandler(c *gin.Context) {
	var req struct {
		State        string `json:"state" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if len(req.CodeVerifier) < 43 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "code_verifier must be at least 43 characters"})
		return
	}
	storePKCEVerifier(req.State, req.CodeVerifier)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetLoginPageDataHandler handles the login page data request
func (ctrl *HmgrController) GetLoginPageDataHandler(c *gin.Context) {
	loginChallenge := c.Query("login_challenge")
	if loginChallenge == "" {
		c.JSON(http.StatusBadRequest, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Missing login_challenge parameter",
		})
		return
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(loginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Failed to get login request",
		})
		return
	}

	if loginRequest.Client.ClientID == "" {
		c.JSON(http.StatusBadRequest, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Invalid login request: missing client ID",
		})
		return
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Client not found",
		})
		return
	}

	tenantIDForOIDC, _ := clientDetails.Metadata["tenant_id"].(string)
	realTenantID, _ := clientDetails.Metadata["c_id"].(string)
	tenantName, _ := clientDetails.Metadata["tenant_name"].(string)

	if tenantIDForOIDC == "" {
		c.JSON(http.StatusBadRequest, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Invalid client configuration",
		})
		return
	}

	clientID := loginRequest.Client.ClientID
	allProviders, err := ctrl.service.GetAllProvidersForTenant(tenantIDForOIDC, realTenantID, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "Failed to get authentication providers",
		})
		return
	}

	if len(allProviders) == 0 {
		c.JSON(http.StatusBadRequest, hydramodels.LoginPageDataResponse{
			Success: false,
			Error:   "No authentication providers configured",
		})
		return
	}

	oidcProviders := make([]hydramodels.OIDCProvider, 0, len(allProviders))
	for _, p := range allProviders {
		providerConfig := map[string]interface{}{"type": p.Type}
		for key, value := range p.Config {
			providerConfig[key] = value
		}
		oidcProviders = append(oidcProviders, hydramodels.OIDCProvider{
			ProviderName: p.ProviderName,
			DisplayName:  p.DisplayName,
			IsActive:     p.IsActive,
			SortOrder:    p.SortOrder,
			Config:       providerConfig,
		})
	}

	c.JSON(http.StatusOK, hydramodels.LoginPageDataResponse{
		ClientID:       strings.TrimSuffix(loginRequest.Client.ClientID, "-main-client"),
		Success:        true,
		LoginChallenge: loginChallenge,
		TenantName:     tenantName,
		ClientName:     clientDetails.ClientName,
		Providers:      oidcProviders,
		BaseURL:        config.AppConfig.BaseURL,
	})
}

// InitiateAuthHandler initiates authentication with a provider
func (ctrl *HmgrController) InitiateAuthHandler(c *gin.Context) {
	providerName := c.Param("provider")

	var req struct {
		LoginChallenge string `json:"login_challenge"`
		OriginDomain   string `json:"origin_domain,omitempty"`
		CodeVerifier   string `json:"code_verifier,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Invalid JSON body",
		})
		return
	}

	if req.LoginChallenge == "" {
		c.JSON(http.StatusBadRequest, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Missing login_challenge",
		})
		return
	}

	// Store the PKCE code_verifier by login_challenge so ExchangeTokenHandler can
	// retrieve it later.
	if req.CodeVerifier != "" {
		storePKCEVerifier(req.LoginChallenge, req.CodeVerifier)
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(req.LoginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Failed to get login request",
		})
		return
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Client not found",
		})
		return
	}

	tenantID, _ := clientDetails.Metadata["tenant_id"].(string)
	providers, err := ctrl.service.GetOIDCProvidersForTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Failed to get OIDC providers",
		})
		return
	}

	var selectedProvider *hydramodels.OIDCProvider
	for _, provider := range providers {
		if strings.EqualFold(provider.ProviderName, providerName) {
			selectedProvider = &provider
			break
		}
	}

	if selectedProvider == nil {
		c.JSON(http.StatusNotFound, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Provider not found",
		})
		return
	}

	if !selectedProvider.IsActive {
		c.JSON(http.StatusBadRequest, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Provider is not active",
		})
		return
	}

	providerConfig := selectedProvider.Config
	clientID, _ := providerConfig["client_id"].(string)
	authURL, _ := providerConfig["auth_url"].(string)
	scopes, _ := providerConfig["scopes"].([]interface{})

	scopeStrings := make([]string, len(scopes))
	for i, scope := range scopes {
		scopeStrings[i] = scope.(string)
	}

	nonce := hydrautils.GenerateCodeVerifier()
	originDomain := req.OriginDomain

	if originDomain == "" {
		originDomain = c.GetHeader("X-Forwarded-Host")
	}
	if originDomain == "" {
		if origin := c.GetHeader("Origin"); origin != "" {
			if u, err := url.Parse(origin); err == nil {
				originDomain = u.Host
			}
		}
	}
	if originDomain == "" {
		if referer := c.GetHeader("Referer"); referer != "" {
			if u, err := url.Parse(referer); err == nil {
				originDomain = u.Host
			}
		}
	}
	if originDomain == "" {
		originDomain = c.Request.Host
	}

	if originDomain != "" && tenantID != "" {
		verifiedDomains, err := hydramodels.GetVerifiedDomainsForTenant(config.DB, tenantID)
		if err == nil && len(verifiedDomains) > 0 {
			isVerified := false
			for _, d := range verifiedDomains {
				if strings.HasSuffix(originDomain, d) || strings.EqualFold(d, originDomain) {
					isVerified = true
					break
				}
			}
			if !isVerified {
				isDev := strings.Contains(originDomain, "localhost") || strings.Contains(originDomain, "127.0.0.1")
				if !isDev {
					c.JSON(http.StatusForbidden, hydramodels.AuthInitiateResponse{
						Success: false,
						Error:   "Origin domain not verified for this tenant",
					})
					return
				}
			}
		}
	}

	stateData := map[string]string{
		"login_challenge": req.LoginChallenge,
		"nonce":           nonce,
		"provider":        providerName,
		"origin_domain":   originDomain,
	}
	stateBytes, err := json.Marshal(stateData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "Failed to generate state",
		})
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	if len(clientDetails.RedirectURIs) == 0 {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{
			Success: false,
			Error:   "No registered redirect URI found for client",
		})
		return
	}
	callbackURL := clientDetails.RedirectURIs[0]

	oauthURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
		authURL,
		clientID,
		url.QueryEscape(callbackURL),
		url.QueryEscape(strings.Join(scopeStrings, " ")),
		url.QueryEscape(state),
	)

	c.JSON(http.StatusOK, hydramodels.AuthInitiateResponse{
		Success:  true,
		AuthURL:  oauthURL,
		State:    state,
		Provider: providerName,
	})
}

// HandleCallbackHandler processes the OAuth callback
func (ctrl *HmgrController) HandleCallbackHandler(c *gin.Context) {
	var req struct {
		Code  string `json:"code"`
		State string `json:"state"`
		Error string `json:"error,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.Error != "" {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{
			Success: false,
			Error:   fmt.Sprintf("OAuth provider error: %s", req.Error),
		})
		return
	}

	if req.Code == "" || req.State == "" {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{
			Success: false,
			Error:   "Missing required parameters: code or state",
		})
		return
	}

	redirectTo, userInfo, err := ctrl.ProcessOAuthCallback(req.Code, req.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{
			Success: false,
			Error:   "Authentication processing failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, hydramodels.CallbackValidationResponse{
		Success:    true,
		RedirectTo: redirectTo,
		UserInfo:   userInfo,
	})
}

// ProcessOAuthCallback processes the OAuth callback logic
func (ctrl *HmgrController) ProcessOAuthCallback(code, receivedState string) (string, *hydramodels.User, error) {
	var stateData map[string]string

	stateBytes, err := base64.URLEncoding.DecodeString(receivedState)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode state: %w", err)
	}

	if err := json.Unmarshal(stateBytes, &stateData); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	loginChallenge := stateData["login_challenge"]
	providerName := stateData["provider"]
	originDomain := stateData["origin_domain"]

	if loginChallenge == "" {
		return "", nil, fmt.Errorf("missing login_challenge in state")
	}
	if providerName == "" {
		return "", nil, fmt.Errorf("missing provider in state")
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(loginChallenge)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get login request: %w", err)
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get client details: %w", err)
	}

	tenantID, _ := clientDetails.Metadata["tenant_id"].(string)
	if tenantID == "" {
		return "", nil, fmt.Errorf("missing tenant_id in client metadata")
	}

	providers, err := ctrl.service.GetOIDCProvidersForTenant(tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get OIDC providers: %w", err)
	}

	var selectedProvider *hydramodels.OIDCProvider
	for _, provider := range providers {
		if strings.EqualFold(provider.ProviderName, providerName) {
			selectedProvider = &provider
			break
		}
	}

	if selectedProvider == nil {
		return "", nil, fmt.Errorf("provider %s not found", providerName)
	}

	if len(clientDetails.RedirectURIs) == 0 {
		return "", nil, fmt.Errorf("no registered redirect URI found for client")
	}
	redirectURI := clientDetails.RedirectURIs[0]

	ctx := context.Background()
	tokenResponse, err := ctrl.service.ExchangeCodeForTokens(ctx, selectedProvider, code, redirectURI)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	accessToken, ok := tokenResponse["access_token"].(string)
	if !ok || accessToken == "" {
		return "", nil, fmt.Errorf("no access token in response")
	}

	userInfo, err := ctrl.service.GetUserInfo(ctx, selectedProvider, accessToken)
	if err != nil {
		// For Microsoft, fall back to decoding the id_token instead of calling Graph API.
		// Graph API requires User.Read permission which may not be granted; the id_token
		// already contains sub/email/name/preferred_username from openid+profile+email scopes.
		if strings.EqualFold(providerName, "microsoft") || strings.EqualFold(providerName, "azure") {
			if idToken, ok := tokenResponse["id_token"].(string); ok && idToken != "" {
				log.Printf("Microsoft Graph userinfo failed (%v), falling back to id_token", err)
				userInfo, err = extractClaimsFromIDToken(idToken)
				if err != nil {
					return "", nil, fmt.Errorf("failed to extract claims from Microsoft id_token: %w", err)
				}
			} else {
				return "", nil, fmt.Errorf("failed to get user info: %w", err)
			}
		} else {
			return "", nil, fmt.Errorf("failed to get user info: %w", err)
		}
	}

	user, userID, err := ctrl.ExtractUserFromProviderResponse(providerName, userInfo)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract user info: %w", err)
	}

	parsedTenantID, err := uuid.Parse(clientDetails.Metadata["c_id"].(string))
	if err != nil {
		return "", nil, fmt.Errorf("invalid tenant ID format (c_id): %w", err)
	}

	clientIDStr, ok := clientDetails.Metadata["tenant_id"].(string)
	if !ok || clientIDStr == "" {
		return "", nil, fmt.Errorf("missing tenant_id in client metadata")
	}

	parsedClientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return "", nil, fmt.Errorf("invalid client ID format (tenant_id): %w", err)
	}

	user.ClientID = parsedClientID
	user.TenantID = parsedTenantID

	user, err = ctrl.service.CreateOrUpdateUser(accessToken, user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create/update user: %w", err)
	}

	acceptResponse, err := ctrl.service.AcceptHydraLoginRequestWithContext(loginChallenge, userID, map[string]interface{}{
		"email":       user.Email,
		"name":        user.Name,
		"username":    user.Username,
		"provider":    user.Provider,
		"provider_id": user.ProviderID,
		"tenant_id":   user.TenantID,
		"project_id":  user.ProjectID,
		"avatar_url":  user.AvatarURL,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to accept login request: %w", err)
	}

	finalRedirectURL := acceptResponse.RedirectTo
	safeOriginDomain := hmgrGetSafeOriginDomainForRedirect(acceptResponse.RedirectTo, originDomain)
	if safeOriginDomain != "" {
		finalRedirectURL = hmgrReplaceRedirectDomain(acceptResponse.RedirectTo, safeOriginDomain)
	}

	return finalRedirectURL, user, nil
}

// ExtractUserFromProviderResponse extracts user information from provider response
func (ctrl *HmgrController) ExtractUserFromProviderResponse(providerName string, userInfo map[string]interface{}) (*hydramodels.User, string, error) {
	var userID, email, name, username, avatarURL, providerUserID string

	switch strings.ToLower(providerName) {
	case "github":
		if id, ok := userInfo["id"].(float64); ok {
			providerUserID = fmt.Sprintf("%.0f", id)
			userID = fmt.Sprintf("github-%.0f", id)
		} else if id, ok := userInfo["id"].(int); ok {
			providerUserID = fmt.Sprintf("%d", id)
			userID = fmt.Sprintf("github-%d", id)
		} else if idStr, ok := userInfo["id"].(string); ok {
			providerUserID = idStr
			userID = fmt.Sprintf("github-%s", idStr)
		}
		if emailVal, exists := userInfo["email"]; exists && emailVal != nil {
			email, _ = emailVal.(string)
		}
		name, _ = userInfo["name"].(string)
		username, _ = userInfo["login"].(string)
		avatarURL, _ = userInfo["avatar_url"].(string)
		if email == "" && username != "" {
			email = fmt.Sprintf("%s@users.noreply.github.com", username)
		}

	case "google":
		if sub, ok := userInfo["sub"].(string); ok && sub != "" {
			providerUserID = sub
			userID = fmt.Sprintf("google-%s", sub)
		}
		email, _ = userInfo["email"].(string)
		name, _ = userInfo["name"].(string)
		if givenName, ok1 := userInfo["given_name"].(string); ok1 {
			if familyName, ok2 := userInfo["family_name"].(string); ok2 {
				name = fmt.Sprintf("%s %s", givenName, familyName)
			}
		}
		if email != "" {
			username = strings.Split(email, "@")[0]
		}
		avatarURL, _ = userInfo["picture"].(string)

	case "linkedin":
		if id, ok := userInfo["id"].(string); ok && id != "" {
			providerUserID = id
			userID = fmt.Sprintf("linkedin-%s", id)
		}
		email, _ = userInfo["emailAddress"].(string)
		if firstName, ok := userInfo["localizedFirstName"].(string); ok {
			if lastName, ok := userInfo["localizedLastName"].(string); ok {
				name = fmt.Sprintf("%s %s", firstName, lastName)
			} else {
				name = firstName
			}
		}
		if email != "" {
			username = strings.Split(email, "@")[0]
		}

	case "microsoft", "azure":
		if id, ok := userInfo["id"].(string); ok && id != "" {
			providerUserID = id
			userID = fmt.Sprintf("microsoft-%s", id)
		} else if oid, ok := userInfo["oid"].(string); ok && oid != "" {
			providerUserID = oid
			userID = fmt.Sprintf("microsoft-%s", oid)
		} else if sub, ok := userInfo["sub"].(string); ok && sub != "" {
			providerUserID = sub
			userID = fmt.Sprintf("microsoft-%s", sub)
		}
		email, _ = userInfo["email"].(string)
		if email == "" {
			email, _ = userInfo["mail"].(string)
		}
		if email == "" {
			email, _ = userInfo["userPrincipalName"].(string)
		}
		if email == "" {
			email, _ = userInfo["preferred_username"].(string)
		}
		name, _ = userInfo["displayName"].(string)
		if name == "" {
			name, _ = userInfo["name"].(string)
		}
		username, _ = userInfo["mailNickname"].(string)
		if username == "" && email != "" {
			username = strings.Split(email, "@")[0]
		}

	default:
		if sub, ok := userInfo["sub"].(string); ok && sub != "" {
			providerUserID = sub
			userID = fmt.Sprintf("%s-%s", providerName, sub)
		} else if id, ok := userInfo["id"].(string); ok && id != "" {
			providerUserID = id
			userID = fmt.Sprintf("%s-%s", providerName, id)
		} else if id, ok := userInfo["id"].(float64); ok {
			providerUserID = fmt.Sprintf("%.0f", id)
			userID = fmt.Sprintf("%s-%.0f", providerName, id)
		}
		email, _ = userInfo["email"].(string)
		name, _ = userInfo["name"].(string)
		username, _ = userInfo["username"].(string)
		if username == "" {
			username, _ = userInfo["preferred_username"].(string)
		}
		avatarURL, _ = userInfo["avatar_url"].(string)
		if avatarURL == "" {
			avatarURL, _ = userInfo["picture"].(string)
		}
	}

	if userID == "" || providerUserID == "" {
		if email != "" {
			hash := sha256.Sum256([]byte(email))
			providerUserID = fmt.Sprintf("email-%x", hash[:8])
			userID = fmt.Sprintf("%s-%s", providerName, providerUserID)
		} else if username != "" {
			hash := sha256.Sum256([]byte(username))
			providerUserID = fmt.Sprintf("username-%x", hash[:8])
			userID = fmt.Sprintf("%s-%s", providerName, providerUserID)
		} else {
			return nil, "", fmt.Errorf("unable to extract user identifier from provider response")
		}
	}

	if username == "" && email != "" {
		username = strings.Split(email, "@")[0]
	}
	if name == "" {
		if username != "" {
			name = username
		} else if email != "" {
			name = email
		}
	}
	if email == "" {
		return nil, "", fmt.Errorf("no email found in provider response from %s", providerName)
	}

	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal user info: %w", err)
	}

	now := time.Now()
	user := &hydramodels.User{
		Email:        email,
		Username:     &username,
		Name:         name,
		Provider:     providerName,
		ProviderID:   providerUserID,
		ProviderData: datatypes.JSON(userInfoJSON),
		AvatarURL:    &avatarURL,
		LastLogin:    &now,
		Active:       true,
	}
	return user, userID, nil
}

// ExchangeTokenHandler handles token exchange requests
func (ctrl *HmgrController) ExchangeTokenHandler(c *gin.Context) {
	var req hydramodels.TokenExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body: " + err.Error()})
		return
	}

	if !strings.HasPrefix(req.Code, "ory_ac_") {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid authorization code format"})
		return
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(req.LoginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve login information"})
		return
	}

	clientID := loginRequest.Client.ClientID
	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve client information"})
		return
	}

	orgID := clientDetails.Metadata["c_id"].(string)

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to tenant database"})
		return
	}

	var client hydramodels.Client
	tenantIDStr := clientDetails.Metadata["tenant_id"].(string)
	if err := tenantDB.Where("tenant_id = ? AND active = ? AND client_id = ?", orgID, true, tenantIDStr).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve client information"})
		return
	}

	clientSecret, err := config.SecretInVault(orgID, client.ProjectID.String(), client.ClientID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve client secret"})
		return
	}

	// Retrieve the stored PKCE code_verifier.
	// Priority order:
	//   1. Stored by state (GenerateLoginURLHandler path — backend-owned PKCE)
	//   2. Stored by login_challenge (server-side flows)
	//   3. Client-supplied in the request body (backward compat while React still owns PKCE)
	codeVerifier := consumePKCEVerifier(req.State)
	if codeVerifier == "" {
		codeVerifier = consumePKCEVerifier(req.LoginChallenge)
	}
	if codeVerifier == "" {
		codeVerifier = req.CodeVerifier
	}

	ctx := context.Background()
	tokens, err := ctrl.ExchangeCodeForHydraTokens(ctx, clientID, clientSecret, req.Code, req.RedirectURI, codeVerifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to exchange code for tokens: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// ExchangeCodeForHydraTokens exchanges an authorization code for tokens with Hydra
func (ctrl *HmgrController) ExchangeCodeForHydraTokens(ctx context.Context, clientID, clientSecret, code, redirectURI, codeVerifier string) (*hydramodels.TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/token", config.AppConfig.HydraPublicURL)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken, _ := tokenResponse["access_token"].(string)
	return &hydramodels.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(tokenResponse["expires_in"].(float64)),
	}, nil
}

// LoginRedirectHandler handles login redirects
func (ctrl *HmgrController) LoginRedirectHandler(c *gin.Context) {
	loginChallenge := c.Query("login_challenge")
	if loginChallenge == "" {
		c.String(http.StatusBadRequest, "Missing login_challenge parameter")
		return
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(loginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{Success: false, Error: "Failed to get login request"})
		return
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil || len(clientDetails.RedirectURIs) == 0 {
		c.JSON(http.StatusInternalServerError, hydramodels.AuthInitiateResponse{Success: false, Error: "No registered redirect URI found for client"})
		return
	}

	var callbackURL string
	for _, uri := range clientDetails.RedirectURIs {
		if strings.HasSuffix(uri, "/oidc/auth/callback") {
			callbackURL = uri
			break
		}
	}
	if callbackURL == "" {
		callbackURL = clientDetails.RedirectURIs[0]
	}

	baseURL := strings.TrimSuffix(callbackURL, "/oidc/auth/callback")

	if tenantIDObj, ok := clientDetails.Metadata["tenant_id"].(string); ok && tenantIDObj != "" {
		verifiedDomains, err := hydramodels.GetVerifiedDomainsForTenant(config.DB, tenantIDObj)
		if err == nil && len(verifiedDomains) > 0 {
			if u, err := url.Parse(baseURL); err == nil {
				host := u.Hostname()
				isVerified := false
				for _, d := range verifiedDomains {
					if strings.EqualFold(d, host) {
						isVerified = true
						break
					}
				}
				if !isVerified {
					isDev := strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1")
					if !isDev {
						c.JSON(http.StatusForbidden, hydramodels.AuthInitiateResponse{Success: false, Error: "Security violation: Redirect host not verified for this tenant"})
						return
					}
				}
			}
		}
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("%s/oidc/login?login_challenge=%s", baseURL, loginChallenge))
}

// ConsentHandler handles consent requests
func (ctrl *HmgrController) ConsentHandler(c *gin.Context) {
	consentChallenge := c.Query("consent_challenge")
	if consentChallenge == "" {
		c.String(http.StatusBadRequest, "Missing consent_challenge parameter")
		return
	}

	consentRequest, err := ctrl.service.GetHydraConsentRequest(consentChallenge)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get consent request")
		return
	}

	acceptResponse, err := ctrl.service.AcceptHydraConsentRequest(consentChallenge, consentRequest)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to complete consent")
		return
	}

	c.Redirect(http.StatusFound, acceptResponse.RedirectTo)
}

// HealthHandler provides a health check endpoint
func (ctrl *HmgrController) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":          "healthy",
		"service":         "oauth-login-service-api",
		"timestamp":       time.Now(),
		"hydra_admin_url": config.AppConfig.HydraAdminURL,
		"base_url":        config.AppConfig.BaseURL,
		"react_app_url":   config.AppConfig.ReactAppURL,
	})
}

// LoginChallengeHandler handles login challenge queries
func (ctrl *HmgrController) LoginChallengeHandler(c *gin.Context) {
	loginChallenge := c.Query("login_challenge")
	if loginChallenge == "" {
		c.String(http.StatusBadRequest, "Missing login_challenge parameter")
		return
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(loginChallenge)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get login request")
		return
	}

	c.JSON(http.StatusOK, loginRequest.Client)
}

// GenerateLoginURLHandler generates a login URL for testing
func (ctrl *HmgrController) GenerateLoginURLHandler(c *gin.Context) {
	var req struct {
		TenantID    string `json:"tenant_id"`
		OrgID       string `json:"org_id"`
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON")
		return
	}

	clients, err := ctrl.service.GetAllHydraClients()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get clients")
		return
	}

	var tenantClientID string
	for _, client := range clients {
		if tenantID, ok := client.Metadata["tenant_id"].(string); ok && tenantID == req.TenantID {
			if orgID, ok := client.Metadata["org_id"].(string); ok && orgID == req.OrgID {
				if clientType, ok := client.Metadata["type"].(string); ok && clientType == "tenant_main_client" {
					tenantClientID = client.ClientID
					break
				}
			}
		}
	}

	if tenantClientID == "" {
		c.String(http.StatusNotFound, "Tenant client not found")
		return
	}

	codeVerifier := hydrautils.GenerateCodeVerifier()
	codeChallenge := hydrautils.GenerateCodeChallenge(codeVerifier)

	// Store code_verifier server-side, keyed by state.
	// The state value will be echoed back in the exchange-token request, allowing
	// retrieval at token exchange time without ever exposing the verifier to the client.
	if req.State != "" {
		storePKCEVerifier(req.State, codeVerifier)
	}

	oauthURL := fmt.Sprintf("%s/oauth2/auth?client_id=%s&response_type=code&scope=openid+profile+email&redirect_uri=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		config.AppConfig.HydraPublicURL,
		tenantClientID,
		url.QueryEscape(req.RedirectURI),
		req.State,
		codeChallenge,
	)

	c.JSON(http.StatusOK, map[string]interface{}{
		"success":          true,
		"tenant_client_id": tenantClientID,
		"oauth_url":        oauthURL,
		"login_endpoint":   fmt.Sprintf("%s/login", config.AppConfig.BaseURL),
		"react_login_url":  fmt.Sprintf("%s/oidc/login", config.AppConfig.ReactAppURL),
	})
}

// --- SAML Handlers ---

// InitiateSAMLAuthHandler initiates SAML authentication with a provider
func (ctrl *HmgrController) InitiateSAMLAuthHandler(c *gin.Context) {
	providerName := c.Param("provider")

	var req struct {
		LoginChallenge string `json:"login_challenge" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.SAMLInitiateResponse{Success: false, Error: "Invalid request: " + err.Error()})
		return
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(req.LoginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.SAMLInitiateResponse{Success: false, Error: "Failed to get login request"})
		return
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, hydramodels.SAMLInitiateResponse{Success: false, Error: "Client not found"})
		return
	}

	realTenantID, _ := clientDetails.Metadata["c_id"].(string)
	if realTenantID == "" {
		c.JSON(http.StatusBadRequest, hydramodels.SAMLInitiateResponse{Success: false, Error: "Invalid client configuration - missing c_id"})
		return
	}

	clientID := loginRequest.Client.ClientID
	samlProvider, err := ctrl.service.GetSAMLProvider(realTenantID, providerName, clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, hydramodels.SAMLInitiateResponse{Success: false, Error: "SAML provider not found: " + err.Error()})
		return
	}

	if !samlProvider.IsActive {
		c.JSON(http.StatusBadRequest, hydramodels.SAMLInitiateResponse{Success: false, Error: "Provider is not active"})
		return
	}

	samlRequest, relayState, err := ctrl.service.CreateSAMLRequest(samlProvider, req.LoginChallenge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.SAMLInitiateResponse{Success: false, Error: "Failed to create SAML request"})
		return
	}

	ssoURL := fmt.Sprintf("%s?SAMLRequest=%s&RelayState=%s",
		samlProvider.SSOURL,
		url.QueryEscape(samlRequest),
		url.QueryEscape(relayState),
	)

	c.JSON(http.StatusOK, hydramodels.SAMLInitiateResponse{
		Success:     true,
		SSOURL:      ssoURL,
		SAMLRequest: samlRequest,
		RelayState:  relayState,
		Provider:    providerName,
	})
}

// HandleSAMLACSHandler handles SAML Assertion Consumer Service (ACS) callback
func (ctrl *HmgrController) HandleSAMLACSHandler(c *gin.Context) {
	var req hydramodels.SAMLCallbackRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid SAML response: " + err.Error()})
		return
	}

	if req.SAMLResponse == "" || req.RelayState == "" {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Missing required SAML parameters"})
		return
	}

	assertion, loginChallenge, providerName, tenantID, _, err := ctrl.service.ValidateSAMLResponse(req.SAMLResponse, req.RelayState)
	if err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid SAML response: " + err.Error()})
		return
	}

	redirectTo, user, err := ctrl.ProcessSAMLAssertion(assertion, loginChallenge, providerName, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Authentication processing failed: " + err.Error()})
		return
	}

	parsedURL, err := url.Parse(redirectTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Failed to generate redirect URL"})
		return
	}

	redirectURI := parsedURL.Query().Get("redirect_uri")
	if redirectURI == "" {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid OAuth redirect URL"})
		return
	}

	frontendURL, err := url.Parse(redirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid frontend URL"})
		return
	}

	frontendBaseURL := fmt.Sprintf("%s://%s", frontendURL.Scheme, frontendURL.Host)
	redirectURL := fmt.Sprintf("%s/oidc/login?login_challenge=%s&success=true&user_id=%s&user_email=%s&user_name=%s&provider=%s&client_id=%s&tenant_id=%s&project_id=%s&provider_id=%s&active=%t",
		frontendBaseURL,
		url.QueryEscape(loginChallenge),
		url.QueryEscape(user.ID.String()),
		url.QueryEscape(user.Email),
		url.QueryEscape(user.Name),
		url.QueryEscape(user.Provider),
		url.QueryEscape(user.ClientID.String()),
		url.QueryEscape(user.TenantID.String()),
		url.QueryEscape(user.ProjectID.String()),
		url.QueryEscape(user.ProviderID),
		user.Active,
	)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Authentication Successful</title></head><body><p>Authentication successful. Redirecting...</p><script>window.location.href = "%s";</script><noscript><a href="%s">Click here to continue</a></noscript></body></html>`, redirectURL, redirectURL))
}

// HandleSAMLACSClientHandler handles client-specific ACS callback
func (ctrl *HmgrController) HandleSAMLACSClientHandler(c *gin.Context) {
	tenantIDParam := c.Param("tenant_id")
	clientIDParam := c.Param("client_id")

	var req hydramodels.SAMLCallbackRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid SAML response: " + err.Error()})
		return
	}

	if req.SAMLResponse == "" {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Missing SAMLResponse"})
		return
	}

	assertion, loginChallenge, providerName, tenantID, clientID, err := ctrl.service.ValidateSAMLResponse(req.SAMLResponse, req.RelayState)
	if err != nil {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid SAML response: " + err.Error()})
		return
	}

	if tenantIDParam != tenantID {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Tenant ID mismatch"})
		return
	}
	if clientIDParam != clientID {
		c.JSON(http.StatusBadRequest, hydramodels.CallbackValidationResponse{Success: false, Error: "Client ID mismatch"})
		return
	}

	redirectTo, user, err := ctrl.ProcessSAMLAssertion(assertion, loginChallenge, providerName, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Authentication processing failed: " + err.Error()})
		return
	}

	parsedURL, err := url.Parse(redirectTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Failed to generate redirect URL"})
		return
	}

	redirectURI := parsedURL.Query().Get("redirect_uri")
	if redirectURI == "" {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid OAuth redirect URL"})
		return
	}

	frontendURL, err := url.Parse(redirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, hydramodels.CallbackValidationResponse{Success: false, Error: "Invalid frontend URL"})
		return
	}

	frontendBaseURL := fmt.Sprintf("%s://%s", frontendURL.Scheme, frontendURL.Host)
	redirectURL := fmt.Sprintf("%s/oidc/login?login_challenge=%s&success=true&user_id=%s&user_email=%s&user_name=%s&provider=%s&client_id=%s&tenant_id=%s&project_id=%s&provider_id=%s&active=%t",
		frontendBaseURL,
		url.QueryEscape(loginChallenge),
		url.QueryEscape(user.ID.String()),
		url.QueryEscape(user.Email),
		url.QueryEscape(user.Name),
		url.QueryEscape(user.Provider),
		url.QueryEscape(user.ClientID.String()),
		url.QueryEscape(user.TenantID.String()),
		url.QueryEscape(user.ProjectID.String()),
		url.QueryEscape(user.ProviderID),
		user.Active,
	)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Authentication Successful</title></head><body><p>Authentication successful. Redirecting...</p><script>window.location.href = "%s";</script><noscript><a href="%s">Click here to continue</a></noscript></body></html>`, redirectURL, redirectURL))
}

// ProcessSAMLAssertion processes a SAML assertion and creates/updates user
func (ctrl *HmgrController) ProcessSAMLAssertion(assertion *hydramodels.SAMLAssertion, loginChallenge, providerName, tenantID string) (string, *hydramodels.User, error) {
	if err := hydrautils.ValidateEmail(assertion.Email); err != nil {
		return "", nil, fmt.Errorf("invalid SAML email: %w", err)
	}

	firstName, err := hydrautils.ValidateSAMLAttribute(assertion.FirstName, "FirstName", 100)
	if err != nil {
		return "", nil, fmt.Errorf("invalid first name: %w", err)
	}

	lastName, err := hydrautils.ValidateSAMLAttribute(assertion.LastName, "LastName", 100)
	if err != nil {
		return "", nil, fmt.Errorf("invalid last name: %w", err)
	}

	nameID, err := hydrautils.SanitizeString(assertion.NameID, 255)
	if err != nil {
		return "", nil, fmt.Errorf("invalid NameID: %w", err)
	}
	if nameID == "" {
		return "", nil, fmt.Errorf("NameID cannot be empty")
	}

	loginRequest, err := ctrl.service.GetHydraLoginRequest(loginChallenge)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get login request: %w", err)
	}

	clientDetails, _, err := ctrl.service.GetHydraClient(loginRequest.Client.ClientID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get client details: %w", err)
	}

	clientIDFromMetadata, _ := clientDetails.Metadata["tenant_id"].(string)
	realTenantID, _ := clientDetails.Metadata["c_id"].(string)

	if realTenantID != tenantID {
		return "", nil, fmt.Errorf("tenant ID mismatch: expected %s, got %s", realTenantID, tenantID)
	}

	name := fmt.Sprintf("%s %s", firstName, lastName)
	if name == " " || name == "" {
		name = assertion.Email
	}
	name, err = hydrautils.ValidateName(name)
	if err != nil {
		return "", nil, fmt.Errorf("invalid user name: %w", err)
	}

	username := assertion.Email
	if strings.Contains(username, "@") {
		username = strings.Split(username, "@")[0]
	}

	parsedTenantID, err := hydrautils.ValidateUUID(realTenantID, "tenant_id")
	if err != nil {
		return "", nil, err
	}

	parsedClientID, err := hydrautils.ValidateUUID(clientIDFromMetadata, "client_id")
	if err != nil {
		return "", nil, err
	}

	user := &hydramodels.User{
		Email:      assertion.Email,
		Username:   &username,
		Name:       name,
		Provider:   "saml-" + providerName,
		ProviderID: nameID,
		ClientID:   parsedClientID,
		TenantID:   parsedTenantID,
		Active:     true,
	}

	user, err = ctrl.service.CreateOrUpdateUser("", user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create/update user: %w", err)
	}

	userID := fmt.Sprintf("saml-%s-%s", providerName, nameID)
	acceptResponse, err := ctrl.service.AcceptHydraLoginRequestWithContext(loginChallenge, userID, map[string]interface{}{
		"email":       user.Email,
		"name":        user.Name,
		"username":    user.Username,
		"provider":    user.Provider,
		"provider_id": user.ProviderID,
		"tenant_id":   user.TenantID,
		"client_id":   user.ClientID,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to accept login request: %w", err)
	}

	return acceptResponse.RedirectTo, user, nil
}

// GetSAMLMetadataHandler returns SP metadata for a tenant and client
func (ctrl *HmgrController) GetSAMLMetadataHandler(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	clientIDStr := c.Param("client_id")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.XML(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		c.XML(http.StatusBadRequest, gin.H{"error": "Invalid client_id format"})
		return
	}

	metadata, err := ctrl.service.GenerateSAMLMetadata(tenantID, clientID)
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{"error": "Failed to generate metadata"})
		return
	}

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, metadata)
}

// CreateSAMLProviderHandler creates a new SAML provider
func (ctrl *HmgrController) CreateSAMLProviderHandler(c *gin.Context) {
	var req hydramodels.SAMLProviderConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing tenant_id"})
		return
	}

	clientIDStr := c.GetString("client_id")
	if clientIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing client_id"})
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid client_id format"})
		return
	}

	attributeMapping, _ := json.Marshal(req.AttributeMapping)
	provider := &hydramodels.SAMLProvider{
		ClientID:         clientID,
		ProviderName:     req.ProviderName,
		DisplayName:      req.DisplayName,
		EntityID:         req.EntityID,
		SSOURL:           req.SSOURL,
		SLOURL:           req.SLOURL,
		Certificate:      req.Certificate,
		MetadataURL:      req.MetadataURL,
		NameIDFormat:     req.NameIDFormat,
		AttributeMapping: attributeMapping,
		IsActive:         req.IsActive,
		SortOrder:        req.SortOrder,
	}

	createdProvider, err := ctrl.service.CreateSAMLProvider(tenantID, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create SAML provider: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "provider": createdProvider})
}

// UpdateSAMLProviderHandler updates an existing SAML provider
func (ctrl *HmgrController) UpdateSAMLProviderHandler(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid provider ID"})
		return
	}

	var req hydramodels.SAMLProviderConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = c.GetHeader("X-Tenant-ID")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing tenant_id"})
		return
	}

	clientID := c.Query("client_id")
	attributeMapping, _ := json.Marshal(req.AttributeMapping)
	updates := &hydramodels.SAMLProvider{
		ProviderName:     req.ProviderName,
		DisplayName:      req.DisplayName,
		EntityID:         req.EntityID,
		SSOURL:           req.SSOURL,
		SLOURL:           req.SLOURL,
		Certificate:      req.Certificate,
		MetadataURL:      req.MetadataURL,
		NameIDFormat:     req.NameIDFormat,
		AttributeMapping: attributeMapping,
		IsActive:         req.IsActive,
		SortOrder:        req.SortOrder,
	}

	updatedProvider, err := ctrl.service.UpdateSAMLProvider(tenantID, providerID, clientID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update SAML provider: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "provider": updatedProvider})
}

// DeleteSAMLProviderHandler deletes a SAML provider
func (ctrl *HmgrController) DeleteSAMLProviderHandler(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid provider ID"})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = c.GetHeader("X-Tenant-ID")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing tenant_id"})
		return
	}

	clientID := c.Query("client_id")
	if err := ctrl.service.DeleteSAMLProvider(tenantID, providerID, clientID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete SAML provider: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "SAML provider deleted successfully"})
}

// GetSAMLProvidersHandler lists all SAML providers for a tenant
func (ctrl *HmgrController) GetSAMLProvidersHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = c.GetHeader("X-Tenant-ID")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing tenant_id"})
		return
	}

	clientID := c.Query("client_id")
	providers, err := ctrl.service.GetSAMLProvidersForTenant(tenantID, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get SAML providers: " + err.Error()})
		return
	}

	response := gin.H{
		"success":   true,
		"providers": providers,
		"count":     len(providers),
		"tenant_id": tenantID,
	}
	if clientID != "" {
		response["filtered_by_client_id"] = clientID
	}

	c.JSON(http.StatusOK, response)
}

// TestSAMLProviderHandler tests SAML provider configuration
func (ctrl *HmgrController) TestSAMLProviderHandler(c *gin.Context) {
	var req struct {
		TenantID     string `json:"tenant_id" binding:"required"`
		ProviderName string `json:"provider_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	provider, err := ctrl.service.GetSAMLProvider(req.TenantID, req.ProviderName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Provider not found: " + err.Error()})
		return
	}

	cfg := ctrl.service.GetConfig()
	metadataURL := fmt.Sprintf("%s/saml/metadata/%s/%s", cfg.BaseURL, provider.TenantID.String(), provider.ClientID.String())
	acsURLShared := fmt.Sprintf("%s/saml/acs", cfg.BaseURL)
	acsURLClient := fmt.Sprintf("%s/saml/acs/%s/%s", cfg.BaseURL, provider.TenantID.String(), provider.ClientID.String())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"provider": gin.H{
			"name":         provider.ProviderName,
			"display_name": provider.DisplayName,
			"entity_id":    provider.EntityID,
			"sso_url":      provider.SSOURL,
			"is_active":    provider.IsActive,
			"client_id":    provider.ClientID.String(),
		},
		"sp_metadata_url": metadataURL,
		"acs_url_client":  acsURLClient,
		"acs_url_shared":  acsURLShared,
	})
}

// --- Admin stub handlers ---

func (ctrl *HmgrController) GetUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetUsers endpoint - to be implemented", "users": []interface{}{}})
}
func (ctrl *HmgrController) CreateUserHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "CreateUser endpoint - to be implemented"})
}
func (ctrl *HmgrController) UpdateUserHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "UpdateUser endpoint - to be implemented", "user_id": c.Param("id")})
}
func (ctrl *HmgrController) DeleteUserHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "DeleteUser endpoint - to be implemented", "user_id": c.Param("id")})
}
func (ctrl *HmgrController) GetTenantsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetTenants endpoint - to be implemented", "tenants": []interface{}{}})
}
func (ctrl *HmgrController) CreateTenantHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "CreateTenant endpoint - to be implemented"})
}
func (ctrl *HmgrController) UpdateTenantHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "UpdateTenant endpoint - to be implemented", "tenant_id": c.Param("id")})
}
func (ctrl *HmgrController) DeleteTenantHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "DeleteTenant endpoint - to be implemented", "tenant_id": c.Param("id")})
}
func (ctrl *HmgrController) GetRolesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetRoles endpoint - to be implemented", "roles": []interface{}{}})
}
func (ctrl *HmgrController) CreateRoleHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "CreateRole endpoint - to be implemented"})
}
func (ctrl *HmgrController) UpdateRoleHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "UpdateRole endpoint - to be implemented", "role_id": c.Param("id")})
}
func (ctrl *HmgrController) DeleteRoleHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "DeleteRole endpoint - to be implemented", "role_id": c.Param("id")})
}
func (ctrl *HmgrController) GetPermissionsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetPermissions endpoint - to be implemented", "permissions": []interface{}{}})
}
func (ctrl *HmgrController) CreatePermissionHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "CreatePermission endpoint - to be implemented"})
}
func (ctrl *HmgrController) AssignUserRoleHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "AssignUserRole endpoint - to be implemented", "user_id": c.Param("id")})
}
func (ctrl *HmgrController) RemoveUserRoleHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "RemoveUserRole endpoint - to be implemented", "user_id": c.Param("id"), "role_id": c.Param("role_id")})
}
func (ctrl *HmgrController) GetProfileHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "GetProfile endpoint - to be implemented", "profile": map[string]interface{}{}})
}
func (ctrl *HmgrController) UpdateProfileHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "UpdateProfile endpoint - to be implemented"})
}

// extractClaimsFromIDToken decodes a JWT id_token without signature verification
// and returns its claims as a map. Used as a fallback when the userinfo endpoint
// is unavailable (e.g. Microsoft Graph 403 due to missing User.Read permission).
func extractClaimsFromIDToken(idToken string) (map[string]interface{}, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode id_token payload: %w", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal id_token claims: %w", err)
	}
	return claims, nil
}

// --- Helper functions ---

func hmgrReplaceRedirectDomain(redirectURL, newDomain string) string {
	u, err := url.Parse(redirectURL)
	if err != nil {
		return redirectURL
	}
	normalizedDomain := hmgrNormalizeHost(newDomain)
	if normalizedDomain == "" {
		return redirectURL
	}
	u.Host = normalizedDomain
	if !strings.Contains(normalizedDomain, "localhost") && !strings.Contains(normalizedDomain, "127.0.0.1") {
		u.Scheme = "https"
	}
	return u.String()
}

func hmgrGetSafeOriginDomainForRedirect(redirectURL, originDomain string) string {
	normalizedOrigin := hmgrNormalizeHost(originDomain)
	if normalizedOrigin == "" {
		return ""
	}

	u, err := url.Parse(redirectURL)
	if err != nil {
		return ""
	}

	redirectURIParam := u.Query().Get("redirect_uri")
	if redirectURIParam != "" {
		parsedRedirectURI, err := url.Parse(redirectURIParam)
		if err != nil {
			return ""
		}
		redirectURIHost := hmgrNormalizeHost(parsedRedirectURI.Host)
		if redirectURIHost == "" || !strings.EqualFold(redirectURIHost, normalizedOrigin) {
			return ""
		}
	}
	return normalizedOrigin
}

func hmgrNormalizeHost(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if strings.Contains(v, "://") {
		if parsed, err := url.Parse(v); err == nil {
			v = parsed.Host
		}
	}
	if strings.Contains(v, "/") {
		if parsed, err := url.Parse("https://" + v); err == nil {
			v = parsed.Host
		}
	}
	return strings.TrimSpace(v)
}
