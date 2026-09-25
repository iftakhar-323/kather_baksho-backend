package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

func GiftRoutes(router *gin.Engine) {
	g := router.Group("/api/gifts")
	{
		g.GET("/recommend", controllers.RecommendGifts)
	}
}