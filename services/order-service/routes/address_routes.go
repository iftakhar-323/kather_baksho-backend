package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

// AddressRoutes exposes the address-book endpoints under /api/addresses
// (REST alias for the original /api/auth/addresses group).
func AddressRoutes(r *gin.Engine) {
	g := r.Group("/api/addresses")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("", controllers.ListAddresses)
		g.POST("", controllers.CreateAddress)
		g.PUT("/:id", controllers.UpdateAddress)
		g.DELETE("/:id", controllers.DeleteAddress)
	}
}