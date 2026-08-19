// Package api implements the Go API server router and handlers.
package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager manages cached stable public content using Valkey (Redis-compatible).
type CacheManager struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewCacheManager initializes a CacheManager. If addr is empty, caching is disabled.
// A nil logger falls back to slog.Default.
func NewCacheManager(addr string, logger *slog.Logger) *CacheManager {
	if logger == nil {
		logger = slog.Default()
	}
	if addr == "" {
		logger.Info("cache disabled: empty address")
		return &CacheManager{logger: logger}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	logger.Info("cache initialized", "addr", addr)
	return &CacheManager{rdb: rdb, logger: logger}
}

// Get retrieves a string value from the cache.
func (c *CacheManager) Get(ctx context.Context, key string) (string, error) {
	if c.rdb == nil {
		return "", redis.Nil
	}
	return c.rdb.Get(ctx, key).Result()
}

// Set stores a string value in the cache with a 24-hour expiration.
func (c *CacheManager) Set(ctx context.Context, key string, val string) error {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Set(ctx, key, val, 24*time.Hour).Err()
}

// InvalidateAll clears all public cached endpoints.
func (c *CacheManager) InvalidateAll(ctx context.Context) {
	if c.rdb == nil {
		return
	}
	c.logger.Info("invalidating public cache")
	keys, err := c.rdb.Keys(ctx, "felicia:public:*").Result()
	if err != nil {
		c.logger.Error("cache invalidate: list keys failed", "err", err)
		return
	}
	if len(keys) > 0 {
		if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
			c.logger.Error("cache invalidate: delete keys failed", "err", err)
		} else {
			c.logger.Info("cache invalidated", "keys", len(keys))
		}
	}
}
