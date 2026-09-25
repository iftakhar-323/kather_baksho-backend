package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"
const CorrelationIDHeader = "X-Correlation-ID"
const RequestIDKey = "request_id"

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "req-fallback-id"
	}
	return hex.EncodeToString(b)
}

// RequestIDMiddleware extracts or injects a unique X-Request-ID and X-Correlation-ID.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		corrID := c.GetHeader(CorrelationIDHeader)

		if reqID == "" && corrID != "" {
			reqID = corrID
		} else if reqID == "" {
			reqID = generateID()
		}

		if corrID == "" {
			corrID = reqID
		}

		c.Set(RequestIDKey, reqID)
		c.Set("correlation_id", corrID)
		c.Writer.Header().Set(RequestIDHeader, reqID)
		c.Writer.Header().Set(CorrelationIDHeader, corrID)
		c.Next()
	}
}
