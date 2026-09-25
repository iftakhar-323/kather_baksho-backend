package controllers

import (
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// Static list of available plant-care experts. In production this would be a
// query against a Staff table.
var EXPERTS = []Expert{
	{Name: "Rina Akter", Specialty: "Indoor plants & low-light care", Rate: 500},
	{Name: "Tareq Aziz", Specialty: "Balcony & rooftop vegetable gardening", Rate: 600},
	{Name: "Nadia Khan", Specialty: "Succulents, cacti, and propagation", Rate: 450},
}

type Expert struct {
	Name      string `json:"name"`
	Specialty string `json:"specialty"`
	Rate      int    `json:"rate"` // BDT per session
}

// GET /api/consultations/experts
func ListExperts(c *gin.Context) {
	c.JSON(http.StatusOK, EXPERTS)
}

type BookConsultationInput struct {
	ExpertName  string `json:"expert_name" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
	ScheduledAt string `json:"scheduled_at" binding:"required"`
	Notes       string `json:"notes"`
}

// POST /api/consultations
func BookConsultation(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input BookConsultationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validExpert := false
	for _, e := range EXPERTS {
		if e.Name == input.ExpertName {
			validExpert = true
			break
		}
	}
	if !validExpert {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expert name selected"})
		return
	}

	con := models.Consultation{
		UserID:      userID,
		ExpertName:  input.ExpertName,
		Topic:       input.Topic,
		ScheduledAt: input.ScheduledAt,
		Notes:       input.Notes,
		Status:      "booked",
	}
	database.DB.Create(&con)
	c.JSON(http.StatusCreated, con)
}

// GET /api/consultations
func GetMyConsultations(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.Consultation
	database.DB.Where("user_id = ?", userID).Order("scheduled_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/consultations/:id/cancel
func CancelConsultation(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var con models.Consultation
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&con).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consultation not found"})
		return
	}
	con.Status = "cancelled"
	database.DB.Save(&con)
	c.JSON(http.StatusOK, con)
}
