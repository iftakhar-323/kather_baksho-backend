package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

// EventRoutes mounts Redis Streams event queue endpoints
func EventRoutes(r *gin.Engine) {
	group := r.Group("/api/events")
	{
		group.GET("/stats", controllers.GetEventBusStats)
		group.POST("/publish", controllers.PublishTestEvent)
	}
}
