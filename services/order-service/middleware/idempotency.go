package middleware

import (
	"bytes"
	"net/http"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware ensures that requests with an Idempotency-Key header are executed at most once.
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			key = c.GetHeader("X-Idempotency-Key")
		}
		if key == "" {
			c.Next()
			return
		}

		userID := c.GetUint("user_id")
		endpoint := c.Request.Method + " " + c.FullPath()

		var existing models.IdempotencyRecord
		err := database.DB.Where("key = ?", key).First(&existing).Error
		if err == nil {
			// Found existing record
			if existing.Status == "COMPLETED" {
				c.Header("X-Cache-Lookup", "HIT")
				c.Header("X-Idempotency-Key", key)
				c.Data(existing.StatusCode, "application/json; charset=utf-8", []byte(existing.ResponseBody))
				c.Abort()
				return
			}

			if existing.Status == "PROCESSING" {
				// If within 30s, treat as active in-flight duplicate
				if time.Since(existing.CreatedAt) < 30*time.Second {
					c.Header("Retry-After", "2")
					c.AbortWithStatusJSON(http.StatusConflict, gin.H{
						"error": "A request with this idempotency key is currently being processed",
					})
					return
				}
			}
		}

		// Insert or mark PROCESSING
		record := models.IdempotencyRecord{
			Key:       key,
			UserID:    userID,
			Endpoint:  endpoint,
			Status:    "PROCESSING",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		database.DB.Save(&record)

		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		statusCode := c.Writer.Status()
		status := "COMPLETED"
		if statusCode >= 500 {
			status = "FAILED"
		}

		database.DB.Model(&models.IdempotencyRecord{}).Where("key = ?", key).Updates(map[string]interface{}{
			"status":        status,
			"status_code":   statusCode,
			"response_body": w.body.String(),
			"updated_at":    time.Now(),
		})
	}
}
