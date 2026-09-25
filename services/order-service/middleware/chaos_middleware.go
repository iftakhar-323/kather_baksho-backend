package middleware

import (
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

type ChaosConfig struct {
	Enabled          bool     `json:"enabled"`
	LatencyMs        int      `json:"latency_ms"`         // Injected delay in ms (e.g. 100-2000ms)
	ErrorRatePercent int      `json:"error_rate_percent"` // Injected failure percentage (0-100)
	TargetPrefixes   []string `json:"target_prefixes"`    // Specific routes, e.g. ["/api/orders"] or empty for all /api/
	FailureCode      int      `json:"failure_code"`       // HTTP status code to inject (500 or 503)
}

type ChaosStats struct {
	TotalRequests    int64             `json:"total_requests"`
	DelayedRequests  int64             `json:"delayed_requests"`
	InjectedFailures int64             `json:"injected_failures"`
	CircuitBreakers  map[string]string `json:"circuit_breakers"`
}

var (
	chaosMu     sync.RWMutex
	chaosConfig = ChaosConfig{
		Enabled:          false,
		LatencyMs:        0,
		ErrorRatePercent: 0,
		TargetPrefixes:   []string{},
		FailureCode:      http.StatusServiceUnavailable,
	}

	totalRequestsAtomic    int64
	delayedRequestsAtomic  int64
	injectedFailuresAtomic int64
)

// GetChaosConfig returns current active chaos settings
func GetChaosConfig() ChaosConfig {
	chaosMu.RLock()
	defer chaosMu.RUnlock()
	return chaosConfig
}

// SetChaosConfig updates chaos settings safely at runtime
func SetChaosConfig(cfg ChaosConfig) {
	chaosMu.Lock()
	defer chaosMu.Unlock()
	if cfg.FailureCode <= 0 {
		cfg.FailureCode = http.StatusServiceUnavailable
	}
	chaosConfig = cfg
}

// ResetChaos resets all chaos rules and clears failure counters
func ResetChaos() {
	chaosMu.Lock()
	chaosConfig = ChaosConfig{
		Enabled:          false,
		LatencyMs:        0,
		ErrorRatePercent: 0,
		TargetPrefixes:   []string{},
		FailureCode:      http.StatusServiceUnavailable,
	}
	chaosMu.Unlock()

	atomic.StoreInt64(&totalRequestsAtomic, 0)
	atomic.StoreInt64(&delayedRequestsAtomic, 0)
	atomic.StoreInt64(&injectedFailuresAtomic, 0)

	// Reset any tripped circuit breakers
	utils.ResetAllCircuitBreakers()
}

// GetChaosStats returns live stats on injected faults
func GetChaosStats() ChaosStats {
	return ChaosStats{
		TotalRequests:    atomic.LoadInt64(&totalRequestsAtomic),
		DelayedRequests:  atomic.LoadInt64(&delayedRequestsAtomic),
		InjectedFailures: atomic.LoadInt64(&injectedFailuresAtomic),
		CircuitBreakers:  utils.GetAllCircuitBreakers(),
	}
}

// ChaosMiddleware intercepts requests to simulate latency, network jitter, and service outages
func ChaosMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Never disrupt chaos management control endpoints or vital system health telemetry
		if path == "/api/chaos/config" ||
			path == "/api/chaos/reset" ||
			path == "/api/chaos/stats" ||
			path == "/api/chaos/trip-breaker" ||
			strings.HasPrefix(path, "/health") ||
			strings.HasPrefix(path, "/metrics") ||
			strings.HasPrefix(path, "/api/docs") {
			c.Next()
			return
		}

		chaosMu.RLock()
		cfg := chaosConfig
		chaosMu.RUnlock()

		if !cfg.Enabled {
			c.Next()
			return
		}

		// Filter by target prefixes if configured
		if len(cfg.TargetPrefixes) > 0 {
			matched := false
			for _, prefix := range cfg.TargetPrefixes {
				if strings.HasPrefix(path, prefix) {
					matched = true
					break
				}
			}
			if !matched {
				c.Next()
				return
			}
		}

		atomic.AddInt64(&totalRequestsAtomic, 1)

		// 1. Injected Latency / Jitter
		if cfg.LatencyMs > 0 {
			atomic.AddInt64(&delayedRequestsAtomic, 1)
			time.Sleep(time.Duration(cfg.LatencyMs) * time.Millisecond)
		}

		// 2. Injected Failures / Packet Drops / 5xx responses
		if cfg.ErrorRatePercent > 0 {
			roll := rand.Intn(100)
			if roll < cfg.ErrorRatePercent {
				atomic.AddInt64(&injectedFailuresAtomic, 1)
				c.AbortWithStatusJSON(cfg.FailureCode, gin.H{
					"error":       "Chaos Engineering: Simulated infrastructure fault / dependency outage",
					"chaos_fault": true,
					"endpoint":    path,
					"status":      cfg.FailureCode,
				})
				return
			}
		}

		c.Next()
	}
}
