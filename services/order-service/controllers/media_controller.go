package controllers

import (
	"io"
	"net/http"
	"path/filepath"

	"kather_baksho/services"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

// GetStorageStatus returns MinIO S3 object storage health
func GetStorageStatus(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	c.JSON(http.StatusOK, services.GetStorageStatus())
}

// UploadMedia handles multipart file upload to MinIO S3
func UploadMedia(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded in 'file' form field"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	fileURL, err := services.UploadMedia(header.Filename, data, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"filename":     header.Filename,
		"size_bytes":   len(data),
		"content_type": contentType,
		"url":          fileURL,
		"storage":      "MinIO S3",
	})
}

// ServeMedia serves or proxies stored media
func ServeMedia(c *gin.Context) {
	if utils.ProxyToService(c, "CATALOG_SERVICE_URL") {
		return
	}

	filename := c.Param("filename")
	cleanName := filepath.Base(filename)

	data, ctype, err := services.ReadMediaFromStorage(cleanName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media object not found"})
		return
	}

	if ctype == "" {
		ctype = "application/octet-stream"
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, ctype, data)
}
