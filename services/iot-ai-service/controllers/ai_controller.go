package controllers

import (
	"net/http"

	"kather_baksho/iot_ai_service/services"

	"github.com/gin-gonic/gin"
)

type DiagnoseInput struct {
	PlantName   string `json:"plant_name"`
	Symptoms    string `json:"symptoms" binding:"required"`
	Environment string `json:"environment"`
}

// POST /api/ai/diagnose
func DiagnosePlantSymptoms(c *gin.Context) {
	var input DiagnoseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide observed symptoms"})
		return
	}

	result := services.DiagnosePlant(input.PlantName, input.Symptoms, input.Environment)
	c.JSON(http.StatusOK, result)
}

type ChatInput struct {
	Question string `json:"question" binding:"required"`
	Context  string `json:"context"`
}

// POST /api/ai/chat
func ChatWithPlantDoctor(c *gin.Context) {
	var input ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a question"})
		return
	}

	answer := services.AnswerPlantCareQuestion(input.Question, input.Context)
	c.JSON(http.StatusOK, gin.H{
		"question": input.Question,
		"answer":   answer,
	})
}
