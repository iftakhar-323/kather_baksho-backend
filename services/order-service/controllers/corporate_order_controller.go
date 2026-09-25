package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// ---------- Corporate orders (B2B bulk) ----------

type corporateItemReq struct {
	ProductID uint `json:"product_id"`
	Quantity  uint `json:"quantity"`
}

// POST /api/corporate/orders
func CreateCorporateOrder(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		QuoteID      uint               `json:"quote_id"`
		CompanyName  string             `json:"company_name"`
		EventType    string             `json:"event_type"`
		BrandingNote string             `json:"branding_note"`
		Items        []corporateItemReq `json:"items"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.CompanyName == "" || len(body.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_name + items required"})
		return
	}

	type priced struct {
		Name      string  `json:"name"`
		Address   string  `json:"address"`
		ProductID uint    `json:"product_id"`
		Qty       uint    `json:"qty"`
		UnitPrice float64 `json:"unit_price"`
		Subtotal  float64 `json:"subtotal"`
	}
	pricedItems := make([]priced, 0, len(body.Items))
	var subtotal float64
	for _, it := range body.Items {
		var p models.Product
		if err := database.DB.First(&p, it.ProductID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
			return
		}
		q := it.Quantity
		if q == 0 {
			q = 1
		}
		unit := p.Price
		if q >= 100 {
			unit *= 0.85
		} else if q >= 50 {
			unit *= 0.90
		} else if q >= 10 {
			unit *= 0.95
		}
		sub := unit * float64(q)
		subtotal += sub
		pricedItems = append(pricedItems, priced{
			Name: p.Name, ProductID: p.ID, Qty: q,
			UnitPrice: unit, Subtotal: sub,
		})
	}
	itemsJSON, _ := json.Marshal(pricedItems)

	o := models.CorporateOrder{
		QuoteID:      body.QuoteID,
		UserID:       uid,
		CompanyName:  body.CompanyName,
		Status:       "pending_approval",
		Items:        string(itemsJSON),
		TotalAmount:  subtotal,
		BrandingNote: body.BrandingNote,
	}
	if err := database.DB.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, o)
}

// GET /api/corporate/orders
func ListCorporateOrders(c *gin.Context) {
	uid := c.GetUint("user_id")
	q := database.DB.Order("created_at desc")
	if !userIsAdmin(c) {
		q = q.Where("user_id = ?", uid)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var list []models.CorporateOrder
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// GET /api/corporate/orders/:id
func GetCorporateOrder(c *gin.Context) {
	uid := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var o models.CorporateOrder
	if err := database.DB.First(&o, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if o.UserID != uid && !userIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not yours"})
		return
	}
	c.JSON(http.StatusOK, o)
}

// PATCH /api/corporate/orders/:id/status  (admin)
func UpdateCorporateOrderStatus(c *gin.Context) {
	if !userIsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	uid := c.GetUint("user_id")
	var body struct {
		Status    string  `json:"status"`
		AdminNote string  `json:"admin_note"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	updates := map[string]interface{}{
		"status":     body.Status,
		"updated_at": time.Now(),
	}
	if body.AdminNote != "" {
		updates["admin_note"] = body.AdminNote
	}
	if body.Status == "approved" || body.Status == "dispatched" || body.Status == "delivered" {
		updates["approved_by"] = uid
	}
	if err := database.DB.Model(&models.CorporateOrder{}).Where("id = ?", id).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/corporate/quote (preview, no DB write)
func CorporateQuote(c *gin.Context) {
	var body struct {
		EventType string             `json:"event_type"`
		Items     []corporateItemReq `json:"items"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var subtotal float64
	var lines []gin.H
	for _, it := range body.Items {
		var p models.Product
		if err := database.DB.First(&p, it.ProductID).Error; err != nil {
			lines = append(lines, gin.H{"product_id": it.ProductID, "error": "not found"})
			continue
		}
		q := it.Quantity
		if q == 0 {
			q = 1
		}
		unit := p.Price
		discount := 0.0
		if q >= 100 {
			discount = 0.15
		} else if q >= 50 {
			discount = 0.10
		} else if q >= 10 {
			discount = 0.05
		}
		if discount > 0 {
			unit *= 1 - discount
		}
		sub := unit * float64(q)
		subtotal += sub
		lines = append(lines, gin.H{
			"product_id": p.ID, "name": p.Name, "quantity": q,
			"unit_price": p.Price, "discount_pct": discount * 100,
			"subtotal": sub,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"event_type": body.EventType,
		"items":      lines,
		"subtotal":   subtotal,
	})
}