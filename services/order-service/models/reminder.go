package models

import "gorm.io/gorm"

type CareReminder struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"index"`
	ProductID    uint   `json:"product_id" gorm:"index"`
	Type         string `json:"type"`          // "watering", "fertilizer", "repotting"
	NextDueDate  string `json:"next_due_date"` // YYYY-MM-DD
	IntervalDays int    `json:"interval_days"` // how often to repeat
	Completed    bool   `json:"completed" gorm:"default:false"`
}