package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CategoryRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	// Public read
	router.GET("/api/categories", controllers.ListCategories)

	// Admin CRUD
	adm := router.Group("/api/admin/categories", auth, admin)
	adm.POST("", controllers.AdminCreateCategory)
	adm.PUT("/:id", controllers.AdminUpdateCategory)
	adm.DELETE("/:id", controllers.AdminDeleteCategory)
}