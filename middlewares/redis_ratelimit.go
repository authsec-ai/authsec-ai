package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisRateLimiter is a distributed sliding-window rate limiter backed by Redis sorted sets.
// Each request is a member scored by its Unix-nanosecond timestamp. On every check the
// limiter trims entries outside the window (ZREMRANGEBYSCORE) and counts the remainder (ZCARD).
// This is safe for multi-replica deployments because Redis serialises all operations.
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter returns a limiter that uses the provided Redis client.
// If client is nil the middleware falls through (fail-open) so the service
// can still run without Redis.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// allow returns true if the request identified by key is within limit/window.
func (rl *RedisRateLimiter) allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	pipe := rl.client.Pipeline()
	// Remove entries outside the sliding window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	// Count remaining entries
	countCmd := pipe.ZCard(ctx, key)
	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	// Set key expiry slightly beyond window to auto-clean idle keys
	pipe.Expire(ctx, key, window+time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return countCmd.Val() < int64(limit), nil
}

// DistributedRateLimitMiddleware is a drop-in replacement for MennovRateLimitMiddleware
// that stores counters in Redis so limits are shared across replicas.
func DistributedRateLimitMiddleware(rl *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fall through if Redis is unavailable — don't block traffic.
		if rl == nil || rl.client == nil {
			c.Next()
			return
		}

		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		cfg := selectRateLimitConfig(path)
		key := fmt.Sprintf("rl:%s:%s", clientIP, cfg.Endpoint)

		allowed, err := rl.allow(c.Request.Context(), key, cfg.Limit, cfg.Window)
		if err != nil {
			// Redis failure — fail open, don't penalise users.
			logrus.WithError(err).Warn("Distributed rate limiter: Redis error, falling through")
			c.Next()
			return
		}

		if !allowed {
			logrus.WithFields(logrus.Fields{
				"request_id": requestID,
				"client_ip":  clientIP,
				"path":       path,
				"method":     c.Request.Method,
				"limit_type": cfg.Endpoint,
				"limit":      cfg.Limit,
				"window":     cfg.Window,
			}).Warn("Distributed rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Maximum %d requests per %v allowed. Please try again later.", cfg.Limit, cfg.Window),
				"limit":   cfg.Limit,
				"window":  cfg.Window.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DistributedStrictRateLimitMiddleware is a stricter per-IP+path variant for
// sensitive auth endpoints, backed by Redis.
func DistributedStrictRateLimitMiddleware(rl *RedisRateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl == nil || rl.client == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		// Normalise path — strip query params and trailing slashes
		if idx := strings.Index(path, "?"); idx != -1 {
			path = path[:idx]
		}
		path = strings.TrimRight(path, "/")

		key := fmt.Sprintf("rl:strict:%s:%s", clientIP, path)

		allowed, err := rl.allow(c.Request.Context(), key, limit, window)
		if err != nil {
			logrus.WithError(err).Warn("Distributed strict rate limiter: Redis error, falling through")
			c.Next()
			return
		}

		if !allowed {
			logrus.WithFields(logrus.Fields{
				"client_ip": clientIP,
				"path":      path,
				"limit":     limit,
				"window":    window,
			}).Warn("Distributed strict rate limit exceeded")

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
