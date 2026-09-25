package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	telegramClient  = &http.Client{Timeout: 10 * time.Second}
	lastAlertTimes  sync.Map // key: "plantID:alertType" -> time.Time
	alertCooldown   = 4 * time.Hour
)

type TelegramMessagePayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"` // "Markdown" or "HTML"
}

// SendTelegramMessage dispatches a formatted message to Telegram API.
func SendTelegramMessage(text string) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" || chatID == "" {
		log.Printf("[Telegram:Mock] ChatID: %s | Message:\n%s", chatID, text)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := TelegramMessagePayload{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := telegramClient.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API error with status code %d", resp.StatusCode)
	}

	log.Printf("[Telegram] Successfully dispatched notification to chat %s", chatID)
	return nil
}

// CheckDebounce returns true if an alert is allowed to be sent (not throttled).
func CheckDebounce(key string) bool {
	now := time.Now()
	if val, ok := lastAlertTimes.Load(key); ok {
		if lastTime, ok := val.(time.Time); ok {
			if now.Sub(lastTime) < alertCooldown {
				return false // Debounced / throttled
			}
		}
	}
	lastAlertTimes.Store(key, now)
	return true
}

// ResetDebounce resets debouncing (useful in unit tests).
func ResetDebounce() {
	lastAlertTimes = sync.Map{}
}

// SendPlantSoilAlert sends an automated Telegram alert when plant soil moisture drops critically.
func SendPlantSoilAlert(plantID uint, plantName, location string, moisture float64) {
	key := fmt.Sprintf("%d:soil", plantID)
	if !CheckDebounce(key) {
		return
	}

	msg := fmt.Sprintf(
		"🚨 *Kather Baksho Plant Care Alert*\n\n"+
			"🌿 *Plant:* %s (#%d)\n"+
			"📍 *Location:* %s\n"+
			"💧 *Soil Moisture:* `%.1f%%` (Critically Dry!)\n\n"+
			"👉 _Immediate watering recommended to prevent root dehydration._\n"+
			"Timestamp: %s",
		plantName, plantID, location, moisture, time.Now().Format("2006-01-02 15:04:05"),
	)

	go func() {
		if err := SendTelegramMessage(msg); err != nil {
			log.Printf("[Telegram] Soil alert failed: %v", err)
		}
	}()
}

// SendPlantHeatAlert sends a Telegram alert when ambient temperature exceeds safe threshold.
func SendPlantHeatAlert(plantID uint, plantName, location string, temp float64) {
	key := fmt.Sprintf("%d:heat", plantID)
	if !CheckDebounce(key) {
		return
	}

	msg := fmt.Sprintf(
		"🔥 *Kather Baksho Thermal Stress Alert*\n\n"+
			"🌿 *Plant:* %s (#%d)\n"+
			"📍 *Location:* %s\n"+
			"🌡️ *Ambient Temp:* `%.1f°C` (Excess Heat!)\n\n"+
			"👉 _Please relocate plant to shaded area or mist foliage._\n"+
			"Timestamp: %s",
		plantName, plantID, location, temp, time.Now().Format("2006-01-02 15:04:05"),
	)

	go func() {
		if err := SendTelegramMessage(msg); err != nil {
			log.Printf("[Telegram] Heat alert failed: %v", err)
		}
	}()
}
