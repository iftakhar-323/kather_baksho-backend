package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

// MediaRoutes registers MinIO S3 object storage routes
func MediaRoutes(r *gin.Engine) {
	group := r.Group("/api/media")
	{
		group.GET("/status", controllers.GetStorageStatus)
		group.POST("/upload", controllers.UploadMedia)
		group.GET("/file/:filename", controllers.ServeMedia)
	}
}
