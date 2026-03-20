// Package controllers – authmgr_controller.go
//
// Merges auth-manager into the authsec monolith.
// All handler logic is ported from:
//   - auth-manager/controllers/token_controller.go
//   - auth-manager/controllers/permission_controller.go
//   - auth-manager/controllers/group_controller.go
//   - auth-manager/controllers/validation_controller.go
//   - auth-manager/controllers/system_controller.go
//
// DB access uses authsec's config.DB (primary GORM DB) and
// config.GetTenantGORMDB(tenantID) (tenant GORM DB).
// All audit logging uses log.Printf (no external audit package needed).
package platform

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	authmgrrepo "github.com/authsec-ai/authsec/internal/authmgr/repo"
	"github.com/authsec-ai/authsec/services"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	hydra "github.com/ory/hydra-client-go/v2"
	"gorm.io/gorm"
)

// ────────────────────────────────────────────────────────────────────────────
// AuthmgrController groups all authmgr handler methods.
// ────────────────────────────────────────────────────────────────────────────

// AuthmgrController is a unified controller for auth-manager endpoints.
type AuthmgrController struct {
	rbacRepo authmgrrepo.RBACRepository
}

// NewAuthmgrController creates an AuthmgrController wired to authsec's tenant DB.
func NewAuthmgrController() *AuthmgrController {
	return &AuthmgrController{
		rbacRepo: authmgrrepo.NewRBACRepository(func(tenantID string) (*gorm.DB, error) {
			return config.GetTenantGORMDB(tenantID)
		}),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Internal helpers (package-level, prefixed authmgr*)
// ────────────────────────────────────────────────────────────────────────────

// authmgrIsAdminPath returns true when the request path is the authmgr admin sub-path.
func authmgrIsAdminPath(path string) bool {
	return strings.Contains(path, "/authmgr/admin")
}

// authmgrGetDBTypeFromPath returns "admin" or "tenant" for logging.
func authmgrGetDBTypeFromPath(path string) string {
	if authmgrIsAdminPath(path) {
		return "admin"
	}
	return "tenant"
}

// authmgrGetStringFromCtx extracts a string claim from the gin context.
func authmgrGetStringFromCtx(c *gin.Context, key string) string {
	val, exists := c.Get(key)
	if !exists {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// authmgrSafeExtractString safely pulls a string from a map[string]interface{}.
func authmgrSafeExtractString(data map[string]interface{}, key string) (string, error) {
	if data == nil {
		return "", fmt.Errorf("data map is nil")
	}
	value, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	s, ok := value.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("key '%s' is not a non-empty string", key)
	}
	return s, nil
}

// authmgrValidateRequiredFields returns an error if any required field is missing.
func authmgrValidateRequiredFields(data map[string]interface{}, required []string) error {
	var missing []string
	for _, f := range required {
		if _, ok := data[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required OIDC fields: %v", missing)
	}
	return nil
}

// authmgrIsSubset returns true when every element of subset is in superset.
func authmgrIsSubset(subset, superset []string) bool {
	set := make(map[string]bool, len(superset))
	for _, s := range superset {
		set[s] = true
	}
	for _, s := range subset {
		if !set[s] {
			return false
		}
	}
	return true
}

// authmgrConvertSlice converts []interface{} to []string.
func authmgrConvertSlice(data interface{}) []string {
	if data == nil {
		return []string{}
	}
	switch v := data.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	}
	return []string{}
}

// ── clientsvc equivalent: DB-based authz lookup ─────────────────────────────

// authmgrAuthz holds authorization data for a user.
type authmgrAuthz struct {
	Roles     []string
	Scopes    []string
	Resources []string
	Groups    []string
}

// authmgrGetAuthz loads roles/scopes/groups for a user by trying the primary DB
// first and then the tenant DB. Uses authsec's config.DB and GetTenantGORMDB.
func authmgrGetAuthz(ctx context.Context, tenantID, projectID, clientID, email string) (*authmgrAuthz, error) {
	if tenantID == "" || email == "" {
		return nil, errors.New("tenantID and email are required")
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("parse tenantID: %w", err)
	}

	// Try primary DB first
	if config.DB != nil {
		authz, err := authmgrLoadAuthzFromDB(ctx, config.DB, tid, tenantID, projectID, clientID, email)
		if err == nil && authz != nil && (len(authz.Roles) > 0 || len(authz.Scopes) > 0) {
			return authz, nil
		}
		log.Printf("[authmgr GetAuthz] primary DB miss for %s/%s, trying tenant DB: %v", tenantID, email, err)
	}

	// Fall back to tenant DB
	tenantDB, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve tenant db: %w", err)
	}
	return authmgrLoadAuthzFromDB(ctx, tenantDB, tid, tenantID, projectID, clientID, email)
}

func authmgrLoadAuthzFromDB(ctx context.Context, db *gorm.DB, tid uuid.UUID, tenantID, projectID, clientID, email string) (*authmgrAuthz, error) {
	var user sharedmodels.User
	if err := db.WithContext(ctx).Where("email = ? AND tenant_id = ?", email, tid).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Load roles via role_bindings
	var roleBindings []struct {
		RoleID   uuid.UUID
		RoleName string
	}
	err := db.WithContext(ctx).
		Table("role_bindings").
		Select("role_bindings.role_id, roles.name as role_name").
		Joins("LEFT JOIN roles ON role_bindings.role_id = roles.id").
		Where("role_bindings.tenant_id = ?", tid).
		Where("role_bindings.user_id = ?", user.ID).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > NOW()").
		Scan(&roleBindings).Error
	if err != nil {
		// Legacy fallback
		_ = db.WithContext(ctx).
			Table("user_roles").
			Select("user_roles.role_id, roles.name as role_name").
			Joins("LEFT JOIN roles ON user_roles.role_id = roles.id").
			Where("user_roles.user_id = ?", user.ID).
			Scan(&roleBindings).Error
	}

	roles := make([]string, 0)
	roleIDs := make([]uuid.UUID, 0)
	for _, rb := range roleBindings {
		if rb.RoleName != "" {
			roles = append(roles, rb.RoleName)
		}
		roleIDs = append(roleIDs, rb.RoleID)
	}

	// Load scopes via permissions
	var scopes []string
	if len(roleIDs) > 0 {
		var perms []struct {
			Resource string
			Action   string
		}
		if err := db.WithContext(ctx).Table("role_permissions").
			Select("permissions.resource, permissions.action").
			Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
			Where("role_permissions.role_id IN ?", roleIDs).
			Scan(&perms).Error; err == nil {
			for _, p := range perms {
				scopes = append(scopes, fmt.Sprintf("%s:%s", p.Resource, p.Action))
			}
		}
	}

	// Groups
	var groups []string
	var groupRecords []struct{ Name string }
	if err := db.WithContext(ctx).
		Table("user_groups").
		Select("groups.name").
		Joins("JOIN groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", user.ID).
		Scan(&groupRecords).Error; err == nil {
		for _, g := range groupRecords {
			groups = append(groups, g.Name)
		}
	}

	return &authmgrAuthz{
		Roles:     roles,
		Scopes:    scopes,
		Resources: nil,
		Groups:    groups,
	}, nil
}

// authmgrLookupClientByEmail returns clientID and projectID for the given tenant+email.
func authmgrLookupClientByEmail(ctx context.Context, tenantID, email string) (string, string, error) {
	if tenantID == "" || email == "" {
		return "", "", errors.New("tenantID and email required")
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("parse tenantID: %w", err)
	}

	// Try primary DB
	if config.DB != nil {
		var user sharedmodels.User
		if err := config.DB.WithContext(ctx).
			Select("client_id", "project_id").
			Where("tenant_id = ? AND email = ?", tid, email).
			First(&user).Error; err == nil {
			return user.ClientID.String(), user.ProjectID.String(), nil
		}
	}

	// Fall back to tenant DB
	tenantDB, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("tenant db: %w", err)
	}
	var user sharedmodels.User
	if err := tenantDB.WithContext(ctx).
		Select("client_id", "project_id").
		Where("tenant_id = ? AND email = ?", tid, email).
		First(&user).Error; err != nil {
		return "", "", fmt.Errorf("client lookup: %w", err)
	}
	return user.ClientID.String(), user.ProjectID.String(), nil
}

// ── JWT helpers ──────────────────────────────────────────────────────────────

// authmgrValidateOIDCToken introspects a Hydra OIDC token and returns claims.
func authmgrValidateOIDCToken(token string) (*sharedmodels.Introspection, error) {
	hydraAdminURL := config.AppConfig.HydraAdminURL
	if hydraAdminURL == "" {
		return nil, errors.New("hydra admin URL not configured")
	}
	if strings.HasPrefix(hydraAdminURL, "http://") {
		hydraAdminURL = hydraAdminURL[7:]
	}
	cfg := hydra.NewConfiguration()
	cfg.Host = hydraAdminURL
	cfg.Scheme = "http"
	client := hydra.NewAPIClient(cfg)

	resp, httpResp, err := client.OAuth2API.IntrospectOAuth2Token(context.Background()).Token(token).Execute()
	if err != nil {
		return nil, fmt.Errorf("introspect: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspect status: %s", httpResp.Status)
	}
	return &sharedmodels.Introspection{
		Active:   &resp.Active,
		Scope:    *resp.Scope,
		ClientID: *resp.ClientId,
		Ext:      resp.Ext,
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// System endpoints (health, profile, auth-status)
// ────────────────────────────────────────────────────────────────────────────

// HealthCheck returns the auth-manager service health.
func (ac *AuthmgrController) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "auth-manager",
		"version": "1.1.1",
	})
}

