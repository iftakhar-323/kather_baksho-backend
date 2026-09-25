package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func OrderExtensionRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()

	// Events (timeline)
	router.GET("/api/orders/:id/events", auth, controllers.ListOrderEvents)
	router.POST("/api/orders/:id/events", auth, controllers.AddOrderEvent)
	router.POST("/api/orders/:id/events/admin", auth, middleware.StaffMiddleware(), controllers.AddOrderEvent)

	// Invoice + receipt (HTML & TypeScript PDF Microservice)
	router.GET("/api/orders/:id/invoice", auth, controllers.InvoiceHTML)
	router.GET("/api/orders/:id/invoice/pdf", auth, controllers.InvoicePDF)
	router.GET("/api/orders/:id/receipt", auth, controllers.ReceiptHTML)

	// Returns / Refunds / Exchanges
	router.POST("/api/returns", auth, controllers.CreateReturnRequest)
	router.GET("/api/returns", auth, controllers.ListReturns)
	router.PATCH("/api/returns/:id", auth, middleware.StaffMiddleware(), controllers.UpdateReturnRequest)

	// Estimated delivery
	router.GET("/api/orders/:id/estimated-delivery", auth, controllers.EstimatedDelivery)
}
