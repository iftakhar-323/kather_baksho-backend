package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// ConnectRedis initializes connection to Redis for pub/sub and stream alerts.
func ConnectRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "redis"
		}
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}
		addr = host + ":" + port
	}

	password := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[IoT-AI-Service:Redis] Warning: Redis connection failed at %s: %v", addr, err)
		return
	}

	RedisClient = client
	log.Printf("[IoT-AI-Service:Redis] Connected to Redis at %s", addr)
}
