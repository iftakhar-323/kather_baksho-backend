package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// ReviewRoutes mounts review endpoints under /api.
func ReviewRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	auth := middleware.AuthMiddleware()

	// Public read — reviews live under the product scope
	router.GET("/api/products/:id/reviews", proxyMW, controllers.ListProductReviews)

	// Logged-in users can post / edit / delete their own reviews
	router.POST("/api/products/:id/reviews", proxyMW, auth, controllers.CreateReview)
	router.PUT("/api/reviews/:id", proxyMW, auth, controllers.UpdateReview)
	router.DELETE("/api/reviews/:id", proxyMW, auth, controllers.DeleteReview)

	// Current user's own reviews (for Profile section)
	router.GET("/api/reviews/mine", proxyMW, auth, controllers.MyReviews)
}
