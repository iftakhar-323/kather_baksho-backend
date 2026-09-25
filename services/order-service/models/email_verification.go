package models

import (
	"time"

	"gorm.io/gorm"
)

// EmailVerification stores one-time codes sent to a user to prove they own
// the email address on file. Multiple unconsumed rows are allowed — each new
// Resend supersedes the previous one (the verify handler picks the freshest
// unconsumed, unexpired row).
type EmailVerification struct {
	gorm.Model
	UserID    uint       `json:"user_id" gorm:"index"`
	Code      string     `json:"code" gorm:"size:8;index"`
	ExpiresAt time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
}