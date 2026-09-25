package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func CommunityExtensionRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	auth := middleware.AuthMiddleware()

	// Follow
	router.POST("/api/community/follow/:user_id", proxyMW, auth, controllers.FollowUser)
	router.DELETE("/api/community/follow/:user_id", proxyMW, auth, controllers.UnfollowUser)
	router.GET("/api/community/followers/:user_id", proxyMW, controllers.ListFollowers)
	router.GET("/api/community/following/:user_id", proxyMW, controllers.ListFollowing)

	// Bookmark
	router.POST("/api/community/bookmark/:post_type/:post_id", proxyMW, auth, controllers.ToggleBookmark)
	router.GET("/api/community/bookmarks", proxyMW, auth, controllers.ListBookmarks)

	// Groups
	router.POST("/api/community/groups", proxyMW, auth, controllers.CreateGroup)
	router.GET("/api/community/groups", proxyMW, controllers.ListGroups)
	router.POST("/api/community/groups/:id/join", proxyMW, auth, controllers.JoinGroup)
	router.DELETE("/api/community/groups/:id/join", proxyMW, auth, controllers.LeaveGroup)
	router.GET("/api/community/groups/:id/members", proxyMW, controllers.ListGroupMembers)

	// Q&A
	router.POST("/api/community/questions", proxyMW, auth, controllers.AskQuestion)
	router.GET("/api/community/questions", proxyMW, controllers.ListQuestions)
	router.GET("/api/community/questions/:id", proxyMW, controllers.GetQuestion)
	router.POST("/api/community/questions/:id/answer", proxyMW, auth, controllers.AnswerQuestion)
	router.PATCH("/api/community/answers/:id/accept", proxyMW, auth, controllers.AcceptAnswer)

	// Leaderboard
	router.GET("/api/community/leaderboard", proxyMW, controllers.Leaderboard)
}
