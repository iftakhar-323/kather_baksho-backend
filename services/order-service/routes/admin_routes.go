package routes

import (
	"kather_baksho/controllers"
	"kather_baksho/middleware"

	"github.com/gin-gonic/gin"
)

// AdminRoutes mounts the back-office endpoints under /api/admin/*.
//
// Most are admin-only. A small fulfillment subset (order list, manual order
// entry and status updates) is also open to "staff" via StaffMiddleware —
// see middleware/staff_middleware.go.
func AdminRoutes(router *gin.Engine) {
	g := router.Group("/api/admin")
	g.Use(middleware.AuthMiddleware())

	admin := middleware.AdminMiddleware()
	staff := middleware.StaffMiddleware()

	// ---- Fulfillment: staff + admin ----
	g.GET("/orders", staff, controllers.GetAllOrders)
	g.POST("/orders", staff, controllers.CreateAdminOrder)
	g.PUT("/orders/:id/status", staff, controllers.UpdateOrderStatus)

	// ---- Admin-only ----
	g.DELETE("/orders/:id", admin, controllers.DeleteOrder)
	g.GET("/analytics", admin, controllers.GetAdminAnalytics)
	g.GET("/analytics/inventory-forecast", admin, controllers.InventoryForecast)

	// reminders
	g.GET("/reminders", admin, controllers.AdminListReminders)
	g.POST("/reminders/:id/complete", admin, controllers.AdminCompleteReminder)

	// subscriptions
	g.GET("/subscriptions", admin, controllers.AdminListSubscriptions)
	g.POST("/subscriptions/:id/cancel", admin, controllers.AdminCancelSubscription)

	// consultations
	g.GET("/consultations", admin, controllers.AdminListConsultations)
	g.POST("/consultations/:id/confirm", admin, controllers.AdminConfirmConsultation)
	g.POST("/consultations/:id/cancel", admin, controllers.AdminCancelConsultation)

	// corporate quotes
	g.GET("/corporate", admin, controllers.GetAllCorporateQuotes)
	g.PUT("/corporate/:id", admin, controllers.UpdateCorporateStatus)
	g.DELETE("/corporate/:id", admin, controllers.DeleteCorporateQuote)

	// users + roles
	g.GET("/users", admin, controllers.AdminListUsers)
	g.PUT("/users/:id/role", admin, controllers.AdminUpdateUserRole)

	// reviews
	g.GET("/reviews", admin, controllers.AdminListReviews)
}
