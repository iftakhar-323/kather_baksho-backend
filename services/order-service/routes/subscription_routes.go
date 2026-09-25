package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func SubscriptionRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	g := router.Group("/api/subscriptions", proxyMW)
	g.Use(middleware.AuthMiddleware())
	{
		g.POST("", controllers.CreateSubscription)
		g.POST("/", controllers.CreateSubscription)
		g.GET("", controllers.GetMySubscriptions)
		g.GET("/", controllers.GetMySubscriptions)
		g.POST("/:id/cancel", controllers.CancelSubscription)
		g.POST("/:id/advance", controllers.AdvanceSubscription)
		g.POST("/:id/pause", controllers.PauseSubscription)
		g.POST("/:id/resume", controllers.ResumeSubscription)
		g.POST("/:id/renew", controllers.RenewSubscription)
		g.GET("/:id/deliveries", controllers.ListSubscriptionDeliveries)
	}
}
