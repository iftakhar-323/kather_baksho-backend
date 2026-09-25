package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func WishlistRoutes(router *gin.Engine) {
	g := router.Group("/api/wishlist")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/", controllers.GetWishlist)
		g.POST("/add", controllers.AddToWishlist)
		g.DELETE("/:id", controllers.RemoveFromWishlist)
	}
}
