package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CouponRoutes(router *gin.Engine) {
	// public-ish apply (just for logged in users)
	g := router.Group("/api/coupons")
	g.Use(middleware.AuthMiddleware())
	{
		g.POST("/apply", controllers.ApplyCoupon)
	}

	// admin create
        admin := router.Group("/api/admin")
        admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
        admin.POST("/coupons", controllers.CreateCoupon)
        admin.GET("/coupons", controllers.ListCoupons)
        admin.DELETE("/coupons/:id", controllers.DeleteCoupon)
}
