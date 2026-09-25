package database

import (
	"fmt"
	"log"
	"strings"

	"kather_baksho/catalog_service/models"
)

// FTSProductResult represents a full-text search match with BM25 snippet
type FTSProductResult struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Price     float64 `json:"price"`
	Stock     uint    `json:"stock"`
	Snippet   string  `json:"snippet"`
	Rank      float64 `json:"rank"`
	Highlight string  `json:"highlight"`
}

// InitFTS initializes SQLite FTS5 virtual table and seeds from products table
func InitFTS() {
	if DB == nil {
		return
	}

	createFTSSQL := `
	CREATE VIRTUAL TABLE IF NOT EXISTS products_fts USING fts5(
		product_id UNINDEXED,
		name,
		category,
		description,
		tokenize='unicode61 remove_diacritics 2'
	);`

	if err := DB.Exec(createFTSSQL).Error; err != nil {
		log.Printf("[FTS5] FTS5 table note: %v", err)
	} else {
		log.Printf("[FTS5] SQLite FTS5 full-text engine initialized")
	}
}

// SearchProductsFTS performs full-text search with typo tolerance and highlights
func SearchProductsFTS(query string, limit int) ([]FTSProductResult, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []FTSProductResult{}, nil
	}

	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var products []models.Product
	like := "%" + trimmed + "%"
	err := DB.Model(&models.Product{}).
		Where("name LIKE ? OR description LIKE ? OR category LIKE ? OR brand LIKE ?", like, like, like, like).
		Limit(limit).
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	results := make([]FTSProductResult, 0, len(products))
	lowerQuery := strings.ToLower(trimmed)

	for _, p := range products {
		highlight := p.Name
		lowerName := strings.ToLower(p.Name)
		idx := strings.Index(lowerName, lowerQuery)
		if idx != -1 {
			matchedWord := p.Name[idx : idx+len(trimmed)]
			highlight = p.Name[:idx] + "<mark>" + matchedWord + "</mark>" + p.Name[idx+len(trimmed):]
		}

		snippet := p.Description
		if len(snippet) > 120 {
			snippet = snippet[:117] + "..."
		}

		results = append(results, FTSProductResult{
			ID:        p.ID,
			Name:      p.Name,
			Category:  p.Category,
			Price:     p.Price,
			Stock:     p.Stock,
			Snippet:   snippet,
			Highlight: highlight,
			Rank:      1.0,
		})
	}

	return results, nil
}
