package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/authsec-ai/auth-manager/pkg/authz"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthConfig holds configuration for authentication
type AuthConfig struct {
	JWTSecret         string
	JWTDefaultSecret  string
	ExpectedIssuer    string
	ExpectedAudience  string
	RequireServerAuth bool
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	jwtSdkSecret := os.Getenv("JWT_SDK_SECRET")
	if jwtSdkSecret == "" {
		panic("CRITICAL: JWT_SDK_SECRET environment variable is not set. Cannot authenticate requests.")
	}
	jwtDefSecret := os.Getenv("JWT_DEF_SECRET")
	if jwtDefSecret == "" {
		panic("CRITICAL: JWT_DEF_SECRET environment variable is not set. Cannot authenticate requests.")
	}

	return &AuthConfig{
		JWTSecret:         jwtSdkSecret,
		JWTDefaultSecret:  jwtDefSecret,
		ExpectedIssuer:    getEnvOrDefault("AUTH_EXPECT_ISS", "authsec-ai/auth-manager"),
		ExpectedAudience:  getEnvOrDefault("AUTH_EXPECT_AUD", "authsec-api"),
		RequireServerAuth: getEnvOrDefault("REQUIRE_SERVER_AUTH", "true") == "true",
	}
}

// AuthMiddleware creates a Gin middleware for JWT authentication and authorization using auth-manager
func AuthMiddleware() gin.HandlerFunc {
	return AuthMiddlewareWithConfig(DefaultAuthConfig())
}

// AuthMiddlewareWithConfig creates a Gin middleware with custom configuration using auth-manager
// AuthMiddlewareWithConfig creates a Gin middleware with custom configuration
func AuthMiddlewareWithConfig(cfg *AuthConfig) gin.HandlerFunc {
	// Return a wrapped middleware that performs authentication locally (supporting multiple secrets)
	return func(c *gin.Context) {
		// Skip auth for CORS preflight requests so the global CORS middleware can respond
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		tokenString, err := extractBearerToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		claims, err := validateJWTToken(tokenString, cfg)
		if err != nil {
			fmt.Printf("WARN: JWT validation failed: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract user info and set context
		// Map claims to UserInfo manually or via helper
		// Note: helper extractUserInfo pulls from context "claims", so we must set that first
		c.Set("claims", claims)

		info, err := extractUserInfo(c)
		if err != nil {
			fmt.Printf("WARN: Failed to extract user info from token: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Set all context values expected by downstream consumers
		setContextValues(c, claims, info)

		// Check admin access for admin paths (server auth)
		if cfg.RequireServerAuth {
			path := c.FullPath()
			if path == "" {
				path = c.Request.URL.Path
			}

			// Check admin access for admin paths
			if strings.HasPrefix(path, "/admin/") {
				if !stringSliceContains(info.Roles, "admin") {
					c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
					c.Abort()
					return
				}
			}
		}

		// Backfill missing identifiers (tenant_id, user_id, etc.) for legacy controllers
		ensureTenantContext(c)
		ensureUserContextIdentifiers(c)

		c.Next()
	}
}

// RequireRole creates middleware that requires a specific role using auth-manager
func RequireRole(role string) gin.HandlerFunc {
	// For now, fallback to custom implementation until auth-manager supports role-based authorization
	return func(c *gin.Context) {
		rolesValue, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found in token"})
			c.Abort()
			return
		}

		roles, err := extractStringSlice(rolesValue)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid roles format in token"})
			c.Abort()
			return
		}

		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":         "Insufficient role",
			"required_role": role,
		})
		c.Abort()
	}
}

// RequireScope creates middleware that requires a specific scope using auth-manager
func RequireScope(scope string) gin.HandlerFunc {
	// For consistency with existing API, maintain custom implementation
	// but align with auth-manager patterns for future migration
	return func(c *gin.Context) {
		scopesValue, exists := c.Get("scopes")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No scopes found in token"})
			c.Abort()
			return
		}

		scopes, err := extractStringSlice(scopesValue)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid scopes format in token"})
			c.Abort()
			return
		}

		for _, s := range scopes {
			if s == scope {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":          "Insufficient scope",
			"required_scope": scope,
		})
		c.Abort()
	}
}

