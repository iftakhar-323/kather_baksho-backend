package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

// HealthRoutes registers health probe endpoints at root and under /api
func HealthRoutes(router *gin.Engine) {
	// Root probes for docker/k8s/load balancers
	router.GET("/health", controllers.HealthCheck)
	router.GET("/health/live", controllers.LivenessCheck)
	router.GET("/health/ready", controllers.ReadinessCheck)

	// API-prefixed probes
	api := router.Group("/api/health")
	{
		api.GET("", controllers.HealthCheck)
		api.GET("/live", controllers.LivenessCheck)
		api.GET("/ready", controllers.ReadinessCheck)
	}
}
