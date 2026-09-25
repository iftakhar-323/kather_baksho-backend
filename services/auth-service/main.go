package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"kather_baksho/auth_service/controllers"
	"kather_baksho/auth_service/database"
	"kather_baksho/auth_service/middleware"
	"kather_baksho/auth_service/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Auth-Service] Starting Identity, 2FA TOTP & User Management Microservice...")

	// 1. Initialize database
	database.ConnectDatabase()
	if err := database.DB.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.EmailVerification{},
	); err != nil {
		log.Fatalf("[Auth-Service] Auto-migrate failed: %v", err)
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
			"service":   "kather_baksho-auth",
			"database":  dbStatus,
			"timestamp": time.Now().UTC(),
		})
	})

	// 5. Auth Routes
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)

		// Protected endpoints
		auth.GET("/me", middleware.AuthMiddleware(), controllers.Me)
		auth.PUT("/profile", middleware.AuthMiddleware(), controllers.UpdateProfile)
		auth.PUT("/change-password", middleware.AuthMiddleware(), controllers.ChangePassword)
		auth.DELETE("/account", middleware.AuthMiddleware(), controllers.DeleteAccount)

		// Address management
		auth.GET("/addresses", middleware.AuthMiddleware(), controllers.ListAddresses)
		auth.POST("/addresses", middleware.AuthMiddleware(), controllers.CreateAddress)
		auth.PUT("/addresses/:id", middleware.AuthMiddleware(), controllers.UpdateAddress)
		auth.DELETE("/addresses/:id", middleware.AuthMiddleware(), controllers.DeleteAddress)
		auth.PUT("/addresses/:id/default", middleware.AuthMiddleware(), controllers.SetDefaultAddress)

		// Email verification
		auth.POST("/verify", controllers.VerifyEmail)
		auth.POST("/resend-verification", controllers.ResendVerification)

		// 2FA TOTP
		auth.POST("/2fa/setup", middleware.AuthMiddleware(), controllers.SetupTOTP)
		auth.POST("/2fa/enable", middleware.AuthMiddleware(), controllers.EnableTOTP)
		auth.POST("/2fa/verify", controllers.VerifyTOTP)
		auth.POST("/2fa/disable", middleware.AuthMiddleware(), controllers.DisableTOTP)
		auth.GET("/2fa/status", middleware.AuthMiddleware(), controllers.StatusTOTP)
	}

	// 5b. Direct Addresses Route Group (/api/addresses)
	addresses := router.Group("/api/addresses", middleware.AuthMiddleware())
	{
		addresses.GET("", controllers.ListAddresses)
		addresses.POST("", controllers.CreateAddress)
		addresses.PUT("/:id", controllers.UpdateAddress)
		addresses.DELETE("/:id", controllers.DeleteAddress)
		addresses.PUT("/:id/default", controllers.SetDefaultAddress)
	}

	// 6. Admin Users Management Routes
	adminUsers := router.Group("/api/admin/users", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		adminUsers.GET("", controllers.AdminListUsers)
		adminUsers.GET("/", controllers.AdminListUsers)
		adminUsers.PUT("/:id/role", controllers.AdminUpdateUserRole)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("[Auth-Service] Microservice listening on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[Auth-Service] Failed to start server: %v", err)
	}
}
