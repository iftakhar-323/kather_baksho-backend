package services

import (
	"strings"
	"testing"
)

func TestLocalBotanicalExpertDiagnosisYellowLeaves(t *testing.T) {
	res := localBotanicalExpertDiagnosis("Monstera Deliciosa", "leaves are turning yellow and drooping", "indoor living room")

	if res.ProviderUsed != "local-botanical-expert-fallback" {
		t.Fatalf("expected provider local-botanical-expert-fallback, got %s", res.ProviderUsed)
	}
	if !strings.Contains(res.Diagnosis, "Overwatering") && !strings.Contains(res.Diagnosis, "Chlorosis") {
		t.Errorf("unexpected diagnosis: %s", res.Diagnosis)
	}
	if len(res.ActionPlan) == 0 {
		t.Errorf("expected non-empty action plan")
	}
	if len(res.RecommendedProducts) == 0 {
		t.Errorf("expected non-empty recommended products")
	}
}

func TestLocalBotanicalExpertDiagnosisPests(t *testing.T) {
	res := localBotanicalExpertDiagnosis("Ficus Lyrata", "white sticky bugs and spider mites on bottom of leaves", "balcony")

	if !strings.Contains(res.Diagnosis, "Pest") {
		t.Errorf("expected pest diagnosis, got %s", res.Diagnosis)
	}
	if res.Severity != "Severe" {
		t.Errorf("expected Severe severity, got %s", res.Severity)
	}
}

func TestLocalBotanicalExpertChat(t *testing.T) {
	ans := localBotanicalExpertChat("How often should I water my snake plant?")
	if !strings.Contains(strings.ToLower(ans), "water") {
		t.Errorf("expected chat response to address watering, got: %s", ans)
	}
}
