package models

import "gorm.io/gorm"

type Coupon struct {
	gorm.Model
	Code             string  `json:"code" gorm:"unique"`
	DiscountPercent  float64 `json:"discount_percent"` // 0-100, e.g. 20 = 20% off
	ExpiresAt        string  `json:"expires_at"`        // ISO date string YYYY-MM-DD; "" = never
	Active           bool    `json:"active" gorm:"default:true"`
	MinOrderTotal    float64 `json:"min_order_total"` // 0 = no minimum
}
