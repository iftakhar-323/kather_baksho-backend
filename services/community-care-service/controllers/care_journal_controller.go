package controllers

import (
	"net/http"
	"strconv"
	"time"

	"kather_baksho/community_care_service/database"
	"kather_baksho/community_care_service/models"

	"github.com/gin-gonic/gin"
)

// AddJournalEntry handles POST /api/journal
func AddJournalEntry(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		ProductID uint   `json:"product_id"`
		Note      string `json:"note"`
		Photo     string `json:"photo"`
		Height    string `json:"height"`
		Health    string `json:"health"`
		EntryDate string `json:"entry_date"`
		Type      string `json:"type"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.ProductID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
		return
	}
	if body.Health == "" {
		body.Health = "ok"
	}
	if body.EntryDate == "" {
		body.EntryDate = time.Now().Format("2006-01-02")
	}
	if body.Type == "" {
		body.Type = "note"
	}
	j := models.GrowthJournal{
		UserID:    uid,
		ProductID: body.ProductID,
		EntryDate: body.EntryDate,
		Note:      body.Note,
		PhotoURL:  body.Photo,
		Type:      body.Type,
	}
	if err := database.DB.Create(&j).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, j)
}

// ListJournal handles GET /api/journal?product_id=1
func ListJournal(c *gin.Context) {
	uid := c.GetUint("user_id")
	q := database.DB.Where("user_id = ?", uid).Order("created_at desc")
	if pid := c.Query("product_id"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	var list []models.GrowthJournal
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// DeleteJournalEntry handles DELETE /api/journal/:id
func DeleteJournalEntry(c *gin.Context) {
	uid := c.GetUint("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := database.DB.Where("user_id = ? AND id = ?", uid, id).Delete(&models.GrowthJournal{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// JournalTimeline handles GET /api/journal/timeline?days=60
func JournalTimeline(c *gin.Context) {
	uid := c.GetUint("user_id")
	days := 60
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)
	var list []models.GrowthJournal
	database.DB.Where("user_id = ? AND created_at >= ?", uid, since).
		Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// AddCareSchedule handles POST /api/care-schedule (admin)
func AddCareSchedule(c *gin.Context) {
	var body struct {
		ProductID   uint   `json:"product_id"`
		Month       int    `json:"month"`
		Action      string `json:"action"`
		Description string `json:"description"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.ProductID == 0 || body.Month < 1 || body.Month > 12 || body.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id, month (1-12), action required"})
		return
	}
	cs := models.CareSchedule{
		ProductID:   body.ProductID,
		Month:       body.Month,
		Action:      body.Action,
		Description: body.Description,
	}
	if err := database.DB.Create(&cs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cs)
}

// ListCareSchedule handles GET /api/care-schedule?product_id=1
func ListCareSchedule(c *gin.Context) {
	q := database.DB.Order("month asc, id asc")
	if pid := c.Query("product_id"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	var list []models.CareSchedule
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// DeleteCareSchedule handles DELETE /api/care-schedule/:id (admin)
func DeleteCareSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	database.DB.Delete(&models.CareSchedule{}, id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type careEntry struct {
	Month       int    `json:"month"`
	Action      string `json:"action"`
	Description string `json:"description"`
	ProductID   uint   `json:"product_id"`
	Product     string `json:"product"`
}

// CareCalendar handles GET /api/care-calendar?month=8
func CareCalendar(c *gin.Context) {
	uid := c.GetUint("user_id")
	month := int(time.Now().Month())
	if m := c.Query("month"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n >= 1 && n <= 12 {
			month = n
		}
	}

	pids := map[uint]bool{}

	var wishRows []models.WishlistItem
	database.DB.Where("user_id = ?", uid).Find(&wishRows)
	for _, w := range wishRows {
		pids[w.ProductID] = true
	}

	var remRows []models.CareReminder
	database.DB.Where("user_id = ?", uid).Find(&remRows)
	for _, r := range remRows {
		pids[r.ProductID] = true
	}

	since := time.Now().AddDate(0, -6, 0)
	var orderRows []models.OrderItem
	database.DB.Table("order_items").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.user_id = ? AND orders.created_at >= ?", uid, since).
		Find(&orderRows)
	for _, it := range orderRows {
		pids[it.ProductID] = true
	}

	if len(pids) == 0 {
		c.JSON(http.StatusOK, []careEntry{})
		return
	}

	idList := make([]uint, 0, len(pids))
	for id := range pids {
		idList = append(idList, id)
	}

	var schedules []models.CareSchedule
	database.DB.Where("product_id IN ? AND month = ?", idList, month).Find(&schedules)

	products := map[uint]string{}
	var prods []models.Product
	database.DB.Where("id IN ?", idList).Find(&prods)
	for _, p := range prods {
		products[p.ID] = p.Name
	}

	out := make([]careEntry, 0, len(schedules))
	for _, s := range schedules {
		out = append(out, careEntry{
			Month:       s.Month,
			Action:      s.Action,
			Description: s.Description,
			ProductID:   s.ProductID,
			Product:     products[s.ProductID],
		})
	}
	c.JSON(http.StatusOK, out)
}
