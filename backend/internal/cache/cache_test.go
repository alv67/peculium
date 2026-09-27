package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNoopClientDegrades(t *testing.T) {
	c := New(nil)
	ctx := context.Background()

	var v map[string]any
	if hit, err := c.GetJSON(ctx, "pc:nope", &v); err != nil || hit {
		t.Fatalf("expected miss without error, got hit=%v err=%v", hit, err)
	}
	if err := c.SetJSON(ctx, "pc:nope", map[string]any{"a": 1}, time.Minute); err != nil {
		t.Fatalf("expected noop set, got err=%v", err)
	}
	if rev, err := c.Rev(ctx); err != nil || rev != 0 {
		t.Fatalf("expected rev 0, got rev=%d err=%v", rev, err)
	}
	if rev, err := c.Bump(ctx); err != nil || rev != 0 {
		t.Fatalf("expected bump 0, got rev=%d err=%v", rev, err)
	}
}

func TestRedisRoundTrip(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer rdb.Close()

	c := New(rdb)
	key := "pc:test:roundtrip"
	rdb.Del(ctx, key, revKey)
	defer rdb.Del(ctx, key)

	if err := c.SetJSON(ctx, key, map[string]int{"n": 42}, time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	var got map[string]int
	hit, err := c.GetJSON(ctx, key, &got)
	if err != nil || !hit {
		t.Fatalf("expected hit, got hit=%v err=%v", hit, err)
	}
	if got["n"] != 42 {
		t.Fatalf("expected 42, got %d", got["n"])
	}

	rdb.Del(ctx, key)
	hit, err = c.GetJSON(ctx, key, &got)
	if err != nil || hit {
		t.Fatalf("expected miss after delete, got hit=%v err=%v", hit, err)
	}
}

func TestRedisRevAndBump(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer rdb.Close()

	c := New(rdb)
	rdb.Del(ctx, revKey)
	defer rdb.Del(ctx, revKey)

	if rev, err := c.Rev(ctx); err != nil || rev != 0 {
		t.Fatalf("expected rev 0, got rev=%d err=%v", rev, err)
	}
	if rev, err := c.Bump(ctx); err != nil || rev != 1 {
		t.Fatalf("expected rev 1, got rev=%d err=%v", rev, err)
	}
	if rev, err := c.Bump(ctx); err != nil || rev != 2 {
		t.Fatalf("expected rev 2, got rev=%d err=%v", rev, err)
	}
	if rev, err := c.Rev(ctx); err != nil || rev != 2 {
		t.Fatalf("expected rev 2, got rev=%d err=%v", rev, err)
	}
}
