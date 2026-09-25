package models

import "gorm.io/gorm"

type Consultation struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"index"`
	ExpertName   string `json:"expert_name"`
	Topic        string `json:"topic"`
	ScheduledAt  string `json:"scheduled_at"` // ISO datetime YYYY-MM-DDTHH:MM
	Status       string `json:"status"`       // "booked" | "completed" | "cancelled"
	Notes        string `json:"notes"`
}