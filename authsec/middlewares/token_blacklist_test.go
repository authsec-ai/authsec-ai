package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockBlacklistChecker implements TokenBlacklistChecker for testing
type mockBlacklistChecker struct {
	blacklisted map[string]bool
	err         error
}

func (m *mockBlacklistChecker) IsTokenBlacklisted(token string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.blacklisted[token], nil
}

func TestTokenBlacklistMiddleware_AllowsNonBlacklistedToken(t *testing.T) {
	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		blacklisted: map[string]bool{},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_BlocksBlacklistedToken(t *testing.T) {
	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		blacklisted: map[string]bool{"revoked-token": true},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for blacklisted token, got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_FailClosedOnError(t *testing.T) {
	// Default behavior: fail-closed (503) when blacklist check errors
	os.Unsetenv("TOKEN_BLACKLIST_FAIL_OPEN")

	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		err: fmt.Errorf("redis connection failed"),
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (fail-closed on error), got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_FailOpenOnError(t *testing.T) {
	// Opt-in fail-open: set TOKEN_BLACKLIST_FAIL_OPEN=true
	os.Setenv("TOKEN_BLACKLIST_FAIL_OPEN", "true")
	defer os.Unsetenv("TOKEN_BLACKLIST_FAIL_OPEN")

	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		err: fmt.Errorf("redis connection failed"),
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (fail-open on error), got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_SkipsWhenNoChecker(t *testing.T) {
	router := gin.New()
	router.Use(TokenBlacklistMiddleware(nil))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no checker, got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_SkipsWhenNoAuthHeader(t *testing.T) {
	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		blacklisted: map[string]bool{},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with no auth header, got %d", w.Code)
	}
}

func TestTokenBlacklistMiddleware_SkipsInvalidFormat(t *testing.T) {
	router := gin.New()
	router.Use(TokenBlacklistMiddleware(&mockBlacklistChecker{
		blacklisted: map[string]bool{},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-Bearer auth, got %d", w.Code)
	}
}
