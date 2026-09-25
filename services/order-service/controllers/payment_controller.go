package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/services/payment"
	"kather_baksho/sms"

	"github.com/gin-gonic/gin"
)

func getPaymentSecret() string {
	sec := os.Getenv("PAYMENT_SECRET")
	if sec == "" {
		sec = "katherbox_default_secure_payment_hmac_secret_2026"
	}
	return sec
}

// ComputeHMAC generates an HMAC-SHA256 signature for payload verification.
func ComputeHMAC(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

type InitiatePaymentInput struct {
	OrderID       uint   `json:"order_id" binding:"required"`
	PaymentMethod string `json:"payment_method"` // bkash, nagad, sslcommerz, card
}

// POST /api/payments/initiate
func InitiatePayment(c *gin.Context) {
	var input InitiatePaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", input.OrderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.PaymentStatus == "Paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is already paid"})
		return
	}

	method := input.PaymentMethod
	if method == "" {
		method = "bkash"
	}

	prov, err := payment.GetProvider(method)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := prov.Initiate(&order, "/api/payments/callback")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment: " + err.Error()})
		return
	}

	// Ensure session_id is populated
	if sess.SessionID == "" {
		sess.SessionID = fmt.Sprintf("PAY-SES-%d-%d", order.ID, time.Now().UnixNano())
	}

	// Always compute signature for the actual session_id
	payload := fmt.Sprintf("%d|%.2f|%s", order.ID, order.TotalPrice, sess.SessionID)
	signature := ComputeHMAC(payload, getPaymentSecret())
	sess.Signature = signature

	c.JSON(http.StatusOK, gin.H{
		"session_id":     sess.SessionID,
		"order_id":       order.ID,
		"amount":         order.TotalPrice,
		"payment_method": method,
		"signature":      signature,
		"gateway_url":    sess.GatewayURL,
		"provider":       prov.Name(),
	})
}

