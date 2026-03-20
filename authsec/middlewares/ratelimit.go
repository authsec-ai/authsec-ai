package middlewares

import (
	"net/http"
	"strings"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimitMiddleware creates rate limiting middleware with different limits for different endpoints
func RateLimitMiddleware() gin.HandlerFunc {
	// General API rate limiter (100 requests per minute per IP)
	generalLimiter := tollbooth.NewLimiter(1.67, nil) // 100 requests per minute = 1.67 per second
	generalLimiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"})
	generalLimiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"})
	generalLimiter.SetMessage(`{"error": "Rate limit exceeded", "message": "Too many requests. Please try again later."}`)

	// Authentication endpoints rate limiter (100 requests per minute per IP)
	authLimiter := tollbooth.NewLimiter(1.67, nil) // 100 requests per minute = 1.67 per second
	authLimiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"})
	authLimiter.SetMethods([]string{"POST"})
	authLimiter.SetMessage(`{"error": "Authentication rate limit exceeded", "message": "Too many authentication attempts. Please wait before trying again."}`)

	// Admin endpoints rate limiter (100 requests per minute per IP)
	adminLimiter := tollbooth.NewLimiter(1.67, nil) // 100 requests per minute = 1.67 per second
	adminLimiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"})
	adminLimiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"})
	adminLimiter.SetMessage(`{"error": "Admin rate limit exceeded", "message": "Too many admin requests. Please try again later."}`)

	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		// Determine which limiter to use based on the path
		var selectedLimiter *limiter.Limiter
		var limitType string

		if strings.HasPrefix(path, "/api/v1/auth/") {
			selectedLimiter = authLimiter
			limitType = "auth"
		} else if strings.HasPrefix(path, "/api/v1/admin/") {
			selectedLimiter = adminLimiter
			limitType = "admin"
		} else {
			selectedLimiter = generalLimiter
			limitType = "general"
		}

		// Check rate limit
		httpError := tollbooth.LimitByRequest(selectedLimiter, c.Writer, c.Request)
		if httpError != nil {
			// Rate limit exceeded
			logrus.WithFields(logrus.Fields{
				"request_id":  requestID,
				"client_ip":   clientIP,
				"path":        path,
				"method":      c.Request.Method,
				"limit_type":  limitType,
				"user_agent":  c.GetHeader("User-Agent"),
			}).Warn("Rate limit exceeded")

			// The tollbooth library already sent the response, so we just need to abort
			//c.Abort()
			return
		}

		// Log successful requests (with rate limit info)
		// Note: tollbooth v7 doesn't expose remaining/reset info directly
		logrus.WithFields(logrus.Fields{
			"request_id": requestID,
			"client_ip":  clientIP,
			"path":       path,
			"method":     c.Request.Method,
			"limit_type": limitType,
		}).Debug("Rate limit check passed")

		c.Next()
	}
}

// TenantRateLimitMiddleware provides per-tenant rate limiting
func TenantRateLimitMiddleware() gin.HandlerFunc {
	// Tenant-specific rate limiter (1000 requests per minute per tenant per IP)
	tenantLimiter := tollbooth.NewLimiter(16.67, nil) // 1000 requests per minute = 16.67 per second
	tenantLimiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"})
	tenantLimiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"})
	tenantLimiter.SetMessage(`{"error": "Tenant rate limit exceeded", "message": "Too many requests for this tenant. Please try again later."}`)

	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			// No tenant specified, skip tenant-specific rate limiting
			c.Next()
			return
		}

		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		// Create a composite key for tenant + IP rate limiting
		tenantIPKey := tenantID + ":" + clientIP

		// Check tenant-specific rate limit
		httpError := tollbooth.LimitByKeys(tenantLimiter, []string{tenantIPKey})
		if httpError != nil {
			// Rate limit exceeded for this tenant
			logrus.WithFields(logrus.Fields{
				"request_id":  requestID,
				"client_ip":   clientIP,
				"tenant_id":   tenantID,
				"path":        path,
				"method":      c.Request.Method,
				"user_agent":  c.GetHeader("User-Agent"),
			}).Warn("Tenant rate limit exceeded")

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "Tenant rate limit exceeded",
				"message":    "Too many requests for this tenant. Please try again later.",
				"request_id": requestID,
				"tenant_id":  tenantID,
			})
			return
		}

		// Log tenant rate limit info
		// Note: tollbooth v7 doesn't expose remaining/reset info directly
		logrus.WithFields(logrus.Fields{
			"request_id": requestID,
			"client_ip":  clientIP,
			"tenant_id":  tenantID,
			"path":       path,
			"method":     c.Request.Method,
		}).Debug("Tenant rate limit check passed")

		c.Next()
	}
}