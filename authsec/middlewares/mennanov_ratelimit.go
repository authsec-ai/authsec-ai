package middlewares

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Simple in-memory store for rate limiting
type inMemoryStore struct {
	mu    sync.RWMutex
	items map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	timestamps []time.Time
	mu         sync.Mutex
}

var (
	// In-memory store for rate limiting
	memStore = &inMemoryStore{
		items: make(map[string]*rateLimitEntry),
	}
)

// cleanup removes expired entries periodically
func (s *inMemoryStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.items {
				entry.mu.Lock()
				// Remove timestamps older than 2 minutes
				validTimestamps := []time.Time{}
				for _, ts := range entry.timestamps {
					if now.Sub(ts) < 2*time.Minute {
						validTimestamps = append(validTimestamps, ts)
					}
				}
				entry.timestamps = validTimestamps

				// Remove entry if no timestamps left
				if len(entry.timestamps) == 0 {
					delete(s.items, key)
				}
				entry.mu.Unlock()
			}
			s.mu.Unlock()
		}
	}()
}

func init() {
	memStore.cleanup()
}

// checkLimit checks if the request exceeds the rate limit using sliding window
func (s *inMemoryStore) checkLimit(key string, limit int, window time.Duration) bool {
	s.mu.Lock()
	entry, exists := s.items[key]
	if !exists {
		entry = &rateLimitEntry{
			timestamps: []time.Time{},
		}
		s.items[key] = entry
	}
	s.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()

	// Remove timestamps outside the window
	validTimestamps := []time.Time{}
	for _, ts := range entry.timestamps {
		if now.Sub(ts) < window {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	entry.timestamps = validTimestamps

	// Check if limit exceeded
	if len(entry.timestamps) >= limit {
		return false // Rate limit exceeded
	}

	// Add current timestamp
	entry.timestamps = append(entry.timestamps, now)
	return true // Request allowed
}

// RateLimitConfig defines rate limiting configuration for different endpoint types
type RateLimitConfig struct {
	Limit    int           // Number of requests allowed
	Window   time.Duration // Time window for the limit
	Endpoint string        // Endpoint type for logging
}

// Common rate limit configurations
var (
	// Authentication endpoints: 10 requests per minute
	AuthRateLimit = RateLimitConfig{
		Limit:    10,
		Window:   time.Minute,
		Endpoint: "auth",
	}

	// General API endpoints: 100 requests per minute
	GeneralRateLimit = RateLimitConfig{
		Limit:    100,
		Window:   time.Minute,
		Endpoint: "general",
	}

	// Admin endpoints: 50 requests per minute
	AdminRateLimit = RateLimitConfig{
		Limit:    50,
		Window:   time.Minute,
		Endpoint: "admin",
	}
)

// MennovRateLimitMiddleware creates rate limiting middleware using in-memory sliding window
func MennovRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		// Determine rate limit config based on path
		config := selectRateLimitConfig(path)

		// Create rate limiter key: IP + endpoint type
		key := fmt.Sprintf("%s:%s", clientIP, config.Endpoint)

		// Check if request is allowed
		allowed := memStore.checkLimit(key, config.Limit, config.Window)

		if !allowed {
			// Rate limit exceeded
			logrus.WithFields(logrus.Fields{
				"request_id": requestID,
				"client_ip":  clientIP,
				"path":       path,
				"method":     c.Request.Method,
				"limit_type": config.Endpoint,
				"limit":      config.Limit,
				"window":     config.Window,
				"user_agent": c.GetHeader("User-Agent"),
			}).Warn("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Maximum %d requests per %v allowed. Please try again later.", config.Limit, config.Window),
				"limit":   config.Limit,
				"window":  config.Window.String(),
			})
			c.Abort()
			return
		}

		// Log successful request
		logrus.WithFields(logrus.Fields{
			"request_id": requestID,
			"client_ip":  clientIP,
			"path":       path,
			"method":     c.Request.Method,
			"limit_type": config.Endpoint,
		}).Debug("Rate limit check passed")

		c.Next()
	}
}

// selectRateLimitConfig determines which rate limit to apply based on path
func selectRateLimitConfig(path string) RateLimitConfig {
	// Authentication endpoints (login, register, forgot password, reset password, OTP)
	authPaths := []string{
		"/uflow/auth/enduser/login",
		"/uflow/auth/enduser/register",
		"/uflow/auth/enduser/forgot-password",
		"/uflow/auth/enduser/reset-password",
		"/uflow/auth/enduser/verify-otp",
		"/uflow/auth/enduser/resend-otp",
		"/uflow/auth/admin/login",
		"/uflow/auth/admin/forgot-password",
		"/uflow/auth/admin/reset-password",
		"/uflow/auth/oidc/",
		"/uflow/auth/saml/",
		"/uflow/auth/webauthn/",
		"/uflow/auth/totp/",
		"/uflow/auth/device/",
		"/uflow/auth/ciba/",
		"/api/v1/auth/",
	}

	for _, authPath := range authPaths {
		if strings.HasPrefix(path, authPath) {
			return AuthRateLimit
		}
	}

	// Admin endpoints
	if strings.HasPrefix(path, "/api/v1/admin/") || strings.HasPrefix(path, "/uflow/admin/") {
		return AdminRateLimit
	}

	// Default to general rate limit
	return GeneralRateLimit
}

// StrictAuthRateLimitMiddleware provides stricter rate limiting for sensitive auth operations
// Use this for endpoints like password reset, OTP verification where even 10/min might be too high
func StrictAuthRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		// Create rate limiter key: IP + path (more granular)
		key := fmt.Sprintf("%s:%s", clientIP, path)

		// Check if request is allowed
		allowed := memStore.checkLimit(key, limit, window)

		if !allowed {
			// Rate limit exceeded
			logrus.WithFields(logrus.Fields{
				"request_id": requestID,
				"client_ip":  clientIP,
				"path":       path,
				"method":     c.Request.Method,
				"limit":      limit,
				"window":     window,
			}).Warn("Strict rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many attempts. Maximum %d requests per %v allowed. Please try again later.", limit, window),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerUserRateLimitMiddleware provides per-user rate limiting (requires authentication)
// Use this to limit authenticated users separately from IP-based limits
func PerUserRateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from token claims
		userID, exists := c.Get("user_id")
		if !exists {
			// Fall back to IP-based limiting if no user ID
			clientIP := c.ClientIP()
			userID = clientIP
		}

		requestID := c.GetString("request_id")
		path := c.Request.URL.Path

		// Create rate limiter key: user ID
		key := fmt.Sprintf("user:%v", userID)

		// Check if request is allowed
		allowed := memStore.checkLimit(key, limit, window)

		if !allowed {
			// Rate limit exceeded
			logrus.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    userID,
				"path":       path,
				"method":     c.Request.Method,
				"limit":      limit,
				"window":     window,
			}).Warn("Per-user rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Maximum %d requests per %v allowed.", limit, window),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
