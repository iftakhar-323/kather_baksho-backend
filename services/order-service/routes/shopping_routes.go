package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func ShoppingRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	// Guest checkout (no auth)
	router.POST("/api/guest-checkout", controllers.GuestCheckout)

	// Quote
	router.GET("/api/quote", controllers.Quote)
	router.POST("/api/quote", controllers.Quote)

	// Gift cards
	router.POST("/api/gift-cards", auth, admin, controllers.CreateGiftCard)
	router.GET("/api/gift-cards", auth, admin, controllers.ListGiftCards)
	router.POST("/api/gift-cards/redeem", auth, controllers.RedeemGiftCard)
	router.GET("/api/gift-cards/balance/:code", controllers.GiftCardBalance)

	// Shipping + Tax rules
	router.GET("/api/shipping-rules", controllers.ListShippingRules)
	router.POST("/api/shipping-rules", auth, admin, controllers.CreateShippingRule)
	router.GET("/api/tax-rules", controllers.ListTaxRules)
	router.POST("/api/tax-rules", auth, admin, controllers.CreateTaxRule)
}