// RequireResource creates middleware that requires access to a specific resource using auth-manager
func RequireResource(resource string) gin.HandlerFunc {
	// Use auth-manager's resource-based approach
	return authz.Require(resource, "access")
}

// Require creates middleware that requires a specific resource and action using auth-manager
// Auth-manager v1.0.1+ automatically fetches permissions from DB for minimal tokens
func Require(resource, action string) gin.HandlerFunc {
	return authz.Require(resource, action)
}

// RequireAll creates middleware that requires all specified needs using auth-manager
func RequireAll(needs ...authz.Need) gin.HandlerFunc {
	return authz.RequireAll(needs...)
}

// RequireAny creates middleware that requires any of the specified needs using auth-manager
func RequireAny(needs ...authz.Need) gin.HandlerFunc {
	return authz.RequireAny(needs...)
}

// RequireHTTPMethod creates middleware that restricts HTTP methods using auth-manager
func RequireHTTPMethod(methods ...string) gin.HandlerFunc {
	// This is more of a general middleware, keep simple implementation
	return func(c *gin.Context) {
		if len(methods) == 0 {
			c.Next()
			return
		}

		currentMethod := c.Request.Method
		for _, method := range methods {
			if strings.EqualFold(method, currentMethod) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":           "HTTP method not allowed",
			"current_method":  currentMethod,
			"allowed_methods": methods,
		})
		c.Abort()
	}
}

// ValidatePermissions creates middleware for permission validation using auth-manager
func ValidatePermissions(scope, resource string, methods ...string) gin.HandlerFunc {
	// Build auth-manager needs
	var needs []authz.Need

	if scope != "" {
		needs = append(needs, authz.Need{
			Resource: "scope",
			Action:   scope,
		})
	}

	if resource != "" {
		needs = append(needs, authz.Need{
			Resource: resource,
			Action:   "access",
		})
	}

	// If we have auth-manager needs, use them
	if len(needs) > 0 {
		authzMiddleware := authz.RequireAll(needs...)

		// Wrap with HTTP method validation if needed
		return func(c *gin.Context) {
			// Check HTTP methods first
			if len(methods) > 0 {
				currentMethod := c.Request.Method
				methodAllowed := false
				for _, method := range methods {
					if strings.EqualFold(method, currentMethod) {
						methodAllowed = true
						break
					}
				}
				if !methodAllowed {
					c.JSON(http.StatusMethodNotAllowed, gin.H{
						"error":           "HTTP method not allowed",
						"current_method":  currentMethod,
						"allowed_methods": methods,
					})
					c.Abort()
					return
				}
			}

			// Then check authorization
			authzMiddleware(c)
		}
	}

	// Fallback to HTTP method validation only
	return RequireHTTPMethod(methods...)
}

// extractBearerToken extracts the Bearer token from Authorization header
func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// validateJWTToken validates the JWT token and returns claims
func validateJWTToken(tokenString string, cfg *AuthConfig) (jwt.MapClaims, error) {
	// Try parsing with different secrets - prioritize environment variables
	// Use configured JWT secrets with minimal fallback
	secrets := []string{
		cfg.JWTSecret,        // Primary JWT secret (JWT_SDK_SECRET from env)
		cfg.JWTDefaultSecret, // Secondary JWT secret (JWT_DEF_SECRET from env)
	}

	// Add only essential fallback for cross-service compatibility
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" && jwtSecret != cfg.JWTSecret && jwtSecret != cfg.JWTDefaultSecret {
		secrets = append(secrets, jwtSecret)
	}

	var lastErr error
	for _, secret := range secrets {
		if secret == "" {
			continue // Skip empty secrets
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// Validate issuer and audience
				if err := validateClaims(claims, cfg); err != nil {
					return nil, err
				}
				return claims, nil
			}
		}
		lastErr = err
	}

	return nil, lastErr
}

