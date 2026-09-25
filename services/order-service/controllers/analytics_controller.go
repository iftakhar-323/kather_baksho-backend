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

// ---------- Analytics dashboard ----------

// GET /api/analytics/summary?days=30
// Returns: total_revenue, total_orders, total_customers, avg_order, daily[]
func AnalyticsSummary(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).
		Where("created_at >= ? AND status <> ?", since, "cancelled").
		Select("COALESCE(SUM(total_price),0)").Row().Scan(&totalRevenue)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Where("created_at >= ?", since).Count(&totalOrders)

	var totalCustomers int64
	database.DB.Model(&models.User{}).Where("created_at >= ?", since).Count(&totalCustomers)

	var avg float64
	if totalOrders > 0 {
		avg = totalRevenue / float64(totalOrders)
	}

	// daily revenue buckets
	type bucket struct {
		Date  string  `json:"date"`
		Total float64 `json:"total"`
		Count int     `json:"count"`
	}
	rows, err := database.DB.Raw(`
		SELECT strftime('%Y-%m-%d', created_at) as date,
		       COALESCE(SUM(total_price),0) as total,
		       COUNT(*) as count
		FROM orders
		WHERE created_at >= ? AND status <> 'cancelled'
		GROUP BY date ORDER BY date ASC
	`, since).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	daily := make([]bucket, 0, days)
	for rows.Next() {
		var b bucket
		if err := rows.Scan(&b.Date, &b.Total, &b.Count); err == nil {
			daily = append(daily, b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"days":            days,
		"total_revenue":   totalRevenue,
		"total_orders":    totalOrders,
		"total_customers": totalCustomers,
		"avg_order":       avg,
		"daily":           daily,
	})
}

// GET /api/analytics/top-customers?limit=10
func TopCustomers(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	type row struct {
		UserID     uint    `json:"user_id"`
		Name       string  `json:"name"`
		Email      string  `json:"email"`
		OrderCount int     `json:"order_count"`
		TotalSpend float64 `json:"total_spend"`
	}
	var rows []row
	database.DB.Raw(`
		SELECT u.id as user_id, COALESCE(u.name,'') as name, u.email as email,
		       COUNT(o.id) as order_count,
		       COALESCE(SUM(o.total_price),0) as total_spend
		FROM users u
		JOIN orders o ON o.user_id = u.id
		WHERE o.status <> 'cancelled'
		GROUP BY u.id
		ORDER BY total_spend DESC
		LIMIT ?`, limit).Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, rows)
}

// GET /api/analytics/inventory  (low-stock report)
func InventoryReport(c *gin.Context) {
	threshold := 5
	if t := c.Query("threshold"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n >= 0 {
			threshold = n
		}
	}
	var ps []models.Product
	database.DB.Where("stock <= ?", threshold).Order("stock asc").Find(&ps)
	if ps == nil {
		ps = []models.Product{}
	}
	c.JSON(http.StatusOK, gin.H{"threshold": threshold, "low_stock": ps})
}

// GET /api/analytics/traffic?days=30
func TrafficReport(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)
	type row struct {
		Date  string `json:"date"`
		Views int    `json:"views"`
	}
	var rows []row
	database.DB.Raw(`
		SELECT strftime('%Y-%m-%d', created_at) as date, COUNT(*) as views
		FROM page_views WHERE created_at >= ? GROUP BY date ORDER BY date ASC`, since).
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "traffic": rows})
}

// GET /api/analytics/categories  (revenue by category)
func CategoryRevenue(c *gin.Context) {
	type row struct {
		Category string  `json:"category"`
		Revenue  float64 `json:"revenue"`
		Count    int     `json:"count"`
	}
	var rows []row
	database.DB.Raw(`
		SELECT COALESCE(p.category,'Uncategorized') as category,
		       COALESCE(SUM(oi.price * oi.quantity),0) as revenue,
		       COUNT(oi.id) as count
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		GROUP BY p.category
		ORDER BY revenue DESC`).Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, rows)
}

