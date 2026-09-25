package models

import "gorm.io/gorm"

type Notification struct {
	gorm.Model
	UserID  uint   `json:"user_id" gorm:"index"`
	Message string `json:"message"`
	Type    string `json:"type"` // order_update / reminder / subscription / promo
	IsRead  bool   `json:"is_read" gorm:"default:false"`
}