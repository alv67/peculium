// Package cache provides a JSON cache on top of Redis with a global data
// revision counter used to invalidate entries when data changes.
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const revKey = "pc:data_rev"

// Cache wraps a Redis client. With a nil client every operation degrades to a
// noop so the application keeps working without Redis.
type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// GetJSON loads key into dst and returns true on a hit. A missing key, a nil
// client or a Redis error is reported as a miss.
func (c *Cache) GetJSON(ctx context.Context, key string, dst any) (bool, error) {
	if c.rdb == nil {
		return false, nil
	}
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON marshals v and stores it under key with the given TTL.
func (c *Cache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	if c.rdb == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Rev returns the current global data revision, 0 when absent.
func (c *Cache) Rev(ctx context.Context) (int64, error) {
	if c.rdb == nil {
		return 0, nil
	}
	n, err := c.rdb.Get(ctx, revKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}

// Bump increments the global data revision and returns the new value.
func (c *Cache) Bump(ctx context.Context) (int64, error) {
	if c.rdb == nil {
		return 0, nil
	}
	return c.rdb.Incr(ctx, revKey).Result()
}
