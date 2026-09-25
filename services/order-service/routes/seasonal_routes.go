package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

func SeasonalRoutes(router *gin.Engine) {
	g := router.Group("/api/seasonal-guide")
	{
		g.GET("/", controllers.SeasonalGuide)
	}
}