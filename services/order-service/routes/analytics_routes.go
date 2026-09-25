package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func AnalyticsRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	router.GET("/api/analytics/summary", auth, admin, controllers.AnalyticsSummary)
	router.GET("/api/analytics/report/pdf", auth, admin, controllers.AnalyticsReportPDF)
	router.GET("/api/analytics/top-customers", auth, admin, controllers.TopCustomers)
	router.GET("/api/analytics/inventory", auth, admin, controllers.InventoryReport)
	router.GET("/api/analytics/inventory-forecast", auth, admin, controllers.InventoryForecast)
	router.GET("/api/analytics/traffic", auth, admin, controllers.TrafficReport)
	router.GET("/api/analytics/categories", auth, admin, controllers.CategoryRevenue)
}
