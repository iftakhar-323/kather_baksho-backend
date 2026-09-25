package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds standard OWASP defense-in-depth headers to every response.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME-sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Mitigate clickjacking attacks
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// Prevent browser reflective XSS
		c.Header("X-XSS-Protection", "1; mode=block")

		// Limit referrer leak across domains
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy for camera/mic
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeLimiter enforces a maximum request body payload to prevent memory exhaustion attacks.
func RequestSizeLimiter(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Payload exceeds maximum allowed size",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
