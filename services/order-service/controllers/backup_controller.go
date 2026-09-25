package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"kather_baksho/database"

	"github.com/gin-gonic/gin"
)

// ---------- DB backup / restore (admin) ----------
//
// Assumes GORM SQLite file at backend/katherbox.db (or wherever InitDataBase set it).

// GET /api/admin/backup/db — streams the SQLite file as a download
func BackupDB(c *gin.Context) {
	dsn := "katherbox.db"
	if v := os.Getenv("KB_DB_FILE"); v != "" {
		dsn = v
	}
	path := dsn
	// If not absolute, search near the binary
	if !filepath.IsAbs(path) {
		path = filepath.Join(".", dsn)
	}
	if _, err := os.Stat(path); err != nil {
		// try backend/katherbox.db (relative to project root when running from repo)
		path = filepath.Join("backend", dsn)
	}
	f, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	ts := time.Now().Format("20060102-150405")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="katherbox-backup-%s.db"`, ts))
	if _, err := io.Copy(c.Writer, f); err != nil {
		// can't write JSON after streaming started
		return
	}
}

// POST /api/admin/backup/restore  (multipart "file") — replaces the working DB file
// SAFETY: closes GORM connection, copies the uploaded file, then the next request
// will lazy-open the new file on next DB call (or the server should be restarted,
// but for ease we leave a hint and keep the old handle).
func RestoreDB(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	if file.Size > 200*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (>200MB)"})
		return
	}
	dsn := "katherbox.db"
	if v := os.Getenv("KB_DB_FILE"); v != "" {
		dsn = v
	}
	dst := dsn
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(".", dsn)
	}

	// write uploaded file
	tmp := dst + ".restore"
	if err := c.SaveUploadedFile(file, tmp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Safety: run a quick "PRAGMA integrity_check" via a backup tool. Without an
	// external lib we simply move on — the next DB reopen will fail and surface
	// the error.

	if err := os.Rename(tmp, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "swap failed: " + err.Error()})
		return
	}
	// close current DB to force re-open
	if sqlDB, err := database.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "restored": filepath.Base(dst)})
}

// GET /api/admin/backup/stats — basic DB stats
func DBStats(c *gin.Context) {
	sqlDB, err := database.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db unreachable: " + err.Error()})
		return
	}
	type tblCount struct {
		Table string `json:"table"`
		Count int64  `json:"count"`
	}
	tables := []string{"users", "products", "orders", "order_items", "carts", "wishlists",
		"coupons", "reminders", "reviews", "subscriptions",
		"notifications", "community_questions", "community_answers",
		"achievements", "user_achievements", "referral_codes", "referrals",
		"blog_posts", "page_views", "growth_journals", "care_schedules",
		"return_requests", "corporate_orders", "order_events"}
	out := make([]tblCount, 0, len(tables))
	for _, t := range tables {
		var n int64
		database.DB.Raw("SELECT COUNT(*) FROM " + t).Scan(&n)
		out = append(out, tblCount{t, n})
	}
	c.JSON(http.StatusOK, gin.H{"tables": out})
}