// validateClaims validates issuer, audience, and timing claims
// Relaxed validation to support tokens from multiple services (auth-manager, user-flow, etc.)
func validateClaims(claims jwt.MapClaims, cfg *AuthConfig) error {
	// Validate issuer - allow multiple trusted issuers for cross-service compatibility
	validIssuers := []string{
		cfg.ExpectedIssuer,
		"authsec-ai/auth-manager", // Always trust auth-manager
		"authsec-ai/user-flow",    // Trust user-flow self-issued tokens
	}

	if iss, ok := claims["iss"].(string); ok {
		isValid := false
		for _, validIss := range validIssuers {
			if iss == validIss {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid issuer: %s", iss)
		}
	}
	// If no issuer claim, reject the token
	if _, ok := claims["iss"]; !ok {
		return fmt.Errorf("token missing required issuer (iss) claim")
	}

	// Validate audience (always enforced; skip only if no audience is configured)
	if audClaim, exists := claims["aud"]; exists {
		if cfg.ExpectedAudience != "" && !validateAudience(audClaim, cfg.ExpectedAudience) {
			return fmt.Errorf("token audience mismatch")
		}
	}

	// Validate timing - strict on expiration
	now := time.Now().Unix()
	if exp, ok := claims["exp"].(float64); ok && now > int64(exp) {
		return fmt.Errorf("token expired")
	}
	if nbf, ok := claims["nbf"].(float64); ok && now < int64(nbf) {
		return fmt.Errorf("token not yet valid")
	}

	return nil
}

// validateAudience validates the audience claim
func validateAudience(aud interface{}, expected string) bool {
	switch v := aud.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// UserInfo holds extracted user information from JWT claims
type UserInfo struct {
	TenantID  string
	ProjectID string
	ClientID  string
	UserID    string
	Email     string
	Roles     []string
	Groups    []string
	Scopes    []string
	Resources []string
}

// extractUserInfo extracts user information from auth-manager claims
func extractUserInfo(c *gin.Context) (*UserInfo, error) {
	info := &UserInfo{}

	// Get claims from auth-manager (exposed in Gin context)
	claims, exists := c.Get("claims")
	if !exists {
		return nil, fmt.Errorf("claims not found in context")
	}

	claimsMap, ok := claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Extract basic user info
	if v, ok := claimsMap["tenant_id"].(string); ok {
		info.TenantID = v
	}
	if v, ok := claimsMap["project_id"].(string); ok {
		info.ProjectID = v
	}
	if v, ok := claimsMap["client_id"].(string); ok {
		info.ClientID = v
	}
	// user_id fallback to sub (trimmed tokens use sub)
	if v, ok := claimsMap["user_id"].(string); ok {
		info.UserID = v
	} else if v, ok := claimsMap["sub"].(string); ok {
		info.UserID = v
	}
	// email_id fallback to email (trimmed tokens use email)
	if v, ok := claimsMap["email_id"].(string); ok {
		info.Email = v
	} else if v, ok := claimsMap["email"].(string); ok {
		info.Email = v
	}

	// Extract roles - handle both []string and []interface{}
	if roles, err := extractStringSlice(claimsMap["roles"]); err == nil {
		info.Roles = roles
	}

	// Extract groups (may be empty in trimmed tokens)
	if groups, err := extractStringSlice(claimsMap["groups"]); err == nil {
		info.Groups = groups
	}

	// Extract scopes - fallback to parsing scope string (trimmed tokens use scope string)
	if scopes, err := extractStringSlice(claimsMap["scopes"]); err == nil && len(scopes) > 0 {
		info.Scopes = scopes
	} else if scopeStr, ok := claimsMap["scope"].(string); ok && scopeStr != "" {
		info.Scopes = strings.Split(scopeStr, " ")
	}

	// Extract resources (may be empty in trimmed tokens)
	if resources, err := extractStringSlice(claimsMap["resources"]); err == nil {
		info.Resources = resources
	}

	return info, nil
}

// extractStringSlice extracts a string slice from JWT claims, handling both []string and []interface{}
func extractStringSlice(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, nil
	case nil:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("invalid type for string slice")
	}
}

func claimString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "", false
		}
		return v, true
	case fmt.Stringer:
		s := v.String()
		if s == "" {
			return "", false
		}
		return s, true
	case nil:
		return "", false
	default:
		s := fmt.Sprint(v)
		if s == "" {
			return "", false
		}
		return s, true
	}
}

func claimStringSlice(value interface{}) []string {
	if value == nil {
		return []string{}
	}

	if slice, err := extractStringSlice(value); err == nil {
		return slice
	}

	return []string{}
}

func stringSliceContains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func parseWithSecret(tokenString string, secret []byte) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
}

