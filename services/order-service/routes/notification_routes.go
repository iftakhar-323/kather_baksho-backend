package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func NotificationRoutes(router *gin.Engine) {
	g := router.Group("/api/notifications")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/", controllers.GetNotifications)
		g.POST("/mark-read", controllers.MarkNotificationsRead)
	}
}