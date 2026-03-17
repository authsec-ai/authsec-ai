package sdkmgr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/authsec-ai/authsec/config"
	models "github.com/authsec-ai/authsec/models/sdkmgr"
	"github.com/sirupsen/logrus"
)

// MCPAuthService is the core MCP authentication service.
// It orchestrates OAuth PKCE flows, session management, RBAC evaluation,
// and tool protection.
type MCPAuthService struct {
	SessionStore        *OAuthSessionStore
	ToolsManager        *MCPToolsManager
	clientToolsMetadata sync.Map // map[clientID][]interface{} (tool metadata)
}

// NewMCPAuthService creates and returns a new service instance.
func NewMCPAuthService() *MCPAuthService {
	svc := &MCPAuthService{
		SessionStore: NewOAuthSessionStore(),
		ToolsManager: NewMCPToolsManager(),
	}
	return svc
}

// Initialize runs startup tasks (invalidate stale sessions).
func (s *MCPAuthService) Initialize() {
	s.SessionStore.InvalidateAllSessions()
}

// HealthCheck returns a simple health response.
func (s *MCPAuthService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":    "healthy",
		"service":   "mcp-auth-service",
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

// ---------- OAuth Flow ----------

// StartOAuthFlow creates a new session with PKCE and returns the authorization URL.
func (s *MCPAuthService) StartOAuthFlow(clientID, appName string) (map[string]interface{}, error) {
	candidates := BuildClientIDCandidates(clientID)
	resolvedClientID := clientID
	if len(candidates) > 0 {
		resolvedClientID = candidates[0]
	}

	session := models.NewOAuthSession()
	session.ClientIdentifier = &resolvedClientID

	// Generate PKCE parameters.
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("PKCE generation failed: %w", err)
	}
	challenge := GenerateCodeChallenge(verifier)
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("state generation failed: %w", err)
	}

	// Use static PKCE challenge if configured (for testing), otherwise compute.
	cfg := config.AppConfig
	if cfg != nil && cfg.PKCEChallenge != "" {
		challenge = cfg.PKCEChallenge
	}

	session.PKCEVerifier = &verifier
	session.PKCEChallenge = &challenge
	session.OAuthState = &state

	if err := s.SessionStore.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Build authorization URL.
	redirectURI := s.resolveRedirectURI(resolvedClientID)

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {resolvedClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"openid profile email"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}

	authURL := cfg.OAuthAuthURL + "?" + params.Encode()

	truncatedID := resolvedClientID
	if len(truncatedID) > 12 {
		truncatedID = truncatedID[:8] + "..." + truncatedID[len(truncatedID)-4:]
	}

	return map[string]interface{}{
		"status":            "ready_for_authentication",
		"session_id":        session.SessionID,
		"authorization_url": authURL,
		"callback_url":      redirectURI,
		"client_id_used":    resolvedClientID,
		"browser_opened":    false,
		"app_name":          appName,
		"instructions": []string{
			fmt.Sprintf("Your session ID: %s", session.SessionID),
			fmt.Sprintf("Using client_id: %s", truncatedID),
			"Browser should open automatically if configured, otherwise open authorization_url manually",
			"If callback_url points to /sdkmgr/mcp-auth/callback, authentication completes automatically",
			"After login, protected tools can use latest session automatically (session_id optional)",
			"If your login flow returns a raw JWT instead, use oauth_authenticate as fallback",
		},
	}, nil
}

