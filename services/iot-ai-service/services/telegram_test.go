package services

import (
	"testing"
)

func TestTelegramDebounce(t *testing.T) {
	ResetDebounce()

	key := "42:soil"

	// First alert should pass
	if !CheckDebounce(key) {
		t.Error("expected first debounce check to return true")
	}

	// Immediate second alert for same key should be throttled
	if CheckDebounce(key) {
		t.Error("expected immediate second debounce check to return false")
	}

	// Different alert type for same plant should pass
	heatKey := "42:heat"
	if !CheckDebounce(heatKey) {
		t.Error("expected different alert type key to return true")
	}
}

func TestSendPlantAlertsMock(t *testing.T) {
	ResetDebounce()

	// Should not panic or error in mock mode
	SendPlantSoilAlert(101, "Fiddle Leaf Fig", "Living Room", 18.5)
	SendPlantHeatAlert(102, "Monstera Deliciosa", "Balcony", 38.2)

	err := SendTelegramMessage("Test automated notification")
	if err != nil {
		t.Fatalf("unexpected error in mock Telegram message send: %v", err)
	}
}
