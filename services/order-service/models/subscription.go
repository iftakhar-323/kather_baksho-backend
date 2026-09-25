package models

import "gorm.io/gorm"

// Subscription is a recurring plant-box plan.
//
// Lifecycle:
//
//	active → paused → active → renewed → cancelled
//
// Status semantics:
//
//	"active"     — billing + dispatch as scheduled.
//	"paused"     — billing suspended, but stays on file. Resumable.
//	"cancelled"  — terminal. Cannot be reactivated (user must create a new sub).
type Subscription struct {
	gorm.Model
	UserID       uint    `json:"user_id" gorm:"index"`
	PlanName     string  `json:"plan_name"`     // "Monthly Plant Box", "Cactus Box", etc.
	IntervalDays int     `json:"interval_days"` // 30, 60, 90
	NextDelivery string  `json:"next_delivery"` // YYYY-MM-DD
	Status       string  `json:"status"`        // "active" | "paused" | "cancelled"
	Price        float64 `json:"price"`         // BDT per delivery
	// Extra controls (Sprint D): pausing stores a pause timestamp; renewing
	// bumps NextDelivery forward by IntervalDays; cancelling is irreversible.
	PausedAt        string `json:"paused_at"` // ISO datetime, "" if not paused
	ResumedAt       string `json:"resumed_at"`
	LastRenewedAt   string `json:"last_renewed_at"`
	CancelledAt     string `json:"cancelled_at"`
	DeliveriesCount uint   `json:"deliveries_count" gorm:"default:0"`
}
