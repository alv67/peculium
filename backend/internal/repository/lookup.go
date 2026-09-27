package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type LookupRepository interface {
	Get(ctx context.Context, query string) ([]byte, error)
	Set(ctx context.Context, query string, results []byte, ttl time.Duration) error
}

type lookupRepo struct {
	rdb *redis.Client
}

// NewLookupCache returns a LookupRepository backed by Redis. With a nil client
// reads degrade to a miss and writes are noops.
func NewLookupCache(rdb *redis.Client) LookupRepository {
	return &lookupRepo{rdb: rdb}
}

func (r *lookupRepo) Get(ctx context.Context, query string) ([]byte, error) {
	if r.rdb == nil {
		return nil, redis.Nil
	}
	return r.rdb.Get(ctx, "pc:lookup:"+query).Bytes()
}

func (r *lookupRepo) Set(ctx context.Context, query string, results []byte, ttl time.Duration) error {
	if r.rdb == nil {
		return nil
	}
	return r.rdb.Set(ctx, "pc:lookup:"+query, results, ttl).Err()
}
