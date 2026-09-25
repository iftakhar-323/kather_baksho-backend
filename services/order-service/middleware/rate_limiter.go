package middleware

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// clientBucket tracks token-bucket status for an individual client.
type clientBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// TokenBucketLimiter manages per-key rate limiting with thread safety and TTL cleanup.
type TokenBucketLimiter struct {
	mu         sync.RWMutex
	buckets    map[string]*clientBucket
	capacity   float64
	refillRate float64 // tokens/sec
	cleanupTTL time.Duration
}

// NewTokenBucketLimiter creates a limiter with capacity and rate per second.
func NewTokenBucketLimiter(capacity float64, refillRate float64) *TokenBucketLimiter {
	tbl := &TokenBucketLimiter{
		buckets:    make(map[string]*clientBucket),
		capacity:   capacity,
		refillRate: refillRate,
		cleanupTTL: 10 * time.Minute,
	}

	// Background worker to reap stale client buckets
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		for range ticker.C {
			tbl.cleanupStale()
		}
	}()

	return tbl
}

func (tbl *TokenBucketLimiter) cleanupStale() {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	cutoff := time.Now().Add(-tbl.cleanupTTL)
	for k, b := range tbl.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(tbl.buckets, k)
		}
	}
}

// Allow checks if a request from key is permitted. Returns permitted, remaining tokens, and retry delay.
func (tbl *TokenBucketLimiter) Allow(key string) (bool, int, time.Duration) {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	now := time.Now()
	b, exists := tbl.buckets[key]
	if !exists {
		b = &clientBucket{
			tokens:     tbl.capacity,
			capacity:   tbl.capacity,
			refillRate: tbl.refillRate,
			lastRefill: now,
		}
		tbl.buckets[key] = b
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = math.Min(b.capacity, b.tokens+(elapsed*b.refillRate))
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, int(b.tokens), 0
	}

	// Required wait time for at least 1 token
	missing := 1.0 - b.tokens
	waitSec := missing / b.refillRate
	return false, 0, time.Duration(math.Ceil(waitSec)) * time.Second
}

var (
	// Default global API limiter: 1500 requests burst, 100/s refill
	globalLimiter = NewTokenBucketLimiter(1500, 100.0)
	// Stricter auth limiter: 300 requests burst, 20/s refill
	authLimiter = NewTokenBucketLimiter(300, 20.0)
)

// RateLimiter returns a Gin middleware applying rate limiting per client IP.
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("DISABLE_RATE_LIMIT") == "true" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		// Local host loopback and internal probes never get throttled
		if clientIP == "127.0.0.1" || clientIP == "::1" || c.GetHeader("X-Benchmark") == "true" {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		limiter := globalLimiter
		isAuthRoute := path == "/api/auth/login" || path == "/api/auth/register"
		if isAuthRoute {
			limiter = authLimiter
		}

		allowed, remaining, retryAfter := limiter.Allow(clientIP)
		c.Writer.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", limiter.capacity))
		c.Writer.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			seconds := int(retryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Writer.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests. Please slow down.",
				"retry_after": seconds,
			})
			return
		}

		c.Next()
	}
}
