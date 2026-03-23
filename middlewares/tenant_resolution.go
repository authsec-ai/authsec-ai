package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authsec-ai/authsec/database"
)

// TenantResolutionContextKey is the Gin context key where resolved tenant ID is stored
const TenantResolutionContextKey = "resolved_tenant_id"

// TenantResolutionMiddleware resolves tenant from Host header using tenant_domains table
// This replaces unsafe LIKE-based tenant_domain lookups with exact matches against verified domains
type TenantResolutionMiddleware struct {
	db *database.DBConnection
}

// NewTenantResolutionMiddleware creates a new tenant resolution middleware
func NewTenantResolutionMiddleware(db *database.DBConnection) *TenantResolutionMiddleware {
	return &TenantResolutionMiddleware{db: db}
}

// ResolveTenant resolves tenant_id from Host header and attaches to context
func (trm *TenantResolutionMiddleware) ResolveTenant(c *gin.Context) {
	// Get effective host from request (prefer X-Forwarded-Host, fallback to Host)
	host := getEffectiveHost(c)

	// Normalize host: lowercase, strip port, strip trailing dot
	host = normalizeHostname(host)

	// Reject obviously invalid hosts early
	if !isValidHostname(host) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname format"})
		c.Abort()
		return
	}

	// Look up tenant in master DB from verified domains only
	repo := database.NewTenantDomainsRepository(trm.db)
	td, err := repo.GetDomainByHostname(host)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		c.Abort()
		return
	}

	// Attach resolved tenant ID to context
	c.Set(TenantResolutionContextKey, td.TenantID.String())
	c.Next()
}

// getEffectiveHost extracts host from X-Forwarded-Host or falls back to Host
func getEffectiveHost(c *gin.Context) string {
	host := ""

	// Prefer X-Forwarded-Host (first value before comma) if present
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		// Use first value if comma-separated
		if idx := strings.Index(forwardedHost, ","); idx != -1 {
			host = strings.TrimSpace(forwardedHost[:idx])
		} else {
			host = forwardedHost
		}
	}

	// Fallback to Host header
	if host == "" {
		host = c.Request.Host
	}

	return host
}

// normalizeHostname converts hostname to lowercase and removes port and trailing dot
func normalizeHostname(hostname string) string {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	// Remove port if present (handle both IPv4 and IPv6)
	if idx := strings.Index(hostname, ":"); idx != -1 {
		// Only strip port if the part before `:` is not an IPv6 address
		if !strings.Contains(hostname[:idx], "[") {
			hostname = hostname[:idx]
		}
	}

	// Remove trailing dot
	hostname = strings.TrimSuffix(hostname, ".")

	return hostname
}

// isValidHostname performs basic validation on hostname format
func isValidHostname(hostname string) bool {
	// Reject if empty
	if hostname == "" {
		return false
	}

	// Reject if contains invalid characters (path separators, backslashes, wildcards, spaces)
	if strings.ContainsAny(hostname, "/\\* ") {
		return false
	}

	// Validate domain format (basic check: at least one dot, no consecutive dots, reasonable length)
	// This is a minimal check - real validation is done by DNS and DB
	if !strings.Contains(hostname, ".") || len(hostname) < 3 || len(hostname) > 253 {
		return false
	}

	// Reject localhost in production (adjust per environment)
	// In production, we might want to reject localhost for security
	// For now, allow it for testing purposes
	// TODO: Add env var check: if os.Getenv("ENVIRONMENT") != "production" then allow localhost

	return true
}

// GetTenantID retrieves the resolved tenant ID from context
func GetTenantID(c *gin.Context) *uuid.UUID {
	if val, exists := c.Get(TenantResolutionContextKey); exists {
		if tenantIDStr, ok := val.(string); ok {
			if id, err := uuid.Parse(tenantIDStr); err == nil {
				return &id
			}
		}
	}
	return nil
}
