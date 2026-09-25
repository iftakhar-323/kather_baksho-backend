package models

import (
	"time"

	"gorm.io/gorm"
)

// =====================================================================
// Sprint D+E+I — additional models for orders, subscriptions,
// plant care, community, loyalty, corporate, blog, analytics.
// All persisted via AutoMigrate in main.go.
// =====================================================================

// --- Order extensions ---

// OrderEvent is one entry in an order's status timeline.
type OrderEvent struct {
	gorm.Model
	OrderID   uint   `json:"order_id" gorm:"index"`
	Event     string `json:"event"`     // "placed", "paid", "packed", "shipped", "delivered", "cancelled", "returned", "refunded", "exchanged", "note"
	Note      string `json:"note"`
	CreatedBy uint   `json:"created_by"` // user_id of who recorded it (0=system)
}

// ReturnRequest captures a return/refund/exchange ask on an order.
type ReturnRequest struct {
	gorm.Model
	OrderID      uint    `json:"order_id" gorm:"index"`
	UserID       uint    `json:"user_id" gorm:"index"`
	Type         string  `json:"type"` // "return" | "refund" | "exchange"
	Reason       string  `json:"reason"`
	ProductID    uint    `json:"product_id"`   // primary product being returned (0 = order-level)
	Quantity     int     `json:"quantity"`     // -1 if order-level
	Items        string  `json:"items"`        // JSON array of {product_id, quantity}
	Status       string  `json:"status"`       // "pending" | "approved" | "rejected" | "completed"
	AdminNote    string  `json:"admin_note"`
	RefundAmount float64 `json:"refund_amount"`
	Notes        string  `json:"notes"`        // free-form user notes
}

// --- Subscription extensions ---

// Add fields to Subscription model via the file `subscription.go`.
// These helpers sit alongside for clarity.

// SubscriptionDelivery tracks each dispatch from a subscription.
type SubscriptionDelivery struct {
	gorm.Model
	SubscriptionID uint      `json:"subscription_id" gorm:"index"`
	DeliveredAt    time.Time `json:"delivered_at"`
	Notes          string    `json:"notes"`
}

// --- Plant care extensions ---

// GrowthJournal is a per-plant log of notes + photos.
type GrowthJournal struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	ProductID uint   `json:"product_id" gorm:"index"`
	EntryDate string `json:"entry_date"`     // YYYY-MM-DD
	Note      string `json:"note"`
	HeightCM  uint   `json:"height_cm"`
	PhotoURL  string `json:"photo_url"`
	Type      string `json:"type"`           // "water", "fertilize", "repot", "note", "bloom", "photo"
}

// CareSchedule is a per-product seasonal care hint (admin-managed).
type CareSchedule struct {
	gorm.Model
	ProductID   uint   `json:"product_id" gorm:"index"`
	Month       int    `json:"month"`           // 1-12
	Action      string `json:"action"`          // "water", "fertilize", "repot"
	Description string `json:"description"`
}

// --- Community extensions ---

// CommunityFollow is a one-way follow relationship.
type CommunityFollow struct {
	gorm.Model
	FollowerID uint `json:"follower_id" gorm:"uniqueIndex:idx_follow_pair"`
	FollowedID uint `json:"followed_id" gorm:"uniqueIndex:idx_follow_pair"`
}

// CommunityBookmark saves a post for later per-user.
type CommunityBookmark struct {
	gorm.Model
	UserID   uint   `json:"user_id" gorm:"uniqueIndex:idx_bm_post_user"`
	PostID   uint   `json:"post_id" gorm:"uniqueIndex:idx_bm_post_user"`
	PostType string `json:"post_type"` // "post" | "product" | "blog" | "question"
}

// CommunityGroup is a named group users can belong to (e.g. "Succulent Lovers").
type CommunityGroup struct {
	gorm.Model
	Name        string `json:"name" gorm:"unique"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	OwnerID     uint   `json:"owner_id" gorm:"index"`
	MemberCount uint   `json:"member_count" gorm:"default:0"`
}

// CommunityGroupMember links a user to a group.
type CommunityGroupMember struct {
	gorm.Model
	GroupID  uint      `json:"group_id" gorm:"uniqueIndex:idx_gm_user_group"`
	UserID   uint      `json:"user_id" gorm:"uniqueIndex:idx_gm_user_group"`
	Role     string    `json:"role" gorm:"default:'member'"`
	JoinedAt time.Time `json:"joined_at"`
}

// CommunityQuestion is a Q&A post (extends CommunityPost via a flag).
type CommunityQuestion struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Tags      string `json:"tags"`              // comma-separated
	ProductID uint   `json:"product_id" gorm:"index"`
	Status    string `json:"status"`             // "open" | "answered" | "resolved"
	Accepted  uint   `json:"accepted_answer_id"` // ID of the accepted comment
}

// CommunityAnswer is a reply to a CommunityQuestion.
type CommunityAnswer struct {
	gorm.Model
	QuestionID uint   `json:"question_id" gorm:"index"`
	UserID     uint   `json:"user_id" gorm:"index"`
	Author     string `json:"author"`
	Body       string `json:"body"`
	IsAccepted bool   `json:"is_accepted" gorm:"default:false"`
}

// --- Loyalty extensions ---

// Achievement is a badge that unlocks when a user hits a milestone.
type Achievement struct {
	gorm.Model
	Code        string `json:"code" gorm:"unique"` // "first_order", "five_orders", etc.
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`               // emoji or font-awesome name
	PointsBonus uint   `json:"points_bonus"`
}

