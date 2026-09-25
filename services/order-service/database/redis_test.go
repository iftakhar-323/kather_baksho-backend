package database

import (
	"testing"
	"time"
)

func TestRedisGracefulDegradationWhenNil(t *testing.T) {
	// Ensure RedisClient is nil to verify non-panicking degradation
	orig := RedisClient
	RedisClient = nil
	defer func() { RedisClient = orig }()

	if val, ok := CacheGet("test-key"); ok || val != "" {
		t.Errorf("expected empty CacheGet when RedisClient is nil, got: %v, %v", val, ok)
	}

	// None of these should panic
	CacheSet("test-key", "value", time.Minute)
	CacheDelete("test-key")
	InvalidateCachePrefix("test:")

	if err := RedisPing(); err != nil {
		t.Errorf("expected nil error on RedisPing when client is nil, got %v", err)
	}
}
