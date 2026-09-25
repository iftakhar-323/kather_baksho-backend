package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func BlogRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	// Public reads
	router.GET("/api/blog/by-id/:id", proxyMW, auth, admin, controllers.AdminGetBlogPost)
	router.GET("/api/blog", proxyMW, controllers.ListBlogPosts)
	router.GET("/api/blog/:slug", proxyMW, controllers.GetBlogPost)

	// Admin write
	router.POST("/api/blog", proxyMW, auth, admin, controllers.CreateBlogPost)
	router.PATCH("/api/blog/by-id/:id", proxyMW, auth, admin, controllers.UpdateBlogPost)
	router.DELETE("/api/blog/by-id/:id", proxyMW, auth, admin, controllers.DeleteBlogPost)
}
