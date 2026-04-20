package middleware

import (
	"crypto/x509"

	"github.com/authsec-ai/authsec/internal/spire/utils"
	"github.com/gin-gonic/gin"
)

// Context keys used across SPIRE middleware
const (
	SpireTenantIDKey   = "spire_tenant_id"
	SpireUserIDKey     = "spire_user_id"
	SpireClaimsKey     = "spire_claims"
	SpireSpiffeIDKey   = "spire_spiffe_id"
	SpireIsAgentKey    = "spire_is_agent"
	SpireClientCertKey = "spire_client_cert"
	SpireAgentIDKey    = "spire_agent_id"
)

// GetSpireTenantID extracts the tenant ID from the Gin context.
func GetSpireTenantID(c *gin.Context) (string, bool) {
	val, exists := c.Get(SpireTenantIDKey)
	if !exists {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// GetSpireUserID extracts the user ID from the Gin context.
func GetSpireUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(SpireUserIDKey)
	if !exists {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// GetSpireClaims extracts the JWT claims from the Gin context.
func GetSpireClaims(c *gin.Context) (*utils.JWTClaims, bool) {
	val, exists := c.Get(SpireClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*utils.JWTClaims)
	return claims, ok
}

// GetSpireSpiffeID extracts the SPIFFE ID from the Gin context.
func GetSpireSpiffeID(c *gin.Context) (string, bool) {
	val, exists := c.Get(SpireSpiffeIDKey)
	if !exists {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// GetSpireIsAgent extracts the agent flag from the Gin context.
func GetSpireIsAgent(c *gin.Context) (bool, bool) {
	val, exists := c.Get(SpireIsAgentKey)
	if !exists {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// GetSpireClientCert extracts the client certificate from the Gin context.
func GetSpireClientCert(c *gin.Context) (*x509.Certificate, bool) {
	val, exists := c.Get(SpireClientCertKey)
	if !exists {
		return nil, false
	}
	cert, ok := val.(*x509.Certificate)
	return cert, ok
}

// GetSpireAgentID extracts the agent database ID from the Gin context.
func GetSpireAgentID(c *gin.Context) (string, bool) {
	val, exists := c.Get(SpireAgentIDKey)
	if !exists {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}
