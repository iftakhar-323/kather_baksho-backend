package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func ReminderRoutes(router *gin.Engine) {
	g := router.Group("/api/reminders")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/", controllers.GetReminders)
		g.POST("/:id/complete", controllers.CompleteReminder)
		g.DELETE("/:id", controllers.DeleteReminder)
	}
}