type PaymentCallbackInput struct {
	OrderID   uint   `json:"order_id" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
	Status    string `json:"status" binding:"required"` // SUCCESS, FAILED, CANCELLED
	Signature string `json:"signature" binding:"required"`
}

// POST /api/payments/callback
// Handles return callbacks and webhook deliveries with HMAC validation and idempotency.
func PaymentCallback(c *gin.Context) {
	var input PaymentCallbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").First(&order, input.OrderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Verify HMAC signature
	payload := fmt.Sprintf("%d|%.2f|%s", order.ID, order.TotalPrice, input.SessionID)
	expectedSignature := ComputeHMAC(payload, getPaymentSecret())
	if !hmac.Equal([]byte(input.Signature), []byte(expectedSignature)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid payment signature"})
		return
	}

	// Idempotency check: if order is already marked as paid, return success without duplicate side effects
	if order.PaymentStatus == "Paid" {
		c.JSON(http.StatusOK, gin.H{
			"message":        "Order already processed",
			"order_id":       order.ID,
			"payment_status": order.PaymentStatus,
			"status":         order.Status,
			"idempotent":     true,
		})
		return
	}

	if input.Status == "SUCCESS" {
		order.PaymentStatus = "Paid"
		order.Status = "Processing"
		database.DB.Save(&order)

		// Record order event timeline
		database.DB.Create(&models.OrderEvent{
			OrderID:   order.ID,
			Event:     "Payment Received",
			Note:      fmt.Sprintf("Payment of BDT %.2f verified via session %s", order.TotalPrice, input.SessionID),
			CreatedBy: order.UserID,
		})

		// Dispatch SMS notification for received payment
		if order.ShippingPhone != "" {
			sms.SendAsync(sms.BuildOrderMessage(order.ShippingPhone, fmt.Sprintf("%d", order.ID), fmt.Sprintf("%.2f", order.TotalPrice), sms.EventPaymentReceived, "bn"))
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Payment verified and order updated",
			"order_id":       order.ID,
			"payment_status": "Paid",
			"status":         "Processing",
		})
		return
	}

	// If failed, restore reserved stock and mark payment failed
	order.PaymentStatus = "Failed"
	order.Status = "Cancelled"
	database.DB.Save(&order)

	for _, item := range order.Items {
		database.DB.Model(&models.Product{}).Where("id = ?", item.ProductID).
			Update("stock", database.DB.Raw("stock + ?", item.Quantity))
	}

	database.DB.Create(&models.OrderEvent{
		OrderID:   order.ID,
		Event:     "Payment Failed",
		Note:      fmt.Sprintf("Payment failed for session %s, inventory restored", input.SessionID),
		CreatedBy: order.UserID,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "Payment failed, stock restored",
		"order_id":       order.ID,
		"payment_status": "Failed",
		"status":         "Cancelled",
	})
}

// GET /api/payments/status/:order_id
func GetPaymentStatus(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var order models.Order
	if err := database.DB.Select("id, total_price, payment_method, payment_status, status").First(&order, oid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":       order.ID,
		"total_price":    order.TotalPrice,
		"payment_method": order.PaymentMethod,
		"payment_status": order.PaymentStatus,
		"order_status":   order.Status,
	})
}

// GET /api/payments/simulate-gateway
func SimulateGenericGateway(c *gin.Context) {
	sessionID := c.Query("session_id")
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>Payment Gateway Simulator</title><style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#0f172a;color:#f8fafc;}.card{background:#1e293b;border:1px solid #334155;padding:2.5rem;border-radius:1rem;width:380px;text-align:center;box-shadow:0 20px 25px -5px rgba(0,0,0,0.5);}h2{color:#10b981;margin-top:0;}p{color:#94a3b8;font-size:0.9rem;}code{background:#0f172a;padding:0.2rem 0.4rem;border-radius:0.25rem;color:#38bdf8;}button{background:#10b981;color:#fff;border:none;padding:0.75rem 1.5rem;border-radius:0.5rem;cursor:pointer;font-weight:600;margin-top:1.5rem;width:100%%;}</style></head><body><div class="card"><h2>Kather Baksho Secure Payment</h2><p>Session ID:<br><code>%s</code></p><p>Authorized test transaction simulation ready.</p><button onclick="window.location.href='/dashboard'">Return to Dashboard</button></div></body></html>`, sessionID)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// GET /api/payments/bkash/simulate
func SimulateBKashGateway(c *gin.Context) {
	paymentID := c.Query("paymentID")
	orderID := c.Query("orderID")
	amount := c.Query("amount")

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>bKash Payment Simulation</title><meta name="viewport" content="width=device-width, initial-scale=1.0"><style>body{font-family:system-ui,sans-serif;background:#f3f4f6;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;}.box{background:#e2136e;color:#fff;border-radius:1rem;width:360px;padding:2rem;box-shadow:0 10px 25px rgba(226,19,110,0.3);text-align:center;}.logo{font-size:2rem;font-weight:900;margin-bottom:1rem;letter-spacing:1px;}.inner{background:#fff;color:#1e293b;border-radius:0.75rem;padding:1.5rem;margin-top:1rem;}input{width:100%%;box-sizing:border-box;padding:0.75rem;border:1px solid #cbd5e1;border-radius:0.5rem;margin:0.5rem 0;font-size:1rem;text-align:center;}button{width:100%%;padding:0.85rem;background:#e2136e;color:#fff;border:none;border-radius:0.5rem;font-size:1rem;font-weight:bold;cursor:pointer;margin-top:0.75rem;}.amount{font-size:1.5rem;font-weight:bold;color:#e2136e;}</style></head><body><div class="box"><div class="logo">bKash</div><p style="margin:0;font-size:0.9rem;">Merchant: <b>Kather Baksho</b></p><div class="inner"><p style="margin:0;color:#64748b;font-size:0.85rem;">Invoice: #%s</p><div class="amount">৳ %s</div><input type="text" placeholder="bKash Mobile No (01XXXXXXXXX)" value="01711223344" readonly><input type="password" placeholder="Enter PIN" value="12345" readonly><button onclick="execute()">CONFIRM PAYMENT</button></div></div><script>async function execute(){const payload={order_id:parseInt("%s"),session_id:"%s",status:"SUCCESS",payment_id:"%s",signature:"mock_bkash_sig"};const res=await fetch('/api/payments/callback',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({order_id:parseInt("%s"),session_id:"%s",status:"SUCCESS",signature:"mock_verified"})});alert("bKash Payment Completed Successfully!");window.location.href="/orders";}</script></body></html>`,
		orderID, amount, orderID, paymentID, paymentID, orderID, paymentID)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// GET /api/payments/sslcommerz/simulate
func SimulateSSLCommerzGateway(c *gin.Context) {
	tranID := c.Query("tran_id")
	orderID := c.Query("order_id")
	amount := c.Query("amount")

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>SSLCommerz Payment Gateway Simulation</title><meta name="viewport" content="width=device-width, initial-scale=1.0"><style>body{font-family:system-ui,sans-serif;background:#f8fafc;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;}.box{background:#fff;border:1px solid #e2e8f0;border-radius:1rem;width:420px;padding:2rem;box-shadow:0 10px 25px rgba(0,0,0,0.06);}.header{display:flex;justify-content:space-between;align-items:center;border-bottom:2px solid #f1f5f9;padding-bottom:1rem;margin-bottom:1.5rem;}.brand{font-size:1.4rem;font-weight:900;color:#0284c7;}.amount{font-size:1.4rem;font-weight:bold;color:#0f172a;}.card-opt{display:flex;gap:0.5rem;margin-bottom:1.5rem;}.opt{flex:1;border:1px solid #cbd5e1;padding:0.75rem;border-radius:0.5rem;text-align:center;font-size:0.85rem;font-weight:600;color:#475569;background:#f8fafc;}button{width:100%%;padding:0.85rem;background:#0284c7;color:#fff;border:none;border-radius:0.5rem;font-size:1rem;font-weight:bold;cursor:pointer;}button.fail{background:#ef4444;margin-top:0.5rem;}</style></head><body><div class="box"><div class="header"><div class="brand">SSLCOMMERZ</div><div class="amount">৳ %s</div></div><p style="color:#64748b;font-size:0.9rem;margin-top:0;">Payment for <b>Kather Baksho Order #%s</b><br><small>Tran ID: %s</small></p><div class="card-opt"><div class="opt" style="border-color:#0284c7;background:#f0f9ff;color:#0284c7;">Cards</div><div class="opt">Mobile Banking</div><div class="opt">Net Banking</div></div><button onclick="pay('SUCCESS')">PAY ৳ %s (DEMO SUCCESS)</button><button class="fail" onclick="pay('FAILED')">SIMULATE FAILED PAYMENT</button></div><script>async function pay(status){await fetch('/api/payments/callback',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({order_id:parseInt("%s"),session_id:"%s",status:status,signature:"mock_verified"})});alert("SSLCommerz Payment " + status);window.location.href="/orders";}</script></body></html>`,
		amount, orderID, tranID, amount, orderID, tranID)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
