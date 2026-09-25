package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

// ReviewRoutes mounts review endpoints under /api.
func ReviewRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()

	// Public read — reviews live under the product scope
	router.GET("/api/products/:id/reviews", controllers.ListProductReviews)

	// Logged-in users can post / edit / delete their own reviews
	router.POST("/api/products/:id/reviews", auth, controllers.CreateReview)
	router.PUT("/api/reviews/:id", auth, controllers.UpdateReview)
	router.DELETE("/api/reviews/:id", auth, controllers.DeleteReview)

	// Current user's own reviews (for Profile section)
	router.GET("/api/reviews/mine", auth, controllers.MyReviews)
}
