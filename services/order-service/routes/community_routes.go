package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func CommunityRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	g := router.Group("/api/community", proxyMW)
	{
		g.GET("/posts", controllers.ListPosts)
		g.GET("/posts/:id/comments", controllers.ListComments)
	}

	auth := router.Group("/api/community", proxyMW)
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/posts", controllers.CreatePost)
		auth.POST("/posts/:id/comments", controllers.AddComment)
		auth.POST("/posts/:id/like", controllers.ToggleLike)
		auth.DELETE("/posts/:id", controllers.DeletePost)
	}
}
