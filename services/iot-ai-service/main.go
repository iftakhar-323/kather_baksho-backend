package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"kather_baksho/iot_ai_service/controllers"
	"kather_baksho/iot_ai_service/database"
	"kather_baksho/iot_ai_service/middleware"
	"kather_baksho/iot_ai_service/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[IoT-AI-Service] Starting Botanical Intelligence & IoT Telemetry Microservice...")

	// 1. Initialize databases & MQTT subscriber
	database.ConnectMongoDB()
	database.ConnectRedis()
	services.StartMQTTSubscriber()

	// 2. Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	router := gin.New()
	router.Use(gin.Recovery())

	// 3. Middlewares
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Correlation-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(middleware.RequestIDMiddleware())

	// 4. Health probe
	router.GET("/health", func(c *gin.Context) {
		mongoStatus := "connected"
		if err := database.MongoPing(); err != nil {
			mongoStatus = "disconnected"
		}

		redisStatus := "connected"
		if database.RedisClient == nil {
			redisStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "kather_baksho-iot-ai",
			"mongodb":   mongoStatus,
			"redis":     redisStatus,
			"timestamp": time.Now().UTC(),
		})
	})

	// 5. IoT Telemetry Routes
	iot := router.Group("/api/iot")
	{
		iot.GET("/plants", controllers.GetMonitoredPlants)
		iot.GET("/telemetry/:plant_id", controllers.GetPlantTelemetryHistory)
		iot.POST("/telemetry", controllers.IngestPlantTelemetry)
		iot.POST("/seed", middleware.AuthMiddleware(), middleware.StaffMiddleware(), controllers.SeedIoTTelemetry)
	}

	// 6. AI Plant Doctor Routes
	ai := router.Group("/api/ai")
	{
		ai.POST("/diagnose", controllers.DiagnosePlantSymptoms)
		ai.POST("/chat", controllers.ChatWithPlantDoctor)
	}

	// 7. ML Comparative Benchmark Routes
	ml := router.Group("/api/ml")
	{
		ml.POST("/compare", controllers.CompareModels)
		ml.GET("/compare", controllers.CompareModels)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}

	log.Printf("[IoT-AI-Service] Microservice listening on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[IoT-AI-Service] Failed to start server: %v", err)
	}
}
