package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// GET all products with advanced filters, sorting, pagination.
//
// Query params (all optional):
//
//	search        — matches name, description, brand
//	category      — top-level: plant / decor / care
//	parent_category — nested category, e.g. "outdoor > flowering"
//	subcategory   — indoor_plant / outdoor_plant / plant_box / decor / soil / fertilizer / care_kit
//	indoor_outdoor — indoor / outdoor / both
//	difficulty    — easy / medium / hard
//	sunlight      — low / medium / bright / direct
//	water         — low / medium / high
//	humidity      — low / medium / high
//	pet_friendly  — true
//	brand         — exact
//	min_price / max_price
//	discount_only — true → only items with a discount
//	sort          — newest | price_asc | price_desc | name_asc | popular (view_count)
//	page, limit
func GetProducts(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	cacheKey := "products:query:" + c.Request.URL.RawQuery
	if c.Request.URL.RawQuery == "" {
		cacheKey = "products:query:default"
	}
	if cached, ok := database.CacheGet(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
		return
	}

	var products []models.Product
	q := database.DB.Model(&models.Product{})

	if s := c.Query("search"); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR description LIKE ? OR brand LIKE ?", like, like, like)
	}
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if pc := c.Query("parent_category"); pc != "" {
		q = q.Where("parent_category = ?", pc)
	}
	if sub := c.Query("subcategory"); sub != "" {
		q = q.Where("subcategory = ?", sub)
	}
	if io := c.Query("indoor_outdoor"); io != "" {
		q = q.Where("indoor_outdoor = ? OR indoor_outdoor = ?", io, "both")
	}
	if d := c.Query("difficulty"); d != "" {
		q = q.Where("difficulty = ?", d)
	}
	if s := c.Query("sunlight"); s != "" {
		q = q.Where("sunlight = ?", s)
	}
	if w := c.Query("water"); w != "" {
		q = q.Where("water = ?", w)
	}
	if h := c.Query("humidity"); h != "" {
		q = q.Where("humidity = ?", h)
	}
	if c.Query("pet_friendly") == "true" {
		q = q.Where("pet_friendly = ?", true)
	}
	if b := c.Query("brand"); b != "" {
		q = q.Where("brand = ?", b)
	}
	if v := c.Query("min_price"); v != "" {
		q = q.Where("price >= ?", v)
	}
	if v := c.Query("max_price"); v != "" {
		q = q.Where("price <= ?", v)
	}
	if c.Query("discount_only") == "true" {
		q = q.Where("discount_pct > 0 OR (compare_at_price > 0 AND compare_at_price > price)")
	}

	// sorting
	switch c.Query("sort") {
	case "price_asc":
		q = q.Order("price asc")
	case "price_desc":
		q = q.Order("price desc")
	case "name_asc":
		q = q.Order("name asc")
	case "popular":
		q = q.Order("view_count desc, id desc")
	case "newest":
		q = q.Order("id desc")
	default:
		// "featured": a stable pseudo-random order so page 1 shows a varied
		// mix of plants / decor / care instead of whichever category was
		// seeded last. Deterministic, so pagination stays consistent.
		q = q.Order("(id * 1103515245 + 12345) % 2147483647")
	}

	// pagination (defaults: page=1, limit=24, capped at 100)
	page := 1
	limit := 24
	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 24
	}
	if limit > 100 {
		limit = 100
	}

	var total int64
	q.Model(&models.Product{}).Count(&total)

	q.Limit(limit).Offset((page - 1) * limit).Find(&products)

	resp := gin.H{
		"items":       products,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
	}

	if jsonBytes, err := json.Marshal(resp); err == nil {
		database.CacheSet(cacheKey, string(jsonBytes), 2*time.Minute)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, resp)
}

// GET single product
func GetProduct(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	var product models.Product
	id := c.Param("id")
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	c.JSON(http.StatusOK, product)
}

