package controllers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"kather_baksho/database"
	"kather_baksho/mailer"
	"kather_baksho/models"
	"kather_baksho/services"
	"kather_baksho/sms"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CheckoutInput struct {
	CouponCode      string `json:"coupon_code"`
	PointsToRedeem  uint   `json:"points_to_redeem"`
	GiftWrap        bool   `json:"gift_wrap"`
	PaymentMethod   string `json:"payment_method"`
	PaymentStatus   string `json:"payment_status"`
	TransactionID   string `json:"transaction_id"`
	ShippingName    string `json:"shipping_name"`
	ShippingPhone   string `json:"shipping_phone"`
	ShippingAddress string `json:"shipping_address"`
	DeliveryNote    string `json:"delivery_note"`
}

// POST /api/orders/checkout - Creates order from user cart
func Checkout(c *gin.Context) {
	userID := c.GetUint("user_id")

	var input CheckoutInput
	_ = c.ShouldBindJSON(&input) // optional body

	var cart models.Cart
	if err := database.DB.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart not found"})
		return
	}
	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	var totalPrice float64
	var orderItems []models.OrderItem
	for _, item := range cart.Items {
		totalPrice += item.Product.Price * float64(item.Quantity)
		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Product.Price,
		})
	}

	// coupon discount
	var couponDiscount float64
	var appliedCouponCode string
	if input.CouponCode != "" {
		var coupon models.Coupon
		if err := database.DB.Where("code = ?", input.CouponCode).First(&coupon).Error; err == nil && coupon.Active {
			valid := true
			if coupon.ExpiresAt != "" {
				exp, err := time.Parse("2006-01-02", coupon.ExpiresAt)
				if err == nil && time.Now().After(exp) {
					valid = false
				}
			}
			if totalPrice < coupon.MinOrderTotal {
				valid = false
			}
			if valid {
				couponDiscount = totalPrice * coupon.DiscountPercent / 100
				appliedCouponCode = coupon.Code
			}
		}
	}

	discount := couponDiscount

	// Green Points redemption: 1 point = 1 BDT discount
	if input.PointsToRedeem > 0 {
		var user models.User
		database.DB.First(&user, userID)
		redeem := float64(input.PointsToRedeem)
		if redeem > float64(user.Points) {
			redeem = float64(user.Points)
		}
		if redeem > totalPrice-discount {
			redeem = totalPrice - discount
		}
		discount += redeem
	}
	finalTotal := totalPrice - discount

	// Gift wrap adds 50 BDT to the final total.
	if input.GiftWrap {
		finalTotal += 50
	}
	if finalTotal < 0 {
		finalTotal = 0
	}

	// Shipping details: prefer what the checkout form sent, fall back to the
	// user's saved profile so an order always has a deliverable address.
	shipName := input.ShippingName
	shipPhone := input.ShippingPhone
	shipAddr := input.ShippingAddress
	if shipName == "" || shipPhone == "" || shipAddr == "" {
		var profile models.User
		if err := database.DB.First(&profile, userID).Error; err == nil {
			if shipName == "" {
				shipName = profile.Name
			}
			if shipPhone == "" {
				shipPhone = profile.Phone
			}
			if shipAddr == "" {
				shipAddr = profile.Address
			}
		}
	}

	payMethod := input.PaymentMethod
	if payMethod == "" {
		payMethod = "cod"
	}
	payStatus := input.PaymentStatus
	if payStatus == "" {
		if payMethod == "cod" {
			payStatus = "Pending COD"
		} else {
			payStatus = "Pending Payment"
		}
	}

	order := models.Order{
		UserID:          userID,
		TotalPrice:      finalTotal,
		Status:          "Pending",
		PaymentMethod:   payMethod,
		PaymentStatus:   payStatus,
		TransactionID:   input.TransactionID,
		ShippingName:    shipName,
		ShippingPhone:   shipPhone,
		ShippingAddress: shipAddr,
		DeliveryNote:    input.DeliveryNote,
		GiftWrap:        input.GiftWrap,
		CouponCode:      appliedCouponCode,
		DiscountAmount:  couponDiscount,
		Items:           orderItems,
	}
	database.DB.Create(&order)

	// Clear user cart items (use Unscoped to hard delete so items don't linger)
	database.DB.Unscoped().Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})

	// Green Points: 1 pt per 10 currency units spent (based on total before discount)
	earned := uint(totalPrice / 10)
	if earned > 0 {
		database.DB.Model(&models.User{}).Where("id = ?", userID).
			Update("points", gorm.Expr("points + ?", earned))
	}
	// deduct redeemed points
	if input.PointsToRedeem > 0 {
		database.DB.Model(&models.User{}).Where("id = ?", userID).
			Update("points", gorm.Expr("CASE WHEN points >= ? THEN points - ? ELSE 0 END",
				input.PointsToRedeem, input.PointsToRedeem))
	}

	// auto-create watering reminders for plant-category items
	for _, item := range cart.Items {
		var p models.Product
		database.DB.First(&p, item.ProductID)
		if p.Category == "plant" {
			next := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
			database.DB.Create(&models.CareReminder{
				UserID:       userID,
				ProductID:    p.ID,
				Type:         "watering",
				NextDueDate:  next,
				IntervalDays: 7,
			})
		}
	}

	database.DB.Preload("Items.Product").First(&order, order.ID)

	// Best-effort order-confirmation email. The send is safe to ignore
	// when no mailer key is configured.
	var u models.User
	if err := database.DB.Select("email").First(&u, userID).Error; err == nil && u.Email != "" {
		_ = mailer.Send(mailer.OrderPlaced(u.Email, uintToStr(order.ID), finalTotal))
	}

	// Dispatch asynchronous SMS confirmation
	if shipPhone != "" {
		sms.SendAsync(sms.BuildOrderMessage(shipPhone, uintToStr(order.ID), fmt.Sprintf("%.2f", finalTotal), sms.EventOrderPlaced, "bn"))
	}

	// Publish asynchronous 'order.created' event to Redis Streams message bus
	_, _ = services.PublishEvent("order.created", map[string]interface{}{
		"order_id":    order.ID,
		"user_id":     userID,
		"total_price": finalTotal,
		"items_count": len(order.Items),
	})

	c.JSON(http.StatusCreated, gin.H{
		"order":          order,
		"subtotal":       totalPrice,
		"discount_total": discount,
		"points_earned":  earned,
	})
}

