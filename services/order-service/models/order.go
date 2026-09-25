package models

import "gorm.io/gorm"

type Order struct {
	gorm.Model
	UserID          uint        `json:"user_id"`
	TotalPrice      float64     `json:"total_price"`
	Status          string      `json:"status"` // Pending, Processing, Packed, On the Way, Delivered
	PaymentMethod   string      `json:"payment_method" gorm:"default:'cod'"`
	PaymentStatus   string      `json:"payment_status" gorm:"default:'Pending'"`
	TransactionID   string      `json:"transaction_id"`
	ShippingName    string      `json:"shipping_name"`
	ShippingPhone   string      `json:"shipping_phone"`
	ShippingAddress string      `json:"shipping_address"`
	DeliveryNote    string      `json:"delivery_note"`
	GiftWrap        bool        `json:"gift_wrap" gorm:"default:false"`
	CouponCode      string      `json:"coupon_code"`
	DiscountAmount  float64     `json:"discount_amount"`
	Items           []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
	gorm.Model
	OrderID   uint    `json:"order_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID;references:ID;<-:false;-:migration"`
	Quantity  uint    `json:"quantity"`
	Price     float64 `json:"price"` // Preserves unit price at the time of purchase
}
