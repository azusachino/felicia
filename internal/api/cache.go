// Package api implements the Go API server router and handlers.
package api

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager manages cached stable public content using Valkey (Redis-compatible).
type CacheManager struct {
	rdb *redis.Client
}

// NewCacheManager initializes a CacheManager. If addr is empty, caching is disabled.
func NewCacheManager(addr string) *CacheManager {
	if addr == "" {
		log.Println("Cache address is empty. Caching is disabled.")
		return &CacheManager{}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	log.Printf("Valkey cache initialized on address: %s", addr)
	return &CacheManager{rdb: rdb}
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
	log.Println("Invalidating all public cached API endpoints...")
	keys, err := c.rdb.Keys(ctx, "felicia:public:*").Result()
	if err != nil {
		log.Printf("Error searching cache keys to invalidate: %v", err)
		return
	}
	if len(keys) > 0 {
		if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
			log.Printf("Error deleting cache keys: %v", err)
		} else {
			log.Printf("Successfully invalidated %d cache keys.", len(keys))
		}
	}
}
