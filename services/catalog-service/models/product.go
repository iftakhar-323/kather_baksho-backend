package models

import "gorm.io/gorm"

// Product represents a single sellable item in the catalogue.
// Fields below support nested categories, search filters, related products,
// frequently-bought-together bundles, discounts, and view tracking.
type Product struct {
	gorm.Model
	Name           string  `json:"name"`
	Slug           string  `json:"slug" gorm:"uniqueIndex"`           // URL-friendly id, auto-generated from name
	SKU            string  `json:"sku" gorm:"uniqueIndex"`            // stock-keeping unit, auto-generated
	Brand          string  `json:"brand"`                              // brand name
	Category       string  `json:"category"`                            // top-level: "plant" | "decor" | "care"
	ParentCategory string  `json:"parent_category"`                    // nested-category: e.g. "outdoor > flowering"
	Subcategory    string  `json:"subcategory"`                         // "indoor_plant","outdoor_plant","plant_box","decor","soil","fertilizer","care_kit"
	IndoorOutdoor  string  `json:"indoor_outdoor"`                      // "indoor", "outdoor", "both", ""
	Difficulty     string  `json:"difficulty"`                          // "easy" | "medium" | "hard"
	Sunlight       string  `json:"sunlight"`                            // "low" | "medium" | "bright" | "direct"
	Water          string  `json:"water"`                              // "low" | "medium" | "high"
	Humidity       string  `json:"humidity"`                            // "low" | "medium" | "high"
	PetFriendly    bool    `json:"pet_friendly" gorm:"default:false"`
	Specifications string  `json:"specifications"`                      // free-form specs text
	Price          float64 `json:"price"`
	CompareAtPrice float64 `json:"compare_at_price"`                   // strike-through price
	DiscountPct    float64 `json:"discount_pct" gorm:"default:0"`      // manual % discount override
	OfferLabel     string  `json:"offer_label"`                         // "Hot", "New", "Limited", etc.
	Stock          uint    `json:"stock"`
	Description    string  `json:"description"`
	ImageURL       string  `json:"image_url"`
	ViewCount      uint    `json:"view_count" gorm:"default:0"`
	RelatedIDs     string  `json:"related_ids"`                         // JSON array of product ids (comma-separated for simplicity)
	FbtIDs         string  `json:"fbt_ids"`                             // frequently-bought-together product ids
}

// PageView records each product page open for analytics.
type PageView struct {
	gorm.Model
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id" gorm:"index"`
	IP        string `json:"ip"`
}