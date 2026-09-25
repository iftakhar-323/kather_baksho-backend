package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"kather_baksho/catalog_service/controllers"
	"kather_baksho/catalog_service/database"
	"kather_baksho/catalog_service/middleware"
	"kather_baksho/catalog_service/models"
	"kather_baksho/catalog_service/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Catalog-Service] Starting Botanical Catalog, Search & Media Microservice...")

	// 1. Initialize databases & storage
	database.ConnectDatabase()
	if err := database.DB.AutoMigrate(
		&models.Product{},
		&models.Category{},
		&models.Review{},
		&models.WishlistItem{},
		&models.PageView{},
	); err != nil {
		log.Fatalf("[Catalog-Service] Auto-migrate failed: %v", err)
	}
	database.InitRedis()
	database.InitFTS()
	services.InitStorage()

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
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-Correlation-ID", "X-Cache"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(middleware.RequestIDMiddleware())

	// 4. Health probe
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if sqlDB, err := database.DB.DB(); err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}

		redisStatus := "connected"
		if database.RedisClient == nil {
			redisStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "kather_baksho-catalog",
			"database":  dbStatus,
			"cache":     redisStatus,
			"timestamp": time.Now().UTC(),
		})
	})

	// 5. Product & Search Routes
	products := router.Group("/api/products")
	{
		products.GET("", controllers.GetProducts)
		products.GET("/", controllers.GetProducts)
		products.GET("/autocomplete", controllers.Autocomplete)
		products.GET("/suggest", controllers.SuggestProducts)
		products.GET("/best-selling", controllers.BestSelling)
		products.GET("/most-viewed", controllers.MostViewed)
		products.GET("/brands", controllers.ListBrands)
		products.GET("/categories", controllers.CategoryTree)
		products.GET("/search", controllers.SearchProducts)
		products.GET("/:id", controllers.GetProduct)
		products.GET("/slug/:slug", controllers.GetProductBySlug)
		products.GET("/related/:id", controllers.GetRelated)
		products.GET("/fbt/:id", controllers.GetFBT)
		products.POST("/:id/view", controllers.TrackView)

		// Direct admin product endpoints
		products.POST("", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.CreateProduct)
		products.POST("/", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.CreateProduct)
		products.PUT("/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.UpdateProduct)
		products.DELETE("/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.DeleteProduct)

		// Product Reviews
		products.GET("/:id/reviews", controllers.ListProductReviews)
		products.POST("/:id/reviews", middleware.AuthMiddleware(), controllers.CreateReview)
		products.DELETE("/:id/reviews", middleware.AuthMiddleware(), controllers.DeleteReview)
	}

	// Direct review endpoints
	router.GET("/api/reviews/mine", middleware.AuthMiddleware(), controllers.MyReviews)
	router.GET("/api/admin/reviews", middleware.AuthMiddleware(), middleware.AdminMiddleware(), controllers.AdminListReviews)

	// 6. Category Routes
	categories := router.Group("/api/categories")
	{
		categories.GET("", controllers.ListCategories)
		categories.GET("/", controllers.ListCategories)
	}

	// 7. Admin Catalog Routes
	adminProducts := router.Group("/api/admin/products", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		adminProducts.POST("", controllers.CreateProduct)
		adminProducts.POST("/", controllers.CreateProduct)
		adminProducts.PUT("/:id", controllers.UpdateProduct)
		adminProducts.DELETE("/:id", controllers.DeleteProduct)
	}

	adminCategories := router.Group("/api/admin/categories", middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		adminCategories.POST("", controllers.AdminCreateCategory)
		adminCategories.POST("/", controllers.AdminCreateCategory)
		adminCategories.PUT("/:id", controllers.AdminUpdateCategory)
		adminCategories.DELETE("/:id", controllers.AdminDeleteCategory)
	}

	// 8. Wishlist Routes
	wishlist := router.Group("/api/wishlist", middleware.AuthMiddleware())
	{
		wishlist.GET("", controllers.GetWishlist)
		wishlist.GET("/", controllers.GetWishlist)
		wishlist.POST("/add", controllers.AddToWishlist)
		wishlist.DELETE("/:id", controllers.RemoveFromWishlist)
	}

	// 9. Media & Object Storage Routes
	media := router.Group("/api/media")
	{
		media.GET("/status", controllers.GetStorageStatus)
		media.POST("/upload", controllers.UploadMedia)
		media.GET("/file/:filename", controllers.ServeMedia)
		media.GET("/serve/:filename", controllers.ServeMedia)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	log.Printf("[Catalog-Service] Microservice listening on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[Catalog-Service] Failed to start server: %v", err)
	}
}
