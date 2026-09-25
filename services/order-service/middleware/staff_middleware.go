package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StaffMiddleware gates fulfillment endpoints: order processing, delivery
// events and return handling. Both "staff" and "admin" pass — admin is a
// superset of staff. Everything that is genuinely admin-only (user roles,
// catalogue, coupons, CMS, analytics, backups) keeps AdminMiddleware.
func StaffMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "staff" && role != "admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Staff access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
