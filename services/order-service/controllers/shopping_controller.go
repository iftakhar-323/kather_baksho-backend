package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// ---------- Guest checkout ----------

func newCode(prefix string) string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// POST /api/guest-checkout
// body: { name, phone, email, address_line1, address_line2, city, region,
//         postal_code, items:[{product_id,quantity}], notes }
type guestItemReq struct {
	ProductID uint `json:"product_id"`
	Quantity  uint `json:"quantity"`
}

func GuestCheckout(c *gin.Context) {
	var body struct {
		Name         string         `json:"name"`
		Phone        string         `json:"phone"`
		Email        string         `json:"email"`
		AddressLine1 string         `json:"address_line1"`
		AddressLine2 string         `json:"address_line2"`
		City         string         `json:"city"`
		Region       string         `json:"region"`
		PostalCode   string         `json:"postal_code"`
		Notes        string         `json:"notes"`
		Items        []guestItemReq `json:"items"`
	}
	if err := c.BindJSON(&body); err != nil || body.Name == "" || len(body.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name + items required"})
		return
	}

	// build order + items
	var subtotal float64
	type priced struct {
		ProductID uint    `json:"product_id"`
		Quantity  uint    `json:"quantity"`
		Price     float64 `json:"price"`
	}
	lines := make([]priced, 0, len(body.Items))
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
		sub := p.Price * float64(q)
		subtotal += sub
		lines = append(lines, priced{ProductID: p.ID, Quantity: q, Price: p.Price})
	}

	o := models.Order{
		UserID:     0, // guest
		TotalPrice: subtotal,
		Status:     "Pending",
	}
	if err := database.DB.Create(&o).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, ln := range lines {
		database.DB.Create(&models.OrderItem{
			OrderID: o.ID, ProductID: ln.ProductID, Quantity: ln.Quantity, Price: ln.Price,
		})
	}

	g := models.GuestOrder{
		OrderID: o.ID, Name: body.Name, Phone: body.Phone, Email: body.Email,
		AddressLine1: body.AddressLine1, AddressLine2: body.AddressLine2,
		City: body.City, Region: body.Region, PostalCode: body.PostalCode,
	}
	database.DB.Create(&g)

	c.JSON(http.StatusCreated, gin.H{
		"order_id":     o.ID,
		"guest_id":     g.ID,
		"subtotal":     subtotal,
		"tracking":     newCode("GUE"),
	})
}

// ---------- Gift cards ----------

// POST /api/gift-cards (admin)
func CreateGiftCard(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Code      string  `json:"code"`
		Balance   float64 `json:"balance"`
		IssuedTo  string  `json:"issued_to"`
		ExpiresAt string  `json:"expires_at"`
	}
	if err := c.BindJSON(&body); err != nil || body.Balance <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "balance > 0 required"})
		return
	}
	if body.Code == "" {
		body.Code = newCode("GC")
	}
	g := models.GiftCard{
		Code: body.Code, Balance: body.Balance, IssuedTo: body.IssuedTo,
		IssuedBy: uid, Active: true, ExpiresAt: body.ExpiresAt,
	}
	if err := database.DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, g)
}

// GET /api/gift-cards (admin)
func ListGiftCards(c *gin.Context) {
	var list []models.GiftCard
	database.DB.Order("created_at desc").Find(&list)
	if list == nil {
		list = []models.GiftCard{}
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/gift-cards/redeem  body: { code, amount }  (auth user)
func RedeemGiftCard(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Code   string  `json:"code"`
		Amount float64 `json:"amount"`
	}
	if err := c.BindJSON(&body); err != nil || body.Code == "" || body.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code + amount required"})
		return
	}
	var g models.GiftCard
	if err := database.DB.Where("code = ? AND active = ?", body.Code, true).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid code"})
		return
	}
	if g.ExpiresAt != "" {
		if exp, err := time.Parse("2006-01-02", g.ExpiresAt); err == nil && time.Now().After(exp) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "card expired"})
			return
		}
	}
	if g.Balance < body.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}
	newBal := g.Balance - body.Amount
	database.DB.Model(&g).Update("balance", newBal)
	// award points equivalent (1 BDT = 1 point)
	database.DB.Model(&models.User{}).Where("id = ?", uid).
		UpdateColumn("points", database.DB.Raw("points + ?", int(body.Amount)))
	c.JSON(http.StatusOK, gin.H{
		"redeemed":     body.Amount,
		"new_balance":  newBal,
		"points_added": int(body.Amount),
	})
}

