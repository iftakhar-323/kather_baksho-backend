package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func ConsultationRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	g := router.Group("/api/consultations", proxyMW)
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/experts", controllers.ListExperts)
		g.POST("", controllers.BookConsultation)
		g.POST("/", controllers.BookConsultation)
		g.GET("", controllers.GetMyConsultations)
		g.GET("/", controllers.GetMyConsultations)
		g.POST("/:id/cancel", controllers.CancelConsultation)
	}
}
