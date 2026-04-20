package middleware

import (
	"net/http"
	"strings"

	"github.com/authsec-ai/authsec/internal/spire/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// JWTAuthMiddleware authenticates requests using Bearer JWT tokens.
// Use this for admin/user-flow endpoints that don't need mTLS.
type JWTAuthMiddleware struct {
	jwtValidator *utils.JWTValidator
	logger       *logrus.Logger
}

// NewJWTAuthMiddleware creates a new JWT-only authentication middleware.
func NewJWTAuthMiddleware(jwtValidator *utils.JWTValidator, logger *logrus.Logger) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		jwtValidator: jwtValidator,
		logger:       logger,
	}
}

// Authenticate returns a Gin middleware that validates JWT Bearer tokens from the Authorization header.
func (m *JWTAuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header with Bearer token required",
				},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
			})
			return
		}

		claims, err := m.jwtValidator.ValidateToken(parts[1])
		if err != nil {
			m.logger.WithError(err).Warn("JWT validation failed")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		if claims.TenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Tenant ID missing in JWT claims",
				},
			})
			return
		}

		// Set Gin context values
		c.Set(SpireTenantIDKey, claims.TenantID)
		c.Set(SpireUserIDKey, claims.UserID)
		c.Set(SpireClaimsKey, claims)
		c.Set(SpireIsAgentKey, false)

		m.logger.WithFields(logrus.Fields{
			"user_id":   claims.UserID,
			"tenant_id": claims.TenantID,
		}).Debug("JWT authenticated")

		c.Next()
	}
}
