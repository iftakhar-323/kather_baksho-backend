package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.Engine) {
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/register", controllers.Register)
		authGroup.POST("/login", controllers.Login)

		// protected
		authGroup.GET("/me", middleware.AuthMiddleware(), controllers.Me)

		// ===== Sprint A — profile + address book =====
		authGroup.PUT("/profile", middleware.AuthMiddleware(), controllers.UpdateProfile)
		authGroup.PUT("/change-password", middleware.AuthMiddleware(), controllers.ChangePassword)
		authGroup.DELETE("/account", middleware.AuthMiddleware(), controllers.DeleteAccount)

		authGroup.GET("/addresses", middleware.AuthMiddleware(), controllers.ListAddresses)
		authGroup.POST("/addresses", middleware.AuthMiddleware(), controllers.CreateAddress)
		authGroup.PUT("/addresses/:id", middleware.AuthMiddleware(), controllers.UpdateAddress)
		authGroup.DELETE("/addresses/:id", middleware.AuthMiddleware(), controllers.DeleteAddress)
		authGroup.PUT("/addresses/:id/default", middleware.AuthMiddleware(), controllers.SetDefaultAddress)

		// Email verification (Sprint F1)
		authGroup.POST("/verify", controllers.VerifyEmail)                     // public
		authGroup.POST("/resend-verification", controllers.ResendVerification) // public

		// 2FA TOTP (Phase 16)
		authGroup.POST("/2fa/setup", middleware.AuthMiddleware(), controllers.SetupTOTP)
		authGroup.POST("/2fa/enable", middleware.AuthMiddleware(), controllers.EnableTOTP)
		authGroup.POST("/2fa/verify", controllers.VerifyTOTP) // public: uses temp_token
		authGroup.POST("/2fa/disable", middleware.AuthMiddleware(), controllers.DisableTOTP)
		authGroup.GET("/2fa/status", middleware.AuthMiddleware(), controllers.StatusTOTP)
	}
}
