package controllers

import (
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// GET /api/wishlist - user-er wishlist
func GetWishlist(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	userID := c.GetUint("user_id")

	var items []models.WishlistItem
	database.DB.Preload("Product").Where("user_id = ?", userID).Find(&items)

	c.JSON(http.StatusOK, items)
}

type WishlistInput struct {
	ProductID uint `json:"product_id" binding:"required"`
}

// POST /api/wishlist/add
func AddToWishlist(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	userID := c.GetUint("user_id")

	var input WishlistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// product exists check
	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// duplicate check
	var existing models.WishlistItem
	if err := database.DB.Where("user_id = ? AND product_id = ?", userID, input.ProductID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Already in wishlist"})
		return
	}

	item := models.WishlistItem{UserID: userID, ProductID: input.ProductID}
	database.DB.Create(&item)

	c.JSON(http.StatusCreated, item)
}

// DELETE /api/wishlist/:id
func RemoveFromWishlist(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	userID := c.GetUint("user_id")
	itemID := c.Param("id")

	var item models.WishlistItem
	if err := database.DB.Where("id = ? AND user_id = ?", itemID, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wishlist item not found"})
		return
	}
	database.DB.Delete(&item)
	c.JSON(http.StatusOK, gin.H{"message": "Removed from wishlist"})
}