// GetProfile returns the authenticated user's profile from JWT claims.
func (ac *AuthmgrController) GetProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":  c.GetString("tenant_id"),
		"project_id": c.GetString("project_id"),
		"client_id":  c.GetString("client_id"),
		"email_id":   c.GetString("email_id"),
		"scopes":     c.MustGet("scopes"),
		"roles":      c.MustGet("roles"),
		"groups":     c.MustGet("groups"),
		"resources":  c.MustGet("resources"),
		"token_type": c.MustGet("token_type"),
	})
}

// GetAuthStatus returns auth debug info for a tenant/email combination.
func (ac *AuthmgrController) GetAuthStatus(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	email := c.Query("email")
	if tenantID == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and email are required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	clientID, projectID, err := authmgrLookupClientByEmail(ctx, tenantID, email)
	if err != nil {
		log.Printf("[authmgr GetAuthStatus] LookupClientByEmail: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found", "details": err.Error()})
		return
	}

	authzData, err := authmgrGetAuthz(ctx, tenantID, projectID, clientID, email)
	if err != nil {
		log.Printf("[authmgr GetAuthStatus] GetAuthz: %v", err)
		authzData = &authmgrAuthz{}
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":           tenantID,
		"email":               email,
		"client_id":           clientID,
		"project_id":          projectID,
		"roles":               authzData.Roles,
		"scopes":              authzData.Scopes,
		"groups":              authzData.Groups,
		"resources":           authzData.Resources,
		"mfa_enabled":         false,
		"webauthn_configured": false,
		"otp_required":        false,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Token endpoints (verifyToken, oidcToken, generateToken)
// ────────────────────────────────────────────────────────────────────────────

// VerifyToken verifies a JWT token and returns enriched claims from the DB.
func (ac *AuthmgrController) VerifyToken(c *gin.Context) {
	var req sharedmodels.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbType := authmgrGetDBTypeFromPath(c.Request.URL.Path)
	log.Printf("[authmgr VerifyToken] db=%s path=%s", dbType, c.Request.URL.Path)

	// Parse without verification to read token_type for key selection
	unverified, _, err := new(jwt.Parser).ParseUnverified(req.Token, jwt.MapClaims{})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	unverifiedClaims, ok := unverified.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	tenantID, _ := unverifiedClaims["tenant_id"].(string)
	projectID, _ := unverifiedClaims["project_id"].(string)
	clientID, _ := unverifiedClaims["client_id"].(string)
	tokenType, _ := unverifiedClaims["token_type"].(string)

	var signingSecret []byte
	if tokenType == "sdk-agent" {
		signingSecret = []byte(config.AppConfig.JWTSdkSecret)
	} else {
		signingSecret = []byte(config.AppConfig.JWTDefSecret)
	}

	token, err := jwt.Parse(req.Token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return signingSecret, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	emailID, _ := claims["email_id"].(string)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	authzData, err := authmgrGetAuthz(ctx, tenantID, projectID, clientID, emailID)
	if err != nil {
		log.Printf("[authmgr VerifyToken] authz fetch failed: %v", err)
		authzData = &authmgrAuthz{}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"tenant_id":  claims["tenant_id"],
		"project_id": claims["project_id"],
		"client_id":  claims["client_id"],
		"email_id":   claims["email_id"],
		"scopes":     authzData.Scopes,
		"roles":      authzData.Roles,
		"groups":     authzData.Groups,
		"resources":  authzData.Resources,
		"token_type": claims["token_type"],
		"issued_at":  claims["iat"],
		"expires_at": claims["exp"],
		"issuer":     claims["iss"],
	})
}

// GenerateToken issues a JWT for the given credentials.
func (ac *AuthmgrController) GenerateToken(c *gin.Context) {
	var req sharedmodels.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbType := authmgrGetDBTypeFromPath(c.Request.URL.Path)
	log.Printf("[authmgr GenerateToken] db=%s", dbType)

	var signingSecret []byte
	var tokenType string
	if req.SecretID != nil {
		signingSecret = []byte(config.AppConfig.JWTSdkSecret)
		tokenType = "sdk-agent"
	} else {
		signingSecret = []byte(config.AppConfig.JWTDefSecret)
		tokenType = "default"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id":  req.TenantID,
		"project_id": req.ProjectID,
		"client_id":  req.ClientID,
		"email_id":   req.EmailID,
		"token_type": tokenType,
		"aud":        "authsec-api",
		"iat":        time.Now().Unix(),
		"nbf":        time.Now().Unix(),
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iss":        "authsec-ai/auth-manager",
	})
	token.Header["kid"] = tokenType

	tokenString, err := token.SignedString(signingSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, sharedmodels.TokenResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   24 * 60 * 60,
	})
}

