package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// Helper: fetch a subscription, checking the caller owns it (or is admin)
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

// POST /api/subscriptions/:id/pause — pause the next delivery
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

// POST /api/subscriptions/:id/resume — resume a paused subscription
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

// POST /api/subscriptions/:id/renew — manual renewal: bumps NextDelivery, logs delivery
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
	// Push next_delivery forward by IntervalDays from today
	today := time.Now()
	if sub.IntervalDays <= 0 {
		sub.IntervalDays = 30
	}
	sub.NextDelivery = today.AddDate(0, 0, sub.IntervalDays).Format("2006-01-02")
	if err := database.DB.Save(sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Log a delivery for this manual renewal
	database.DB.Create(&models.SubscriptionDelivery{
		SubscriptionID: sub.ID,
		DeliveredAt:    time.Now(),
		Notes:          fmt.Sprintf("Manual renewal (count=%d)", sub.DeliveriesCount),
	})
	c.JSON(http.StatusOK, sub)
}

// GET /api/subscriptions/:id/deliveries — list past deliveries
func ListSubscriptionDeliveries(c *gin.Context) {
	sub, ok := loadSubForUser(c)
	if !ok {
		return
	}
	var ds []models.SubscriptionDelivery
	database.DB.Where("subscription_id = ?", sub.ID).Order("delivered_at desc").Find(&ds)
	c.JSON(http.StatusOK, ds)
}