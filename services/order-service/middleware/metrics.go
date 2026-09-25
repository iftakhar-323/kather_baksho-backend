package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by KatherBox backend.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies in seconds.",
			Buckets: []float64{0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests.",
		},
	)
)

// PrometheusMetricsMiddleware records request counts, latencies, and in-flight counts.
func PrometheusMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ignore metric scrape path from metrics recording to prevent recursion
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(c.Writer.Status())

		// Use route pattern (matched path) if available to avoid high cardinality
		pattern := c.FullPath()
		if pattern == "" {
			pattern = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, pattern, statusStr).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, pattern, statusStr).Observe(duration)
	}
}

// PrometheusHandler exposes the standard Prometheus metrics scrape endpoint.
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
