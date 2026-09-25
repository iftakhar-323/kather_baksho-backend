package controllers

import (
	"net/http"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

type SubscriptionInput struct {
	PlanName     string  `json:"plan_name" binding:"required"`
	IntervalDays int     `json:"interval_days" binding:"required"`
	Price        float64 `json:"price"`
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

// POST /api/subscriptions/:id/advance — useful for "Deliver now" demo button.
// In production this would be a cron job.
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