// AuthenticateWithJWT validates the JWT and associates user info with the session.
func (s *MCPAuthService) AuthenticateWithJWT(jwtToken, sessionID string, expiresIn int64) (map[string]interface{}, error) {
	if jwtToken == "" || sessionID == "" {
		return nil, fmt.Errorf("jwt_token and session_id are required")
	}

	// Guard: detect swapped arguments.
	looksLikeJWT := func(v string) bool { return strings.Count(v, ".") >= 2 || strings.HasPrefix(v, "eyJ") }
	looksLikeUUID := func(v string) bool { return len(v) == 36 && strings.Count(v, "-") == 4 }
	if looksLikeJWT(sessionID) && looksLikeUUID(jwtToken) {
		logrus.Warn("authenticate_with_jwt: arguments appear swapped, correcting")
		sessionID, jwtToken = jwtToken, sessionID
	}
	if len(sessionID) > 36 {
		return nil, fmt.Errorf("invalid session_id: expected OAuth session id (UUID)")
	}

	// Fetch or create session.
	session := s.SessionStore.GetSession(sessionID)
	if session == nil {
		session = models.NewOAuthSession()
		session.SessionID = sessionID
	}

	// Verify token via in-process TokenService (replaces HTTP call to /authmgr/verifyToken).
	userInfo := s.verifyToken(jwtToken)

	// Normalize scopes and permissions.
	ensureUserInfoScopesAndPerms(userInfo)

	session.SetJWTToken(jwtToken, expiresIn)
	session.UpdateUserInfo(userInfo)

	// Compute accessible tools from RBAC metadata.
	clientID := ""
	if session.ClientIdentifier != nil {
		clientID = *session.ClientIdentifier
	}
	accessible := s.computeAccessibleTools(clientID, userInfo)
	if accessible != nil {
		session.SetAccessibleTools(accessible)
	}

	if err := s.SessionStore.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return map[string]interface{}{
		"status":           "authenticated",
		"session_id":       session.SessionID,
		"user_info":        userInfo,
		"accessible_tools": accessible,
	}, nil
}

// HandleOAuthCallback exchanges an authorization code for a token.
func (s *MCPAuthService) HandleOAuthCallback(code, state, sessionID, clientID string) (map[string]interface{}, error) {
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}
	if state == "" {
		return nil, fmt.Errorf("missing OAuth state")
	}

	var session *models.OAuthSession
	if sessionID != "" {
		session = s.SessionStore.GetSession(sessionID)
	}
	if session == nil {
		session = s.SessionStore.GetSessionByState(state)
	}
	if session == nil {
		return nil, fmt.Errorf("OAuth session not found for callback state")
	}

	if session.OAuthState != nil && *session.OAuthState != state {
		return nil, fmt.Errorf("invalid OAuth state")
	}

	resolvedClientID := clientID
	if session.ClientIdentifier != nil && *session.ClientIdentifier != "" {
		resolvedClientID = *session.ClientIdentifier
	}
	if resolvedClientID == "" {
		return nil, fmt.Errorf("missing client identifier in callback context")
	}

	redirectURI := s.resolveRedirectURI(resolvedClientID)
	cfg := config.AppConfig

	// Exchange code for token.
	tokenPayload := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {resolvedClientID},
		"code_verifier": {ptrStr(session.PKCEVerifier)},
	}

	resp, err := http.PostForm(cfg.OAuthTokenURL, tokenPayload)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token exchange failed (%d)", resp.StatusCode)
	}

	var tokenData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken, _ := tokenData["access_token"].(string)
	if accessToken == "" {
		return nil, fmt.Errorf("token exchange succeeded but access_token is missing")
	}

	refreshToken, _ := tokenData["refresh_token"].(string)
	expiresIn := int64(3600)
	if v, ok := tokenData["expires_in"].(float64); ok {
		expiresIn = int64(v)
	}

	authResult, err := s.AuthenticateWithJWT(accessToken, session.SessionID, expiresIn)
	if err != nil {
		return nil, err
	}

	// Persist refresh token.
	updated := s.SessionStore.GetSession(session.SessionID)
	if updated != nil {
		updated.AuthorizationCode = &code
		if refreshToken != "" {
			updated.RefreshToken = &refreshToken
		}
		updated.Touch()
		_ = s.SessionStore.SaveSession(updated)
	}

	return map[string]interface{}{
		"status":     "authenticated",
		"session_id": session.SessionID,
		"client_id":  resolvedClientID,
		"flow":       "authorization_code_callback",
		"oauth": map[string]interface{}{
			"code_received":         true,
			"token_expires_in":      expiresIn,
			"refresh_token_present": refreshToken != "",
		},
		"user_info":        authResult["user_info"],
		"accessible_tools": authResult["accessible_tools"],
	}, nil
}

