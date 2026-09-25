package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/database"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// SearchProducts handles BM25 full-text search with typo-tolerant prefix matching
func SearchProducts(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	results, err := database.SearchProductsFTS(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"count":   len(results),
		"results": results,
	})
}
