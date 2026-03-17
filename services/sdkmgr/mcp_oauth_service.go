package sdkmgr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MCPOAuthService manages OAuth flows for connecting to external MCP servers.
// Translates sdk-manager's mcp_oauth_api.py.
type MCPOAuthService struct {
	states sync.Map // map[state]oauthStateData (TTL 15 min)
}

type oauthStateData struct {
	ServerID       string
	ConversationID string
	TenantID       string
	TokenURL       string
	ClientID       string
	CodeVerifier   string
	RedirectURI    string
	CreatedAt      time.Time
}

// wellKnownMCPServer describes a known MCP server's OAuth configuration.
type wellKnownMCPServer struct {
	Name          string
	WellKnownURL  string
	RequiresOAuth *bool // nil means yes (default), false means no
}

// wellKnownMCPServers contains known MCP servers with their OAuth discovery URLs.
var wellKnownMCPServers = map[string]wellKnownMCPServer{
	"api.githubcopilot.com": {Name: "GitHub Copilot", WellKnownURL: "https://api.githubcopilot.com/.well-known/oauth-authorization-server"},
	"mcp.notion.com":        {Name: "Notion", WellKnownURL: "https://mcp.notion.com/.well-known/oauth-authorization-server"},
	"mcp.sentry.dev":        {Name: "Sentry", WellKnownURL: "https://mcp.sentry.dev/.well-known/oauth-authorization-server"},
	"mcp.linear.app":        {Name: "Linear", WellKnownURL: "https://mcp.linear.app/.well-known/oauth-authorization-server"},
	"mcp.figma.com":         {Name: "Figma", WellKnownURL: "https://mcp.figma.com/.well-known/oauth-authorization-server"},
	"mcp.intercom.com":      {Name: "Intercom", WellKnownURL: "https://mcp.intercom.com/.well-known/oauth-authorization-server"},
	"mcp.neon.tech":         {Name: "Neon", WellKnownURL: "https://mcp.neon.tech/.well-known/oauth-authorization-server"},
	"mcp.supabase.com":      {Name: "Supabase", WellKnownURL: "https://mcp.supabase.com/.well-known/oauth-authorization-server"},
	"mcp.paypal.com":        {Name: "PayPal", WellKnownURL: "https://mcp.paypal.com/.well-known/oauth-authorization-server"},
	"mcp.squareup.com":      {Name: "Square", WellKnownURL: "https://mcp.squareup.com/.well-known/oauth-authorization-server"},
	"api.ahrefs.com":        {Name: "Ahrefs", WellKnownURL: "https://api.ahrefs.com/.well-known/oauth-authorization-server"},
	"mcp.asana.com":         {Name: "Asana", WellKnownURL: "https://mcp.asana.com/.well-known/oauth-authorization-server"},
	"mcp.atlassian.com":     {Name: "Atlassian", WellKnownURL: "https://mcp.atlassian.com/.well-known/oauth-authorization-server"},
	"mcp.wix.com":           {Name: "Wix", WellKnownURL: "https://mcp.wix.com/.well-known/oauth-authorization-server"},
	"mcp.webflow.com":       {Name: "Webflow", WellKnownURL: "https://mcp.webflow.com/.well-known/oauth-authorization-server"},
	"mcp.globalping.dev":    {Name: "Globalping", WellKnownURL: "https://mcp.globalping.dev/.well-known/oauth-authorization-server"},
	// Open servers (no OAuth required)
	"mcp.deepwiki.com":      {Name: "DeepWiki", RequiresOAuth: boolPtr(false)},
	"mcp.api.coingecko.com": {Name: "CoinGecko", RequiresOAuth: boolPtr(false)},
	"mcp.semgrep.ai":        {Name: "Semgrep", RequiresOAuth: boolPtr(false)},
	"remote.mcpservers.org": {Name: "MCP Servers Community", RequiresOAuth: boolPtr(false)},
}

func boolPtr(b bool) *bool { return &b }

// NewMCPOAuthService creates a new service instance.
func NewMCPOAuthService() *MCPOAuthService {
	svc := &MCPOAuthService{}
	// Start background cleanup goroutine for expired states.
	go svc.cleanupExpiredStates()
	return svc
}

func (s *MCPOAuthService) cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.states.Range(func(key, value interface{}) bool {
			if data, ok := value.(oauthStateData); ok {
				if now.Sub(data.CreatedAt) > 15*time.Minute {
					s.states.Delete(key)
				}
			}
			return true
		})
	}
}

