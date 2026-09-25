package models

import "gorm.io/gorm"

// User is the central auth/account record.
// Points is the "Green Points" loyalty balance earned on every order.
type User struct {
	gorm.Model
	Name          string `json:"name"`
	Email         string `json:"email" gorm:"unique"`
	Password      string `json:"-"`
	Role          string `json:"role" gorm:"default:customer"` // "customer" or "admin"
	Points        uint   `json:"points" gorm:"default:0"`      // Green Points loyalty balance
	EmailVerified bool   `json:"email_verified" gorm:"default:false"`

	// ===== Profile extras (Sprint A) =====
	Phone   string `json:"phone" gorm:"default:''"`   // user-facing profile phone
	Address string `json:"address" gorm:"default:''"` // legacy single-address string (kept for back-compat)

	// AvatarURL is an optional profile-picture link. When empty the frontend
	// renders a deterministic generated avatar from the name/email.
	AvatarURL string `json:"avatar_url" gorm:"default:''"`

	// ===== 2FA TOTP (Phase 16) =====
	TOTPSecret        string `json:"-" gorm:"default:''"`
	TOTPEnabled       bool   `json:"totp_enabled" gorm:"default:false"`
	TOTPRecoveryCodes string `json:"-" gorm:"default:''"` // Comma-separated recovery codes
}

// Address is a saved delivery address for a user.
// One user can have many addresses; one can be marked as default for checkout.
type Address struct {
	gorm.Model
	UserID     uint   `json:"user_id" gorm:"index"`
	Label      string `json:"label"`     // e.g. "Home", "Office"
	Recipient  string `json:"recipient"` // contact name at this address
	Phone      string `json:"phone"`
	Line1      string `json:"line1"` // street / house / road
	Line2      string `json:"line2"` // apt / floor (optional)
	City       string `json:"city"`
	Region     string `json:"region"` // state / division
	PostalCode string `json:"postal_code"`
	Country    string `json:"country" gorm:"default:'Bangladesh'"`
	IsDefault  bool   `json:"is_default" gorm:"default:false"`
}
