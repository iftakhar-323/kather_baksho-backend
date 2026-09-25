package controllers

import (
	"net/http"
	"strings"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

type ApplyCouponInput struct {
	Code         string  `json:"code" binding:"required"`
	OrderTotal   float64 `json:"order_total" binding:"required"`
}

// POST /api/coupons/apply - coupon code validate and return discount
func ApplyCoupon(c *gin.Context) {
	var input ApplyCouponInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	code := strings.ToUpper(strings.TrimSpace(input.Code))
	var coupon models.Coupon
	if err := database.DB.Where("code = ?", code).First(&coupon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid coupon code"})
		return
	}
	if !coupon.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coupon is not active"})
		return
	}
	if coupon.ExpiresAt != "" {
		exp, err := time.Parse("2006-01-02", coupon.ExpiresAt)
		if err == nil && time.Now().After(exp) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Coupon has expired"})
			return
		}
	}
	if input.OrderTotal < coupon.MinOrderTotal {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Order total below minimum",
			"min_order_total": coupon.MinOrderTotal,
		})
		return
	}

	discount := input.OrderTotal * coupon.DiscountPercent / 100
	c.JSON(http.StatusOK, gin.H{
		"code":              coupon.Code,
		"discount_percent":  coupon.DiscountPercent,
		"discount_amount":   discount,
		"new_total":         input.OrderTotal - discount,
	})
}

// POST /api/coupons - admin creates
func CreateCoupon(c *gin.Context) {
	var coupon models.Coupon
	if err := c.ShouldBindJSON(&coupon); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	coupon.Code = strings.ToUpper(strings.TrimSpace(coupon.Code))
	database.DB.Create(&coupon)
	c.JSON(http.StatusCreated, coupon)
}
// ListCoupons returns every coupon. Admin-only convenience for the admin UI.
func ListCoupons(c *gin.Context) {
        var rows []models.Coupon
        if err := database.DB.Order("id desc").Find(&rows).Error; err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusOK, gin.H{"items": rows, "total": len(rows)})
}

// DELETE /api/admin/coupons/:id - admin deletes a coupon
func DeleteCoupon(c *gin.Context) {
	id := c.Param("id")
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coupon not found"})
		return
	}
	database.DB.Delete(&coupon)
	c.JSON(http.StatusOK, gin.H{"message": "Coupon deleted successfully"})
}