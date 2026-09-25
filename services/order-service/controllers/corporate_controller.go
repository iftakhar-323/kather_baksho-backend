package controllers

import (
	"encoding/json"
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

type CreateCorporateInput struct {
	CompanyName   string  `json:"company_name" binding:"required"`
	ContactName   string  `json:"contact_name" binding:"required"`
	ContactEmail  string  `json:"contact_email" binding:"required"`
	ContactPhone  string  `json:"contact_phone"`
	Recipients    string  `json:"recipients" binding:"required"` // JSON array string
	Message       string  `json:"message"`
	BudgetPerGift float64 `json:"budget_per_gift"`
}

// POST /api/corporate
func CreateCorporateQuote(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input CreateCorporateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var recipientsArray []map[string]interface{}
	if err := json.Unmarshal([]byte(input.Recipients), &recipientsArray); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipients JSON format"})
		return
	}
	count := len(recipientsArray)

	quote := models.CorporateQuote{
		UserID:        userID,
		CompanyName:   input.CompanyName,
		ContactName:   input.ContactName,
		ContactEmail:  input.ContactEmail,
		ContactPhone:  input.ContactPhone,
		Recipients:    input.Recipients,
		Message:       input.Message,
		BudgetPerGift: input.BudgetPerGift,
		TotalEstimate: input.BudgetPerGift * float64(count),
		Status:        "pending",
	}
	database.DB.Create(&quote)
	c.JSON(http.StatusCreated, quote)
}

// GET /api/corporate/mine
func GetMyCorporateQuotes(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.CorporateQuote
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// GET /api/admin/corporate  (admin)
func GetAllCorporateQuotes(c *gin.Context) {
	var list []models.CorporateQuote
	database.DB.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

type UpdateCorporateStatusInput struct {
	Status     string `json:"status"`
	AdminNotes string `json:"admin_notes"`
}

// PUT /api/admin/corporate/:id  (admin)
func UpdateCorporateStatus(c *gin.Context) {
	id := c.Param("id")
	var q models.CorporateQuote
	if err := database.DB.First(&q, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	var input UpdateCorporateStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Status != "" {
		q.Status = input.Status
	}
	if input.AdminNotes != "" {
		q.AdminNotes = input.AdminNotes
	}
	database.DB.Save(&q)

	// notify the user
	database.DB.Create(&models.Notification{
		UserID:  q.UserID,
		Message: "Your corporate gift quote for " + q.CompanyName + " is now " + q.Status,
		Type:    "corporate",
	})
	c.JSON(http.StatusOK, q)
}

// DELETE /api/admin/corporate/:id (admin)
func DeleteCorporateQuote(c *gin.Context) {
	id := c.Param("id")
	var q models.CorporateQuote
	if err := database.DB.First(&q, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	database.DB.Delete(&q)
	c.JSON(http.StatusOK, gin.H{"message": "Quote deleted successfully"})
}