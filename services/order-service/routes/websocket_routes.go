package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

// WebSocketRoutes registers real-time bidirectional WebSocket endpoints.
func WebSocketRoutes(router *gin.Engine) {
	router.GET("/ws/orders/:id/track", controllers.TrackOrderLiveWebSocket)
}
