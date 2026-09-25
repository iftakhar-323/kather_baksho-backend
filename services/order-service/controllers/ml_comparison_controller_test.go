package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCompareModelsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/ml/compare", CompareModels)

	req, _ := http.NewRequest(http.MethodPost, "/api/ml/compare", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp ModelComparisonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ModelA.ModelName != "MobilePlantNet-v3" {
		t.Errorf("expected Model A to be MobilePlantNet-v3, got %s", resp.ModelA.ModelName)
	}
	if resp.ModelB.ModelName != "DeepBotanist-ResNet50" {
		t.Errorf("expected Model B to be DeepBotanist-ResNet50, got %s", resp.ModelB.ModelName)
	}
	if resp.ModelA.LatencyMS <= 0 || resp.ModelB.LatencyMS <= 0 {
		t.Errorf("expected positive latencies, got A=%f, B=%f", resp.ModelA.LatencyMS, resp.ModelB.LatencyMS)
	}
	if resp.ModelA.ThroughputIPS <= 0 {
		t.Errorf("expected positive throughput for Model A")
	}
}