// ensureUserContextIdentifiers backfills missing user-related identifiers (like user_id)
func ensureUserContextIdentifiers(c *gin.Context) {
	// First, extract all important claims from the JWT into context
	if claimsVal, exists := c.Get("claims"); exists {
		switch claims := claimsVal.(type) {
		case jwt.MapClaims:
			// Extract email - check both "email" and "email_id" claims
			// Always set if found in claims, even if already in context
			emailFound := false
			if emailID, ok := claimString(claims["email_id"]); ok && emailID != "" {
				c.Set("email", emailID)
				c.Set("email_id", emailID)
				emailFound = true
			}
			if email, ok := claimString(claims["email"]); ok && email != "" && !emailFound {
				c.Set("email", email)
				c.Set("email_id", email) // backward compatibility
				emailFound = true
			}

			// Extract sub/user_id - check standard "sub" claim first
			if sub, ok := claimString(claims["sub"]); ok && sub != "" {
				c.Set("sub", sub)
				c.Set("user_id", sub) // backward compatibility
			} else if userID, ok := claimString(claims["user_id"]); ok && userID != "" {
				c.Set("user_id", userID)
				c.Set("sub", userID)
			}
		case map[string]interface{}:
			// Extract email - check both "email" and "email_id" claims
			emailFound := false
			if emailID := stringFromAny(claims["email_id"]); emailID != "" {
				c.Set("email", emailID)
				c.Set("email_id", emailID)
				emailFound = true
			}
			if email := stringFromAny(claims["email"]); email != "" && !emailFound {
				c.Set("email", email)
				c.Set("email_id", email)
				emailFound = true
			}

			// Extract sub/user_id - check both variants
			userIDFound := false
			if sub := stringFromAny(claims["sub"]); sub != "" {
				c.Set("sub", sub)
				c.Set("user_id", sub)
				userIDFound = true
			}
			if userID := stringFromAny(claims["user_id"]); userID != "" && !userIDFound {
				c.Set("user_id", userID)
				c.Set("sub", userID)
				userIDFound = true
			}
		}
	}

	if _, err := ResolveUserID(c); err != nil {
		normalizeUserInfoContext(c)
		return
	}

	normalizeUserInfoContext(c)
}

// ResolveUserID ensures a non-empty user_id is present in the Gin context.
// It first checks for an existing value and falls back to looking up the user by email
// in the master database when necessary. When resolved successfully, it also backfills
// related context (claims and user_info) to keep downstream handlers consistent.
func ResolveUserID(c *gin.Context) (string, error) {
	// Check user_id first, then sub (trimmed tokens use sub)
	if userID := getContextString(c, "user_id"); userID != "" {
		return userID, nil
	}
	if userID := getContextString(c, "sub"); userID != "" {
		c.Set("user_id", userID) // Backfill for compatibility
		return userID, nil
	}

	return resolveUserIDFromEmail(c)
}

func resolveUserIDFromEmail(c *gin.Context) (string, error) {
	// Try email first (standard JWT claim), then email_id (legacy)
	email := getContextString(c, "email")
	if email == "" {
		email = getContextString(c, "email_id")
	}
	if email == "" {
		return "", fmt.Errorf("email identifier not available in context")
	}

	dbConn := config.GetDatabase()
	if dbConn == nil || dbConn.DB == nil {
		return "", fmt.Errorf("database connection not available")
	}

	userRepo := database.NewUserRepository(dbConn)
	user, err := userRepo.GetUserByEmail(email)
	if err != nil || user == nil {
		if err == nil {
			err = fmt.Errorf("user not found")
		}
		return "", fmt.Errorf("failed to resolve user by email: %w", err)
	}

	userID := strings.TrimSpace(user.ID.String())
	if userID == "" {
		return "", fmt.Errorf("resolved user has empty identifier")
	}

	c.Set("user_id", userID)

	if claimsVal, exists := c.Get("claims"); exists {
		switch claims := claimsVal.(type) {
		case jwt.MapClaims:
			claims["user_id"] = userID
		case map[string]interface{}:
			claims["user_id"] = userID
			c.Set("claims", claims)
		}
	}

	userInfo := GetUserInfo(c)
	if userInfo == nil {
		userInfo = &UserInfo{}
	}
	userInfo.UserID = userID
	if userInfo.Email == "" {
		userInfo.Email = user.Email
	}
	if user.TenantID != uuid.Nil && userInfo.TenantID == "" {
		userInfo.TenantID = user.TenantID.String()
	}
	if user.ProjectID != uuid.Nil && userInfo.ProjectID == "" {
		userInfo.ProjectID = user.ProjectID.String()
	}
	if user.ClientID != uuid.Nil && userInfo.ClientID == "" {
		userInfo.ClientID = user.ClientID.String()
	}

	if rolesVal, exists := c.Get("roles"); exists {
		if roles, err := extractStringSlice(rolesVal); err == nil {
			userInfo.Roles = roles
		}
	}
	if groupsVal, exists := c.Get("groups"); exists {
		if groups, err := extractStringSlice(groupsVal); err == nil {
			userInfo.Groups = groups
		}
	}
	if scopesVal, exists := c.Get("scopes"); exists {
		if scopes, err := extractStringSlice(scopesVal); err == nil {
			userInfo.Scopes = scopes
		}
	}
	if resourcesVal, exists := c.Get("resources"); exists {
		if resources, err := extractStringSlice(resourcesVal); err == nil {
			userInfo.Resources = resources
		}
	}

	c.Set("user_info", userInfo)

	return userID, nil
}