// GET /api/gift-cards/balance/:code  (no auth — for checkout preview)
func GiftCardBalance(c *gin.Context) {
	code := c.Param("code")
	var g models.GiftCard
	if err := database.DB.Where("code = ? AND active = ?", code, true).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": g.Code, "balance": g.Balance, "expires_at": g.ExpiresAt})
}

// ---------- Shipping + Tax quote ----------

// quoteFromRules resolves a ShippingRule by subtotal (cheapest matching first)
func quoteShip(subtotal float64) (fee float64, ruleName string) {
	var rules []models.ShippingRule
	database.DB.Where("active = ?", true).Order("min_subtotal desc").Find(&rules)
	for _, r := range rules {
		if subtotal >= r.MinSubtotal {
			return r.FlatFee, r.Name
		}
	}
	// default fallback
	return 60, "Standard"
}

func quoteTax(subtotal float64) (tax float64, ruleName string) {
	var rules []models.TaxRule
	database.DB.Where("active = ?", true).Order("min_subtotal desc").Find(&rules)
	for _, r := range rules {
		if subtotal >= r.MinSubtotal {
			return subtotal * r.Percent / 100, r.Name
		}
	}
	return 0, "None"
}

// POST /api/quote  body: { subtotal }
// or  GET /api/quote?subtotal=1200
func Quote(c *gin.Context) {
	subtotal := 0.0
	if s := c.Query("subtotal"); s != "" {
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			subtotal = n
		}
	} else {
		var body struct {
			Subtotal float64 `json:"subtotal"`
		}
		_ = c.BindJSON(&body)
		subtotal = body.Subtotal
	}
	ship, shipRule := quoteShip(subtotal)
	tax, taxRule := quoteTax(subtotal)
	c.JSON(http.StatusOK, gin.H{
		"subtotal":       subtotal,
		"shipping":       ship,
		"shipping_rule":  shipRule,
		"tax":            tax,
		"tax_rule":       taxRule,
		"grand_total":    subtotal + ship + tax,
	})
}

// POST /api/shipping-rules (admin)
func CreateShippingRule(c *gin.Context) {
	var body struct {
		Name        string  `json:"name"`
		MinSubtotal float64 `json:"min_subtotal"`
		FlatFee     float64 `json:"flat_fee"`
		Active      bool    `json:"active"`
	}
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	r := models.ShippingRule{
		Name: body.Name, MinSubtotal: body.MinSubtotal, FlatFee: body.FlatFee, Active: body.Active,
	}
	database.DB.Create(&r)
	c.JSON(http.StatusCreated, r)
}

// GET /api/shipping-rules
func ListShippingRules(c *gin.Context) {
	var list []models.ShippingRule
	database.DB.Order("min_subtotal asc").Find(&list)
	if list == nil {
		list = []models.ShippingRule{}
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/tax-rules (admin)
func CreateTaxRule(c *gin.Context) {
	var body struct {
		Name        string  `json:"name"`
		MinSubtotal float64 `json:"min_subtotal"`
		Percent     float64 `json:"percent"`
		Active      bool    `json:"active"`
	}
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	r := models.TaxRule{
		Name: body.Name, MinSubtotal: body.MinSubtotal, Percent: body.Percent, Active: body.Active,
	}
	database.DB.Create(&r)
	c.JSON(http.StatusCreated, r)
}

// GET /api/tax-rules
func ListTaxRules(c *gin.Context) {
	var list []models.TaxRule
	database.DB.Order("min_subtotal asc").Find(&list)
	if list == nil {
		list = []models.TaxRule{}
	}
	c.JSON(http.StatusOK, list)
}