// AnalyticsReportPDF renders an executive PDF report via the TypeScript worker microservice.
// GET /api/analytics/report/pdf
func AnalyticsReportPDF(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).
		Where("created_at >= ? AND status <> ?", since, "cancelled").
		Select("COALESCE(SUM(total_price),0)").Row().Scan(&totalRevenue)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Where("created_at >= ?", since).Count(&totalOrders)

	type catRow struct {
		Category string  `json:"category"`
		Revenue  float64 `json:"revenue"`
		Count    int     `json:"count"`
	}
	var catRows []catRow
	database.DB.Raw(`
		SELECT COALESCE(p.category,'Uncategorized') as category,
		       COALESCE(SUM(oi.price * oi.quantity),0) as revenue,
		       COUNT(oi.id) as count
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		GROUP BY p.category
		ORDER BY revenue DESC LIMIT 8`).Scan(&catRows)

	breakdown := make([]map[string]interface{}, 0, len(catRows))
	for _, cr := range catRows {
		breakdown = append(breakdown, map[string]interface{}{
			"category": cr.Category,
			"count":    cr.Count,
			"revenue":  cr.Revenue,
		})
	}

	payload := map[string]interface{}{
		"title":             fmt.Sprintf("Business Performance Report (%d Days Window)", days),
		"generatedBy":       "Admin Analytics Engine",
		"startDate":         since.Format("02 Jan 2006"),
		"endDate":           time.Now().Format("02 Jan 2006"),
		"totalRevenue":      totalRevenue,
		"totalOrders":       totalOrders,
		"categoryBreakdown": breakdown,
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
	resp, err := client.Post(workerURL+"/api/v1/reports/sales-pdf", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		if workerURL != "http://localhost:8083" {
			resp, err = client.Post("http://localhost:8083/api/v1/reports/sales-pdf", "application/json", bytes.NewBuffer(jsonData))
		}
	}

	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "TypeScript worker report service temporarily unavailable",
			"details": fmt.Sprintf("%v", err),
		})
		return
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read generated report PDF"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=\"executive_analytics_report.pdf\"")
	c.Header("X-Generated-By", "kather_baksho-worker-ts")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// InventoryForecast calculates predictive stock runout, burn rate, and draft PO.
// GET /api/analytics/inventory-forecast
func InventoryForecast(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 180 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)

	// Retrieve all active products
	var products []models.Product
	if err := database.DB.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	// Query units sold per product in the last `days`
	type salesRow struct {
		ProductID uint    `json:"product_id"`
		UnitsSold float64 `json:"units_sold"`
	}
	var sales []salesRow
	database.DB.Raw(`
		SELECT oi.product_id, COALESCE(SUM(oi.quantity), 0) as units_sold
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.created_at >= ? AND o.status <> 'cancelled'
		GROUP BY oi.product_id
	`, since).Scan(&sales)

	salesMap := make(map[uint]float64)
	for _, s := range sales {
		salesMap[s.ProductID] = s.UnitsSold
	}

	type ItemForecast struct {
		ProductID      uint    `json:"product_id"`
		Name           string  `json:"name"`
		Category       string  `json:"category"`
		CurrentStock   uint    `json:"current_stock"`
		UnitsSold30d   float64 `json:"units_sold_30d"`
		DailyBurnRate  float64 `json:"daily_burn_rate"`
		RunwayDays     float64 `json:"runway_days"`
		RiskLevel      string  `json:"risk_level"` // "CRITICAL", "WARNING", "HEALTHY"
		SuggestedOrder int     `json:"suggested_order"`
		EstimatedCost  float64 `json:"estimated_cost"`
	}

	forecasts := make([]ItemForecast, 0, len(products))
	criticalCount := 0
	warningCount := 0
	totalRestockCost := 0.0

	for _, p := range products {
		sold := salesMap[p.ID]
		dailyRate := sold / float64(days)
		if dailyRate == 0 {
			dailyRate = 0.05 // baseline low default
		}

		runway := float64(p.Stock) / dailyRate
		risk := "HEALTHY"
		suggested := 0

		if p.Stock <= 0 || runway < 5.0 {
			risk = "CRITICAL"
			criticalCount++
			suggested = int(dailyRate * 30) // 30-day replenishment buffer
			if suggested < 10 {
				suggested = 10
			}
		} else if runway <= 14.0 {
			risk = "WARNING"
			warningCount++
			suggested = int(dailyRate * 20)
			if suggested < 5 {
				suggested = 5
			}
		}

		cost := float64(suggested) * p.Price * 0.65 // estimated 65% wholesale cost
		totalRestockCost += cost

		forecasts = append(forecasts, ItemForecast{
			ProductID:      p.ID,
			Name:           p.Name,
			Category:       p.Category,
			CurrentStock:   p.Stock,
			UnitsSold30d:   sold,
			DailyBurnRate:  dailyRate,
			RunwayDays:     runway,
			RiskLevel:      risk,
			SuggestedOrder: suggested,
			EstimatedCost:  cost,
		})
	}

	// Draft Purchase Order structure
	poNumber := fmt.Sprintf("PO-%d-%04d", time.Now().Year(), time.Now().Unix()%10000)
	c.JSON(http.StatusOK, gin.H{
		"analyzed_period_days": days,
		"total_products":       len(products),
		"critical_items_count": criticalCount,
		"warning_items_count":  warningCount,
		"total_restock_cost":   totalRestockCost,
		"draft_po_number":      poNumber,
		"forecasts":            forecasts,
	})
}