// ensureTenantContext makes sure tenant_id is available even when omitted from JWTs
func ensureTenantContext(c *gin.Context) {
	if tenantID := getContextString(c, "tenant_id"); tenantID != "" {
		setTenantContext(c, tenantID)
		return
	}

	// Attempt to recover from claims directly
	if claimsVal, exists := c.Get("claims"); exists {
		switch claims := claimsVal.(type) {
		case jwt.MapClaims:
			if tenantID, ok := claimString(claims["tenant_id"]); ok {
				setTenantContext(c, tenantID)
				return
			}
			if tenantID, ok := claimString(claims["validated_tenant_id"]); ok {
				setTenantContext(c, tenantID)
				return
			}
		case map[string]interface{}:
			if tenantID := stringFromAny(claims["tenant_id"]); tenantID != "" {
				setTenantContext(c, tenantID)
				return
			}
			if tenantID := stringFromAny(claims["validated_tenant_id"]); tenantID != "" {
				setTenantContext(c, tenantID)
				return
			}
		}
	}

	// Fall back to tenant lookup using client_id when available
	clientID := getContextString(c, "client_id")
	if clientID == "" {
		// Try extracting from claims if missing in context
		if claimsVal, exists := c.Get("claims"); exists {
			switch claims := claimsVal.(type) {
			case jwt.MapClaims:
				if cid, ok := claimString(claims["client_id"]); ok {
					clientID = cid
				}
			case map[string]interface{}:
				clientID = stringFromAny(claims["client_id"])
			}
		}
	}

	if clientID == "" {
		return
	}

	if config.Database == nil || config.Database.DB == nil {
		return
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return
	}

	var tenantUUID uuid.UUID
	err = config.Database.DB.QueryRow(
		"SELECT tenant_id FROM tenant_mappings WHERE client_id = $1",
		clientUUID,
	).Scan(&tenantUUID)
	if err == nil {
		setTenantContext(c, tenantUUID.String())
	}
}

// setTenantContext normalizes tenant identifiers across context helpers
func setTenantContext(c *gin.Context, tenantID string) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}

	c.Set("tenant_id", tenantID)
	c.Set("validated_tenant_id", tenantID)

	if claimsVal, exists := c.Get("claims"); exists {
		switch claims := claimsVal.(type) {
		case jwt.MapClaims:
			claims["tenant_id"] = tenantID
			claims["validated_tenant_id"] = tenantID
		case map[string]interface{}:
			claims["tenant_id"] = tenantID
			claims["validated_tenant_id"] = tenantID
			c.Set("claims", claims)
		}
	}

	normalizeUserInfoContext(c)
}

// normalizeUserInfoContext ensures cached user_info is updated with resolved identifiers
func normalizeUserInfoContext(c *gin.Context) {
	userInfo := GetUserInfo(c)
	if userInfo == nil {
		return
	}

	if userInfo.TenantID == "" {
		if tenantID := getContextString(c, "tenant_id"); tenantID != "" {
			userInfo.TenantID = tenantID
		}
	}
	if userInfo.ClientID == "" {
		if clientID := getContextString(c, "client_id"); clientID != "" {
			userInfo.ClientID = clientID
		}
	}
	if userInfo.ProjectID == "" {
		if projectID := getContextString(c, "project_id"); projectID != "" {
			userInfo.ProjectID = projectID
		}
	}

	c.Set("user_info", userInfo)
}