// POST create product
func CreateProduct(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Auto-generate SKU and Slug if absent.
	if product.SKU == "" {
		product.SKU = generateSKU(product.Category, product.Name)
	}
	if product.Slug == "" {
		product.Slug = generateSlug(product.Name)
	}

	// Ensure unique slug (checking both active and soft-deleted products)
	baseSlug := product.Slug
	for i := 1; ; i++ {
		var count int64
		database.DB.Unscoped().Model(&models.Product{}).Where("slug = ?", product.Slug).Count(&count)
		if count == 0 {
			break
		}
		product.Slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	// Ensure unique SKU (checking both active and soft-deleted products)
	baseSKU := product.SKU
	for i := 1; ; i++ {
		var count int64
		database.DB.Unscoped().Model(&models.Product{}).Where("sku = ?", product.SKU).Count(&count)
		if count == 0 {
			break
		}
		product.SKU = fmt.Sprintf("%s-%d", baseSKU, i)
	}

	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product: " + err.Error()})
		return
	}
	database.InvalidateCachePrefix("products:")
	c.JSON(http.StatusCreated, product)
}

// PUT update product
func UpdateProduct(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	var product models.Product
	id := c.Param("id")
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product: " + err.Error()})
		return
	}
	database.InvalidateCachePrefix("products:")
	c.JSON(http.StatusOK, product)
}

// DELETE product
func DeleteProduct(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	var product models.Product
	id := c.Param("id")
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	if err := database.DB.Delete(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product: " + err.Error()})
		return
	}
	database.InvalidateCachePrefix("products:")
	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

// =====================================================================
// Helpers — slug + SKU generation (pure code, no external lib).
// =====================================================================

// slugRE matches any run of non-alphanumeric chars (after lowercasing) so we can
// collapse them to a single dash.
var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug turns "Snake Plant  'Laurentii'" → "snake-plant-laurentii".
func generateSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "product-" + time.Now().Format("20060102150405")
	}
	return s
}

// generateSKU returns "<CAT>-<hash>-<rand>" e.g. "PLANT-A3F1-7K".
// Both pieces are derived from name + category so re-creating the same
// product produces a stable-enough id for the catalogue.
func generateSKU(category, name string) string {
	cat := strings.ToUpper(category)
	if cat == "" {
		cat = "GEN"
	}
	if len(cat) > 4 {
		cat = cat[:4]
	}
	hash := 0
	for _, r := range strings.ToLower(name) {
		hash = hash*31 + int(r)
	}
	if hash < 0 {
		hash = -hash
	}
	short := strings.ToUpper(strconv.FormatInt(int64(hash)%0x10000, 16))
	for len(short) < 4 {
		short = "0" + short
	}
	return cat + "-" + short + "-" + strconv.FormatInt(time.Now().UnixNano()%1000, 10)
}

// =====================================================================
// Sprint D — Product extensions
// =====================================================================

// GET /api/products/autocomplete?q=...
//
// Returns up to 8 lightweight product cards (id + name + category) for
// typeahead. Pure code — just a LIKE on name/brand.
func Autocomplete(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, []models.Product{})
		return
	}
	like := "%" + strings.ToLower(q) + "%"
	var out []models.Product
	database.DB.
		Where("LOWER(name) LIKE ? OR LOWER(brand) LIKE ?", like, like).
		Order("view_count desc").
		Limit(8).
		Find(&out)
	c.JSON(http.StatusOK, out)
}

// GET /api/products/suggest?q=...
//
// Richer than autocomplete — includes subcategory + price + emoji placeholder.
func SuggestProducts(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	like := "%" + strings.ToLower(q) + "%"
	var out []models.Product
	database.DB.
		Where("LOWER(name) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(description) LIKE ?", like, like, like).
		Order("view_count desc").
		Limit(8).
		Find(&out)
	results := make([]gin.H, 0, len(out))
	for _, p := range out {
		results = append(results, gin.H{
			"id":          p.ID,
			"name":        p.Name,
			"slug":        p.Slug,
			"category":    p.Category,
			"subcategory": p.Subcategory,
			"price":       p.Price,
		})
	}
	c.JSON(http.StatusOK, results)
}

