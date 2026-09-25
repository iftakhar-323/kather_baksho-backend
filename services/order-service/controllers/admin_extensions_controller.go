package controllers

import (
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// ============ Admin: Reminders ============

// GET /api/admin/reminders - all pending reminders across users
func AdminListReminders(c *gin.Context) {
	var list []models.CareReminder
	database.DB.Order("completed asc, next_due_date asc").Find(&list)
	// hydrate user and product info manually
	type Out struct {
		models.CareReminder
		UserEmail   string `json:"user_email"`
		ProductName string `json:"product_name"`
	}
	out := make([]Out, 0, len(list))
	for _, r := range list {
		var u models.User
		database.DB.First(&u, r.UserID)
		var p models.Product
		database.DB.First(&p, r.ProductID)
		out = append(out, Out{CareReminder: r, UserEmail: u.Email, ProductName: p.Name})
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/admin/reminders/:id/complete
func AdminCompleteReminder(c *gin.Context) {
	id := c.Param("id")
	var r models.CareReminder
	if err := database.DB.First(&r, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}
	r.Completed = true
	database.DB.Save(&r)
	c.JSON(http.StatusOK, gin.H{"message": "Reminder marked completed"})
}

// ============ Admin: Subscriptions ============

// GET /api/admin/subscriptions
func AdminListSubscriptions(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	var list []models.Subscription
	database.DB.Order("created_at desc").Find(&list)

	type Out struct {
		models.Subscription
		UserEmail string `json:"user_email"`
	}
	out := make([]Out, 0, len(list))
	for _, s := range list {
		var u models.User
		database.DB.First(&u, s.UserID)
		out = append(out, Out{Subscription: s, UserEmail: u.Email})
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/admin/subscriptions/:id/cancel
func AdminCancelSubscription(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")
	var s models.Subscription
	if err := database.DB.First(&s, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}
	database.DB.Unscoped().Delete(&s)

	database.DB.Create(&models.Notification{
		UserID:  s.UserID,
		Message: "Your " + s.PlanName + " subscription was cancelled.",
		Type:    "subscription",
	})
	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled"})
}

// ============ Admin: Consultations ============

// GET /api/admin/consultations
func AdminListConsultations(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	var list []models.Consultation
	database.DB.Order("scheduled_at desc").Find(&list)

	type Out struct {
		models.Consultation
		UserEmail string `json:"user_email"`
	}
	out := make([]Out, 0, len(list))
	for _, con := range list {
		var u models.User
		database.DB.First(&u, con.UserID)
		out = append(out, Out{Consultation: con, UserEmail: u.Email})
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/admin/consultations/:id/confirm
func AdminConfirmConsultation(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")
	var con models.Consultation
	if err := database.DB.First(&con, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consultation not found"})
		return
	}
	con.Status = "confirmed"
	database.DB.Save(&con)

	database.DB.Create(&models.Notification{
		UserID:  con.UserID,
		Message: "Your consultation with " + con.ExpertName + " is confirmed.",
		Type:    "consultation",
	})
	c.JSON(http.StatusOK, gin.H{"message": "Consultation confirmed"})
}

// POST /api/admin/consultations/:id/cancel
func AdminCancelConsultation(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")

	var con models.Consultation
	if err := database.DB.First(&con, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consultation not found"})
		return
	}
	database.DB.Unscoped().Delete(&con)

	database.DB.Create(&models.Notification{
		UserID:  con.UserID,
		Message: "Your consultation with " + con.ExpertName + " was cancelled.",
		Type:    "consultation",
	})
	c.JSON(http.StatusOK, gin.H{"message": "Consultation cancelled"})
}

// ============ Admin Dashboard Aggregator ============

// GET /api/admin/analytics
// Returns the aggregate stats the Admin DashboardTab renders:
//
//	revenue, total_orders, total_users, total_products, total_reminders,
//	top_products[], orders_by_status[]
func GetAdminAnalytics(c *gin.Context) {
	var revenue float64
	database.DB.Model(&models.Order{}).
		Where("status <> ?", "cancelled").
		Select("COALESCE(SUM(total_price),0)").Row().Scan(&revenue)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Count(&totalOrders)

	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)

	var totalProducts int64
	database.DB.Model(&models.Product{}).Count(&totalProducts)

	var totalReminders int64
	database.DB.Model(&models.CareReminder{}).Where("completed = ?", false).Count(&totalReminders)

	// Top-selling products (top 5)
	type TopProduct struct {
		ProductID uint    `json:"product_id"`
		Name      string  `json:"name"`
		Sold      int     `json:"sold"`
		Revenue   float64 `json:"revenue"`
	}
	var topProducts []TopProduct
	database.DB.Raw(`
		SELECT p.id as product_id, p.name as name,
		       COALESCE(SUM(oi.quantity),0) as sold,
		       COALESCE(SUM(oi.price * oi.quantity),0) as revenue
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN orders o ON o.id = oi.order_id AND o.status <> 'cancelled'
		GROUP BY p.id
		HAVING sold > 0
		ORDER BY sold DESC
		LIMIT 5`).Scan(&topProducts)
	if topProducts == nil {
		topProducts = []TopProduct{}
	}

	// Orders grouped by status
	type StatusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	var ordersByStatus []StatusCount
	database.DB.Raw(`
		SELECT COALESCE(status,'unknown') as status, COUNT(*) as count
		FROM orders GROUP BY status ORDER BY count DESC`).Scan(&ordersByStatus)
	if ordersByStatus == nil {
		ordersByStatus = []StatusCount{}
	}

	c.JSON(http.StatusOK, gin.H{
		"revenue":          revenue,
		"total_orders":     totalOrders,
		"total_users":      totalUsers,
		"total_products":   totalProducts,
		"total_reminders":  totalReminders,
		"top_products":     topProducts,
		"orders_by_status": ordersByStatus,
	})
}
