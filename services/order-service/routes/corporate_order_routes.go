package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CorporateOrderRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	router.POST("/api/corporate/orders", auth, controllers.CreateCorporateOrder)
	router.GET("/api/corporate/orders", auth, controllers.ListCorporateOrders)
	router.GET("/api/corporate/orders/:id", auth, controllers.GetCorporateOrder)
	router.PATCH("/api/corporate/orders/:id/status", auth, admin, controllers.UpdateCorporateOrderStatus)
	router.POST("/api/corporate/quote", controllers.CorporateQuote)
}