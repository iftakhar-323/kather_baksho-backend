package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func PaymentRoutes(router *gin.Engine) {
	payments := router.Group("/api/payments")
	{
		// Callback from payment gateway (verified via HMAC signature)
		payments.POST("/callback", controllers.PaymentCallback)
		payments.POST("/webhook", controllers.PaymentCallback)

		// Public simulation endpoints for interactive dev/testing
		payments.GET("/simulate-gateway", controllers.SimulateGenericGateway)
		payments.GET("/bkash/simulate", controllers.SimulateBKashGateway)
		payments.GET("/sslcommerz/simulate", controllers.SimulateSSLCommerzGateway)

		// Authenticated payment initiation and status check
		authenticated := payments.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			authenticated.POST("/initiate", controllers.InitiatePayment)
			authenticated.GET("/status/:order_id", controllers.GetPaymentStatus)
		}
	}
}
