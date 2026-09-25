package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPrometheusMetricsExposition(t *testing.T) {
	r := gin.New()
	r.Use(PrometheusMetricsMiddleware())
	r.GET("/api/test-metric", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/metrics", PrometheusHandler())

	// Perform a request to generate metrics
	req1 := httptest.NewRequest("GET", "/api/test-metric", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// Scrape metrics
	req2 := httptest.NewRequest("GET", "/metrics", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for /metrics, got %d", w2.Code)
	}

	body := w2.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("expected metrics output to contain 'http_requests_total', got: %s", body)
	}
	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Fatalf("expected metrics output to contain 'http_request_duration_seconds', got: %s", body)
	}
}