// OIDCToken exchanges an OIDC token from Hydra for a JWT.
func (ac *AuthmgrController) OIDCToken(c *gin.Context) {
	var req sharedmodels.OIDCTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tokenResp, err := services.IssueOIDCJWT(ctx, req.OidcToken)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "missing required field") {
			status = http.StatusBadRequest
		} else if strings.Contains(err.Error(), "failed to generate token") {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenResp)
}

// ────────────────────────────────────────────────────────────────────────────
// Permission check endpoints
// ────────────────────────────────────────────────────────────────────────────

// CheckPermission checks if the authenticated user has a permission for a resource+action.
func (ac *AuthmgrController) CheckPermission(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	userIDStr := authmgrGetStringFromCtx(c, "user_id")
	if tenantIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	resource := c.Query("resource")
	action := c.Query("scope")
	if resource == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource and scope are required"})
		return
	}

	hasPerm, err := ac.rbacRepo.CheckPermission(c.Request.Context(), tenantID, userID, resource, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":        userIDStr,
		"tenant_id":      tenantIDStr,
		"resource":       resource,
		"scope":          action,
		"has_permission": hasPerm,
	})
}

// CheckRole checks if the authenticated user has a specific role.
func (ac *AuthmgrController) CheckRole(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	userIDStr := authmgrGetStringFromCtx(c, "user_id")
	if tenantIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	roleName := c.Query("role")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}

	hasRole, err := ac.rbacRepo.CheckRole(c.Request.Context(), tenantID, userID, roleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "role check failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userIDStr,
		"role":     roleName,
		"has_role": hasRole,
	})
}

