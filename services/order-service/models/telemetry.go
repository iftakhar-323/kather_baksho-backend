package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// PlantTelemetry represents an IoT soil/climate sensor reading stored in MongoDB.
type PlantTelemetry struct {
	ID              bson.ObjectID `json:"id" bson:"_id,omitempty"`
	PlantID         uint          `json:"plant_id" bson:"plant_id"`
	PlantName       string        `json:"plant_name" bson:"plant_name"`
	Location        string        `json:"location" bson:"location"`
	SoilMoisturePct float64       `json:"soil_moisture_pct" bson:"soil_moisture_pct"`
	AmbientTempC    float64       `json:"ambient_temp_c" bson:"ambient_temp_c"`
	HumidityPct     float64       `json:"humidity_pct" bson:"humidity_pct"`
	LightLux        float64       `json:"light_lux" bson:"light_lux"`
	BatteryPct      float64       `json:"battery_pct" bson:"battery_pct"`
	Status          string        `json:"status" bson:"status"` // Optimal, Needs Water, Low Light, High Heat
	AlertMessage    string        `json:"alert_message" bson:"alert_message"`
	RecordedAt      time.Time     `json:"recorded_at" bson:"recorded_at"`
}

// AuditLog represents an unstructured activity or security audit entry stored in MongoDB.
type AuditLog struct {
	ID        bson.ObjectID          `json:"id" bson:"_id,omitempty"`
	Action    string                 `json:"action" bson:"action"`
	UserID    uint                   `json:"user_id" bson:"user_id"`
	UserEmail string                 `json:"user_email" bson:"user_email"`
	Entity    string                 `json:"entity" bson:"entity"`
	EntityID  string                 `json:"entity_id" bson:"entity_id"`
	Details   map[string]interface{} `json:"details" bson:"details"`
	IPAddress string                 `json:"ip_address" bson:"ip_address"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
}
