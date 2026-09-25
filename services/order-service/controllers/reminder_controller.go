package controllers

import (
	"net/http"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// GET /api/reminders - user-er upcoming care reminders
func GetReminders(c *gin.Context) {
	userID := c.GetUint("user_id")

	var list []models.CareReminder
	database.DB.Preload("Product").
		Where("user_id = ? AND completed = ?", userID, false).
		Order("next_due_date asc").
		Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/reminders/:id/complete - mark as done and schedule the next one
func CompleteReminder(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var r models.CareReminder
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&r).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}

	r.Completed = true
	database.DB.Save(&r)

	// schedule next occurrence
	if r.IntervalDays > 0 {
		next := time.Now().AddDate(0, 0, r.IntervalDays).Format("2006-01-02")
		database.DB.Create(&models.CareReminder{
			UserID:       r.UserID,
			ProductID:    r.ProductID,
			Type:         r.Type,
			NextDueDate:  next,
			IntervalDays: r.IntervalDays,
			Completed:    false,
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reminder completed; next one scheduled."})
}

// DELETE /api/reminders/:id - permanently delete a reminder
func DeleteReminder(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var r models.CareReminder
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&r).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}

	database.DB.Delete(&r)
	c.JSON(http.StatusOK, gin.H{"message": "Reminder deleted"})
}