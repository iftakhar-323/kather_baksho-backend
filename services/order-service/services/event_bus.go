package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"kather_baksho/database"

	"github.com/redis/go-redis/v9"
)

const (
	EventStreamName = "kb:events:stream"
	EventGroupName  = "kb_workers"
	EventDLQName    = "kb:events:dlq"
	MaxEventRetries = 3
)

// AppEvent represents an event dispatched to the message bus
type AppEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	Timestamp  string                 `json:"timestamp"`
	RetryCount int                    `json:"retry_count"`
}

// InitEventBus initializes the Redis Stream consumer group and starts background worker
func InitEventBus() {
	if database.RedisClient == nil {
		log.Println("[EventBus] Redis not available, event-driven message queue running in no-op mode.")
		return
	}

	ctx := context.Background()

	// Ensure the stream and consumer group exist
	err := database.RedisClient.XGroupCreateMkStream(ctx, EventStreamName, EventGroupName, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Printf("[EventBus] Warning initializing consumer group: %v", err)
	} else {
		log.Printf("[EventBus] Initialized Redis Stream consumer group '%s' on '%s'", EventGroupName, EventStreamName)
	}

	// Start asynchronous background consumer
	go startEventConsumerWorker()
}

// PublishEvent pushes an event to the Redis Stream
func PublishEvent(eventType string, payload map[string]interface{}) (string, error) {
	if database.RedisClient == nil {
		log.Printf("[EventBus] Redis offline. Simulating local event publish for '%s'", eventType)
		return "local-simulated-id", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload error: %w", err)
	}

	res, err := database.RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: EventStreamName,
		Values: map[string]interface{}{
			"type":        eventType,
			"payload":     string(payloadBytes),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"retry_count": 0,
		},
	}).Result()

	if err != nil {
		log.Printf("[EventBus] Error publishing event '%s': %v", eventType, err)
		return "", err
	}

	log.Printf("[EventBus] Published event '%s' [ID: %s]", eventType, res)
	return res, nil
}

// startEventConsumerWorker continuously reads and handles events with ack and DLQ
func startEventConsumerWorker() {
	consumerName := fmt.Sprintf("worker-%d", time.Now().UnixNano()%10000)
	ctx := context.Background()

	log.Printf("[EventBus] Worker '%s' started listening on stream '%s'", consumerName, EventStreamName)

	for {
		if database.RedisClient == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Read events for this consumer group
		streams, err := database.RedisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    EventGroupName,
			Consumer: consumerName,
			Streams:  []string{EventStreamName, ">"},
			Count:    5,
			Block:    2 * time.Second,
		}).Result()

		if err != nil {
			if err != redis.Nil {
				time.Sleep(1 * time.Second)
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				processMessageWithRetries(ctx, msg)
			}
		}
	}
}

// processMessageWithRetries handles a message or routes to DLQ on failure
func processMessageWithRetries(ctx context.Context, msg redis.XMessage) {
	eventType, _ := msg.Values["type"].(string)
	payloadRaw, _ := msg.Values["payload"].(string)
	retriesStr, _ := msg.Values["retry_count"].(string)
	retryCount, _ := strconv.Atoi(retriesStr)

	var payload map[string]interface{}
	_ = json.Unmarshal([]byte(payloadRaw), &payload)

	// Execute event handler
	err := handleEvent(eventType, payload)
	if err == nil {
		// Successful processing -> ACK
		_ = database.RedisClient.XAck(ctx, EventStreamName, EventGroupName, msg.ID).Err()
		return
	}

	// Failed processing -> Retry or DLQ
	log.Printf("[EventBus] Handler error for msg %s: %v (Attempt %d/%d)", msg.ID, err, retryCount+1, MaxEventRetries)

	if retryCount+1 >= MaxEventRetries {
		// Route to Dead Letter Queue (DLQ)
		_ = database.RedisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: EventDLQName,
			Values: map[string]interface{}{
				"original_id": msg.ID,
				"type":        eventType,
				"payload":     payloadRaw,
				"error":       err.Error(),
				"failed_at":   time.Now().UTC().Format(time.RFC3339),
			},
		}).Err()

		// Acknowledge from main queue so it does not poison future reads
		_ = database.RedisClient.XAck(ctx, EventStreamName, EventGroupName, msg.ID).Err()
		log.Printf("[EventBus] Message %s routed to Dead-Letter Queue '%s'", msg.ID, EventDLQName)
	} else {
		// Increment retry count
		msg.Values["retry_count"] = retryCount + 1
	}
}

// handleEvent executes domain specific asynchronous logic
func handleEvent(eventType string, payload map[string]interface{}) error {
	switch eventType {
	case "order.created":
		orderID := payload["order_id"]
		userID := payload["user_id"]
		total := payload["total_price"]
		log.Printf("[EventBus:AsyncWorker] Processing 'order.created' for Order #%v (User #%v, Total: %v BDT)", orderID, userID, total)
		// Simulates async processing without blocking caller
		time.Sleep(50 * time.Millisecond)
		return nil

	case "inventory.low_stock":
		prodID := payload["product_id"]
		stock := payload["stock_remaining"]
		log.Printf("[EventBus:AsyncWorker] Low stock alert for Product #%v (Remaining: %v)", prodID, stock)
		return nil

	case "telemetry.alert":
		plantID := payload["plant_id"]
		status := payload["botanical_status"]
		log.Printf("[EventBus:AsyncWorker] IoT Botanical alert for Plant #%v: %v", plantID, status)
		return nil

	default:
		log.Printf("[EventBus:AsyncWorker] Received unknown event type: %s", eventType)
		return nil
	}
}

// GetEventBusStats returns real-time metrics of the Redis Stream
func GetEventBusStats() (map[string]interface{}, error) {
	if database.RedisClient == nil {
		return map[string]interface{}{
			"status": "offline",
			"stream": EventStreamName,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	streamLen, err := database.RedisClient.XLen(ctx, EventStreamName).Result()
	if err != nil {
		streamLen = 0
	}

	dlqLen, err := database.RedisClient.XLen(ctx, EventDLQName).Result()
	if err != nil {
		dlqLen = 0
	}

	pendingInfo, err := database.RedisClient.XPending(ctx, EventStreamName, EventGroupName).Result()
	pendingCount := int64(0)
	if err == nil && pendingInfo != nil {
		pendingCount = pendingInfo.Count
	}

	return map[string]interface{}{
		"status":        "healthy",
		"stream":        EventStreamName,
		"group":         EventGroupName,
		"stream_length": streamLen,
		"pending_count": pendingCount,
		"dlq_stream":    EventDLQName,
		"dlq_length":    dlqLen,
	}, nil
}