// GET /api/orders - user-er nijer shob order dekhbe
func GetMyOrders(c *gin.Context) {
	userID := c.GetUint("user_id")

	var orders []models.Order
	database.DB.Preload("Items.Product").Where("user_id = ?", userID).Order("id desc").Find(&orders)

	c.JSON(http.StatusOK, orders)
}

// GET /api/orders/:id - single order details
func GetOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Items.Product").Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

type UpdateStatusInput struct {
	Status string `json:"status" binding:"required"`
}

// PUT /api/orders/:id/status - admin order status update korbe
func UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")

	var input UpdateStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	oldStatus := order.Status
	order.Status = input.Status
	if err := database.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	// Stock restoration on cancellation, re-deduction on reinstatement
	if input.Status == "Cancelled" && oldStatus != "Cancelled" {
		for _, it := range order.Items {
			database.DB.Model(&models.Product{}).
				Where("id = ?", it.ProductID).
				Update("stock", gorm.Expr("stock + ?", it.Quantity))
		}
	} else if oldStatus == "Cancelled" && input.Status != "Cancelled" {
		for _, it := range order.Items {
			database.DB.Model(&models.Product{}).
				Where("id = ?", it.ProductID).
				Update("stock", gorm.Expr("CASE WHEN stock >= ? THEN stock - ? ELSE 0 END", it.Quantity, it.Quantity))
		}
	}

	// notify user + console-log (simulated SMS/email per spec)
	database.DB.Create(&models.Notification{
		UserID:  order.UserID,
		Message: "Order #" + uintToStr(order.ID) + " is now " + order.Status,
		Type:    "order_update",
	})
	log.Printf("Notify user %d: order #%d status changed to %s", order.UserID, order.ID, order.Status)

	if order.ShippingPhone != "" {
		switch input.Status {
		case "Processing", "Shipped":
			sms.SendAsync(sms.BuildOrderMessage(order.ShippingPhone, uintToStr(order.ID), "", sms.EventOutForDelivery, "bn"))
		case "Delivered":
			sms.SendAsync(sms.BuildOrderMessage(order.ShippingPhone, uintToStr(order.ID), "", sms.EventOrderDelivered, "bn"))
		}
	}

	c.JSON(http.StatusOK, order)
}

