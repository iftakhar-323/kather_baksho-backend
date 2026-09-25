package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

func ChaosRoutes(router *gin.Engine) {
	chaosGroup := router.Group("/api/chaos")
	{
		chaosGroup.GET("/config", controllers.GetChaosConfig)
		chaosGroup.POST("/config", controllers.UpdateChaosConfig)
		chaosGroup.POST("/reset", controllers.ResetChaos)
		chaosGroup.GET("/stats", controllers.GetChaosStats)
		chaosGroup.POST("/trip-breaker", controllers.TripCircuitBreaker)
		chaosGroup.GET("/probe", controllers.ChaosProbe)
	}
}