// POST /api/products/:id/view — increments view counter, logs a PageView row.
func TrackView(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	p.ViewCount++
	database.DB.Model(&p).Update("view_count", p.ViewCount)
	uid := uint(0)
	if v, ok := c.Get("user_id"); ok {
		uid = v.(uint)
	}
	database.DB.Create(&models.PageView{
		UserID:    uid,
		ProductID: p.ID,
		IP:        c.ClientIP(),
	})
	c.JSON(http.StatusOK, gin.H{"view_count": p.ViewCount})
}

// GET /api/products/slug/:slug — lookup by slug for SEO-friendly URLs.
func GetProductBySlug(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	slug := c.Param("slug")
	var p models.Product
	if err := database.DB.Where("slug = ?", slug).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// GET /api/products/related/:id — uses RelatedIDs (comma-separated).
func GetRelated(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	ids := parseIDList(p.RelatedIDs)
	if len(ids) == 0 {
		// fallback: same subcategory
		var fallback []models.Product
		database.DB.Where("subcategory = ? AND id <> ?", p.Subcategory, p.ID).
			Order("view_count desc").Limit(4).Find(&fallback)
		c.JSON(http.StatusOK, fallback)
		return
	}
	var rel []models.Product
	database.DB.Where("id IN ?", ids).Find(&rel)
	c.JSON(http.StatusOK, rel)
}

// GET /api/products/fbt/:id — frequently-bought-together.
func GetFBT(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	ids := parseIDList(p.FbtIDs)
	if len(ids) == 0 {
		c.JSON(http.StatusOK, []models.Product{})
		return
	}
	var out []models.Product
	database.DB.Where("id IN ?", ids).Find(&out)
	c.JSON(http.StatusOK, out)
}

// GET /api/products/best-selling — top 10 by total order quantity.
func BestSelling(c *gin.Context) {
	type Row struct {
		ProductID   uint    `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
		Subcategory string  `json:"subcategory"`
		ImageURL    string  `json:"image_url"`
		TotalQty    uint    `json:"total_qty"`
	}
	var rows []Row
	database.DB.
		Table("order_items").
		Select(`products.id, products.name, products.slug, products.price,
       products.category, products.subcategory, products.image_url,
       SUM(order_items.quantity) AS total_qty`).
		Joins("JOIN products ON products.id = order_items.product_id").
		Group("products.id").
		Order("total_qty desc").
		Limit(10).
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GET /api/products/most-viewed — top 10 by view_count.
func MostViewed(c *gin.Context) {
	var rows []models.Product
	database.DB.Order("view_count desc").Limit(10).Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// GET /api/products/brands — distinct brand names for filter dropdown.
func ListBrands(c *gin.Context) {
	var brands []string
	database.DB.Model(&models.Product{}).
		Where("brand <> ''").
		Distinct("brand").
		Pluck("brand", &brands)
	c.JSON(http.StatusOK, brands)
}

// GET /api/products/categories — nested category tree built from products.
func CategoryTree(c *gin.Context) {
	type Node struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
		Kids  []Node `json:"kids,omitempty"`
	}
	var products []models.Product
	database.DB.Find(&products)
	tree := map[string]*Node{}
	for _, p := range products {
		top := p.Category
		if top == "" {
			top = "Other"
		}
		if _, ok := tree[top]; !ok {
			tree[top] = &Node{Name: top}
		}
		tree[top].Count++
		pc := p.ParentCategory
		if pc == "" {
			pc = p.Subcategory
		}
		if pc == "" {
			continue
		}
		found := false
		for i := range tree[top].Kids {
			if tree[top].Kids[i].Name == pc {
				tree[top].Kids[i].Count++
				found = true
				break
			}
		}
		if !found {
			tree[top].Kids = append(tree[top].Kids, Node{Name: pc, Count: 1})
		}
	}
	out := make([]Node, 0, len(tree))
	for _, v := range tree {
		out = append(out, *v)
	}
	c.JSON(http.StatusOK, out)
}

// parseIDList splits "1,2,3" into []uint.
func parseIDList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil && n > 0 {
			out = append(out, uint(n))
		}
	}
	return out
}
