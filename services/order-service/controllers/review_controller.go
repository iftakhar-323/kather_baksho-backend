package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// GET /api/products/:id/reviews?page=1&limit=20
// Returns reviews + aggregate (avg, count, histogram) for the product.
func ListProductReviews(c *gin.Context) {
	pid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	page := 1
	limit := 20
	if v := c.Query("page"); v != "" {
		n, _ := strconv.Atoi(v)
		if n > 0 {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		n, _ := strconv.Atoi(v)
		if n > 0 && n <= 100 {
			limit = n
		}
	}

	var reviews []models.Review
	database.DB.Where("product_id = ?", pid).
		Order("created_at desc").
		Limit(limit).Offset((page - 1) * limit).
		Find(&reviews)

	var total int64
	database.DB.Model(&models.Review{}).Where("product_id = ?", pid).Count(&total)

	// avg rating
	var avg float64
	database.DB.Model(&models.Review{}).
		Where("product_id = ?", pid).
		Select("COALESCE(AVG(rating),0)").Row().Scan(&avg)

	// histogram 1..5
	hist := map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	type Row struct {
		Rating int   `json:"rating"`
		Count  int64 `json:"count"`
	}
	var rows []Row
	database.DB.Model(&models.Review{}).
		Select("rating, COUNT(*) as count").
		Where("product_id = ?", pid).Group("rating").Scan(&rows)
	for _, r := range rows {
		if r.Rating >= 1 && r.Rating <= 5 {
			hist[r.Rating] = r.Count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":   reviews,
		"page":      page,
		"limit":     limit,
		"total":     total,
		"avg":       avg,
		"count":     total,
		"histogram": hist,
	})
}

// POST /api/products/:id/reviews   (auth)
func CreateReview(c *gin.Context) {
	uid := c.GetUint("user_id")
	name, _ := c.Get("user_name")
	nameStr, _ := name.(string)
	pid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	var body struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Rating < 1 || body.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}

	// confirm product exists
	var p models.Product
	if err := database.DB.First(&p, pid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	// upsert — one review per user/product
	var existing models.Review
	if err := database.DB.Where("user_id = ? AND product_id = ?", uid, pid).First(&existing).Error; err == nil {
		existing.Rating = body.Rating
		existing.Comment = body.Comment
		existing.UserName = nameStr
		database.DB.Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	r := models.Review{
		UserID: uid, ProductID: uint(pid),
		Rating: body.Rating, Comment: body.Comment, UserName: nameStr,
	}
	database.DB.Create(&r)
	c.JSON(http.StatusCreated, r)
}

// PUT /api/reviews/:id   (auth — author only)
func UpdateReview(c *gin.Context) {
	uid := c.GetUint("user_id")
	rid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var existing models.Review
	if err := database.DB.First(&existing, rid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}
	if existing.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only edit your own reviews"})
		return
	}
	var body struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Rating < 1 || body.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}
	existing.Rating = body.Rating
	existing.Comment = body.Comment
	database.DB.Save(&existing)
	c.JSON(http.StatusOK, existing)
}

// DELETE /api/reviews/:id   (auth — author or admin)
func DeleteReview(c *gin.Context) {
	uid := c.GetUint("user_id")
	role, _ := c.Get("role")
	rid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var existing models.Review
	if err := database.DB.First(&existing, rid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
		return
	}
	if role != "admin" && existing.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}
	database.DB.Delete(&existing)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// GET /api/reviews/mine   (auth) — current user's reviews
func MyReviews(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rs []models.Review
	database.DB.Where("user_id = ?", uid).Order("created_at desc").Find(&rs)
	if rs == nil {
		rs = []models.Review{}
	}
	c.JSON(http.StatusOK, rs)
}

// GET /api/admin/reviews   (admin only) — all reviews across all products
func AdminListReviews(c *gin.Context) {
	var list []models.Review
	database.DB.Order("created_at desc").Limit(100).Find(&list)
	c.JSON(http.StatusOK, gin.H{
		"reviews": list,
		"count":   len(list),
	})
}