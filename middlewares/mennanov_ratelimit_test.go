package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMennovRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test router with rate limiting
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(MennovRateLimitMiddleware())

	// Test endpoint
	router.POST("/test/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("AllowsRequestsWithinLimit", func(t *testing.T) {
		// Auth endpoints have 10 req/min limit
		// Send 5 requests - should all succeed
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "/test/login", nil)
			req.RemoteAddr = "192.168.1.100:1234"
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code, "Request %d should succeed", i+1)
		}
	})

	// Reset for this test
	time.Sleep(100 * time.Millisecond)

	t.Run("BlocksRequestsExceedingLimit", func(t *testing.T) {
		// Create new router to isolate test
		testRouter := gin.New()
		testRouter.Use(RequestIDMiddleware())
		testRouter.Use(MennovRateLimitMiddleware())
		testRouter.POST("/uflow/auth/enduser/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Send 11 requests from same IP (limit is 10)
		successCount := 0
		blockedCount := 0
		for i := 0; i < 11; i++ {
			req, _ := http.NewRequest("POST", "/uflow/auth/enduser/login", nil)
			req.RemoteAddr = "192.168.1.101:5678"
			resp := httptest.NewRecorder()
			testRouter.ServeHTTP(resp, req)
			if resp.Code == http.StatusOK {
				successCount++
			} else if resp.Code == http.StatusTooManyRequests {
				blockedCount++
				// Verify error response
				var errResp map[string]interface{}
				json.Unmarshal(resp.Body.Bytes(), &errResp)
				assert.Equal(t, "Rate limit exceeded", errResp["error"])
				assert.Contains(t, errResp["message"], "10 requests")
			}
		}
		assert.Equal(t, 10, successCount, "Should allow exactly 10 requests")
		assert.Equal(t, 1, blockedCount, "Should block 11th request")
	})

	t.Run("DifferentIPsHaveSeparateLimits", func(t *testing.T) {
		testRouter := gin.New()
		testRouter.Use(RequestIDMiddleware())
		testRouter.Use(MennovRateLimitMiddleware())
		testRouter.POST("/uflow/auth/enduser/register", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// IP 1 sends 10 requests
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("POST", "/uflow/auth/enduser/register", nil)
			req.RemoteAddr = "192.168.1.200:1111"
			resp := httptest.NewRecorder()
			testRouter.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code)
		}

		// IP 2 should still be able to send requests
		req, _ := http.NewRequest("POST", "/uflow/auth/enduser/register", nil)
		req.RemoteAddr = "192.168.1.201:2222"
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code, "Different IP should have separate limit")
	})

	t.Run("AuthEndpointsHaveStricterLimit", func(t *testing.T) {
		testRouter := gin.New()
		testRouter.Use(RequestIDMiddleware())
		testRouter.Use(MennovRateLimitMiddleware())

		testRouter.POST("/uflow/auth/enduser/forgot-password", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		testRouter.GET("/api/v1/general/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Auth endpoint: 10 req/min
		authSuccessCount := 0
		for i := 0; i < 15; i++ {
			req, _ := http.NewRequest("POST", "/uflow/auth/enduser/forgot-password", nil)
			req.RemoteAddr = "192.168.1.202:3333"
			resp := httptest.NewRecorder()
			testRouter.ServeHTTP(resp, req)
			if resp.Code == http.StatusOK {
				authSuccessCount++
			}
		}
		assert.Equal(t, 10, authSuccessCount, "Auth endpoint should allow 10 requests")

		// General endpoint: 100 req/min (should allow more)
		generalSuccessCount := 0
		for i := 0; i < 50; i++ {
			req, _ := http.NewRequest("GET", "/api/v1/general/health", nil)
			req.RemoteAddr = "192.168.1.203:4444"
			resp := httptest.NewRecorder()
			testRouter.ServeHTTP(resp, req)
			if resp.Code == http.StatusOK {
				generalSuccessCount++
			}
		}
		assert.Equal(t, 50, generalSuccessCount, "General endpoint should allow all 50 requests")
	})
}

func TestStrictAuthRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EnforcesStricterLimit", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())
		// Apply strict limit: 3 requests per minute
		router.Use(StrictAuthRateLimitMiddleware(3, time.Minute))

		router.POST("/reset-password", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Send 5 requests
		successCount := 0
		blockedCount := 0
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "/reset-password", nil)
			req.RemoteAddr = "192.168.1.250:9999"
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code == http.StatusOK {
				successCount++
			} else if resp.Code == http.StatusTooManyRequests {
				blockedCount++
			}
		}
		assert.Equal(t, 3, successCount, "Should allow exactly 3 requests")
		assert.Equal(t, 2, blockedCount, "Should block remaining 2 requests")
	})
}

func TestSlidingWindowBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("AllowsRequestsAfterWindowExpires", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestIDMiddleware())
		// Use very short window for testing: 2 seconds, 2 requests max
		router.Use(StrictAuthRateLimitMiddleware(2, 2*time.Second))

		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Send 2 requests - should succeed
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("POST", "/test", nil)
			req.RemoteAddr = "192.168.1.251:8888"
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code)
		}

		// 3rd request should fail
		req, _ := http.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "192.168.1.251:8888"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusTooManyRequests, resp.Code)

		// Wait for window to expire
		time.Sleep(2100 * time.Millisecond)

		// Should be able to send requests again
		req, _ = http.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "192.168.1.251:8888"
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code, "Should allow request after window expires")
	})
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(MennovRateLimitMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", bytes.NewBuffer(nil))
		req.RemoteAddr = "127.0.0.1:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	}
}
