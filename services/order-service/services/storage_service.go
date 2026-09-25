package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	MinioEndpoint = "minio:9000"
	MinioBucket   = "kather-baksho-media"
	MinioUser     = "kather_baksho_admin"
	MinioPassword = "kather_baksho_s3_secret"

	localMediaMu    sync.RWMutex
	localMediaCache = make(map[string][]byte)
	localMediaTypes = make(map[string]string)
)

// InitStorage verifies MinIO connection and ensures the default media bucket exists
func InitStorage() {
	if ep := os.Getenv("MINIO_ENDPOINT"); ep != "" {
		MinioEndpoint = ep
	}

	url := fmt.Sprintf("http://%s/minio/health/live", MinioEndpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("[MinIO] Notice: MinIO connection check (%s): %v. Running with local fallback.", url, err)
		return
	}
	_ = resp.Body.Close()

	// Ensure bucket exists in MinIO
	bucketURL := fmt.Sprintf("http://%s/%s", MinioEndpoint, MinioBucket)
	bucketReq, _ := http.NewRequest("PUT", bucketURL, nil)
	bucketReq.SetBasicAuth(MinioUser, MinioPassword)
	bResp, bErr := client.Do(bucketReq)
	if bErr == nil && bResp != nil {
		_ = bResp.Body.Close()
	}

	log.Printf("[MinIO] Connected to MinIO S3 Object Storage successfully at %s (bucket: %s)", MinioEndpoint, MinioBucket)
}

// UploadMedia stores a binary file into MinIO S3 object storage with in-memory fallback
func UploadMedia(filename string, data []byte, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	cleanName := filepath.Base(filename)
	objectKey := fmt.Sprintf("%d_%s", time.Now().Unix(), cleanName)

	// Save in memory cache immediately to guarantee instant retrieval
	localMediaMu.Lock()
	localMediaCache[objectKey] = data
	localMediaTypes[objectKey] = contentType
	localMediaMu.Unlock()

	targetURL := fmt.Sprintf("http://%s/%s/%s", MinioEndpoint, MinioBucket, objectKey)
	req, err := http.NewRequest("PUT", targetURL, bytes.NewReader(data))
	if err == nil {
		req.Header.Set("Content-Type", contentType)
		req.SetBasicAuth(MinioUser, MinioPassword)
		client := &http.Client{Timeout: 3 * time.Second}
		resp, doErr := client.Do(req)
		if doErr == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}

	return fmt.Sprintf("/api/media/file/%s", objectKey), nil
}

// GetStorageStatus returns current MinIO health and stats
func GetStorageStatus() map[string]interface{} {
	endpoint := MinioEndpoint
	url := fmt.Sprintf("http://%s/minio/health/live", endpoint)
	client := &http.Client{Timeout: 1 * time.Second}

	resp, err := client.Get(url)
	isHealthy := err == nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		_ = resp.Body.Close()
	}

	status := "offline"
	if isHealthy {
		status = "healthy"
	}

	localMediaMu.RLock()
	cachedCount := len(localMediaCache)
	localMediaMu.RUnlock()

	return map[string]interface{}{
		"status":           status,
		"service":          "MinIO S3 Compatible Object Storage",
		"endpoint":         endpoint,
		"bucket":           MinioBucket,
		"console_url":      "http://localhost:9006",
		"uploaded_objects": cachedCount,
	}
}

// ReadMediaFromStorage reads stored media bytes from cache or MinIO
func ReadMediaFromStorage(objectKey string) ([]byte, string, error) {
	// Check fast memory cache first
	localMediaMu.RLock()
	data, exists := localMediaCache[objectKey]
	ctype := localMediaTypes[objectKey]
	localMediaMu.RUnlock()

	if exists {
		return data, ctype, nil
	}

	// Fallback to MinIO S3 API
	targetURL := fmt.Sprintf("http://%s/%s/%s", MinioEndpoint, MinioBucket, objectKey)
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.SetBasicAuth(MinioUser, MinioPassword)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("object not found")
	}
	defer resp.Body.Close()

	readData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return readData, resp.Header.Get("Content-Type"), nil
}
