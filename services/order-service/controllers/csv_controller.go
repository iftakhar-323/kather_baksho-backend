package controllers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// ---------- CSV import/export (admin) ----------

// GET /api/admin/products/export
func ExportProductsCSV(c *gin.Context) {
	var list []models.Product
	database.DB.Order("id asc").Find(&list)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="products.csv"`)
	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = w.Write([]string{
		"id", "name", "sku", "slug", "category", "parent_category", "brand",
		"price", "compare_at_price", "discount_pct", "offer_label",
		"stock", "difficulty", "sunlight", "water", "humidity", "pet_friendly",
		"description", "image_url",
	})
	for _, p := range list {
		_ = w.Write([]string{
			strconv.Itoa(int(p.ID)),
			p.Name, p.SKU, p.Slug,
			p.Category, p.ParentCategory, p.Brand,
			fmt.Sprintf("%.2f", p.Price),
			fmt.Sprintf("%.2f", p.CompareAtPrice),
			fmt.Sprintf("%.2f", p.DiscountPct),
			p.OfferLabel,
			strconv.FormatUint(uint64(p.Stock), 10),
			p.Difficulty, p.Sunlight, p.Water, p.Humidity,
			strconv.FormatBool(p.PetFriendly),
			p.Description, p.ImageURL,
		})
	}
}

// POST /api/admin/products/import (multipart, field "file")
func ImportProductsCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty or malformed CSV"})
		return
	}
	header := rows[0]
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := []string{"name", "price"}
	for _, r := range required {
		if _, ok := idx[r]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing required column: " + r})
			return
		}
	}
	created, updated := 0, 0
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		get := func(col string) string {
			i, ok := idx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		name := get("name")
		if name == "" {
			continue
		}
		price, _ := strconv.ParseFloat(get("price"), 64)
		stockI, _ := strconv.Atoi(get("stock"))
		if stockI < 0 {
			stockI = 0
		}
		stock := uint(stockI)
		discount, _ := strconv.ParseFloat(get("discount_pct"), 64)
		cap, _ := strconv.ParseFloat(get("compare_at_price"), 64)
		pet, _ := strconv.ParseBool(get("pet_friendly"))

		var p models.Product
		sku := get("sku")
		if sku != "" {
			if err := database.DB.Where("sku = ?", sku).First(&p).Error; err == nil {
				// update path
				p.Name = name
				p.Slug = get("slug")
				p.Category = get("category")
				p.ParentCategory = get("parent_category")
				p.Brand = get("brand")
				p.Price = price
				p.CompareAtPrice = cap
				p.DiscountPct = discount
				p.OfferLabel = get("offer_label")
				p.Stock = stock
				p.Difficulty = get("difficulty")
				p.Sunlight = get("sunlight")
				p.Water = get("water")
				p.Humidity = get("humidity")
				p.PetFriendly = pet
				p.Description = get("description")
				p.ImageURL = get("image_url")
				p.UpdatedAt = time.Now()
				database.DB.Save(&p)
				updated++
				continue
			}
		}
		// create path
		p = models.Product{
			Name: name, SKU: sku, Slug: get("slug"),
			Category: get("category"), ParentCategory: get("parent_category"),
			Brand: get("brand"), Price: price, CompareAtPrice: cap,
			DiscountPct: discount, OfferLabel: get("offer_label"),
			Stock: stock,
			Difficulty: get("difficulty"), Sunlight: get("sunlight"),
			Water: get("water"), Humidity: get("humidity"),
			PetFriendly: pet,
			Description: get("description"), ImageURL: get("image_url"),
		}
		if p.SKU == "" {
			p.SKU = generateSKU(p.Category, name)
		}
		if p.Slug == "" {
			p.Slug = generateSlug(name)
		}
		database.DB.Create(&p)
		created++
	}
	c.JSON(http.StatusOK, gin.H{"created": created, "updated": updated})
}