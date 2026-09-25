package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	RedisCtx    = context.Background()
)

// InitRedis connects to Redis if configured.
// Supports REDIS_ADDR (e.g. "redis:6379" or "localhost:6379") or REDIS_HOST/REDIS_PORT.
// Degrades gracefully if Redis is unavailable or unconfigured.
func InitRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		host := os.Getenv("REDIS_HOST")
		if host != "" {
			port := os.Getenv("REDIS_PORT")
			if port == "" {
				port = "6379"
			}
			addr = host + ":" + port
		}
	}

	if addr == "" {
		log.Println("[Cache] Redis not configured (REDIS_ADDR empty). Running with direct DB queries.")
		return
	}

	password := os.Getenv("REDIS_PASSWORD")
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(RedisCtx, 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[Cache] Redis ping failed (%v). Running without Redis cache.", err)
		client.Close()
		return
	}

	RedisClient = client
	log.Printf("[Cache] Connected to Redis successfully at %s", addr)
}

// RedisPing returns ping status for health checks.
func RedisPing() error {
	if RedisClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(RedisCtx, 1*time.Second)
	defer cancel()
	return RedisClient.Ping(ctx).Err()
}

// CacheGet fetches a string value from Redis. Returns false on cache miss or when Redis is absent.
func CacheGet(key string) (string, bool) {
	if RedisClient == nil {
		return "", false
	}
	val, err := RedisClient.Get(RedisCtx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

// CacheSet stores a value with TTL. Fails silently if Redis is offline.
func CacheSet(key string, val interface{}, ttl time.Duration) {
	if RedisClient == nil {
		return
	}
	_ = RedisClient.Set(RedisCtx, key, val, ttl).Err()
}

// CacheDelete removes specific keys.
func CacheDelete(keys ...string) {
	if RedisClient == nil || len(keys) == 0 {
		return
	}
	_ = RedisClient.Del(RedisCtx, keys...).Err()
}

// InvalidateCachePrefix clears all keys starting with prefix (e.g. "products:", "categories:").
func InvalidateCachePrefix(prefix string) {
	if RedisClient == nil {
		return
	}
	iter := RedisClient.Scan(RedisCtx, 0, prefix+"*", 0).Iterator()
	var keys []string
	for iter.Next(RedisCtx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		_ = RedisClient.Del(RedisCtx, keys...).Err()
	}
}