// UserAchievement records that a user has unlocked an Achievement.
type UserAchievement struct {
	gorm.Model
	UserID        uint   `json:"user_id" gorm:"uniqueIndex:idx_ua_user_code"`
	AchievementID uint   `json:"achievement_id"`
	Code          string `json:"code" gorm:"uniqueIndex:idx_ua_user_code"`
	UnlockedAt    time.Time `json:"unlocked_at"`
}

// ReferralCode is the unique referral token attached to a user.
type ReferralCode struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"uniqueIndex"`
	Code      string `json:"code" gorm:"uniqueIndex"`
	Uses      uint   `json:"uses" gorm:"default:0"`
}

// Referral records when someone signs up using a referral code.
type Referral struct {
	gorm.Model
	ReferrerID  uint `json:"referrer_id" gorm:"index"` // user who shared
	ReferredID  uint `json:"referred_id" gorm:"uniqueIndex"` // user who signed up
	ReferrerReward uint `json:"referrer_reward" gorm:"default:100"` // points credited to referrer
	ReferredReward uint `json:"referred_reward" gorm:"default:50"`  // points credited to new user
}

// MembershipTier defines the thresholds (Bronze/Silver/Gold).
type MembershipTier struct {
	gorm.Model
	Name        string  `json:"name" gorm:"unique"` // "Bronze" | "Silver" | "Gold" | "Platinum"
	MinSpend    float64 `json:"min_spend"`           // lifetime spend threshold
	MinOrders   uint    `json:"min_orders"`
	DiscountPct float64 `json:"discount_pct"`         // automatic % discount on every order
	Icon        string  `json:"icon"`
	Color       string  `json:"color"`
}

// UserMembership is the per-user tier record (auto-upserted from order totals).
type UserMembership struct {
	gorm.Model
	UserID      uint      `json:"user_id" gorm:"uniqueIndex"`
	Tier        string    `json:"tier"`
	TotalSpend  float64   `json:"total_spend"`
	Points      uint      `json:"points"`
	LastUpdated time.Time `json:"last_updated"`
}

// CouponReward redeems loyalty points for a coupon code.
type CouponReward struct {
	gorm.Model
	Title     string  `json:"title"`
	PointsCost int    `json:"points_cost"`
	Percent   float64 `json:"percent"`
	Active    bool    `json:"active" gorm:"default:true"`
}

// --- Corporate extensions ---

// CorporateOrder is a confirmed/approved bulk order from the corporate portal.
type CorporateOrder struct {
	gorm.Model
	QuoteID      uint    `json:"quote_id" gorm:"index"`
	UserID       uint    `json:"user_id" gorm:"index"`
	CompanyName  string  `json:"company_name"`
	Status       string  `json:"status"` // "pending_approval" | "approved" | "dispatched" | "delivered" | "rejected"
	Items        string  `json:"items"`  // JSON array of {name, address, product_id, qty}
	TotalAmount  float64 `json:"total_amount"`
	BrandingNote string  `json:"branding_note"` // custom wrapping/branding request
	AdminNote    string  `json:"admin_note"`
	ApprovedBy   uint    `json:"approved_by"`
}

// --- Blog / Encyclopedia ---

// BlogPost is a piece of long-form content. category determines which section.
type BlogPost struct {
	gorm.Model
	Slug        string `json:"slug" gorm:"uniqueIndex"`
	Title       string `json:"title"`
	Excerpt     string `json:"excerpt"`
	Content     string `json:"content"`     // markdown
	CoverURL    string `json:"cover_url"`
	Category    string `json:"category"`    // "blog" | "encyclopedia" | "care_guide"
	PlantID     uint   `json:"plant_id"`    // for encyclopedia entries — link to product
	Author      string `json:"author"`
	AuthorID    uint   `json:"author_id"`
	Published   bool   `json:"published" gorm:"default:true"`
	ViewCount   uint   `json:"view_count" gorm:"default:0"`
}

// --- Analytics ---

// PageView records each product page open for analytics.
type PageView struct {
	gorm.Model
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id" gorm:"index"`
	IP        string `json:"ip"`
}

// --- Shopping ---

// GuestOrder captures an order placed without auth (name/phone/address in body).
// The order itself is the same Order + OrderItem pair with UserID=0 (anonymous).
// We keep a separate table to remember the buyer info for receipt printing.
type GuestOrder struct {
	gorm.Model
	OrderID       uint   `json:"order_id" gorm:"index"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
}

// GiftCard is a stored-value card with a code + balance.
type GiftCard struct {
	gorm.Model
	Code      string  `json:"code" gorm:"uniqueIndex"`
	Balance   float64 `json:"balance"`
	IssuedTo  string  `json:"issued_to"`      // recipient name/email
	IssuedBy  uint    `json:"issued_by"`       // user_id of buyer (0=admin)
	Active    bool    `json:"active" gorm:"default:true"`
	ExpiresAt string  `json:"expires_at"`      // YYYY-MM-DD; "" = never
}

// ShippingRule is an admin-editable shipping rule.
type ShippingRule struct {
	gorm.Model
	Name          string  `json:"name"`
	MinSubtotal   float64 `json:"min_subtotal"`   // free over this subtotal
	FlatFee       float64 `json:"flat_fee"`       // flat fee in BDT
	Active        bool    `json:"active" gorm:"default:true"`
}

// TaxRule is an admin-editable tax rule.
type TaxRule struct {
	gorm.Model
	Name        string  `json:"name"`
	MinSubtotal float64 `json:"min_subtotal"` // taxable above this
	Percent     float64 `json:"percent"`      // e.g. 5 = 5%
	Active      bool    `json:"active" gorm:"default:true"`
}
