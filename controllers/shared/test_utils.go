package shared

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// setTokenClaimsInContext sets JWT claims in the Gin context for testing
// This mimics what the auth middleware does after validating a token
// It sets both the "claims" and "tenant_id" keys as the middleware does
func setTokenClaimsInContext(c *gin.Context, tenantID string, userID string) {
	claims := jwt.MapClaims{
		"tenant_id": tenantID,
		"sub":       userID,
	}
	c.Set("claims", claims)
	// Also set tenant_id directly as the middleware does
	c.Set("tenant_id", tenantID)
}

// setTokenClaimsWithProjectInContext sets JWT claims including project_id for testing
func setTokenClaimsWithProjectInContext(c *gin.Context, tenantID, userID, projectID string) {
	claims := jwt.MapClaims{
		"tenant_id":  tenantID,
		"sub":        userID,
		"project_id": projectID,
	}
	c.Set("claims", claims)
	// Also set tenant_id directly as the middleware does
	c.Set("tenant_id", tenantID)
}
