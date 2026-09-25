package models

import (
	"time"

	"gorm.io/gorm"
)

// User stub for relations
type User struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
}

type WishlistItem struct {
	ID        uint `json:"id" gorm:"primaryKey"`
	UserID    uint `json:"user_id"`
	ProductID uint `json:"product_id"`
}

type CareReminder struct {
	ID        uint `json:"id" gorm:"primaryKey"`
	UserID    uint `json:"user_id"`
	ProductID uint `json:"product_id"`
}

type OrderItem struct {
	ID        uint `json:"id" gorm:"primaryKey"`
	OrderID   uint `json:"order_id"`
	ProductID uint `json:"product_id"`
}

// Notification for system alerts
type Notification struct {
	gorm.Model
	UserID  uint   `json:"user_id" gorm:"index"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Read    bool   `json:"read" gorm:"default:false"`
}

// --- Community Post Models ---

type CommunityPost struct {
	gorm.Model
	UserID   uint   `json:"user_id" gorm:"index"`
	Author   string `json:"author"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	ImageURL string `json:"image_url"`
	Category string `json:"category"` // "show-off" | "tip" | "question" | "story"
}

type CommunityComment struct {
	gorm.Model
	PostID uint   `json:"post_id" gorm:"index"`
	UserID uint   `json:"user_id" gorm:"index"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

type CommunityLike struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	PostID    uint      `json:"post_id" gorm:"uniqueIndex:idx_post_user"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_post_user"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Community Extensions ---

type CommunityFollow struct {
	gorm.Model
	FollowerID uint `json:"follower_id" gorm:"uniqueIndex:idx_follow_pair"`
	FollowedID uint `json:"followed_id" gorm:"uniqueIndex:idx_follow_pair"`
}

type CommunityBookmark struct {
	gorm.Model
	UserID   uint   `json:"user_id" gorm:"uniqueIndex:idx_bm_post_user"`
	PostID   uint   `json:"post_id" gorm:"uniqueIndex:idx_bm_post_user"`
	PostType string `json:"post_type"` // "post" | "product" | "blog" | "question"
}

type CommunityGroup struct {
	gorm.Model
	Name        string `json:"name" gorm:"unique"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	OwnerID     uint   `json:"owner_id" gorm:"index"`
	MemberCount uint   `json:"member_count" gorm:"default:0"`
}

type CommunityGroupMember struct {
	gorm.Model
	GroupID  uint      `json:"group_id" gorm:"uniqueIndex:idx_gm_user_group"`
	UserID   uint      `json:"user_id" gorm:"uniqueIndex:idx_gm_user_group"`
	Role     string    `json:"role" gorm:"default:'member'"`
	JoinedAt time.Time `json:"joined_at"`
}

type CommunityQuestion struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Tags      string `json:"tags"`
	ProductID uint   `json:"product_id" gorm:"index"`
	Status    string `json:"status"`             // "open" | "answered" | "resolved"
	Accepted  uint   `json:"accepted_answer_id"` // ID of the accepted answer
}

type CommunityAnswer struct {
	gorm.Model
	QuestionID uint   `json:"question_id" gorm:"index"`
	UserID     uint   `json:"user_id" gorm:"index"`
	Author     string `json:"author"`
	Body       string `json:"body"`
	IsAccepted bool   `json:"is_accepted" gorm:"default:false"`
}

// --- Botanical Care Models ---

type GrowthJournal struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	ProductID uint   `json:"product_id" gorm:"index"`
	EntryDate string `json:"entry_date"` // YYYY-MM-DD
	Note      string `json:"note"`
	HeightCM  uint   `json:"height_cm"`
	PhotoURL  string `json:"photo_url"`
	Type      string `json:"type"` // "water", "fertilize", "repot", "note", "bloom", "photo"
}

type CareSchedule struct {
	gorm.Model
	ProductID   uint   `json:"product_id" gorm:"index"`
	Month       int    `json:"month"`  // 1-12
	Action      string `json:"action"` // "water", "fertilize", "repot"
	Description string `json:"description"`
}

// --- Blog Model ---

type BlogPost struct {
	gorm.Model
	Slug      string `json:"slug" gorm:"uniqueIndex"`
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	Content   string `json:"content"`
	CoverURL  string `json:"cover_url"`
	Author    string `json:"author"`
	AuthorID  uint   `json:"author_id"`
	PlantID   uint   `json:"plant_id"`
	Category  string `json:"category"`  // "care_guide" | "diy" | "botany_news" | "species_spotlight"
	Tags      string `json:"tags"`      // comma-separated
	ReadTime  int    `json:"read_time"` // minutes
	Published bool   `json:"published" gorm:"default:true"`
	ViewCount int    `json:"view_count" gorm:"default:0"`
}

// --- Consultations ---

type Consultation struct {
	gorm.Model
	UserID      uint   `json:"user_id" gorm:"index"`
	ExpertName  string `json:"expert_name"`
	Topic       string `json:"topic"`
	ScheduledAt string `json:"scheduled_at"` // ISO datetime YYYY-MM-DDTHH:MM
	Status      string `json:"status"`       // "booked" | "completed" | "cancelled" | "confirmed"
	Notes       string `json:"notes"`
}

// --- Subscriptions ---

type Subscription struct {
	gorm.Model
	UserID          uint    `json:"user_id" gorm:"index"`
	PlanName        string  `json:"plan_name"`
	IntervalDays    int     `json:"interval_days"`
	NextDelivery    string  `json:"next_delivery"`
	Status          string  `json:"status"`
	Price           float64 `json:"price"`
	PausedAt        string  `json:"paused_at"`
	ResumedAt       string  `json:"resumed_at"`
	LastRenewedAt   string  `json:"last_renewed_at"`
	CancelledAt     string  `json:"cancelled_at"`
	DeliveriesCount uint    `json:"deliveries_count" gorm:"default:0"`
}

type SubscriptionDelivery struct {
	gorm.Model
	SubscriptionID uint      `json:"subscription_id" gorm:"index"`
	DeliveredAt    time.Time `json:"delivered_at"`
	Notes          string    `json:"notes"`
}
