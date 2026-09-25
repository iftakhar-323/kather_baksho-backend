package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"kather_baksho/iot_ai_service/database"

	"github.com/redis/go-redis/v9"
)

const (
	EventStreamName = "kb:events:stream"
)

// PublishTelemetryAlert emits an asynchronous alert event to the Redis Stream.
func PublishTelemetryAlert(plantID uint, plantName, status, alertMsg string) {
	if database.RedisClient == nil {
		return
	}

	payload := map[string]interface{}{
		"plant_id":         plantID,
		"plant_name":       plantName,
		"botanical_status": status,
		"alert_message":    alertMsg,
		"emitted_at":       time.Now().UTC().Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[IoT-AI:EventPublisher] Failed to serialize payload: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = database.RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: EventStreamName,
		Values: map[string]interface{}{
			"type":      "telemetry.alert",
			"payload":   string(payloadBytes),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}).Err()

	if err != nil {
		log.Printf("[IoT-AI:EventPublisher] Warning: Failed to publish telemetry.alert: %v", err)
	} else {
		log.Printf("[IoT-AI:EventPublisher] Emitted telemetry.alert for Plant #%d to Redis Stream", plantID)
	}
}
