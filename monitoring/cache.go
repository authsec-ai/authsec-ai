package monitoring

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// CacheManager handles Redis caching operations
type CacheManager struct {
	client *redis.Client
	logger *logrus.Logger
	ctx    context.Context
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Key        string        `json:"key"`
	Value      interface{}   `json:"value"`
	TTL        time.Duration `json:"ttl"`
	CreatedAt  time.Time     `json:"created_at"`
	AccessedAt time.Time     `json:"accessed_at"`
}

// NewCacheManager creates a new Redis cache manager
func NewCacheManager(redisURL string) (*CacheManager, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &CacheManager{
		client: client,
		logger: GetLogger(),
		ctx:    ctx,
	}, nil
}

// scanKeys uses Redis SCAN (non-blocking) instead of KEYS to find matching keys
func (cm *CacheManager) scanKeys(pattern string) ([]string, error) {
	var allKeys []string
	var cursor uint64
	for {
		keys, nextCursor, err := cm.client.Scan(cm.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return allKeys, nil
}

// Set stores a value in cache with TTL
func (cm *CacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = cm.client.Set(cm.ctx, key, data, ttl).Err()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to set cache value")
		return err
	}

	cm.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl.String(),
	}).Debug("Cache value set")

	return nil
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(key string, dest interface{}) (bool, error) {
	data, err := cm.client.Get(cm.ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Key not found
	}
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to get cache value")
		return false, err
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to unmarshal cache value")
		return false, err
	}

	cm.logger.WithFields(logrus.Fields{
		"key": key,
	}).Debug("Cache value retrieved")

	return true, nil
}

// Delete removes a key from cache
func (cm *CacheManager) Delete(key string) error {
	err := cm.client.Del(cm.ctx, key).Err()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to delete cache value")
		return err
	}

	cm.logger.WithFields(logrus.Fields{
		"key": key,
	}).Debug("Cache value deleted")

	return nil
}

// Exists checks if a key exists in cache
func (cm *CacheManager) Exists(key string) (bool, error) {
	count, err := cm.client.Exists(cm.ctx, key).Result()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to check cache existence")
		return false, err
	}

	exists := count > 0
	cm.logger.WithFields(logrus.Fields{
		"key":    key,
		"exists": exists,
	}).Debug("Cache existence checked")

	return exists, nil
}

// Expire sets expiration time for a key
func (cm *CacheManager) Expire(key string, ttl time.Duration) error {
	err := cm.client.Expire(cm.ctx, key, ttl).Err()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to set cache expiration")
		return err
	}

	cm.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl.String(),
	}).Debug("Cache expiration set")

	return nil
}

// GetTTL returns the remaining TTL for a key
func (cm *CacheManager) GetTTL(key string) (time.Duration, error) {
	ttl, err := cm.client.TTL(cm.ctx, key).Result()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to get cache TTL")
		return 0, err
	}

	return ttl, nil
}

// Increment increments a numeric value in cache
func (cm *CacheManager) Increment(key string) (int64, error) {
	value, err := cm.client.Incr(cm.ctx, key).Result()
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"key":   key,
			"error": err.Error(),
		}).Error("Failed to increment cache value")
		return 0, err
	}

	cm.logger.WithFields(logrus.Fields{
		"key":   key,
		"value": value,
	}).Debug("Cache value incremented")

	return value, nil
}

// HealthCheck performs a health check on the Redis connection
func (cm *CacheManager) HealthCheck() error {
	return cm.client.Ping(cm.ctx).Err()
}

// CacheTenantConfig caches tenant configuration
func (cm *CacheManager) CacheTenantConfig(tenantID string, config interface{}) error {
	key := "tenant:config:" + tenantID
	return cm.Set(key, config, 30*time.Minute) // Cache for 30 minutes
}

// GetTenantConfig retrieves cached tenant configuration
func (cm *CacheManager) GetTenantConfig(tenantID string, dest interface{}) (bool, error) {
	key := "tenant:config:" + tenantID
	return cm.Get(key, dest)
}

// CacheUserPermissions caches user permissions
func (cm *CacheManager) CacheUserPermissions(tenantID, userID string, permissions interface{}) error {
	key := "user:permissions:" + tenantID + ":" + userID
	return cm.Set(key, permissions, 15*time.Minute) // Cache for 15 minutes
}

// GetUserPermissions retrieves cached user permissions
func (cm *CacheManager) GetUserPermissions(tenantID, userID string, dest interface{}) (bool, error) {
	key := "user:permissions:" + tenantID + ":" + userID
	return cm.Get(key, dest)
}

// CacheAuthToken caches authentication token validation results
func (cm *CacheManager) CacheAuthToken(token string, claims interface{}, ttl time.Duration) error {
	key := "auth:token:" + token
	return cm.Set(key, claims, ttl)
}

// GetAuthToken retrieves cached authentication token
func (cm *CacheManager) GetAuthToken(token string, dest interface{}) (bool, error) {
	key := "auth:token:" + token
	return cm.Get(key, dest)
}

// InvalidateTenantCache invalidates all cache entries for a tenant
func (cm *CacheManager) InvalidateTenantCache(tenantID string) error {
	pattern := "tenant:*:" + tenantID
	keys, err := cm.scanKeys(pattern)
	if err != nil {
		cm.logger.WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to find tenant cache keys")
		return err
	}

	if len(keys) > 0 {
		err = cm.client.Del(cm.ctx, keys...).Err()
		if err != nil {
			cm.logger.WithFields(logrus.Fields{
				"tenant_id": tenantID,
				"keys":      keys,
				"error":     err.Error(),
			}).Error("Failed to delete tenant cache keys")
			return err
		}

		cm.logger.WithFields(logrus.Fields{
			"tenant_id":    tenantID,
			"keys_deleted": len(keys),
		}).Info("Tenant cache invalidated")
	}

	return nil
}

// InvalidateUserCache invalidates all cache entries for a user
func (cm *CacheManager) InvalidateUserCache(tenantID, userID string) error {
	patterns := []string{
		"user:permissions:" + tenantID + ":" + userID,
		"auth:token:*", // This is broad, but safer than complex pattern matching
	}

	for _, pattern := range patterns {
		keys, err := cm.scanKeys(pattern)
		if err != nil {
			continue // Skip errors for individual patterns
		}

		if len(keys) > 0 {
			cm.client.Del(cm.ctx, keys...)
		}
	}

	cm.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Info("User cache invalidated")

	return nil
}
