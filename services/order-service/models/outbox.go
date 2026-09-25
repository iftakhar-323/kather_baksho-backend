package models

import (
	"time"

	"gorm.io/gorm"
)

// OutboxEvent represents an event persisted transactionally with database changes.
// It is processed asynchronously by the Outbox Worker to guarantee zero event loss (at-least-once delivery).
type OutboxEvent struct {
	gorm.Model
	EventType    string    `json:"event_type" gorm:"index"`
	AggregateID  string    `json:"aggregate_id" gorm:"index"`
	Payload      string    `json:"payload" gorm:"type:text"`
	Status       string    `json:"status" gorm:"default:'pending';index"` // 'pending', 'published', 'failed'
	RetryCount   int       `json:"retry_count" gorm:"default:0"`
	LastAttempt  time.Time `json:"last_attempt"`
	ErrorMessage string    `json:"error_message"`
}