// getContextString retrieves a normalized string value from Gin context
func getContextString(c *gin.Context, key string) string {
	value, exists := c.Get(key)
	if !exists {
		return ""
	}
	return stringFromAny(value)
}

// stringFromAny attempts to coerce an interface{} to string
func stringFromAny(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case uuid.UUID:
		return v.String()
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []byte:
		return strings.TrimSpace(string(v))
	case nil:
		return ""
	default:
		s := fmt.Sprint(v)
		return strings.TrimSpace(s)
	}
}

// setContextValues sets user information in Gin context
func setContextValues(c *gin.Context, claims jwt.MapClaims, userInfo *UserInfo) {
	// Set individual fields for backward compatibility
	c.Set("tenant_id", userInfo.TenantID)
	c.Set("project_id", userInfo.ProjectID)
	c.Set("client_id", userInfo.ClientID)
	c.Set("user_id", userInfo.UserID)
	c.Set("email", userInfo.Email)
	c.Set("email_id", userInfo.Email) // backward compatibility
	c.Set("roles", userInfo.Roles)
	c.Set("groups", userInfo.Groups)
	c.Set("scopes", userInfo.Scopes)
	c.Set("resources", userInfo.Resources)

	// Set token timing information
	c.Set("issued_at", claims["iat"])
	c.Set("expires_at", claims["exp"])
	c.Set("issuer", claims["iss"])
	c.Set("token_type", claims["token_type"])

	// Set full claims for advanced usage
	c.Set("claims", claims)
}

// performServerAuthorization performs server-side authorization check
func performServerAuthorization(c *gin.Context, tokenString string, userInfo *UserInfo) error {
	// Get the requested resource and action from context or request
	resource := c.Request.URL.Path
	method := c.Request.Method

	// Check if user has required permissions for this resource/action
	return checkPermissions(c, userInfo, resource, method)
}

// checkPermissions checks if user has permissions for the requested resource and action
func checkPermissions(c *gin.Context, userInfo *UserInfo, resource, method string) error {
	// Get claims from context
	claimsAny, ok := c.Get("claims")
	if !ok {
		return fmt.Errorf("no claims found in context")
	}
	claims, ok := claimsAny.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid claims type")
	}

	// Convert HTTP method to action
	action := mapHTTPMethodToAction(method)

	// Try auth-manager v0.9.0 style permission checking first (JWT-based)
	if hasPerm(claims, resource, action) {
		return nil
	}

	// Try scope-based checking as fallback
	scope := resource + ":" + action
	if hasScope(claims, scope) {
		return nil
	}

	// If database is available, fall back to database-based checking
	if config.DB != nil {
		// Parse tenant ID if present
		var tenantID *uuid.UUID
		if userInfo.TenantID != "" {
			if id, err := uuid.Parse(userInfo.TenantID); err == nil {
				tenantID = &id
			}
		}

		// Check resource method access using the RBAC system
		hasAccess, err := models.CheckResourceMethodAccess(config.DB, userInfo.Roles, method, resource, tenantID)
		if err != nil {
			return fmt.Errorf("permission check failed: %v", err)
		}

		if hasAccess {
			return nil
		}
	}

	return fmt.Errorf("insufficient permissions for %s %s", method, resource)
}

