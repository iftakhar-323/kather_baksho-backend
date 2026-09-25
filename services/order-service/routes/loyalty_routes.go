package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

func LoyaltyRoutes(router *gin.Engine) {
	auth := middleware.AuthMiddleware()

	// Achievements
	router.GET("/api/loyalty/achievements", controllers.ListAchievements)
	router.GET("/api/loyalty/achievements/me", auth, controllers.MyAchievements)
	router.POST("/api/loyalty/achievements/:id/claim", auth, controllers.ClaimAchievement)
	router.POST("/api/loyalty/check-achievements", auth, controllers.CheckAchievements)

	// Membership tier
	router.GET("/api/loyalty/tier", auth, controllers.MyTier)

	// Referrals
	router.GET("/api/loyalty/referral", auth, controllers.MyReferralCode)
	router.POST("/api/loyalty/referral/redeem", auth, controllers.RedeemReferral)
	router.GET("/api/loyalty/referrals/me", auth, controllers.MyReferrals)

	// Rewards
	router.GET("/api/loyalty/rewards", controllers.ListRewards)
	router.POST("/api/loyalty/rewards/:id/redeem", auth, controllers.RedeemReward)
}