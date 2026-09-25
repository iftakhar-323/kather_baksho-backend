package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kather_baksho/database"
	"kather_baksho/middleware"
	"kather_baksho/models"
	"kather_baksho/routes"
	"kather_baksho/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (silently ignore when missing so production
	// environments that set vars in the shell still work).
	_ = godotenv.Load()

	database.ConnectDatabase()
	database.InitRedis()
	database.ConnectMongoDB()
	services.InitEventBus()
	services.StartOutboxWorker()
	services.InitStorage()
	if err := database.DB.AutoMigrate(
		&models.Cart{},
		&models.CartItem{},
		&models.Order{},
		&models.OrderItem{},
		&models.Coupon{},
		&models.CareReminder{},
		&models.CorporateQuote{},
		&models.CorporateOrder{},
		&models.OrderEvent{},
		&models.OutboxEvent{},
		&models.ReturnRequest{},
		&models.IdempotencyRecord{},
		&models.Achievement{},
		&models.UserAchievement{},
		&models.ReferralCode{},
		&models.Referral{},
		&models.MembershipTier{},
		&models.CouponReward{},
		&models.GuestOrder{},
		&models.GiftCard{},
		&models.ShippingRule{},
		&models.TaxRule{},
		&models.UserMembership{},
	); err != nil {
		log.Fatalf("[Order-Service] Auto-migrate failed: %v", err)
	}

	// Initialize OpenTelemetry Distributed Tracing
	shutdownTracer := middleware.InitTracer("order-service")
	defer shutdownTracer(context.Background())

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestSizeLimiter(10 << 20))
	router.Use(middleware.OpenTelemetryMiddleware("order-service"))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.RateLimiter())
	router.Use(middleware.PrometheusMetricsMiddleware())
	router.Use(middleware.IdempotencyMiddleware())
	router.Use(middleware.ChaosMiddleware())

	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:5174",
		"http://localhost:80",
		"http://localhost",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:5174",
	}
	if customOrigins := os.Getenv("ALLOWED_ORIGINS"); customOrigins != "" {
		if customOrigins == "*" {
			allowedOrigins = []string{"*"}
		} else {
			allowedOrigins = append(allowedOrigins, strings.Split(customOrigins, ",")...)
		}
	}

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "Idempotency-Key", "X-Idempotency-Key", "traceparent", "tracestate"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After", "X-Cache-Lookup", "X-Idempotency-Key", "traceparent", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false
	} else {
		corsConfig.AllowOrigins = allowedOrigins
	}

	router.Use(cors.New(corsConfig))

	// Prometheus Metrics endpoint
	router.GET("/metrics", middleware.PrometheusHandler())
	router.GET("/api/metrics", middleware.PrometheusHandler())

	routes.HealthRoutes(router)

	routes.ProductRoutes(router)
	routes.AuthRoutes(router)
	routes.CartRoutes(router)
	routes.OrderRoutes(router)
	routes.PaymentRoutes(router)
	routes.AIRoutes(router)
	routes.MLRoutes(router)
	routes.WishlistRoutes(router)
	routes.NotificationRoutes(router)
	routes.CouponRoutes(router)
	routes.ReminderRoutes(router)
	routes.GiftRoutes(router)
	routes.SeasonalRoutes(router)
	routes.SubscriptionRoutes(router)
	routes.ConsultationRoutes(router)
	routes.CorporateRoutes(router)
	routes.CommunityRoutes(router)
	routes.AdminRoutes(router)
	// Sprint D-I extensions
	routes.OrderExtensionRoutes(router)
	routes.CareJournalRoutes(router)
	routes.CommunityExtensionRoutes(router)
	routes.LoyaltyRoutes(router)
	routes.CorporateOrderRoutes(router)
	routes.BlogRoutes(router)
	routes.AnalyticsRoutes(router)
	routes.ShoppingRoutes(router)
	routes.CSVRoutes(router)
	routes.BackupRoutes(router)
	routes.AddressRoutes(router)
	routes.ReviewRoutes(router)
	routes.CategoryRoutes(router)
	routes.TelemetryRoutes(router)
	routes.WebSocketRoutes(router)
	routes.DocsRoutes(router)
	routes.EventRoutes(router)
	routes.MediaRoutes(router)
	routes.ChaosRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:    port,
		Handler: router,
	}

	go func() {
		log.Printf("[Order-Service] Microservice listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Order-Service] Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Order-Service] Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[Order-Service] Server forced to shutdown: %v", err)
	}
	log.Println("[Order-Service] Server exited cleanly.")
}
