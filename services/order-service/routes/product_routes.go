package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func ProductRoutes(router *gin.Engine) {
	productGroup := router.Group("/api/products")
	{
		// List + filter + sort
		productGroup.GET("", controllers.GetProducts)
		productGroup.GET("/", controllers.GetProducts)
		productGroup.GET("/autocomplete", controllers.Autocomplete)
		productGroup.GET("/suggest", controllers.SuggestProducts)
		productGroup.GET("/best-selling", controllers.BestSelling)
		productGroup.GET("/most-viewed", controllers.MostViewed)
		productGroup.GET("/brands", controllers.ListBrands)
		productGroup.GET("/categories", controllers.CategoryTree)
		productGroup.GET("/search", controllers.SearchProducts)

		// Single
		productGroup.GET("/:id", controllers.GetProduct)
		productGroup.GET("/slug/:slug", controllers.GetProductBySlug)
		productGroup.GET("/related/:id", controllers.GetRelated)
		productGroup.GET("/fbt/:id", controllers.GetFBT)

		// Tracking
		productGroup.POST("/:id/view", controllers.TrackView)

		// Admin CRUD
		productGroup.POST("", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.CreateProduct)
		productGroup.POST("/", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.CreateProduct)
		productGroup.PUT("/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.UpdateProduct)
		productGroup.DELETE("/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.DeleteProduct)
	}
}
