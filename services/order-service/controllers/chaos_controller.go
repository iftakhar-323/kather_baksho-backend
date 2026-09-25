package controllers

import (
	"net/http"

	"kather_baksho/middleware"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// GET /api/chaos/config
func GetChaosConfig(c *gin.Context) {
	cfg := middleware.GetChaosConfig()
	c.JSON(http.StatusOK, cfg)
}

// POST /api/chaos/config
func UpdateChaosConfig(c *gin.Context) {
	var cfg middleware.ChaosConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	middleware.SetChaosConfig(cfg)
	c.JSON(http.StatusOK, gin.H{
		"message": "Chaos configuration updated successfully",
		"config":  middleware.GetChaosConfig(),
	})
}

// POST /api/chaos/reset
func ResetChaos(c *gin.Context) {
	middleware.ResetChaos()
	c.JSON(http.StatusOK, gin.H{
		"message": "Chaos experiments halted. All settings and circuit breakers reset to healthy baseline.",
		"config":  middleware.GetChaosConfig(),
		"stats":   middleware.GetChaosStats(),
	})
}

// GET /api/chaos/stats
func GetChaosStats(c *gin.Context) {
	stats := middleware.GetChaosStats()
	c.JSON(http.StatusOK, stats)
}

// POST /api/chaos/trip-breaker
func TripCircuitBreaker(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cb := utils.GetCircuitBreaker(input.Name)
	if cb == nil {
		// Auto-register if not yet created so chaos can test arbitrary named dependencies
		cb = utils.NewCircuitBreaker(utils.CircuitBreakerConfig{
			Name:             input.Name,
			FailureThreshold: 3,
		})
	}

	// Trip it by forcing threshold failures
	for i := 0; i < 5; i++ {
		_ = cb.Execute(func() error {
			return utils.ErrCircuitBreakerOpen
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker manually tripped for chaos drill",
		"breaker": input.Name,
		"state":   cb.State().String(),
	})
}

// GET /api/chaos/probe
func ChaosProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Chaos probe executed without fault injection",
	})
}
