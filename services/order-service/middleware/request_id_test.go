package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddlewareGenerated(t *testing.T) {
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		reqID, exists := c.Get(RequestIDKey)
		if !exists || reqID == "" {
			t.Errorf("expected request_id in context")
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get(RequestIDHeader)
	if respID == "" {
		t.Fatalf("expected %s header in response", RequestIDHeader)
	}
	if len(respID) < 16 {
		t.Fatalf("unexpected short request id: %s", respID)
	}
}

func TestRequestIDMiddlewarePropagated(t *testing.T) {
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	customID := "custom-trace-uuid-12345"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, customID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get(RequestIDHeader)
	if respID != customID {
		t.Fatalf("expected propagated ID %s, got %s", customID, respID)
	}
}
