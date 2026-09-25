package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CSVRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	router.GET("/api/admin/products/export", auth, admin, controllers.ExportProductsCSV)
	router.POST("/api/admin/products/import", auth, admin, controllers.ImportProductsCSV)
}