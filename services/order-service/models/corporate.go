package models

import "gorm.io/gorm"

type CorporateQuote struct {
	gorm.Model
	UserID        uint    `json:"user_id" gorm:"index"`
	CompanyName   string  `json:"company_name"`
	ContactName   string  `json:"contact_name"`
	ContactEmail  string  `json:"contact_email"`
	ContactPhone  string  `json:"contact_phone"`
	Recipients    string  `json:"recipients"`      // JSON-encoded list of {name, address, product_id}
	Message       string  `json:"message"`         // gift message card text
	BudgetPerGift float64 `json:"budget_per_gift"` // BDT per recipient
	TotalEstimate float64 `json:"total_estimate"`
	Status        string  `json:"status"` // "pending" | "quoted" | "accepted" | "delivered"
	AdminNotes    string  `json:"admin_notes"`
}
