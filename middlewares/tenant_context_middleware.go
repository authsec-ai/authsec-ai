package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/authsec-ai/authsec/config"
	"github.com/gin-gonic/gin"
)

// TenantContextMiddleware extracts tenant_id, email, and client_id from request body
// and sets them in the Gin context for use by logging middleware.
// It preserves the request body so handlers can still read it.
func TenantContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process POST requests with JSON body
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		// Only process requests with JSON content type
		contentType := c.GetHeader("Content-Type")
		if contentType != "application/json" && contentType != "" {
			c.Next()
			return
		}

		// Read the request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Restore the body so handlers can read it
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Try to parse as JSON to extract tenant_id, email, client_id
		var requestData map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
			// If it's not valid JSON, just continue
			c.Next()
			return
		}

		// Extract tenant_id and email from request
		tenantID, tenantIDExists := requestData["tenant_id"].(string)
		email, emailExists := requestData["email"].(string)

		// If tenant_id is not provided but email is, look it up from the database
		if (!tenantIDExists || tenantID == "") && emailExists && email != "" {
			if lookedUpTenantID := lookupTenantIDByEmail(email); lookedUpTenantID != "" {
				tenantID = lookedUpTenantID
				tenantIDExists = true
				log.Printf("[TenantContext] Looked up tenant_id '%s' for email '%s'", tenantID, email)
			}
		}

		// Set tenant_id in context
		if tenantIDExists && tenantID != "" {
			c.Set("tenant_id", tenantID)
		}

		// Set email in context (which serves as user identifier)
		if emailExists && email != "" {
			c.Set("email_id", email)
			// Also use email as user_id if no explicit user_id is provided
			if _, exists := requestData["user_id"]; !exists {
				c.Set("user_id", email)
			}
		}

		// Extract and set client_id if present
		if clientID, ok := requestData["client_id"].(string); ok && clientID != "" {
			c.Set("client_id", clientID)
			// Use client_id as user_id if available
			c.Set("user_id", clientID)
		}

		// Extract and set user_id if explicitly provided
		if userID, ok := requestData["user_id"].(string); ok && userID != "" {
			c.Set("user_id", userID)
		}

		c.Next()
	}
}

// lookupTenantIDByEmail queries the global database to find the tenant_id for a given email
func lookupTenantIDByEmail(email string) string {
	// Connect to global DB
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		log.Printf("[TenantContext] Failed to connect to global DB for tenant lookup: %v", err)
		return ""
	}

	// Query the tenants table for the email
	var tenant sharedmodels.Tenant
	if err := globalDB.Where("email = ?", email).Select("tenant_id").First(&tenant).Error; err != nil {
		// Not found or error - this is not critical, just log at debug level
		log.Printf("[TenantContext] Could not find tenant_id for email '%s': %v", email, err)
		return ""
	}

	// Return the tenant_id as string
	return tenant.TenantID.String()
}