// hasPerm checks if the JWT claims contain permission for the given resource and action
// This implementation follows auth-manager v0.9.0
func hasPerm(claims jwt.MapClaims, r, a string) bool {
	// perms is expected to be []{ r:string, a:[]string }
	switch arr := claims["perms"].(type) {
	case []any:
		for _, p := range arr {
			if m, ok := p.(map[string]any); ok {
				if mr, _ := m["r"].(string); mr == r {
					switch acts := m["a"].(type) {
					case []any:
						for _, v := range acts {
							s, _ := v.(string)
							if s == a {
								return true
							}
						}
					case []string:
						for _, act := range acts {
							if act == a {
								return true
							}
						}
					}
				}
			}
		}
	case []map[string]any:
		for _, m := range arr {
			if mr, _ := m["r"].(string); mr == r {
				if acts, ok := m["a"].([]any); ok {
					for _, v := range acts {
						s, _ := v.(string)
						if s == a {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// hasScope checks if the JWT claims contain the given scope with wildcard support
// This implementation follows auth-manager v0.9.0
func hasScope(claims jwt.MapClaims, needed string) bool {
	// Prefer canonical "scope" (space-delimited string)
	if s, ok := claims["scope"].(string); ok && s != "" {
		fields := strings.Fields(s)
		if wildcardMatch(fields, needed) {
			return true
		}
	}

	// Fallback to "scopes" ([]string)
	switch arr := claims["scopes"].(type) {
	case []any:
		have := make([]string, 0, len(arr))
		for _, v := range arr {
			if sv, _ := v.(string); sv != "" {
				have = append(have, sv)
			}
		}
		return wildcardMatch(have, needed)
	case []string:
		return wildcardMatch(arr, needed)
	}
	return false
}

// mapHTTPMethodToAction converts HTTP methods to standard actions
func mapHTTPMethodToAction(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "read"
	case "POST":
		return "write"
	case "PUT":
		return "write"
	case "PATCH":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// wildcardMatch supports "resource:action" with *, e.g. "invoice:*", "*:read", "*:".
func wildcardMatch(have []string, needed string) bool {
	np := strings.SplitN(needed, ":", 2)
	if len(np) != 2 {
		return false
	}
	nr, na := np[0], np[1]

	for _, h := range have {
		hp := strings.SplitN(h, ":", 2)
		if len(hp) != 2 {
			continue
		}
		hr, ha := hp[0], hp[1]

		resourceMatch := (hr == nr || hr == "*" || nr == "*")
		actionMatch := (ha == na || ha == "*" || na == "*")

		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}

// ValidateJWTToken is a public wrapper for validateJWTToken for testing
func ValidateJWTToken(tokenString string, cfg *AuthConfig) (jwt.MapClaims, error) {
	return validateJWTToken(tokenString, cfg)
}

// ExtractUserInfo extracts user information from auth-manager claims in gin context
func ExtractUserInfo(c *gin.Context) (*UserInfo, error) {
	return extractUserInfo(c)
}

// GetUserInfo extracts user information from Gin context (for testing)
func GetUserInfo(c *gin.Context) *UserInfo {
	if userInfo, exists := c.Get("user_info"); exists {
		if ui, ok := userInfo.(*UserInfo); ok {
			return ui
		}
	}
	return nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WebSocketAuthMiddleware creates a middleware for WebSocket authentication
// that supports both Authorization header and query parameter token
// Use this for WebSocket endpoints since browsers can't send custom headers on WS upgrade
func WebSocketAuthMiddleware() gin.HandlerFunc {
	cfg := DefaultAuthConfig()

	return func(c *gin.Context) {
		var tokenString string

		// First try Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// If no header, try query parameter
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required. Pass token as query param: ?token=<JWT>"})
			c.Abort()
			return
		}

		// Validate the token
		claims, err := validateJWTToken(tokenString, cfg)
		if err != nil {
			fmt.Printf("Token validation error: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set claims in context (similar to AuthMiddleware)
		c.Set("claims", claims)

		// Extract and set individual claims
		if tenantID, ok := claims["tenant_id"].(string); ok {
			c.Set("tenant_id", tenantID)
		}
		if projectID, ok := claims["project_id"].(string); ok {
			c.Set("project_id", projectID)
		}
		if clientID, ok := claims["client_id"].(string); ok {
			c.Set("client_id", clientID)
		}
		if userID, ok := claims["sub"].(string); ok {
			c.Set("sub", userID)
			c.Set("user_id", userID)
		}
		if email, ok := claims["email"].(string); ok {
			c.Set("email", email)
			c.Set("email_id", email)
		}
		if jti, ok := claims["jti"].(string); ok {
			c.Set("jti", jti)
		}

		// Backfill context
		ensureTenantContext(c)
		ensureUserContextIdentifiers(c)

		c.Next()
	}
}

// ExtractTenantFromPath is a middleware that extracts the tenant_id from the URL path
// and sets it in the Gin context. This is used for routes where tenant_id is part of the URL.
func ExtractTenantFromPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	}
}
