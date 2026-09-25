package controllers

import (
	"net/http"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getOrCreateCart retrieves existing active cart or initializes a new one
func getOrCreateCart(userID uint) models.Cart {
	var cart models.Cart
	result := database.DB.Preload("Items.Product").Where("user_id = ?", userID).First(&cart)

	if result.Error != nil {
		// Cart does not exist, create a new cart for the user
		cart = models.Cart{UserID: userID}
		database.DB.Create(&cart)
	}
	return cart
}

// GET /api/cart - Fetch current user's active cart with populated items
func GetCart(c *gin.Context) {
	userID := c.GetUint("user_id")
	cart := getOrCreateCart(userID)
	c.JSON(http.StatusOK, cart)
}

type AddToCartInput struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  uint `json:"quantity" binding:"required"`
}

// POST /api/cart/add - Add product to user's cart with atomic inventory reservation
func AddToCart(c *gin.Context) {
	userID := c.GetUint("user_id")

	var input AddToCartInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify product exists
	database.EnsureAttached(database.DB)
	var product models.Product
	if err := database.DB.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	cart := getOrCreateCart(userID)

	// Check if product already exists in cart and compute requested quantity
	var existingItem models.CartItem
	err := database.DB.Where("cart_id = ? AND product_id = ?", cart.ID, input.ProductID).First(&existingItem).Error

	totalRequested := input.Quantity
	if err == nil {
		totalRequested += existingItem.Quantity
	}

	// Verify available stock considering existing cart reservation
	available := product.Stock
	if err == nil {
		available = product.Stock + existingItem.Quantity
	}

	if totalRequested > available {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "Not enough stock",
			"available": available,
		})
		return
	}

	// Atomically deduct inventory with condition to guarantee zero overselling
	res := database.DB.Model(&models.Product{}).
		Where("id = ? AND stock >= ?", product.ID, input.Quantity).
		Update("stock", gorm.Expr("stock - ?", input.Quantity))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not enough stock available",
		})
		return
	}

	if err == nil {
		// Existing item in cart: increment quantity
		existingItem.Quantity = totalRequested
		database.DB.Save(&existingItem)
	} else {
		// New item: create cart item record
		newItem := models.CartItem{
			CartID:    cart.ID,
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
		}
		database.DB.Create(&newItem)
	}

	updatedCart := getOrCreateCart(userID)
	database.InvalidateCachePrefix("products:")
	c.JSON(http.StatusOK, updatedCart)
}

type UpdateCartItemInput struct {
	Quantity uint `json:"quantity" binding:"required"`
}

// PUT /api/cart/item/:id - Update item quantity and adjust reserved inventory
func UpdateCartItem(c *gin.Context) {
	itemID := c.Param("id")

	var input UpdateCartItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var item models.CartItem
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
		return
	}

	database.EnsureAttached(database.DB)
	var product models.Product
	if err := database.DB.First(&product, item.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Calculate required stock delta
	needed := input.Quantity
	available := product.Stock + item.Quantity // Include currently reserved quantity in available inventory

	if needed > available {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "Not enough stock",
			"available": available,
		})
		return
	}

	// Apply inventory delta atomically
	delta := int(needed) - int(item.Quantity)
	if delta > 0 {
		res := database.DB.Model(&models.Product{}).
			Where("id = ? AND stock >= ?", product.ID, uint(delta)).
			Update("stock", gorm.Expr("stock - ?", uint(delta)))
		if res.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Not enough stock available",
			})
			return
		}
	} else if delta < 0 {
		release := uint(-delta)
		database.DB.Model(&models.Product{}).
			Where("id = ?", product.ID).
			Update("stock", gorm.Expr("stock + ?", release))
	}

	item.Quantity = input.Quantity
	database.DB.Save(&item)
	database.InvalidateCachePrefix("products:")

	c.JSON(http.StatusOK, item)
}

// DELETE /api/cart/item/:id - Remove item from cart and restore reserved inventory
func RemoveCartItem(c *gin.Context) {
	itemID := c.Param("id")

	var item models.CartItem
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
		return
	}

	// Restore reserved stock back to available product inventory
	database.DB.Model(&models.Product{}).Where("id = ?", item.ProductID).
		Update("stock", gorm.Expr("stock + ?", item.Quantity))

	database.DB.Delete(&item)
	database.InvalidateCachePrefix("products:")
	c.JSON(http.StatusOK, gin.H{"message": "Item removed from cart"})
}
