package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

func CareJournalRoutes(router *gin.Engine) {
	proxyMW := func(c *gin.Context) {
		if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
			c.Abort()
			return
		}
	}

	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	// Growth Journal
	router.GET("/api/journal", proxyMW, auth, controllers.ListJournal)
	router.POST("/api/journal", proxyMW, auth, controllers.AddJournalEntry)
	router.DELETE("/api/journal/:id", proxyMW, auth, controllers.DeleteJournalEntry)
	router.GET("/api/journal/timeline", proxyMW, auth, controllers.JournalTimeline)

	// Care Schedule
	router.GET("/api/care-schedule", proxyMW, controllers.ListCareSchedule)
	router.POST("/api/care-schedule", proxyMW, auth, admin, controllers.AddCareSchedule)
	router.DELETE("/api/care-schedule/:id", proxyMW, auth, admin, controllers.DeleteCareSchedule)

	// Care Calendar (aggregated, per-user)
	router.GET("/api/care-calendar", proxyMW, auth, controllers.CareCalendar)
}
