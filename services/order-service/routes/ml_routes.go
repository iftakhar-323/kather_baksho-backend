package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

func MLRoutes(router *gin.Engine) {
	ml := router.Group("/api/ml")
	{
		ml.POST("/compare", controllers.CompareModels)
		ml.GET("/compare", controllers.CompareModels)
	}
}
