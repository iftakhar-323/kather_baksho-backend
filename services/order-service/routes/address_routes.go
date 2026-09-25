package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// AddressRoutes exposes the address-book endpoints under /api/addresses
// (REST alias for the original /api/auth/addresses group).
func AddressRoutes(r *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	g := r.Group("/api/addresses", proxyMW)
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("", controllers.ListAddresses)
		g.POST("", controllers.CreateAddress)
		g.PUT("/:id", controllers.UpdateAddress)
		g.DELETE("/:id", controllers.DeleteAddress)
	}
}