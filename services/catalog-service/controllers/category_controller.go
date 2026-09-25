package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"kather_baksho/catalog_service/database"
	"kather_baksho/catalog_service/models"

	"github.com/gin-gonic/gin"
)

func slugifyCategory(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == ' ' || r == '-' || r == '_':
			out = append(out, '-')
		}
	}
	res := string(out)
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	return strings.Trim(res, "-")
}

// GET /api/categories
func ListCategories(c *gin.Context) {
	cacheKey := "categories:all"
	if cached, ok := database.CacheGet(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	var cats []models.Category
	database.DB.Order("position asc, name asc").Find(&cats)
	if cats == nil {
		cats = []models.Category{}
	}

	if jsonBytes, err := json.Marshal(cats); err == nil {
		database.CacheSet(cacheKey, string(jsonBytes), 5*time.Minute)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, cats)
}

// POST /api/admin/categories   (admin)
func AdminCreateCategory(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Parent   string `json:"parent"`
		Icon     string `json:"icon"`
		Position int    `json:"position"`
		Active   *bool  `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	slug := body.Slug
	if slug == "" {
		slug = slugifyCategory(body.Name)
	}
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not derive slug"})
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	cat := models.Category{
		Name: body.Name, Slug: slug, Parent: body.Parent,
		Icon: body.Icon, Position: body.Position, Active: active,
	}
	if err := database.DB.Create(&cat).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.InvalidateCachePrefix("categories:")
	c.JSON(http.StatusCreated, cat)
}

// PUT /api/admin/categories/:id   (admin)
func AdminUpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var body struct {
		Name     *string `json:"name"`
		Slug     *string `json:"slug"`
		Parent   *string `json:"parent"`
		Icon     *string `json:"icon"`
		Position *int    `json:"position"`
		Active   *bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name != nil {
		cat.Name = *body.Name
	}
	if body.Slug != nil && *body.Slug != "" {
		cat.Slug = slugifyCategory(*body.Slug)
	}
	if body.Parent != nil {
		cat.Parent = *body.Parent
	}
	if body.Icon != nil {
		cat.Icon = *body.Icon
	}
	if body.Position != nil {
		cat.Position = *body.Position
	}
	if body.Active != nil {
		cat.Active = *body.Active
	}
	database.DB.Save(&cat)
	database.InvalidateCachePrefix("categories:")
	c.JSON(http.StatusOK, cat)
}

// DELETE /api/admin/categories/:id   (admin)
func AdminDeleteCategory(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	database.DB.Delete(&cat)
	database.InvalidateCachePrefix("categories:")
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