// CheckRoleResource checks if the user has a role scoped to a specific resource.
func (ac *AuthmgrController) CheckRoleResource(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	userIDStr := authmgrGetStringFromCtx(c, "user_id")
	if tenantIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	roleName := c.Query("role")
	resourceIDStr := c.Query("resource_id")
	scopeType := c.Query("scope_type")
	if roleName == "" || resourceIDStr == "" || scopeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role, resource_id, and scope_type are required"})
		return
	}

	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource_id"})
		return
	}

	hasRole, err := ac.rbacRepo.CheckRoleResource(c.Request.Context(), tenantID, userID, roleName, scopeType, resourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "role resource check failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userIDStr,
		"role":        roleName,
		"resource_id": resourceIDStr,
		"scope_type":  scopeType,
		"has_role":    hasRole,
	})
}

// CheckPermissionScoped checks permission with an optional scope ID.
func (ac *AuthmgrController) CheckPermissionScoped(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	userIDStr := authmgrGetStringFromCtx(c, "user_id")
	if tenantIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	resource := c.Query("resource")
	action := c.Query("scope")
	scopeIDStr := c.Query("scope_id")
	scopeType := c.Query("scope_type")
	if resource == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource and scope are required"})
		return
	}

	var scopeID *uuid.UUID
	if scopeIDStr != "" {
		parsed, err := uuid.Parse(scopeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
			return
		}
		scopeID = &parsed
	}

	hasPerm, err := ac.rbacRepo.CheckPermissionWithScope(c.Request.Context(), tenantID, userID, resource, action, scopeType, scopeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}

	resp := gin.H{
		"user_id":        userIDStr,
		"tenant_id":      tenantIDStr,
		"resource":       resource,
		"scope":          action,
		"has_permission": hasPerm,
	}
	if scopeID != nil {
		resp["scope_id"] = scopeIDStr
		resp["scope_type"] = scopeType
	}
	c.JSON(http.StatusOK, resp)
}

