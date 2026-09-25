package controllers

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// In-memory fallback dataset in case MongoDB is temporarily starting or offline
var fallbackMonitoredPlants = []models.PlantTelemetry{
	{
		PlantID:         1,
		PlantName:       "Monstera Deliciosa (Swiss Cheese)",
		Location:        "Living Room Corner",
		SoilMoisturePct: 48.5,
		AmbientTempC:    24.2,
		HumidityPct:     62.0,
		LightLux:        820.0,
		BatteryPct:      94.0,
		Status:          "Optimal",
		AlertMessage:    "Plant environment is balanced and healthy.",
		RecordedAt:      time.Now(),
	},
	{
		PlantID:         2,
		PlantName:       "Fiddle Leaf Fig",
		Location:        "Balcony Garden",
		SoilMoisturePct: 21.0,
		AmbientTempC:    29.5,
		HumidityPct:     55.0,
		LightLux:        1450.0,
		BatteryPct:      88.0,
		Status:          "Needs Water",
		AlertMessage:    "Soil moisture critically low (21%)! Watering required.",
		RecordedAt:      time.Now(),
	},
	{
		PlantID:         3,
		PlantName:       "Sansevieria Snake Plant",
		Location:        "Bedroom Desk",
		SoilMoisturePct: 35.0,
		AmbientTempC:    23.0,
		HumidityPct:     58.0,
		LightLux:        320.0,
		BatteryPct:      98.0,
		Status:          "Optimal",
		AlertMessage:    "Hardy plant performing well in ambient bedroom light.",
		RecordedAt:      time.Now(),
	},
	{
		PlantID:         4,
		PlantName:       "Peace Lily (Spathiphyllum)",
		Location:        "Study Room Shelf",
		SoilMoisturePct: 18.5,
		AmbientTempC:    25.1,
		HumidityPct:     50.0,
		LightLux:        95.0,
		BatteryPct:      79.0,
		Status:          "Low Light",
		AlertMessage:    "Low light detected (95 lux). Move closer to window for flowering.",
		RecordedAt:      time.Now(),
	},
}

// IngestPlantTelemetry receives an IoT sensor payload and saves it into MongoDB.
// POST /api/iot/telemetry
func IngestPlantTelemetry(c *gin.Context) {
	if utils.ProxyToService(c, "IOT_AI_SERVICE_URL") {
		return
	}

	var payload models.PlantTelemetry
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telemetry payload: " + err.Error()})
		return
	}

	if payload.PlantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plant_id is required"})
		return
	}

	if payload.RecordedAt.IsZero() {
		payload.RecordedAt = time.Now().UTC()
	}

	// Compute automated botanical status & alert
	if payload.SoilMoisturePct < 25.0 {
		payload.Status = "Needs Water"
		payload.AlertMessage = "Soil moisture critically low! Immediate watering required."
	} else if payload.AmbientTempC > 34.0 {
		payload.Status = "High Heat"
		payload.AlertMessage = "Excessive heat detected. Shield from direct midday scorching sun."
	} else if payload.LightLux < 120.0 {
		payload.Status = "Low Light"
		payload.AlertMessage = "Light levels too dim for photosynthesis. Relocate closer to light."
	} else {
		payload.Status = "Optimal"
		payload.AlertMessage = "Plant environment is balanced and healthy."
	}

	if database.MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		coll := database.MongoDB.Collection("plant_telemetry")
		_, err := coll.InsertOne(ctx, payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist to MongoDB: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "IoT telemetry recorded successfully",
		"telemetry": payload,
		"database":  "mongodb",
	})
}

