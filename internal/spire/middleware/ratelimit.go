package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter tracks per-IP request rates using a token bucket algorithm.
type RateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewRateLimiter creates a RateLimiter.
// r is the refill rate (events per second), b is the burst size.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.ips[ip] = limiter
	}
	return limiter
}

// Middleware returns a Gin middleware that rate-limits by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		// Use X-Forwarded-For if behind a proxy (Gin's ClientIP already handles this,
		// but we also check explicitly for consistency)
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			ip = xff
		}

		if !rl.getLimiter(ip).Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		c.Next()
	}
}

// Pre-configured rate limiters for different endpoint tiers.
var (
	// BootstrapLimiter: 60 requests/minute, burst 20 -- for node attestation, agent renewal, PKI provision, entries lookup.
	// Must accommodate fleet-wide agent restarts (15+ agents x 3 bootstrap calls each).
	BootstrapLimiter = NewRateLimiter(rate.Limit(60.0/60.0), 20)

	// StandardLimiter: 30 requests/minute, burst 10 -- for authenticated CRUD endpoints.
	StandardLimiter = NewRateLimiter(rate.Limit(30.0/60.0), 10)

	// SensitiveLimiter: 10 requests/minute, burst 5 -- for JWT issuance, delegation.
	SensitiveLimiter = NewRateLimiter(rate.Limit(10.0/60.0), 5)
)
