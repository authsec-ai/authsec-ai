package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/authsec-ai/authsec/config"
	"github.com/gin-gonic/gin"
)

// ValidateTenantFromToken ensures the tenant_id in URL/body matches the JWT token's tenant_id
// This prevents tenant spoofing attacks where a user tries to access another tenant's data
func ValidateTenantFromToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant_id from URL parameter
		urlTenantID := c.Param("tenant_id")
		if urlTenantID == "" {
			// No tenant_id in URL, skip validation
			c.Next()
			return
		}

		// Get tenant_id from JWT token (set by AuthMiddleware after validation)
		tokenTenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Tenant information not found in authentication token",
			})
			c.Abort()
			return
		}

		tokenTenantStr, ok := tokenTenantID.(string)
		if !ok || tokenTenantStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid tenant information in authentication token",
			})
			c.Abort()
			return
		}

		// Compare tenant IDs (case-insensitive)
		if !strings.EqualFold(urlTenantID, tokenTenantStr) {
			// EXCEPTION: Allow admins to access other tenants only if explicitly enabled
			if isAdminUser(c) && os.Getenv("ADMIN_CROSS_TENANT_ACCESS") == "true" {
				// Admin accessing another tenant - log it for audit
				logCrossTenantAccess(c, tokenTenantStr, urlTenantID)
				c.Next()
				return
			}

			// Regular user trying to access wrong tenant → BLOCK
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied: Tenant mismatch",
				"message": "You can only access resources from your own tenant",
			})
			c.Abort()
			return
		}

		// Validation passed - URL tenant matches token tenant
		c.Next()
	}
}

// MustMatchTenantFromToken is a stricter version that doesn't allow admin bypass
// Use this for sensitive operations where even admins must use their own tenant
func MustMatchTenantFromToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlTenantID := c.Param("tenant_id")
		if urlTenantID == "" {
			c.Next()
			return
		}

		tokenTenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Tenant not found in token",
			})
			c.Abort()
			return
		}

		tokenTenantStr, ok := tokenTenantID.(string)
		if !ok || tokenTenantStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid tenant in token",
			})
			c.Abort()
			return
		}

		if !strings.EqualFold(urlTenantID, tokenTenantStr) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Tenant mismatch - access denied",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetTenantIDFromToken safely extracts tenant_id from the validated JWT token
// Returns empty string and false if tenant_id is not found or invalid
// This should be used by all controllers instead of c.Param("tenant_id")
func GetTenantIDFromToken(c *gin.Context) (string, bool) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		return "", false
	}

	tenantID, ok := tenantIDVal.(string)
	if !ok || tenantID == "" {
		return "", false
	}

	return tenantID, true
}

// GetValidatedTenantID extracts tenant_id from token and returns error response if not found
// Use this in controllers for cleaner error handling
func GetValidatedTenantID(c *gin.Context) (string, error) {
	tenantID, ok := GetTenantIDFromToken(c)
	if !ok {
		return "", ErrTenantNotFoundInToken
	}
	return tenantID, nil
}

// isAdminUser checks if the authenticated user has an admin role
func isAdminUser(c *gin.Context) bool {
	rolesAny, exists := c.Get("roles")
	if !exists {
		return false
	}

	roles, err := extractStringSlice(rolesAny)
	if err != nil {
		return false
	}

	for _, role := range roles {
		if strings.EqualFold(role, "admin") ||
			strings.EqualFold(role, "administrator") ||
			strings.EqualFold(role, "super_admin") {
			return true
		}
	}
	return false
}

// logCrossTenantAccess logs when an admin accesses another tenant's data
func logCrossTenantAccess(c *gin.Context, ownTenantID, accessedTenantID string) {
	if config.AuditLogger == nil {
		return
	}

	requestID, _ := c.Get("request_id")
	requestIDStr, _ := requestID.(string)

	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	// Log the cross-tenant access for security audit using LogAdminAction
	config.AuditLogger.LogAdminAction(
		requestIDStr,
		accessedTenantID, // The tenant being accessed
		userIDStr,
		"cross_tenant_access",
		"tenant",
		accessedTenantID,
		c.Request.Method,
		c.Request.URL.Path,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		200, // Status code for successful cross-tenant access
		0,   // Duration (not measured here)
		map[string]string{"admin_tenant": ownTenantID},
		map[string]string{"accessed_tenant": accessedTenantID},
		"", // No error
	)
}

// Custom errors for tenant validation
var (
	ErrTenantNotFoundInToken = &TenantValidationError{
		Message:    "Tenant ID not found in authentication token",
		StatusCode: http.StatusUnauthorized,
	}
	ErrInvalidTenantInToken = &TenantValidationError{
		Message:    "Invalid tenant ID in authentication token",
		StatusCode: http.StatusUnauthorized,
	}
	ErrTenantMismatch = &TenantValidationError{
		Message:    "Tenant ID mismatch - access denied",
		StatusCode: http.StatusForbidden,
	}
)

// TenantValidationError represents a tenant validation error
type TenantValidationError struct {
	Message    string
	StatusCode int
}

func (e *TenantValidationError) Error() string {
	return e.Message
}