// CheckOAuthScopePermission checks if an OAuth scope name grants a specific permission.
func (ac *AuthmgrController) CheckOAuthScopePermission(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	if tenantIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}

	scopeName := c.Query("scope_name")
	resource := c.Query("resource")
	action := c.Query("action")
	if scopeName == "" || resource == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope_name, resource, and action are required"})
		return
	}

	hasPerm, err := ac.rbacRepo.CheckOAuthScope(c.Request.Context(), tenantID, scopeName, resource, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oauth scope check failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":      tenantIDStr,
		"scope_name":     scopeName,
		"resource":       resource,
		"action":         action,
		"has_permission": hasPerm,
	})
}

// ListUserPermissions returns all permissions for the authenticated user.
func (ac *AuthmgrController) ListUserPermissions(c *gin.Context) {
	tenantIDStr := authmgrGetStringFromCtx(c, "tenant_id")
	userIDStr := authmgrGetStringFromCtx(c, "user_id")
	if tenantIDStr == "" || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant ID"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	perms, err := ac.rbacRepo.GetUserPermissions(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve permissions"})
		return
	}

	permStrings := make([]string, len(perms))
	for i, p := range perms {
		permStrings[i] = p.Resource + ":" + p.Action
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userIDStr,
		"tenant_id":   tenantIDStr,
		"permissions": permStrings,
		"count":       len(permStrings),
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Validation endpoints
// ────────────────────────────────────────────────────────────────────────────

// ValidateToken returns token claims on successful auth middleware pass-through.
func (ac *AuthmgrController) ValidateToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Token validation successful",
		"service":   "auth-manager",
		"tenant_id": c.GetString("tenant_id"),
		"client_id": c.GetString("client_id"),
		"scopes":    c.MustGet("scopes"),
		"roles":     c.MustGet("roles"),
	})
}

// ValidateScope checks that the token has the required 'read' scope.
func (ac *AuthmgrController) ValidateScope(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":        "Scope validation successful",
		"required_scope": "read",
		"user_scopes":    c.MustGet("scopes"),
	})
}

// ValidateResource checks that the token has the required 'api' resource.
func (ac *AuthmgrController) ValidateResource(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":           "Resource validation successful",
		"required_resource": "api",
		"user_resources":    c.MustGet("resources"),
	})
}

// ValidatePermissions checks combined scope+resource requirements.
func (ac *AuthmgrController) ValidatePermissions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":           "Permission validation successful",
		"required_scope":    "write",
		"required_resource": "api",
		"user_scopes":       c.MustGet("scopes"),
		"user_resources":    c.MustGet("resources"),
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Group management endpoints
// ────────────────────────────────────────────────────────────────────────────

