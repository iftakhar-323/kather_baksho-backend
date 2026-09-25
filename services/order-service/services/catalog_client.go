package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
)

// ProductDTO represents product data retrieved across the service boundary.
type ProductDTO struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	SKU            string  `json:"sku"`
	Brand          string  `json:"brand"`
	Category       string  `json:"category"`
	Price          float64 `json:"price"`
	CompareAtPrice float64 `json:"compare_at_price"`
	DiscountPct    float64 `json:"discount_pct"`
	Stock          uint    `json:"stock"`
	ImageURL       string  `json:"image_url"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// GetCatalogProduct fetches product info either from catalog-service via HTTP,
// or via attached database fallback.
func GetCatalogProduct(productID uint) (*ProductDTO, error) {
	catalogURL := os.Getenv("CATALOG_SERVICE_URL")
	if catalogURL != "" {
		reqURL := fmt.Sprintf("%s/api/products/%d", catalogURL, productID)
		resp, err := httpClient.Get(reqURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var product ProductDTO
			if err := json.NewDecoder(resp.Body).Decode(&product); err == nil {
				return &product, nil
			}
		}
	}

	// Fallback to local attached database
	database.EnsureAttached(database.DB)
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		return nil, err
	}

	return &ProductDTO{
		ID:             product.ID,
		Name:           product.Name,
		Slug:           product.Slug,
		SKU:            product.SKU,
		Brand:          product.Brand,
		Category:       product.Category,
		Price:          product.Price,
		CompareAtPrice: product.CompareAtPrice,
		DiscountPct:    product.DiscountPct,
		Stock:          product.Stock,
		ImageURL:       product.ImageURL,
	}, nil
}
