package models

import "gorm.io/gorm"

// Review is a product review left by a user.
// One review per (UserID, ProductID) pair — re-submitting updates the row.
type Review struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index;uniqueIndex:idx_review_user_product"`
	ProductID uint   `json:"product_id" gorm:"index;uniqueIndex:idx_review_user_product"`
	Rating    int    `json:"rating" gorm:"index"` // 1..5
	Comment   string `json:"comment"`
	UserName  string `json:"user_name"` // denormalised for fast list rendering
}