// CheckOAuthRequirements checks if a server URL requires OAuth and fetches its config.
func (s *MCPOAuthService) CheckOAuthRequirements(serverURL string) map[string]interface{} {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return map[string]interface{}{"requires_oauth": false, "error": err.Error()}
	}

	hostname := parsed.Host
	if hostname == "" {
		hostname = strings.Split(parsed.Path, "/")[0]
	}

	server, known := wellKnownMCPServers[hostname]
	if !known {
		return map[string]interface{}{
			"requires_oauth": false,
			"message":        "OAuth configuration not found for this server. It may still work without authentication.",
		}
	}

	// Open server — no OAuth needed.
	if server.RequiresOAuth != nil && !*server.RequiresOAuth {
		return map[string]interface{}{
			"requires_oauth": false,
			"name":           server.Name,
			"message":        fmt.Sprintf("%s does not require OAuth authentication", server.Name),
		}
	}

	// Fetch .well-known metadata.
	if server.WellKnownURL != "" {
		metadata, err := s.fetchWellKnown(server.WellKnownURL)
		if err == nil {
			regURL, _ := metadata["registration_endpoint"].(string)
			requiresClientID := regURL == ""

			return map[string]interface{}{
				"requires_oauth": true,
				"name":           server.Name,
				"config": map[string]interface{}{
					"auth_url":                        metadata["authorization_endpoint"],
					"token_url":                       metadata["token_endpoint"],
					"registration_url":                regURL,
					"client_id":                       nil,
					"scope":                           "",
					"requires_client_id":              requiresClientID,
					"supports_dynamic_registration":   regURL != "",
				},
				"hostname": hostname,
				"metadata": metadata,
			}
		}
		logrus.WithError(err).Warn("failed to fetch .well-known config")
	}

	return map[string]interface{}{
		"requires_oauth": false,
		"message":        "OAuth configuration not found for this server. It may still work without authentication.",
	}
}

func (s *MCPOAuthService) fetchWellKnown(wellKnownURL string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(wellKnownURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("well-known returned %d", resp.StatusCode)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

// Authorize initiates the OAuth authorization flow for an MCP server.
// Returns the authorization URL and state for the frontend to open in a popup.
func (s *MCPOAuthService) Authorize(
	serverID, conversationID, tenantID, authURL, tokenURL string,
	clientID, registrationURL, scope, redirectURI *string,
) (map[string]interface{}, error) {
	resolvedClientID := ""
	if clientID != nil {
		resolvedClientID = *clientID
	}

	// Dynamic client registration (RFC 7591) if no client_id.
	if resolvedClientID == "" && registrationURL != nil && *registrationURL != "" {
		logrus.Info("no client_id provided, attempting dynamic client registration")
		redirect := "http://localhost:8000/sdkmgr/playground/oauth/callback"
		if redirectURI != nil && *redirectURI != "" {
			redirect = *redirectURI
		}
		clientInfo, err := s.registerDynamicClient(*registrationURL, redirect)
		if err != nil || clientInfo == nil {
			return nil, fmt.Errorf("client ID required but dynamic registration failed; please provide a client_id")
		}
		if cid, ok := clientInfo["client_id"].(string); ok && cid != "" {
			resolvedClientID = cid
		} else {
			return nil, fmt.Errorf("dynamic registration returned no client_id")
		}
	} else if resolvedClientID == "" {
		return nil, fmt.Errorf("client ID is required for OAuth flow")
	}

	// Generate PKCE + state.
	codeVerifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("PKCE generation failed: %w", err)
	}
	codeChallenge := GenerateCodeChallenge(codeVerifier)
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("state generation failed: %w", err)
	}

	// Store state data.
	rd := ""
	if redirectURI != nil {
		rd = *redirectURI
	}
	s.states.Store(state, oauthStateData{
		ServerID:       serverID,
		ConversationID: conversationID,
		TenantID:       tenantID,
		TokenURL:       tokenURL,
		ClientID:       resolvedClientID,
		CodeVerifier:   codeVerifier,
		RedirectURI:    rd,
		CreatedAt:      time.Now(),
	})

	// Build authorization URL.
	params := url.Values{}
	params.Set("client_id", resolvedClientID)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	if redirectURI != nil && *redirectURI != "" {
		params.Set("redirect_uri", *redirectURI)
	}
	if scope != nil && *scope != "" {
		params.Set("scope", *scope)
	}

	fullAuthURL := authURL + "?" + params.Encode()

	return map[string]interface{}{
		"authorization_url": fullAuthURL,
		"state":             state,
	}, nil
}

