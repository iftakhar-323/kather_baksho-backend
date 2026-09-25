package routes

import (
	"kather_baksho/controllers"

	"github.com/gin-gonic/gin"
)

// DocsRoutes registers Swagger UI and OpenAPI 3.0 specification endpoints
func DocsRoutes(r *gin.Engine) {
	// OpenAPI 3.0 JSON specification
	r.GET("/api/docs/openapi.json", controllers.ServeOpenAPISpec)

	// Interactive Swagger UI documentation
	r.GET("/docs", controllers.ServeSwaggerUI)
	r.GET("/docs/index.html", controllers.ServeSwaggerUI)
	r.GET("/swagger", controllers.ServeSwaggerUI)
	r.GET("/swagger/index.html", controllers.ServeSwaggerUI)
}
