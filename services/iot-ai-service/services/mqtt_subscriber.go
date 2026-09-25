package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"kather_baksho/iot_ai_service/database"
	"kather_baksho/iot_ai_service/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MQTTTelemetryPayload struct {
	DeviceID        string  `json:"device_id"`
	PlantID         uint    `json:"plant_id"`
	PlantName       string  `json:"plant_name"`
	Location        string  `json:"location"`
	SoilMoisturePct float64 `json:"soil_moisture_pct"`
	AmbientTempC    float64 `json:"ambient_temp_c"`
	HumidityPct     float64 `json:"humidity_pct"`
	LightLux        float64 `json:"light_lux"`
	BatteryPct      float64 `json:"battery_pct"`
}

var MQTTClient mqtt.Client

// StartMQTTSubscriber initializes the background connection to Mosquitto MQTT broker.
func StartMQTTSubscriber() {
	brokerURI := os.Getenv("MQTT_BROKER_URL")
	if brokerURI == "" {
		brokerURI = "tcp://mosquitto:1883"
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetClientID("kather_baksho_iot_subscriber_" + strconv.FormatInt(time.Now().UnixMilli(), 10))
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetKeepAlive(30 * time.Second)

	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("[MQTT] Successfully connected to broker at %s", brokerURI)
		topic := "kather_baksho/plants/+/telemetry"
		token := c.Subscribe(topic, 1, handleIncomingTelemetry)
		if token.Wait() && token.Error() != nil {
			log.Printf("[MQTT] Failed to subscribe to topic '%s': %v", topic, token.Error())
		} else {
			log.Printf("[MQTT] Subscribed to live plant telemetry topic: %s", topic)
		}
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v. Reconnecting in background...", err)
	}

	client := mqtt.NewClient(opts)
	MQTTClient = client

	// Start connection in background non-blocking goroutine
	go func() {
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			log.Printf("[MQTT] Initial connection to %s failed (will retry): %v", brokerURI, token.Error())
		}
	}()
}

func handleIncomingTelemetry(client mqtt.Client, msg mqtt.Message) {
	var payload MQTTTelemetryPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("[MQTT] Error unmarshalling payload on %s: %v", msg.Topic(), err)
		return
	}

	// Fallback plant ID from topic if missing: kather_baksho/plants/{plant_id}/telemetry
	if payload.PlantID == 0 {
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) >= 3 {
			if parsedID, err := strconv.ParseUint(parts[2], 10, 32); err == nil {
				payload.PlantID = uint(parsedID)
			}
		}
	}

	if payload.PlantName == "" {
		payload.PlantName = fmt.Sprintf("Botanical Planter #%d", payload.PlantID)
	}
	if payload.Location == "" {
		payload.Location = "Smart Garden"
	}
	if payload.BatteryPct == 0 {
		payload.BatteryPct = 100.0
	}

	// Evaluate plant wellness status
	status := "Optimal"
	alertMsg := "Plant environment is balanced and healthy."

	if payload.SoilMoisturePct < 22.0 {
		status = "Needs Water"
		alertMsg = fmt.Sprintf("Critically dry soil (%.1f%%)! Immediate watering required.", payload.SoilMoisturePct)
		SendPlantSoilAlert(payload.PlantID, payload.PlantName, payload.Location, payload.SoilMoisturePct)
	} else if payload.AmbientTempC > 36.0 {
		status = "High Heat"
		alertMsg = fmt.Sprintf("High temperature alert (%.1f°C)! Relocate to shaded spot.", payload.AmbientTempC)
		SendPlantHeatAlert(payload.PlantID, payload.PlantName, payload.Location, payload.AmbientTempC)
	} else if payload.LightLux < 150.0 {
		status = "Low Light"
		alertMsg = fmt.Sprintf("Low sunlight (%.0f lux). Consider moving closer to light source.", payload.LightLux)
	}

	record := models.PlantTelemetry{
		ID:              bson.NewObjectID(),
		PlantID:         payload.PlantID,
		PlantName:       payload.PlantName,
		Location:        payload.Location,
		SoilMoisturePct: payload.SoilMoisturePct,
		AmbientTempC:    payload.AmbientTempC,
		HumidityPct:     payload.HumidityPct,
		LightLux:        payload.LightLux,
		BatteryPct:      payload.BatteryPct,
		Status:          status,
		AlertMessage:    alertMsg,
		RecordedAt:      time.Now().UTC(),
	}

	// 1. Save to MongoDB
	if database.MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		coll := database.MongoDB.Collection("plant_telemetry")
		if _, err := coll.InsertOne(ctx, record); err != nil {
			log.Printf("[MQTT] MongoDB insert failed for plant %d: %v", record.PlantID, err)
		}
	}

	// 2. Cache latest telemetry in Redis & publish to live stream
	if database.RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		jsonBytes, _ := json.Marshal(record)
		cacheKey := fmt.Sprintf("iot:telemetry:%d:latest", record.PlantID)
		database.RedisClient.Set(ctx, cacheKey, jsonBytes, 24*time.Hour)
		database.RedisClient.Publish(ctx, "iot:telemetry:stream", jsonBytes)
	}

	log.Printf("[MQTT] Received telemetry for Plant #%d (%s): Moisture=%.1f%% Temp=%.1f°C Status=%s",
		record.PlantID, record.PlantName, record.SoilMoisturePct, record.AmbientTempC, record.Status)
}