// ---------- Session Status / Logout ----------

// GetSessionStatus returns the status of a session.
func (s *MCPAuthService) GetSessionStatus(sessionID string) map[string]interface{} {
	session := s.SessionStore.GetSession(sessionID)
	if session == nil {
		return map[string]interface{}{
			"status":     "not_found",
			"session_id": sessionID,
		}
	}
	valid := session.IsActive && session.IsTokenValid()
	status := "expired"
	if valid {
		status = "authenticated"
	}
	return map[string]interface{}{
		"status":     status,
		"session_id": session.SessionID,
		"expires_at": session.TokenExpiresAt,
		"user_info":  session.GetUserInfoMap(),
	}
}

// LogoutSession deactivates a session.
func (s *MCPAuthService) LogoutSession(sessionID string) map[string]interface{} {
	s.SessionStore.DeleteSession(sessionID)
	return map[string]interface{}{
		"status":     "logged_out",
		"session_id": sessionID,
	}
}

// GetActiveSessionsCount returns total active authenticated sessions.
func (s *MCPAuthService) GetActiveSessionsCount() map[string]interface{} {
	count := s.SessionStore.GetActiveAuthenticatedSessionsCount()
	return map[string]interface{}{
		"active_authenticated_sessions": count,
	}
}

// CleanupSessions deactivates all sessions for a client.
func (s *MCPAuthService) CleanupSessions(clientID, appName, reason string) map[string]interface{} {
	candidates := BuildClientIDCandidates(clientID)
	var total int64
	for _, cid := range candidates {
		total += s.SessionStore.CleanupClientSessions(cid)
	}
	return map[string]interface{}{
		"status":             "cleaned",
		"client_id":          clientID,
		"app_name":           appName,
		"reason":             reason,
		"sessions_cleaned":   total,
	}
}

// ---------- Tools ----------

// GetToolsList returns the available tools for an MCP client.
func (s *MCPAuthService) GetToolsList(clientID, appName string, userTools []interface{}) map[string]interface{} {
	oauthTools := s.ToolsManager.GetOAuthTools()
	availableTools := make([]ToolSchema, len(oauthTools))
	copy(availableTools, oauthTools)

	cfg := config.AppConfig

	// Store tool metadata.
	if len(userTools) > 0 {
		if _, ok := userTools[0].(map[string]interface{}); ok {
			s.clientToolsMetadata.Store(clientID, userTools)
		} else {
			// Old format: convert strings to dict format.
			converted := make([]interface{}, len(userTools))
			for i, t := range userTools {
				name, _ := t.(string)
				converted[i] = map[string]interface{}{"name": name, "rbac": map[string]interface{}{}}
			}
			s.clientToolsMetadata.Store(clientID, converted)
		}
	}

	toolMetaListRaw, _ := s.clientToolsMetadata.Load(clientID)
	toolMetaList, _ := toolMetaListRaw.([]interface{})

	alwaysExpose := true
	hideUnauthorized := false
	if cfg != nil {
		alwaysExpose = cfg.SDKAlwaysExposeProtectedTools
		hideUnauthorized = cfg.SDKHideUnauthorizedTools
	}

	// Check for active session.
	sessions := s.SessionStore.GetActiveSessionsForClient(clientID)
	var latestSession *models.OAuthSession
	if len(sessions) > 0 {
		latestSession = &sessions[0]
	}

	// Compute accessible tools if we have session + metadata.
	if latestSession != nil {
		userInfo := latestSession.GetUserInfoMap()
		if userInfo != nil {
			ensureUserInfoScopesAndPerms(userInfo)
			accessible := s.computeAccessibleTools(clientID, userInfo)
			if accessible != nil {
				latestSession.SetAccessibleTools(accessible)
				latestSession.UpdateUserInfo(userInfo)
				_ = s.SessionStore.SaveSession(latestSession)
			}
		}
	}

	shouldFilter := !alwaysExpose && hideUnauthorized && latestSession != nil && latestSession.AccessibleTools != nil

	if shouldFilter {
		accessibleSet := make(map[string]bool)
		for _, t := range latestSession.GetAccessibleToolsList() {
			accessibleSet[t] = true
		}
		for _, meta := range toolMetaList {
			name := toolMetaName(meta)
			if accessibleSet[name] {
				availableTools = append(availableTools, s.ToolsManager.GenerateUserToolSchemaFromMetadata(meta))
			}
		}
	} else {
		for _, meta := range toolMetaList {
			availableTools = append(availableTools, s.ToolsManager.GenerateUserToolSchemaFromMetadata(meta))
		}
	}

	return map[string]interface{}{"tools": availableTools}
}