// CreateGroup creates one or more groups in the tenant database.
func (ac *AuthmgrController) CreateGroup(c *gin.Context) {
	var req sharedmodels.UserDefinedGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant db unavailable", "details": err.Error()})
		return
	}

	var created []sharedmodels.Group
	for _, name := range req.Groups {
		var existing sharedmodels.Group
		if db.Where("name = ? AND (tenant_id = ? OR tenant_id IS NULL)", name, tenantID).First(&existing).Error == nil {
			continue
		}
		g := sharedmodels.Group{TenantID: &tenantID, Name: name}
		if err := db.Create(&g).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group", "details": err.Error()})
			return
		}
		created = append(created, g)
	}

	log.Printf("[authmgr CreateGroup] tenant=%s created %d groups", req.TenantID, len(created))
	c.JSON(http.StatusCreated, gin.H{"message": "groups created", "groups": created, "count": len(created)})
}

// ListGroups lists all groups for a tenant.
func (ac *AuthmgrController) ListGroups(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var groups []sharedmodels.Group
	if err := db.Where("tenant_id = ? OR tenant_id IS NULL", tid).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups, "count": len(groups)})
}

// GetGroup retrieves a specific group by ID.
func (ac *AuthmgrController) GetGroup(c *gin.Context) {
	idParam := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND (tenant_id = ? OR tenant_id IS NULL)", uint(groupID), tid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

// UpdateGroup updates a group's name or description.
func (ac *AuthmgrController) UpdateGroup(c *gin.Context) {
	idParam := c.Param("id")
	var req struct {
		TenantID    string `json:"tenant_id" binding:"required"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND tenant_id = ?", uint(groupID), tid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if err := db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

// DeleteGroup removes a group from the tenant database.
func (ac *AuthmgrController) DeleteGroup(c *gin.Context) {
	idParam := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND tenant_id = ?", uint(groupID), tid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	if err := db.Delete(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

// AddUsersToGroup adds users to a group.
func (ac *AuthmgrController) AddUsersToGroup(c *gin.Context) {
	idParam := c.Param("id")
	var req struct {
		TenantID string   `json:"tenant_id" binding:"required"`
		UserIDs  []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND tenant_id = ?", uint(groupID), tid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	added := 0
	for _, userIDStr := range req.UserIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}
		var user sharedmodels.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, tid).First(&user).Error; err != nil {
			continue
		}
		if err := db.Model(&user).Association("Groups").Append(&group); err == nil {
			added++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "users added",
		"group_id":    groupID,
		"users_added": added,
		"users_total": len(req.UserIDs),
	})
}

// RemoveUsersFromGroup removes users from a group.
func (ac *AuthmgrController) RemoveUsersFromGroup(c *gin.Context) {
	idParam := c.Param("id")
	var req struct {
		TenantID string   `json:"tenant_id" binding:"required"`
		UserIDs  []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(req.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND tenant_id = ?", uint(groupID), tid).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	removed := 0
	for _, userIDStr := range req.UserIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}
		var user sharedmodels.User
		if err := db.Where("id = ? AND tenant_id = ?", userID, tid).First(&user).Error; err != nil {
			continue
		}
		if err := db.Model(&user).Association("Groups").Delete(&group); err == nil {
			removed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "users removed",
		"group_id":      groupID,
		"users_removed": removed,
		"users_total":   len(req.UserIDs),
	})
}

// ListGroupUsers returns users belonging to a group.
func (ac *AuthmgrController) ListGroupUsers(c *gin.Context) {
	idParam := c.Param("id")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	groupID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	db, err := config.GetTenantGORMDB(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var group sharedmodels.Group
	if err := db.Where("id = ? AND tenant_id = ?", uint(groupID), tid).Preload("Users").First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	var users []sharedmodels.User
	if err := db.Model(&group).Association("Users").Find(&users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type UserInfo struct {
		ID     uuid.UUID `json:"id"`
		Email  string    `json:"email"`
		Name   string    `json:"name"`
		Active bool      `json:"active"`
	}
	infos := make([]UserInfo, len(users))
	for i, u := range users {
		infos[i] = UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Active: u.Active}
	}

	c.JSON(http.StatusOK, gin.H{
		"group_id":   groupID,
		"group_name": group.Name,
		"users":      infos,
		"count":      len(infos),
	})
}