// registerDynamicClient performs RFC 7591 dynamic client registration.
func (s *MCPOAuthService) registerDynamicClient(registrationURL, redirectURI string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"client_name":                  "AuthSec MCP Playground",
		"redirect_uris":               []string{redirectURI},
		"grant_types":                  []string{"authorization_code", "refresh_token"},
		"response_types":              []string{"code"},
		"token_endpoint_auth_method":  "none",
		"application_type":            "web",
	}
	bodyBytes, _ := json.Marshal(body)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(registrationURL, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// HandleCallback processes the OAuth callback, exchanges the code for tokens,
// and returns HTML with postMessage for the popup window.
func (s *MCPOAuthService) HandleCallback(code, state, oauthError, errorDesc string) (string, int) {
	// OAuth error from provider.
	if oauthError != "" {
		msg := errorDesc
		if msg == "" {
			msg = oauthError
		}
		return s.callbackErrorHTML(oauthError, msg), http.StatusOK
	}

	// Validate state.
	raw, ok := s.states.LoadAndDelete(state)
	if !ok || state == "" {
		return s.callbackErrorHTML("invalid_state", "Invalid or expired OAuth state. Please try again."), http.StatusOK
	}
	data := raw.(oauthStateData)

	// Expired?
	if time.Since(data.CreatedAt) > 15*time.Minute {
		return s.callbackErrorHTML("state_expired", "OAuth session expired. Please try again."), http.StatusOK
	}

	if code == "" {
		return s.callbackErrorHTML("no_code", "No authorization code received."), http.StatusOK
	}

	// Exchange code for token.
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", data.ClientID)
	form.Set("code_verifier", data.CodeVerifier)
	if data.RedirectURI != "" {
		form.Set("redirect_uri", data.RedirectURI)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(data.TokenURL, form)
	if err != nil {
		logrus.WithError(err).Error("token exchange HTTP error")
		return s.callbackErrorHTML("token_exchange_failed", "Token exchange request failed."), http.StatusOK
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logrus.WithField("status", resp.StatusCode).WithField("body", string(bodyBytes)).Error("token exchange failed")
		return s.callbackErrorHTML("token_exchange_failed", fmt.Sprintf("Token exchange failed (HTTP %d).", resp.StatusCode)), http.StatusOK
	}

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return s.callbackErrorHTML("token_parse_failed", "Failed to parse token response."), http.StatusOK
	}

	accessToken, _ := tokenResp["access_token"].(string)
	refreshToken, _ := tokenResp["refresh_token"].(string)
	expiresIn := 3600.0
	if ei, ok := tokenResp["expires_in"].(float64); ok {
		expiresIn = ei
	}

	if accessToken == "" {
		return s.callbackErrorHTML("no_access_token", "No access token received from server."), http.StatusOK
	}

	tokenExpiry := time.Now().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339)

	html := fmt.Sprintf(`<html>
<body>
  <h2>Authorization Successful!</h2>
  <p>You can close this window.</p>
  <script>
    window.opener.postMessage({
      type: 'oauth_success',
      server_id: '%s',
      conversation_id: '%s',
      tenant_id: '%s',
      access_token: '%s',
      refresh_token: '%s',
      token_expiry: '%s'
    }, '*');
    setTimeout(() => window.close(), 1000);
  </script>
</body>
</html>`, data.ServerID, data.ConversationID, data.TenantID, accessToken, refreshToken, tokenExpiry)

	return html, http.StatusOK
}

func (s *MCPOAuthService) callbackErrorHTML(errorCode, message string) string {
	return fmt.Sprintf(`<html>
<body>
  <h2>OAuth Error</h2>
  <p>%s</p>
  <script>
    window.opener.postMessage({
      type: 'oauth_error',
      error: '%s',
      error_description: '%s'
    }, '*');
    window.close();
  </script>
</body>
</html>`, message, errorCode, message)
}

// RefreshToken refreshes an expired OAuth access token.
func (s *MCPOAuthService) RefreshToken(refreshToken, tokenURL, clientID string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(tokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	accessToken, _ := tokenResp["access_token"].(string)
	newRefreshToken, _ := tokenResp["refresh_token"].(string)
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}
	expiresIn := 3600.0
	if ei, ok := tokenResp["expires_in"].(float64); ok {
		expiresIn = ei
	}

	return map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_expiry":  time.Now().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339),
	}, nil
}