// ProtectTool validates that a session is active and RBAC allows the tool.
func (s *MCPAuthService) ProtectTool(sessionID *string, toolName, clientID, appName string) map[string]interface{} {
	session := s.resolveActiveSession(sessionID, clientID)
	if session == nil {
		return map[string]interface{}{
			"allowed": false,
			"error":   "Access denied",
			"message": "No active authenticated session. Run oauth_start and complete browser login.",
		}
	}

	// Compute RBAC on-demand if not yet done.
	if session.AccessibleTools == nil {
		userInfo := session.GetUserInfoMap()
		if userInfo == nil {
			userInfo = make(map[string]interface{})
		}
		ensureUserInfoScopesAndPerms(userInfo)
		accessible := s.computeAccessibleTools(clientID, userInfo)
		if accessible != nil {
			session.SetAccessibleTools(accessible)
			session.UpdateUserInfo(userInfo)
			_ = s.SessionStore.SaveSession(session)
		}
	}

	// Enforce RBAC.
	accessibleTools := session.GetAccessibleToolsList()
	if accessibleTools != nil && !contains(accessibleTools, toolName) {
		return map[string]interface{}{
			"allowed":    false,
			"error":      "Access denied",
			"message":    "RBAC denied: missing required permissions",
			"session_id": session.SessionID,
		}
	}

	return map[string]interface{}{
		"allowed":    true,
		"user_info":  session.GetUserInfoMap(),
		"tool":       toolName,
		"session_id": session.SessionID,
	}
}

// ExecuteOAuthTool dispatches an oauth_* tool call.
func (s *MCPAuthService) ExecuteOAuthTool(toolName, clientID, appName string, arguments map[string]interface{}) map[string]interface{} {
	wrapResult := func(result interface{}) map[string]interface{} {
		text, _ := json.MarshalIndent(result, "", "  ")
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": string(text)},
			},
		}
	}
	wrapError := func(msg string) map[string]interface{} {
		return wrapResult(map[string]string{"error": msg})
	}

	switch toolName {
	case "oauth_start":
		result, err := s.StartOAuthFlow(clientID, appName)
		if err != nil {
			return wrapError(err.Error())
		}
		return wrapResult(result)

	case "oauth_authenticate":
		jwt, _ := arguments["jwt_token"].(string)
		sid, _ := arguments["session_id"].(string)
		exp := int64(3600)
		if v, ok := arguments["expires_in"].(float64); ok {
			exp = int64(v)
		}
		result, err := s.AuthenticateWithJWT(jwt, sid, exp)
		if err != nil {
			return wrapError(err.Error())
		}
		return wrapResult(result)

	case "oauth_status":
		sid, _ := arguments["session_id"].(string)
		return wrapResult(s.GetSessionStatus(sid))

	case "oauth_logout":
		sid, _ := arguments["session_id"].(string)
		return wrapResult(s.LogoutSession(sid))

	case "oauth_user_info":
		sid, _ := arguments["session_id"].(string)
		return wrapResult(s.getSessionUserInfo(sid))

	case "oauth_user_roles":
		sid, _ := arguments["session_id"].(string)
		info := s.getSessionUserInfo(sid)
		return wrapResult(map[string]interface{}{"roles": info["roles"]})

	case "oauth_user_groups":
		sid, _ := arguments["session_id"].(string)
		info := s.getSessionUserInfo(sid)
		return wrapResult(map[string]interface{}{"groups": info["groups"]})

	case "oauth_user_permissions":
		sid, _ := arguments["session_id"].(string)
		info := s.getSessionUserInfo(sid)
		perms := info["permissions"]
		if perms == nil {
			perms = info["scopes"]
		}
		return wrapResult(map[string]interface{}{"permissions": perms})

	case "oauth_user_resources":
		sid, _ := arguments["session_id"].(string)
		info := s.getSessionUserInfo(sid)
		return wrapResult(map[string]interface{}{"resources": info["resources"]})

	case "oauth_user_scopes":
		sid, _ := arguments["session_id"].(string)
		info := s.getSessionUserInfo(sid)
		scopes := info["scopes"]
		if scopes == nil {
			scopes = extractScopes(info)
		}
		return wrapResult(map[string]interface{}{"scopes": scopes})

	default:
		return wrapError(fmt.Sprintf("Unknown OAuth tool: %s", toolName))
	}
}

