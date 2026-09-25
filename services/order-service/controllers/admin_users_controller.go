package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// GET /api/admin/users
// List every user account. Search via ?q= filters by name/email substring.
func AdminListUsers(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	var users []models.User
	q := c.Query("q")
	tx := database.DB
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name LIKE ? OR email LIKE ?", like, like)
	}
	if err := tx.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Don't ship passwords even though they're hashed.
	for i := range users {
		users[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"items": users})
}

// PUT /api/admin/users/:id/role  { role }
// Promote or demote a user. Body: { role: "customer" | "staff" | "admin" }
func AdminUpdateUserRole(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role required"})
		return
	}
	switch body.Role {
	case "customer", "staff", "admin":
		// ok
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be customer/staff/admin"})
		return
	}
	res := database.DB.Model(&models.User{}).Where("id = ?", id).Update("role", body.Role)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": id, "role": body.Role})
}