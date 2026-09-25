package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogEntry represents a JSON log record for an HTTP request.
type StructuredLogEntry struct {
	Timestamp string  `json:"timestamp"`
	Level     string  `json:"level"`
	RequestID string  `json:"request_id,omitempty"`
	Method    string  `json:"method"`
	Path      string  `json:"path"`
	Status    int     `json:"status"`
	LatencyMS float64 `json:"latency_ms"`
	ClientIP  string  `json:"client_ip"`
	UserAgent string  `json:"user_agent,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// StructuredLogger emits structured JSON log entries for each completed HTTP request.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		reqID, _ := c.Get(RequestIDKey)
		reqIDStr, _ := reqID.(string)

		level := "INFO"
		if status >= 500 {
			level = "ERROR"
		} else if status >= 400 {
			level = "WARN"
		}

		var errMsg string
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		entry := StructuredLogEntry{
			Timestamp: start.UTC().Format(time.RFC3339Nano),
			Level:     level,
			RequestID: reqIDStr,
			Method:    c.Request.Method,
			Path:      path,
			Status:    status,
			LatencyMS: float64(latency.Microseconds()) / 1000.0,
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Error:     errMsg,
		}

		data, err := json.Marshal(entry)
		if err == nil {
			fmt.Fprintln(os.Stdout, string(data))
		}
	}
}
