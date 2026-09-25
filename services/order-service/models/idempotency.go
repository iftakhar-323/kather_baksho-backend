package models

import (
	"time"
)

// IdempotencyRecord stores request hashes and cached responses to prevent duplicate operations.
type IdempotencyRecord struct {
	Key          string    `gorm:"primaryKey;size:128" json:"key"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Endpoint     string    `gorm:"size:255" json:"endpoint"`
	Status       string    `gorm:"size:32;default:'PROCESSING'" json:"status"` // PROCESSING, COMPLETED, FAILED
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `gorm:"type:text" json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
