package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDistributedRateLimitMiddleware_NilLimiterFallsThrough(t *testing.T) {
	router := gin.New()
	router.Use(DistributedRateLimitMiddleware(nil))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when limiter is nil, got %d", w.Code)
	}
}

func TestDistributedRateLimitMiddleware_NilClientFallsThrough(t *testing.T) {
	rl := NewRedisRateLimiter(nil)

	router := gin.New()
	router.Use(DistributedRateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when Redis client is nil, got %d", w.Code)
	}
}

func TestDistributedStrictRateLimitMiddleware_NilFallsThrough(t *testing.T) {
	router := gin.New()
	router.Use(DistributedStrictRateLimitMiddleware(nil, 5, 60e9))
	router.GET("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when limiter is nil, got %d", w.Code)
	}
}
