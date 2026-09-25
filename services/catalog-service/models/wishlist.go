package models

import "gorm.io/gorm"

type WishlistItem struct {
	gorm.Model
	UserID    uint    `json:"user_id" gorm:"index"`
	ProductID uint    `json:"product_id" gorm:"index"`
	Product   Product `json:"Product" gorm:"foreignKey:ProductID"`
}
