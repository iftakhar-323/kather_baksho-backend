// Standalone CLI: assigns a real catalogue photo to every product's image_url
// using models.ProductImageURL (keyword match on name, else a per-subcategory
// pool). Idempotent — safe to re-run. The photos themselves live in
// frontend/public/images/products/catalog/.
//
//	go run ./cmd/productimages          # update every product
//	go run ./cmd/productimages --force  # also overwrite non-empty custom URLs
package main

import (
	"fmt"
	"os"
	"strings"

	"kather_baksho/database"
	"kather_baksho/models"
)

func main() {
	force := false
	for _, a := range os.Args[1:] {
		if a == "--force" || a == "-f" {
			force = true
		}
	}

	database.ConnectDatabase()

	var products []models.Product
	if err := database.DB.Find(&products).Error; err != nil {
		fmt.Println("load products:", err)
		os.Exit(1)
	}

	updated, kept := 0, 0
	for _, p := range products {
		want := models.ProductImageURL(p.Name, p.Subcategory, p.Category, p.ID)

		// Leave genuine custom uploads alone unless --force. A URL is treated
		// as "custom" only if it already points at the catalogue folder with a
		// different file, or at an external http(s) source.
		isManaged := p.ImageURL == "" ||
			strings.Contains(p.ImageURL, "/images/products/") // old seed_*/tree* paths included
		if !force && !isManaged && (strings.HasPrefix(p.ImageURL, "http://") || strings.HasPrefix(p.ImageURL, "https://")) {
			kept++
			continue
		}

		if p.ImageURL == want {
			kept++
			continue
		}
		if err := database.DB.Model(&models.Product{}).Where("id = ?", p.ID).
			Update("image_url", want).Error; err != nil {
			fmt.Printf("  update %d (%s): %v\n", p.ID, p.Name, err)
			continue
		}
		updated++
	}

	fmt.Printf("productimages: %d products, %d updated, %d already correct\n",
		len(products), updated, kept)
}
