package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func BackupRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	router.GET("/api/admin/backup/db", auth, admin, controllers.BackupDB)
	router.POST("/api/admin/backup/restore", auth, admin, controllers.RestoreDB)
	router.GET("/api/admin/backup/stats", auth, admin, controllers.DBStats)
}