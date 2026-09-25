package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func CorporateRoutes(router *gin.Engine) {
	g := router.Group("/api/corporate")
	g.Use(middleware.AuthMiddleware())
	{
		g.POST("/", controllers.CreateCorporateQuote)
		g.GET("/mine", controllers.GetMyCorporateQuotes)
	}
}