package models

import (
	"time"

	"gorm.io/gorm"
)

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
	PostID   uint   `json:"post_id" gorm:"index"`
	UserID   uint   `json:"user_id" gorm:"index"`
	Author   string `json:"author"`
	Body     string `json:"body"`
}

type CommunityLike struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	PostID    uint      `json:"post_id" gorm:"uniqueIndex:idx_post_user"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_post_user"`
	CreatedAt time.Time `json:"created_at"`
}