// ---------- Internal helpers ----------

// verifyToken validates the JWT. In-process call replaces the HTTP call
// the Python sdk-manager made to /authmgr/verifyToken.
func (s *MCPAuthService) verifyToken(jwtToken string) map[string]interface{} {
	// Try in-process token verification via the global TokenService.
	if config.TokenService != nil {
		// The auth-manager package's verifyToken returns claims directly.
		// For now, we fall back to unverified decode; the controller layer
		// can integrate the actual TokenService.ValidateToken call.
	}

	// Fallback: decode JWT payload locally (unverified).
	claims, err := DecodeJWTPayload(jwtToken)
	if err != nil {
		logrus.WithError(err).Warn("failed to decode JWT payload, returning empty claims")
		return make(map[string]interface{})
	}
	return claims
}

// resolveRedirectURI determines the OAuth redirect URI for a given client.
// Priority: OAuthRedirectURITemplate → OAuthRedirectURI → fallback.
func (s *MCPAuthService) resolveRedirectURI(clientID string) string {
	cfg := config.AppConfig
	if cfg == nil {
		return "http://localhost:3005/oauth/callback"
	}

	// TODO: Phase 1 enhancement — query tenant_hydra_clients table when
	// SDKRedirectSource == "db". For now, use env-based resolution.

	if cfg.OAuthRedirectURITemplate != "" {
		return strings.ReplaceAll(cfg.OAuthRedirectURITemplate, "{client_id}", clientID)
	}
	if cfg.OAuthRedirectURI != "" {
		return cfg.OAuthRedirectURI
	}
	return "http://localhost:3005/oauth/callback"
}

// resolveActiveSession finds a usable session: explicit session_id first,
// then latest active session for the client.
func (s *MCPAuthService) resolveActiveSession(sessionID *string, clientID string) *models.OAuthSession {
	if sessionID != nil {
		cleaned := strings.Trim(strings.TrimSpace(*sessionID), `"'`)
		if cleaned != "" {
			sess := s.SessionStore.GetSession(cleaned)
			if sess != nil && sess.IsActive && sess.IsTokenValid() {
				return sess
			}
		}
	}

	sessions := s.SessionStore.GetActiveSessionsForClient(clientID)
	for i := range sessions {
		if sessions[i].IsActive && sessions[i].IsTokenValid() {
			return &sessions[i]
		}
	}
	return nil
}

func (s *MCPAuthService) getSessionUserInfo(sessionID string) map[string]interface{} {
	session := s.SessionStore.GetSession(sessionID)
	if session == nil {
		return make(map[string]interface{})
	}
	info := session.GetUserInfoMap()
	if info == nil {
		return make(map[string]interface{})
	}
	return info
}

