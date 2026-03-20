package middlewares

import (
	"log"
	"net/http"
	"strings"

	"github.com/authsec-ai/authsec/config"
	"github.com/gin-gonic/gin"
)

// matchesWildcardOrigin checks if an origin matches a wildcard pattern like *.app.authsec.dev
func matchesWildcardOrigin(origin, pattern string) bool {
	// Normalize origin to host-only (strip protocol)
	originHost := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")

	// pattern is expected to be host-only (no protocol) and may start with '*.'
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		if originHost == suffix {
			return true
		}
		if strings.HasSuffix(originHost, "."+suffix) {
			return true
		}
	}

	return false
}

func CORSMiddleware() gin.HandlerFunc {
	// Parse allowed origins from config
	// Support entries like: "*", "https://app.authsec.dev", "https://*.app.authsec.dev" or simply "*.app.authsec.dev"
	allowedOrigins := strings.Split(config.AppConfig.CorsAllowOrigin, ",")
	originSet := make(map[string]struct{}, len(allowedOrigins))
	wildcardPatterns := make([]string, 0)
	allowAll := false

	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}

		// Normalize and detect wildcard patterns even if protocol is included (e.g. https://*.app.authsec.dev)
		hostOnly := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
		if strings.HasPrefix(hostOnly, "*.") {
			// store host-only wildcard (no protocol) for pattern matching
			wildcardPatterns = append(wildcardPatterns, hostOnly)
			continue
		}

		// Keep exact origin matching (including protocol) for straightforward matches
		originSet[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		// Get the request's Origin header
		origin := c.GetHeader("Origin")

		if origin == "" {
			// No Origin header; proceed without CORS headers
			c.Next()
			return
		}

		// Check if the origin is allowed
		allowed := allowAll // Only explicit "*" allows all origins; empty config blocks all
		if !allowed {
			// Check exact match first (origin may include protocol)
			if _, ok := originSet[origin]; ok {
				allowed = true
			} else {
				// Try matching against origin sets that were added without protocol (normalize)
				// e.g., config "https://app.authsec.dev" vs stored host-only entries
				originHost := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
				if _, ok := originSet[originHost]; ok {
					allowed = true
				} else {
					// Check wildcard patterns (host-only)
					for _, pattern := range wildcardPatterns {
						if matchesWildcardOrigin(origin, pattern) {
							allowed = true
							break
						}
					}
				}
			}
		}

		if !allowed {
			log.Printf("CORS: Blocked origin: %s. Configured origins: %v, AllowAll: %v", origin, allowedOrigins, allowAll)
		}

		if allowed {
			// Always echo the specific origin to support credentials
			// Never use "*" with Access-Control-Allow-Credentials: true
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, HEAD")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With, X-CSRF-Token, X-Tenant-ID, tenant_id, Client-Id, client-id, X-Client-Id, X-Client-ID")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours
			c.Writer.Header().Set("Vary", "Origin")                  // Important for caching
		} else {
			// Silently block unallowed origin
		}

		// Handle preflight OPTIONS requests
		if c.Request.Method == http.MethodOptions {
			if allowed {
				c.AbortWithStatus(http.StatusOK)
			} else {
				c.AbortWithStatus(http.StatusForbidden) // Block unallowed origins
			}
			return
		}

		c.Next()
	}
}
