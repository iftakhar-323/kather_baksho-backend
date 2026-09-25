package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"gorm.io/gorm"
)

// SaveOutboxEvent records an event within an existing database transaction.
func SaveOutboxEvent(tx *gorm.DB, eventType string, aggregateID string, payload map[string]interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox event payload: %w", err)
	}

	event := models.OutboxEvent{
		EventType:   eventType,
		AggregateID: aggregateID,
		Payload:     string(payloadBytes),
		Status:      "pending",
		RetryCount:  0,
	}

	if tx != nil {
		return tx.Create(&event).Error
	}
	return database.DB.Create(&event).Error
}

// StartOutboxWorker runs a background polling loop that flushes pending outbox events to Redis Streams.
func StartOutboxWorker() {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			if database.DB == nil {
				continue
			}
			processPendingOutboxEvents()
		}
	}()
	log.Println("[Outbox] Background Transactional Outbox worker started.")
}

func processPendingOutboxEvents() {
	var events []models.OutboxEvent
	if err := database.DB.Where("status = ?", "pending").Order("id ASC").Limit(25).Find(&events).Error; err != nil {
		return
	}

	for _, evt := range events {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(evt.Payload), &payload); err != nil {
			database.DB.Model(&evt).Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": fmt.Sprintf("invalid json: %v", err),
			})
			continue
		}

		_, err := PublishEvent(evt.EventType, payload)
		now := time.Now()
		if err != nil {
			database.DB.Model(&evt).Updates(map[string]interface{}{
				"retry_count":   evt.RetryCount + 1,
				"last_attempt":  now,
				"error_message": err.Error(),
				"status": func() string {
					if evt.RetryCount >= 5 {
						return "failed"
					}
					return "pending"
				}(),
			})
		} else {
			database.DB.Model(&evt).Updates(map[string]interface{}{
				"status":       "published",
				"last_attempt": now,
			})
		}
	}
}
