package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

// TelemetryRoutes registers IoT plant sensor & telemetry endpoints.
func TelemetryRoutes(router *gin.Engine) {
	// Public / authenticated IoT monitoring routes
	iot := router.Group("/api/iot")
	{
		iot.GET("/plants", controllers.GetMonitoredPlants)
		iot.GET("/telemetry/:plant_id", controllers.GetPlantTelemetryHistory)
		iot.POST("/telemetry", controllers.IngestPlantTelemetry)
		iot.POST("/seed", middleware.AuthMiddleware(), middleware.StaffMiddleware(), controllers.SeedIoTTelemetry)
	}
}
