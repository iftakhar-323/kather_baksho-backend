package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// ---------- helpers ----------

func userIsAdmin(c *gin.Context) bool {
	if v, ok := c.Get("role"); ok && v == "admin" {
		return true
	}
	return false
}

// userIsStaff is true for both "staff" and "admin" — the fulfillment tier.
func userIsStaff(c *gin.Context) bool {
	if v, ok := c.Get("role"); ok && (v == "staff" || v == "admin") {
		return true
	}
	return false
}

// ---------- Order Event timeline ----------

// POST /api/orders/:id/events
func AddOrderEvent(c *gin.Context) {
	uid := c.GetUint("user_id")
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	var body struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.Status == "" {
		body.Status = "note"
	}
	ev := models.OrderEvent{
		OrderID:   uint(oid),
		Event:     body.Status,
		Note:      body.Message,
		CreatedBy: uid,
	}
	if err := database.DB.Create(&ev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if body.Status != "note" {
		database.DB.Model(&models.Order{}).Where("id = ?", oid).Update("status", body.Status)
	}
	c.JSON(http.StatusCreated, ev)
}

// GET /api/orders/:id/events
func ListOrderEvents(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var evs []models.OrderEvent
	database.DB.Where("order_id = ?", oid).Order("created_at desc").Find(&evs)
	c.JSON(http.StatusOK, evs)
}

// ---------- Estimated Delivery ----------
// EstimatedDelivery returns a status + estimated delivery window for the order.
// Local rule (no external APIs): Dhaka=2 days, other BD cities=4 days, remote=7 days
// If order already shipped/delivered, returns the actual recorded event date.
func EstimatedDelivery(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	uid := c.GetUint("user_id")
	var o models.Order
	if err := database.DB.First(&o, oid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if o.UserID != uid && !userIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not yours"})
		return
	}
	// If order has shipped/delivered/cancelled, prefer the recorded event
	var lastShip models.OrderEvent
	hasShip := database.DB.Where("order_id = ? AND event in ?", oid, []string{"shipped", "delivered"}).
		Order("created_at desc").First(&lastShip).Error == nil
	now := time.Now()
	type est struct {
		Status       string `json:"status"`
		Earliest     string `json:"earliest"`
		Latest       string `json:"latest"`
		BusinessDays int    `json:"business_days"`
		ShippedAt    string `json:"shipped_at,omitempty"`
		DeliveredAt  string `json:"delivered_at,omitempty"`
		Note         string `json:"note"`
	}
	out := est{Status: o.Status}
	switch o.Status {
	case "delivered":
		var ev models.OrderEvent
		if database.DB.Where("order_id = ? AND event = ?", oid, "delivered").Order("created_at desc").First(&ev).Error == nil {
			out.DeliveredAt = ev.CreatedAt.Format("02 Jan 2006")
			out.Status = "delivered"
		}
		out.Note = "Your order has been delivered."
	case "cancelled":
		out.Note = "Order cancelled — no delivery expected."
	default:
		// Look up the address from the linked GuestOrder (if guest checkout)
		var guest models.GuestOrder
		hasGuest := database.DB.Where("order_id = ?", oid).First(&guest).Error == nil
		addr := ""
		if hasGuest {
			addr = guest.City + " " + guest.Region + " " + guest.AddressLine1
		}
		// Local rule: Dhaka=2, major cities=4, remote=7, unknown=4
		transit := 4
		if containsCI(addr, "dhaka") {
			transit = 2
		} else if containsCI(addr, "chittagong") || containsCI(addr, "chattogram") || containsCI(addr, "sylhet") || containsCI(addr, "rajshahi") || containsCI(addr, "khulna") || containsCI(addr, "barisal") || containsCI(addr, "barishal") || containsCI(addr, "rangpur") || containsCI(addr, "mymensingh") {
			transit = 4
		} else if containsCI(addr, "cox") || containsCI(addr, "bandarban") || containsCI(addr, "rangamati") || containsCI(addr, "khagrachari") || containsCI(addr, "st. martin") {
			transit = 7
		} else if addr == "" {
			transit = 4
		}
		out.BusinessDays = transit
		out.Earliest = now.AddDate(0, 0, transit).Format("02 Jan 2006")
		out.Latest = now.AddDate(0, 0, transit+1).Format("02 Jan 2006")
		if hasShip {
			out.ShippedAt = lastShip.CreatedAt.Format("02 Jan 2006 15:04")
			earliest := lastShip.CreatedAt.AddDate(0, 0, transit)
			latest := lastShip.CreatedAt.AddDate(0, 0, transit+1)
			out.Earliest = earliest.Format("02 Jan 2006")
			out.Latest = latest.Format("02 Jan 2006")
			out.Note = "Shipped — delivery window below."
		} else {
			out.Note = "Estimated delivery window (Dhaka 2d · other cities 4d · remote 7d)."
		}
	}
	c.JSON(http.StatusOK, out)
}

// containsCI helper is defined in gift_controller.go (shared).

// ---------- Return / Refund / Exchange ----------

// POST /api/returns
func CreateReturnRequest(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		OrderID   uint   `json:"order_id"`
		ProductID uint   `json:"product_id"`
		Quantity  int    `json:"quantity"`
		Reason    string `json:"reason"`
		Type      string `json:"type"` // refund | exchange
		Notes     string `json:"notes"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.Type == "" {
		body.Type = "refund"
	}
	if body.Quantity <= 0 {
		body.Quantity = 1
	}
	r := models.ReturnRequest{
		UserID:    uid,
		OrderID:   body.OrderID,
		ProductID: body.ProductID,
		Quantity:  body.Quantity,
		Reason:    body.Reason,
		Type:      body.Type,
		Notes:     body.Notes,
		Status:    "pending",
	}
	if err := database.DB.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&models.OrderEvent{
		OrderID:   body.OrderID,
		Event:     "return_requested",
		Note:      fmt.Sprintf("%s requested (%s)", body.Type, body.Reason),
		CreatedBy: uid,
	})
	c.JSON(http.StatusCreated, r)
}

// GET /api/returns
func ListReturns(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rs []models.ReturnRequest
	q := database.DB.Order("created_at desc")
	if !userIsStaff(c) {
		q = q.Where("user_id = ?", uid)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Find(&rs)
	c.JSON(http.StatusOK, rs)
}

// PATCH /api/returns/:id  (staff / admin)
func UpdateReturnRequest(c *gin.Context) {
	if !userIsStaff(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "staff only"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Status       string  `json:"status"`
		AdminNote    string  `json:"admin_note"`
		RefundAmount float64 `json:"refund_amount"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	updates := map[string]interface{}{"status": body.Status, "updated_at": time.Now()}
	if body.AdminNote != "" {
		updates["admin_note"] = body.AdminNote
	}
	if body.RefundAmount > 0 {
		updates["refund_amount"] = body.RefundAmount
	}
	if err := database.DB.Model(&models.ReturnRequest{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var r models.ReturnRequest
	database.DB.First(&r, id)
	if r.OrderID > 0 {
		database.DB.Create(&models.OrderEvent{
			OrderID: r.OrderID,
			Event:   "return_" + body.Status,
			Note:    fmt.Sprintf("Return #%d → %s", r.ID, body.Status),
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// shipToLine renders an optional "Ship to" paragraph for the invoice.
func shipToLine(o models.Order) string {
	if o.ShippingAddress == "" && o.ShippingName == "" {
		return ""
	}
	parts := o.ShippingName
	if o.ShippingPhone != "" {
		parts += " · " + o.ShippingPhone
	}
	line := fmt.Sprintf("<p><strong>Ship to:</strong> %s<br>%s", parts, o.ShippingAddress)
	if o.DeliveryNote != "" {
		line += fmt.Sprintf("<br><em class=\"muted\">Note: %s</em>", o.DeliveryNote)
	}
	return line + "</p>"
}

// ---------- Invoice HTML (browser-printable) ----------

func InvoiceHTML(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}
	uid := c.GetUint("user_id")
	var o models.Order
	if err := database.DB.First(&o, oid).Error; err != nil {
		c.String(http.StatusNotFound, "order not found")
		return
	}
	if o.UserID != uid && !userIsAdmin(c) {
		c.String(http.StatusForbidden, "not yours")
		return
	}
	var items []models.OrderItem
	database.DB.Where("order_id = ?", oid).Find(&items)
	type row struct {
		Name     string
		Quantity uint
		Price    float64
		Subtotal float64
	}
	rows := make([]row, 0, len(items))
	var total float64
	for _, it := range items {
		var p models.Product
		database.DB.First(&p, it.ProductID)
		sub := float64(it.Quantity) * it.Price
		total += sub
		rows = append(rows, row{p.Name, it.Quantity, it.Price, sub})
	}
	if total == 0 {
		total = o.TotalPrice
	}
	buyer := fmt.Sprintf("Customer #%d", o.UserID)
	if o.UserID > 0 {
		var u models.User
		if database.DB.First(&u, o.UserID).Error == nil {
			buyer = u.Name
			if buyer == "" {
				buyer = u.Email
			}
		}
	}
	html := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Invoice #%d</title>
<style>
body{font-family:Arial,sans-serif;max-width:780px;margin:30px auto;color:#222}
h1{color:#2d6a4f;border-bottom:2px solid #2d6a4f;padding-bottom:8px}
table{width:100%%;border-collapse:collapse;margin-top:18px}
th,td{padding:10px 8px;border-bottom:1px solid #ddd;text-align:left}
th{background:#f6f6f6}
.total{font-weight:700;font-size:18px;margin-top:18px;text-align:right}
.muted{color:#777;font-size:13px}
.btn{background:#2d6a4f;color:#fff;border:none;padding:10px 18px;border-radius:6px;cursor:pointer;margin-top:14px}
@media print{.btn{display:none}}
</style></head><body>
<h1>Kather Baksho — Invoice</h1>
<p class="muted">Invoice #INV-%05d · %s</p>
<p><strong>Billed to:</strong> %s</p>
%s
<table>
<tr><th>Item</th><th>Qty</th><th>Price</th><th>Subtotal</th></tr>
`, oid, oid, o.CreatedAt.Format("02 Jan 2006"), buyer, shipToLine(o))
	for _, r := range rows {
		html += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>BDT %.2f</td><td>BDT %.2f</td></tr>",
			r.Name, r.Quantity, r.Price, r.Subtotal)
	}
	html += fmt.Sprintf(`</table>
<p class="total">Total: BDT %.2f</p>
<p class="muted">Status: %s</p>
<button class="btn" onclick="window.print()">Print / Save as PDF</button>
</body></html>`, total, o.Status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// InvoicePDF generates and streams an official branded PDF invoice rendered by the TypeScript worker microservice.
// GET /api/orders/:id/invoice/pdf
func InvoicePDF(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	uid := c.GetUint("user_id")
	var o models.Order
	if err := database.DB.First(&o, oid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if o.UserID != uid && !userIsAdmin(c) && !userIsStaff(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var items []models.OrderItem
	database.DB.Where("order_id = ?", oid).Find(&items)

	type workerItem struct {
		ID          uint    `json:"id"`
		ProductID   uint    `json:"productId"`
		ProductName string  `json:"productName"`
		Quantity    uint    `json:"quantity"`
		Price       float64 `json:"price"`
		Subtotal    float64 `json:"subtotal"`
	}

	workerItems := make([]workerItem, 0, len(items))
	var calculatedSubtotal float64
	for _, it := range items {
		var p models.Product
		database.DB.First(&p, it.ProductID)
		sub := float64(it.Quantity) * it.Price
		calculatedSubtotal += sub
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("Botanical Item #%d", it.ProductID)
		}
		workerItems = append(workerItems, workerItem{
			ID:          it.ID,
			ProductID:   it.ProductID,
			ProductName: name,
			Quantity:    it.Quantity,
			Price:       it.Price,
			Subtotal:    sub,
		})
	}

	buyerName := fmt.Sprintf("Customer #%d", o.UserID)
	buyerEmail := ""
	if o.UserID > 0 {
		var u models.User
		if database.DB.First(&u, o.UserID).Error == nil {
			if u.Name != "" {
				buyerName = u.Name
			}
			buyerEmail = u.Email
		}
	}
	if o.ShippingName != "" {
		buyerName = o.ShippingName
	}

	payload := map[string]interface{}{
		"orderId":         o.ID,
		"orderNumber":     fmt.Sprintf("INV-%06d", o.ID),
		"customerName":    buyerName,
		"customerEmail":   buyerEmail,
		"customerPhone":   o.ShippingPhone,
		"shippingAddress": o.ShippingAddress,
		"paymentMethod":   o.PaymentMethod,
		"paymentStatus":   o.PaymentStatus,
		"createdAt":       o.CreatedAt.Format(time.RFC3339),
		"items":           workerItems,
		"subtotal":        calculatedSubtotal,
		"discount":        o.DiscountAmount,
		"deliveryFee":     0.0,
		"total":           o.TotalPrice,
	}

	workerURL := os.Getenv("WORKER_TS_URL")
	if workerURL == "" {
		workerURL = "http://worker-ts:8083"
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode payload"})
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(workerURL+"/api/v1/invoices/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Fallback to local host port if workerURL is not resolvable outside docker
		if workerURL != "http://localhost:8083" {
			resp, err = client.Post("http://localhost:8083/api/v1/invoices/generate", "application/json", bytes.NewBuffer(jsonData))
		}
	}

	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "TypeScript worker invoice service temporarily unavailable",
			"details": fmt.Sprintf("%v", err),
		})
		return
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read generated PDF stream"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"invoice_order_%d.pdf\"", o.ID))
	c.Header("X-Generated-By", "kather_baksho-worker-ts")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ---------- Receipt (compact) ----------

func ReceiptHTML(c *gin.Context) {
	oid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}
	uid := c.GetUint("user_id")
	var o models.Order
	if err := database.DB.First(&o, oid).Error; err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}
	if o.UserID != uid && !userIsAdmin(c) {
		c.String(http.StatusForbidden, "not yours")
		return
	}
	buyer := fmt.Sprintf("Customer #%d", o.UserID)
	if o.UserID > 0 {
		var u models.User
		if database.DB.First(&u, o.UserID).Error == nil {
			buyer = u.Name
			if buyer == "" {
				buyer = u.Email
			}
		}
	}
	html := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Receipt #%d</title>
<style>body{font-family:Arial;max-width:520px;margin:30px auto;padding:20px;border:1px solid #ddd;border-radius:8px}
h2{color:#2d6a4f;text-align:center}
.row{display:flex;justify-content:space-between;padding:6px 0;border-bottom:1px dotted #ddd}
.bt{font-weight:700;font-size:18px;margin-top:12px}
.muted{color:#888;font-size:12px;text-align:center;margin-top:18px}</style></head><body>
<h2>Kather Baksho — Receipt</h2>
<p class="muted">Order #%05d · %s</p>
<div class="row"><span>Customer</span><span>%s</span></div>
<div class="row bt"><span>Total Paid</span><span>BDT %.2f</span></div>
<div class="row"><span>Status</span><span>%s</span></div>
<p class="muted">Thank you for supporting Kather Baksho 🌿</p>
</body></html>`, oid, oid, o.CreatedAt.Format("02 Jan 2006 15:04"),
		buyer, o.TotalPrice, o.Status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
