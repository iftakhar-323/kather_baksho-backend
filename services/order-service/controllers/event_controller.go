package controllers

import (
	"net/http"

	"kather_baksho/services"

	"github.com/gin-gonic/gin"
)

// GetEventBusStats exposes event stream length and consumer group stats
func GetEventBusStats(c *gin.Context) {
	stats, err := services.GetEventBusStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// PublishTestEvent publishes a custom test event to the stream
func PublishTestEvent(c *gin.Context) {
	var req struct {
		Type    string                 `json:"type" binding:"required"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: 'type' is required"})
		return
	}

	if req.Payload == nil {
		req.Payload = map[string]interface{}{}
	}

	msgID, err := services.PublishEvent(req.Type, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"event_id": msgID,
		"type":     req.Type,
		"message":  "Event published to Redis Stream successfully",
	})
}
