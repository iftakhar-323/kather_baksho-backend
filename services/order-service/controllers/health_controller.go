package controllers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"kather_baksho/database"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

var appStartTime = time.Now()

// HealthCheck provides a lightweight general status check.
// GET /health and GET /api/health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"service":   "kather_baksho-backend",
		"version":   "1.0.0",
		"uptime":    time.Since(appStartTime).Round(time.Second).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// LivenessCheck answers Kubernetes and Docker liveness probes.
// GET /health/live and GET /api/health/live
func LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// ReadinessCheck answers readiness probes by inspecting database connectivity and system stats.
// GET /health/ready and GET /api/health/ready
func ReadinessCheck(c *gin.Context) {
	var dbStatus = "connected"
	var dbErr string

	sqlDB, err := database.DB.DB()
	if err != nil {
		dbStatus = "disconnected"
		dbErr = err.Error()
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "disconnected"
		dbErr = err.Error()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := gin.H{
		"status": dbStatus,
		"uptime": time.Since(appStartTime).Round(time.Second).String(),
		"database": gin.H{
			"status": dbStatus,
			"error":  dbErr,
		},
		"system": gin.H{
			"alloc_mb":   fmt.Sprintf("%.2f MB", float64(m.Alloc)/(1024*1024)),
			"sys_mb":     fmt.Sprintf("%.2f MB", float64(m.Sys)/(1024*1024)),
			"goroutines": runtime.NumGoroutine(),
			"num_gc":     m.NumGC,
			"go_version": runtime.Version(),
		},
		"circuit_breakers": utils.GetAllCircuitBreakers(),
	}

	var redisStatus = "disabled"
	if database.RedisClient != nil {
		if err := database.RedisPing(); err != nil {
			redisStatus = "unreachable: " + err.Error()
		} else {
			redisStatus = "connected"
		}
	}
	stats["cache"] = gin.H{
		"provider": "redis",
		"status":   redisStatus,
	}

	var mongoStatus = "disabled"
	if database.MongoClient != nil {
		if err := database.MongoPing(); err != nil {
			mongoStatus = "unreachable: " + err.Error()
		} else {
			mongoStatus = "connected"
		}
	}
	stats["nosql_database"] = gin.H{
		"provider": "mongodb",
		"status":   mongoStatus,
	}

	if dbStatus != "connected" {
		c.JSON(http.StatusServiceUnavailable, stats)
		return
	}

	c.JSON(http.StatusOK, stats)
}
