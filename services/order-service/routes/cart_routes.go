package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CartRoutes(router *gin.Engine) {
	cartGroup := router.Group("/api/cart")
	cartGroup.Use(middleware.AuthMiddleware())
	{
		cartGroup.GET("/", controllers.GetCart)
		cartGroup.POST("/add", controllers.AddToCart)
		cartGroup.PUT("/item/:id", controllers.UpdateCartItem)
		cartGroup.DELETE("/item/:id", controllers.RemoveCartItem)
	}
}
