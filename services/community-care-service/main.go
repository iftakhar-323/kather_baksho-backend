package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"kather_baksho/community_care_service/controllers"
	"kather_baksho/community_care_service/database"
	"kather_baksho/community_care_service/middleware"
	"kather_baksho/community_care_service/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Community-Care-Service] Starting Community, Botanical Care & Subscriptions Microservice...")

	// 1. Initialize database
	database.ConnectDatabase()
	if err := database.DB.AutoMigrate(
		&models.CommunityPost{},
		&models.CommunityComment{},
		&models.CommunityLike{},
		&models.CommunityFollow{},
		&models.CommunityBookmark{},
		&models.CommunityGroup{},
		&models.CommunityGroupMember{},
		&models.CommunityQuestion{},
		&models.CommunityAnswer{},
		&models.GrowthJournal{},
		&models.CareSchedule{},
		&models.BlogPost{},
		&models.Consultation{},
		&models.Subscription{},
		&models.SubscriptionDelivery{},
		&models.Notification{},
	); err != nil {
		log.Fatalf("[Community-Care-Service] Auto-migrate failed: %v", err)
	}

	// 2. Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	router := gin.New()
	router.Use(gin.Recovery())

	// 3. Middlewares
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Correlation-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(middleware.RequestIDMiddleware())

	// 4. Health probe
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if sqlDB, err := database.DB.DB(); err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "kather_baksho-community-care",
			"database":  dbStatus,
			"timestamp": time.Now().UTC(),
		})
	})

	auth := middleware.AuthMiddleware()
	admin := middleware.AdminMiddleware()

	// 5. Community Routes (/api/community)
	comm := router.Group("/api/community")
	{
		// Posts & Comments (Public reads)
		comm.GET("/posts", controllers.ListPosts)
		comm.GET("/posts/:id/comments", controllers.ListComments)

		// Social & Q&A (Public reads)
		comm.GET("/followers/:user_id", controllers.ListFollowers)
		comm.GET("/following/:user_id", controllers.ListFollowing)
		comm.GET("/groups", controllers.ListGroups)
		comm.GET("/groups/:id/members", controllers.ListGroupMembers)
		comm.GET("/questions", controllers.ListQuestions)
		comm.GET("/questions/:id", controllers.GetQuestion)
		comm.GET("/leaderboard", controllers.Leaderboard)

		// Authenticated mutations
		comm.POST("/posts", auth, controllers.CreatePost)
		comm.POST("/posts/:id/comments", auth, controllers.AddComment)
		comm.POST("/posts/:id/like", auth, controllers.ToggleLike)
		comm.DELETE("/posts/:id", auth, controllers.DeletePost)

		comm.POST("/follow/:user_id", auth, controllers.FollowUser)
		comm.DELETE("/follow/:user_id", auth, controllers.UnfollowUser)

		comm.POST("/bookmark/:post_type/:post_id", auth, controllers.ToggleBookmark)
		comm.GET("/bookmarks", auth, controllers.ListBookmarks)

		comm.POST("/groups", auth, controllers.CreateGroup)
		comm.POST("/groups/:id/join", auth, controllers.JoinGroup)
		comm.DELETE("/groups/:id/join", auth, controllers.LeaveGroup)

		comm.POST("/questions", auth, controllers.AskQuestion)
		comm.POST("/questions/:id/answer", auth, controllers.AnswerQuestion)
		comm.PATCH("/answers/:id/accept", auth, controllers.AcceptAnswer)
	}

	// 6. Blog Routes (/api/blog)
	blog := router.Group("/api/blog")
	{
		blog.GET("", controllers.ListBlogPosts)
		blog.GET("/:slug", controllers.GetBlogPost)
		blog.GET("/by-id/:id", auth, admin, controllers.AdminGetBlogPost)
		blog.POST("", auth, admin, controllers.CreateBlogPost)
		blog.PATCH("/by-id/:id", auth, admin, controllers.UpdateBlogPost)
		blog.DELETE("/by-id/:id", auth, admin, controllers.DeleteBlogPost)
	}

	// 7. Growth Journal Routes (/api/journal)
	journal := router.Group("/api/journal", auth)
	{
		journal.GET("", controllers.ListJournal)
		journal.POST("", controllers.AddJournalEntry)
		journal.DELETE("/:id", controllers.DeleteJournalEntry)
		journal.GET("/timeline", controllers.JournalTimeline)
	}

	// 8. Care Schedule Routes (/api/care-schedule)
	careSched := router.Group("/api/care-schedule")
	{
		careSched.GET("", controllers.ListCareSchedule)
		careSched.POST("", auth, admin, controllers.AddCareSchedule)
		careSched.DELETE("/:id", auth, admin, controllers.DeleteCareSchedule)
	}

	// 9. Care Calendar Route (/api/care-calendar)
	router.GET("/api/care-calendar", auth, controllers.CareCalendar)

	// 10. Consultation Routes (/api/consultations)
	consultations := router.Group("/api/consultations", auth)
	{
		consultations.GET("/experts", controllers.ListExperts)
		consultations.POST("", controllers.BookConsultation)
		consultations.POST("/", controllers.BookConsultation)
		consultations.GET("", controllers.GetMyConsultations)
		consultations.GET("/", controllers.GetMyConsultations)
		consultations.POST("/:id/cancel", controllers.CancelConsultation)

		// Admin consultation management endpoints
		consultations.GET("/admin", admin, controllers.AdminListConsultations)
		consultations.POST("/admin/:id/confirm", admin, controllers.AdminConfirmConsultation)
		consultations.POST("/admin/:id/cancel", admin, controllers.AdminCancelConsultation)
	}

	// 11. Subscription Routes (/api/subscriptions)
	subs := router.Group("/api/subscriptions", auth)
	{
		subs.POST("", controllers.CreateSubscription)
		subs.POST("/", controllers.CreateSubscription)
		subs.GET("", controllers.GetMySubscriptions)
		subs.GET("/", controllers.GetMySubscriptions)
		subs.POST("/:id/cancel", controllers.CancelSubscription)
		subs.POST("/:id/advance", controllers.AdvanceSubscription)
		subs.POST("/:id/pause", controllers.PauseSubscription)
		subs.POST("/:id/resume", controllers.ResumeSubscription)
		subs.POST("/:id/renew", controllers.RenewSubscription)
		subs.GET("/:id/deliveries", controllers.ListSubscriptionDeliveries)

		// Admin subscription management endpoints
		subs.GET("/admin", admin, controllers.AdminListSubscriptions)
		subs.POST("/admin/:id/cancel", admin, controllers.AdminCancelSubscription)
	}

	// 12. Direct admin routes mirror for backward compatibility
	adminRoutes := router.Group("/api/admin", auth, admin)
	{
		adminRoutes.GET("/consultations", controllers.AdminListConsultations)
		adminRoutes.POST("/consultations/:id/confirm", controllers.AdminConfirmConsultation)
		adminRoutes.POST("/consultations/:id/cancel", controllers.AdminCancelConsultation)

		adminRoutes.GET("/subscriptions", controllers.AdminListSubscriptions)
		adminRoutes.POST("/subscriptions/:id/cancel", controllers.AdminCancelSubscription)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	log.Printf("[Community-Care-Service] Microservice listening on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[Community-Care-Service] Failed to start server: %v", err)
	}
}
