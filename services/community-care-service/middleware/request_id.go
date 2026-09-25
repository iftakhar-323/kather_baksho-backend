package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDMiddleware injects X-Request-ID and propagates X-Correlation-ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Header("X-Request-ID", reqID)
		c.Set("request_id", reqID)

		corrID := c.GetHeader("X-Correlation-ID")
		if corrID == "" {
			corrID = reqID
		}
		c.Header("X-Correlation-ID", corrID)
		c.Set("correlation_id", corrID)

		c.Next()
	}
}
