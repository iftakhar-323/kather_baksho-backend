package models

import "gorm.io/gorm"

// Category is an admin-curated taxonomy entry.
// Slug is what links to Product.Category / ParentCategory / Subcategory.
type Category struct {
	gorm.Model
	Name     string `json:"name" gorm:"unique"`       // "Plant"
	Slug     string `json:"slug" gorm:"uniqueIndex"`  // "plant"
	Parent   string `json:"parent"`                   // parent slug, "" for top-level
	Icon     string `json:"icon"`                     // emoji
	Position int    `json:"position" gorm:"default:0"` // sort order
	Active   bool   `json:"active" gorm:"default:true"`
}