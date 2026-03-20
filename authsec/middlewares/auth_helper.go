package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetTenantIDSafely extracts tenant_id from context with fallback
func GetTenantIDSafely(c *gin.Context) (string, bool) {
	// Try context first
	if tenantID, exists := c.Get("tenant_id"); exists && tenantID != nil {
		if tid, ok := tenantID.(string); ok && tid != "" {
			return tid, true
		}
	}

	// Try claims directly
	if claims, exists := c.Get("claims"); exists {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			if tid, exists := claimsMap["tenant_id"]; exists {
				if tidStr, ok := tid.(string); ok && tidStr != "" {
					c.Set("tenant_id", tidStr)
					return tidStr, true
				} else if tidNum, ok := tid.(float64); ok {
					c.Set("tenant_id", fmt.Sprintf("%.0f", tidNum))
					return fmt.Sprintf("%.0f", tidNum), true
				}
			}
		}
	}

	return "", false
}
