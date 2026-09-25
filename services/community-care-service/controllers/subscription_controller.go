package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/community_care_service/database"
	"kather_baksho/community_care_service/models"

	"github.com/gin-gonic/gin"
)

type SubscriptionInput struct {
	PlanName     string  `json:"plan_name" binding:"required"`
	IntervalDays int     `json:"interval_days" binding:"required"`
	Price        float64 `json:"price"`
}

func userIsAdmin(c *gin.Context) bool {
	if v, ok := c.Get("role"); ok && v == "admin" {
		return true
	}
	return false
}

func loadSubForUser(c *gin.Context) (*models.Subscription, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	var sub models.Subscription
	if err := database.DB.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return nil, false
	}
	uid := c.GetUint("user_id")
	if sub.UserID != uid && !userIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not yours"})
		return nil, false
	}
	return &sub, true
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

// POST /api/subscriptions
func CreateSubscription(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input SubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.IntervalDays <= 0 {
		input.IntervalDays = 30
	}
	sub := models.Subscription{
		UserID:       userID,
		PlanName:     input.PlanName,
		IntervalDays: input.IntervalDays,
		Price:        input.Price,
		NextDelivery: time.Now().AddDate(0, 0, input.IntervalDays).Format("2006-01-02"),
		Status:       "active",
	}
	database.DB.Create(&sub)
	c.JSON(http.StatusCreated, sub)
}

// GET /api/subscriptions
func GetMySubscriptions(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.Subscription
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/subscriptions/:id/cancel
func CancelSubscription(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var sub models.Subscription
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}
	sub.Status = "cancelled"
	sub.CancelledAt = time.Now().Format(time.RFC3339)
	database.DB.Save(&sub)
	c.JSON(http.StatusOK, sub)
}

// POST /api/subscriptions/:id/advance
func AdvanceSubscription(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var sub models.Subscription
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}
	sub.NextDelivery = time.Now().AddDate(0, 0, sub.IntervalDays).Format("2006-01-02")
	sub.DeliveriesCount += 1
	sub.LastRenewedAt = time.Now().Format(time.RFC3339)
	database.DB.Save(&sub)
	c.JSON(http.StatusOK, sub)
}

// POST /api/subscriptions/:id/pause
func PauseSubscription(c *gin.Context) {
	sub, ok := loadSubForUser(c)
	if !ok {
		return
	}
	if sub.Status == "paused" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already paused"})
		return
	}
	if sub.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot pause a cancelled subscription"})
		return
	}
	sub.Status = "paused"
	sub.PausedAt = nowISO()
	if err := database.DB.Save(sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// POST /api/subscriptions/:id/resume
func ResumeSubscription(c *gin.Context) {
	sub, ok := loadSubForUser(c)
	if !ok {
		return
	}
	if sub.Status != "paused" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription is not paused"})
		return
	}
	sub.Status = "active"
	sub.PausedAt = ""
	sub.ResumedAt = nowISO()
	if err := database.DB.Save(sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// POST /api/subscriptions/:id/renew
func RenewSubscription(c *gin.Context) {
	sub, ok := loadSubForUser(c)
	if !ok {
		return
	}
	if sub.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "renew a cancelled subscription by creating a new one"})
		return
	}
	sub.Status = "active"
	sub.PausedAt = ""
	sub.LastRenewedAt = nowISO()
	sub.DeliveriesCount = sub.DeliveriesCount + 1
	today := time.Now()
	if sub.IntervalDays <= 0 {
		sub.IntervalDays = 30
	}
	sub.NextDelivery = today.AddDate(0, 0, sub.IntervalDays).Format("2006-01-02")
	if err := database.DB.Save(sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&models.SubscriptionDelivery{
		SubscriptionID: sub.ID,
		DeliveredAt:    time.Now(),
		Notes:          fmt.Sprintf("Manual renewal (count=%d)", sub.DeliveriesCount),
	})
	c.JSON(http.StatusOK, sub)
}

// GET /api/subscriptions/:id/deliveries
func ListSubscriptionDeliveries(c *gin.Context) {
	sub, ok := loadSubForUser(c)
	if !ok {
		return
	}
	var ds []models.SubscriptionDelivery
	database.DB.Where("subscription_id = ?", sub.ID).Order("delivered_at desc").Find(&ds)
	c.JSON(http.StatusOK, ds)
}

// Admin: List subscriptions
// GET /api/subscriptions/admin / /api/admin/subscriptions
func AdminListSubscriptions(c *gin.Context) {
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

// Admin: Cancel subscription
// POST /api/subscriptions/admin/:id/cancel / /api/admin/subscriptions/:id/cancel
func AdminCancelSubscription(c *gin.Context) {
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