// GetPlantTelemetry returns the latest snapshot of all monitored plants.
// GET /api/iot/plants
func GetMonitoredPlants(c *gin.Context) {
	if utils.ProxyToService(c, "IOT_AI_SERVICE_URL") {
		return
	}

	if database.MongoDB == nil {
		c.Header("X-Mongo-Fallback", "active")
		c.JSON(http.StatusOK, gin.H{
			"plants":    fallbackMonitoredPlants,
			"source":    "memory_fallback",
			"timestamp": time.Now().UTC(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	coll := database.MongoDB.Collection("plant_telemetry")

	// Get latest entry for distinct plant_ids
	plantIDs := []uint{1, 2, 3, 4}
	var results []models.PlantTelemetry

	for _, pid := range plantIDs {
		var item models.PlantTelemetry
		opts := options.FindOne().SetSort(bson.D{{Key: "recorded_at", Value: -1}})
		err := coll.FindOne(ctx, bson.M{"plant_id": pid}, opts).Decode(&item)
		if err == nil {
			results = append(results, item)
		}
	}

	if len(results) == 0 {
		c.Header("X-Mongo-Fallback", "seeded_memory")
		c.JSON(http.StatusOK, gin.H{
			"plants":    fallbackMonitoredPlants,
			"source":    "fallback",
			"timestamp": time.Now().UTC(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plants":    results,
		"source":    "mongodb",
		"timestamp": time.Now().UTC(),
	})
}

// GetPlantTelemetryHistory returns historical time-series data for a specific plant.
// GET /api/iot/telemetry/:plant_id
func GetPlantTelemetryHistory(c *gin.Context) {
	if utils.ProxyToService(c, "IOT_AI_SERVICE_URL") {
		return
	}

	pidStr := c.Param("plant_id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plant_id"})
		return
	}

	limit := 24
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	if database.MongoDB == nil {
		c.Header("X-Mongo-Fallback", "synthetic_history")
		// Generate synthetic 24h curve for visualization
		var history []models.PlantTelemetry
		baseTime := time.Now().Add(-24 * time.Hour)
		for i := 0; i < limit; i++ {
			t := baseTime.Add(time.Duration(i) * time.Hour)
			moisture := 55.0 - float64(i)*1.2 + math.Sin(float64(i))*3.0
			history = append(history, models.PlantTelemetry{
				PlantID:         uint(pid),
				PlantName:       "Monstera Deliciosa",
				SoilMoisturePct: math.Round(moisture*10) / 10,
				AmbientTempC:    math.Round((24.0+math.Sin(float64(i)/3.0)*4.0)*10) / 10,
				HumidityPct:     math.Round((60.0+math.Cos(float64(i)/3.0)*8.0)*10) / 10,
				LightLux:        math.Max(50.0, math.Round((500.0+math.Sin(float64(i)/2.0)*450.0)*10)/10),
				Status:          "Optimal",
				RecordedAt:      t,
			})
		}
		c.JSON(http.StatusOK, gin.H{"plant_id": pid, "history": history, "source": "synthetic"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	coll := database.MongoDB.Collection("plant_telemetry")
	opts := options.Find().
		SetSort(bson.D{{Key: "recorded_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := coll.Find(ctx, bson.M{"plant_id": uint(pid)}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MongoDB query failed: " + err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var history []models.PlantTelemetry
	if err := cursor.All(ctx, &history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode documents: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plant_id": pid,
		"count":    len(history),
		"history":  history,
		"source":   "mongodb",
	})
}

// SeedIoTTelemetry populates MongoDB with realistic sensor readings.
// POST /api/iot/seed
func SeedIoTTelemetry(c *gin.Context) {
	if utils.ProxyToService(c, "IOT_AI_SERVICE_URL") {
		return
	}

	if database.MongoDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MongoDB connection not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := database.MongoDB.Collection("plant_telemetry")

	plants := []struct {
		ID       uint
		Name     string
		Location string
		BaseSoil float64
	}{
		{ID: 1, Name: "Monstera Deliciosa", Location: "Living Room Window", BaseSoil: 48.0},
		{ID: 2, Name: "Fiddle Leaf Fig", Location: "Sunny Balcony", BaseSoil: 22.0},
		{ID: 3, Name: "Snake Plant", Location: "Master Bedroom", BaseSoil: 38.0},
		{ID: 4, Name: "Peace Lily", Location: "Office Desk", BaseSoil: 26.0},
	}

	now := time.Now().UTC()
	var docs []interface{}

	for _, p := range plants {
		for hour := 24; hour >= 0; hour-- {
			recTime := now.Add(-time.Duration(hour) * time.Hour)
			soil := p.BaseSoil + float64(hour)*0.6 + math.Sin(float64(hour))*2.5
			if soil > 85.0 {
				soil = 85.0
			} else if soil < 12.0 {
				soil = 12.0
			}
			temp := 23.5 + math.Sin(float64(hour)/3.5)*4.5
			lux := math.Max(60.0, 600.0+math.Sin(float64(hour)/2.0)*500.0)
			status := "Optimal"
			alert := "Environment healthy."
			if soil < 25.0 {
				status = "Needs Water"
				alert = "Soil moisture critically low!"
			}

			docs = append(docs, models.PlantTelemetry{
				PlantID:         p.ID,
				PlantName:       p.Name,
				Location:        p.Location,
				SoilMoisturePct: math.Round(soil*10) / 10,
				AmbientTempC:    math.Round(temp*10) / 10,
				HumidityPct:     math.Round((58.0+math.Cos(float64(hour)/4.0)*10.0)*10) / 10,
				LightLux:        math.Round(lux),
				BatteryPct:      95.0,
				Status:          status,
				AlertMessage:    alert,
				RecordedAt:      recTime,
			})
		}
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed telemetry: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "MongoDB IoT plant sensor telemetry seeded successfully",
		"insertedCount": len(res.InsertedIDs),
		"plants":        len(plants),
	})
}
