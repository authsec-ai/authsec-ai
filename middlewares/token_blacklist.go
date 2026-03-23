package middlewares

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenBlacklistChecker interface to avoid circular dependency
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(tokenString string) (bool, error)
}

// TokenBlacklistMiddleware checks if the access token has been blacklisted
func TokenBlacklistMiddleware(checker TokenBlacklistChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			// No checker provided, skip blacklist check
			c.Next()
			return
		}
		
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token, skip blacklist check (will fail in auth middleware)
			c.Next()
			return
		}
		
		// Extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, skip blacklist check (will fail in auth middleware)
			c.Next()
			return
		}
		
		tokenString := parts[1]
		
		// Check if token is blacklisted
		blacklisted, err := checker.IsTokenBlacklisted(tokenString)
		if err != nil {
			// Fail-closed by default: if we cannot verify the blacklist, reject the request
			// Operators can set TOKEN_BLACKLIST_FAIL_OPEN=true to override
			if os.Getenv("TOKEN_BLACKLIST_FAIL_OPEN") == "true" {
				log.Printf("WARNING: Token blacklist check failed (fail-open enabled): %v", err)
				c.Next()
				return
			}
			log.Printf("ERROR: Token blacklist check failed (fail-closed): %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
			})
			c.Abort()
			return
		}
		
		if blacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token has been revoked",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
