package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTokenBucketLimiterAllow(t *testing.T) {
	// Limiter with capacity 3, refill 1 token/sec
	limiter := NewTokenBucketLimiter(3, 1.0)
	key := "client-ip-1"

	for i := 0; i < 3; i++ {
		allowed, remaining, _ := limiter.Allow(key)
		if !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
		expectedRemaining := 2 - i
		if remaining != expectedRemaining {
			t.Errorf("request %d: expected remaining %d, got %d", i+1, expectedRemaining, remaining)
		}
	}

	// 4th request must be rejected (exhausted)
	allowed, _, retryAfter := limiter.Allow(key)
	if allowed {
		t.Fatalf("expected 4th request to be denied")
	}
	if retryAfter <= 0 {
		t.Errorf("expected positive retryAfter, got %v", retryAfter)
	}

	// Wait for refill of 1 token
	time.Sleep(1100 * time.Millisecond)
	allowed, _, _ = limiter.Allow(key)
	if !allowed {
		t.Fatalf("expected request to be allowed after token refill")
	}
}

func TestRateLimiterMiddlewareHTTP(t *testing.T) {
	r := gin.New()
	r.Use(RateLimiter())
	r.GET("/api/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest("GET", "/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Errorf("expected X-RateLimit-Limit header")
	}
}