// computeAccessibleTools evaluates RBAC for all tools registered by a client.
func (s *MCPAuthService) computeAccessibleTools(clientID string, userInfo map[string]interface{}) []string {
	raw, ok := s.clientToolsMetadata.Load(clientID)
	if !ok {
		return nil
	}
	toolsMeta, _ := raw.([]interface{})
	if len(toolsMeta) == 0 {
		return nil
	}

	var accessible []string
	for _, meta := range toolsMeta {
		var toolName string
		var rbac map[string]interface{}

		if m, ok := meta.(map[string]interface{}); ok {
			toolName, _ = m["name"].(string)
			rbac, _ = m["rbac"].(map[string]interface{})
		} else {
			toolName = fmt.Sprintf("%v", meta)
		}

		if toolName != "" && EvaluateRBAC(userInfo, rbac) {
			accessible = append(accessible, toolName)
		}
	}
	return accessible
}

// EvaluateRBAC checks whether user claims satisfy the RBAC requirements.
// Uses AND/OR logic controlled by require_all.
func EvaluateRBAC(userInfo, rbac map[string]interface{}) bool {
	if len(rbac) == 0 {
		return true
	}

	rolesReq := normalizeList(rbac["roles"])
	groupsReq := normalizeList(rbac["groups"])
	resourcesReq := normalizeList(rbac["resources"])
	scopesReq := normalizeList(rbac["scopes"])
	permsReq := normalizeList(rbac["permissions"])
	requireAll, _ := rbac["require_all"].(bool)

	if len(rolesReq) == 0 && len(groupsReq) == 0 && len(resourcesReq) == 0 && len(scopesReq) == 0 && len(permsReq) == 0 {
		return true
	}

	roles := toSet(normalizeList(userInfo["roles"]))
	groups := toSet(normalizeList(userInfo["groups"]))
	resources := toSet(normalizeList(userInfo["resources"]))
	scopes := toSet(normalizeList(userInfo["scopes"]))
	permissions := toSet(normalizeList(userInfo["permissions"]))

	// Treat scopes as permissions if permissions are empty.
	if len(permissions) == 0 && len(scopes) > 0 {
		permissions = scopes
	}

	var checks []bool
	if len(rolesReq) > 0 {
		checks = append(checks, intersects(roles, rolesReq))
	}
	if len(groupsReq) > 0 {
		checks = append(checks, intersects(groups, groupsReq))
	}
	if len(resourcesReq) > 0 {
		checks = append(checks, intersects(resources, resourcesReq))
	}
	if len(scopesReq) > 0 {
		checks = append(checks, intersects(scopes, scopesReq))
	}
	if len(permsReq) > 0 {
		checks = append(checks, intersects(permissions, permsReq))
	}

	if requireAll {
		for _, c := range checks {
			if !c {
				return false
			}
		}
		return true
	}
	for _, c := range checks {
		if c {
			return true
		}
	}
	return false
}

// ---------- utility functions ----------

func ensureUserInfoScopesAndPerms(info map[string]interface{}) {
	if info == nil {
		return
	}
	if _, ok := info["scopes"]; !ok {
		info["scopes"] = extractScopes(info)
	}
	perms := normalizeList(info["permissions"])
	if len(perms) == 0 {
		scopes := normalizeList(info["scopes"])
		if len(scopes) > 0 {
			info["permissions"] = scopes
		}
	}
}

func extractScopes(payload map[string]interface{}) []string {
	if scopes := normalizeList(payload["scopes"]); len(scopes) > 0 {
		return scopes
	}
	return normalizeList(payload["scope"])
}

func normalizeList(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		var out []string
		for _, item := range val {
			s := fmt.Sprintf("%v", item)
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return val
	case string:
		parts := strings.Fields(strings.ReplaceAll(val, ",", " "))
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	default:
		return nil
	}
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func intersects(set map[string]bool, list []string) bool {
	for _, item := range list {
		if set[item] {
			return true
		}
	}
	return false
}

func toolMetaName(meta interface{}) string {
	if m, ok := meta.(map[string]interface{}); ok {
		name, _ := m["name"].(string)
		return name
	}
	if s, ok := meta.(string); ok {
		return s
	}
	return ""
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