// (uint -> string helper to avoid extra strconv import line)
func uintToStr(n uint) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// GET /api/orders/all - shob order dekhe admin (admin only)
func GetAllOrders(c *gin.Context) {
	var orders []models.Order
	database.DB.Preload("Items.Product").Order("id desc").Find(&orders)
	c.JSON(http.StatusOK, orders)
}

// DELETE /api/orders/:id - admin order delete korte parbe
func DeleteOrder(c *gin.Context) {
	orderID := c.Param("id")
	var order models.Order
	if err := database.DB.Preload("Items").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// restore stock so totals stay consistent only if not already Cancelled
	if order.Status != "Cancelled" {
		for _, it := range order.Items {
			database.DB.Model(&models.Product{}).
				Where("id = ?", it.ProductID).
				Update("stock", gorm.Expr("stock + ?", it.Quantity))
		}
	}

	// Delete order items and parent order record
	database.DB.Where("order_id = ?", order.ID).Delete(&models.OrderItem{})
	database.DB.Delete(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Order deleted"})
}

// AdminOrderItemInput — single line the admin enters into the new order form.
type AdminOrderItemInput struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  uint `json:"quantity" binding:"required,min=1"`
}

// CreateAdminOrderInput — payload for POST /api/admin/orders.
// Admin can place an order on behalf of any user (e.g. phone orders).
type CreateAdminOrderInput struct {
	UserID   uint                  `json:"user_id" binding:"required"`
	Items    []AdminOrderItemInput `json:"items" binding:"required,min=1,dive"`
	Status   string                `json:"status"`
	GiftWrap bool                  `json:"gift_wrap"`
}

// POST /api/admin/orders - admin creates an order on behalf of a user.
func CreateAdminOrder(c *gin.Context) {
	var input CreateAdminOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// user must exist
	var user models.User
	if err := database.DB.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// resolve products and compute total, with stock check
	var orderItems []models.OrderItem
	var totalPrice float64
	for _, line := range input.Items {
		var p models.Product
		if err := database.DB.First(&p, line.ProductID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product " + uintToStr(line.ProductID) + " not found"})
			return
		}
		if p.Stock < line.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock for " + p.Name})
			return
		}
		orderItems = append(orderItems, models.OrderItem{
			ProductID: p.ID,
			Quantity:  line.Quantity,
			Price:     p.Price,
		})
		totalPrice += p.Price * float64(line.Quantity)
	}

	status := input.Status
	if status == "" {
		status = "Pending"
	}
	if status != "Pending" && status != "Processing" && status != "Packed" &&
		status != "On the Way" && status != "Delivered" && status != "Cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	order := models.Order{
		UserID:     input.UserID,
		TotalPrice: totalPrice,
		Status:     status,
		GiftWrap:   input.GiftWrap,
		Items:      orderItems,
	}
	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// decrement stock for each line
	for _, line := range input.Items {
		database.DB.Model(&models.Product{}).
			Where("id = ?", line.ProductID).
			Update("stock", gorm.Expr("CASE WHEN stock >= ? THEN stock - ? ELSE stock END",
				line.Quantity, line.Quantity))
	}

	// notify the user
	database.DB.Create(&models.Notification{
		UserID:  order.UserID,
		Message: "Order #" + uintToStr(order.ID) + " was placed on your behalf (status: " + order.Status + ")",
		Type:    "order_update",
	})

	database.DB.Preload("Items.Product").First(&order, order.ID)
	c.JSON(http.StatusCreated, gin.H{"order": order})
}
