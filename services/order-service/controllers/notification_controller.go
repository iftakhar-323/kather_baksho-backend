package controllers

import (
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// GET /api/notifications - user-er notifications (newest first)
func GetNotifications(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.Notification
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/notifications/mark-read
func MarkNotificationsRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	database.DB.Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Update("is_read", true)
	c.JSON(http.StatusOK, gin.H{"message": "All marked as read"})
}