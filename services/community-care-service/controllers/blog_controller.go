package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"kather_baksho/community_care_service/database"
	"kather_baksho/community_care_service/models"

	"github.com/gin-gonic/gin"
)

// ListBlogPosts handles GET /api/blog
func ListBlogPosts(c *gin.Context) {
	page := 1
	limit := 10
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	q := database.DB.Where("published = ?", true).Order("created_at desc")
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if s := c.Query("q"); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		q = q.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ?", like, like)
	}
	var total int64
	q.Model(&models.BlogPost{}).Count(&total)
	var list []models.BlogPost
	q.Offset((page - 1) * limit).Limit(limit).Find(&list)
	c.JSON(http.StatusOK, gin.H{
		"posts": list, "page": page, "limit": limit, "total": total,
		"pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetBlogPost handles GET /api/blog/:slug
func GetBlogPost(c *gin.Context) {
	slug := c.Param("slug")
	var p models.BlogPost
	if err := database.DB.Where("slug = ? AND published = ?", slug, true).
		First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	database.DB.Model(&p).UpdateColumn("view_count", p.ViewCount+1)
	p.ViewCount++
	c.JSON(http.StatusOK, p)
}

// AdminGetBlogPost handles GET /api/blog/by-id/:id
func AdminGetBlogPost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p models.BlogPost
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// CreateBlogPost handles POST /api/blog (admin)
func CreateBlogPost(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Excerpt   string `json:"excerpt"`
		Content   string `json:"content"`
		CoverURL  string `json:"cover_url"`
		Category  string `json:"category"`
		PlantID   uint   `json:"plant_id"`
		Author    string `json:"author"`
		Published bool   `json:"published"`
	}
	if err := c.BindJSON(&body); err != nil || body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	if body.Slug == "" {
		body.Slug = slugify(body.Title)
	}
	if body.Category == "" {
		body.Category = "blog"
	}
	p := models.BlogPost{
		Slug: body.Slug, Title: body.Title, Excerpt: body.Excerpt, Content: body.Content,
		CoverURL: body.CoverURL, Category: body.Category, PlantID: body.PlantID,
		Author: body.Author, AuthorID: uid, Published: body.Published,
	}
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// UpdateBlogPost handles PATCH /api/blog/by-id/:id (admin)
func UpdateBlogPost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	body["updated_at"] = time.Now()
	if err := database.DB.Model(&models.BlogPost{}).Where("id = ?", id).
		Updates(body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteBlogPost handles DELETE /api/blog/by-id/:id (admin)
func DeleteBlogPost(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	database.DB.Delete(&models.BlogPost{}, id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			prevDash = false
		} else if !prevDash && len(out) > 0 {
			out = append(out, '-')
			prevDash = true
		}
	}
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "post"
	}
	return string(out)
}
