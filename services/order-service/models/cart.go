package models

import "gorm.io/gorm"

type Cart struct {
	gorm.Model
	UserID uint       `json:"user_id"`
	Items  []CartItem `json:"items" gorm:"foreignKey:CartID"`
}

type CartItem struct {
	gorm.Model
	CartID    uint    `json:"cart_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID;references:ID;<-:false;-:migration"`
	Quantity  uint    `json:"quantity"`
}
