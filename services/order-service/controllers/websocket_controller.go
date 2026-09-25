package controllers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all local & frontend origins
	},
}

// GPSCheckpoint represents a delivery milestone on the Dhaka transit map.
type GPSCheckpoint struct {
	Step            int     `json:"step"`
	TotalSteps      int     `json:"total_steps"`
	Location        string  `json:"location"`
	Status          string  `json:"status"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	SpeedKMH        float64 `json:"speed_kmh"`
	ETAMinutes      int     `json:"eta_minutes"`
	ProgressPercent int     `json:"progress_percent"`
	Note            string  `json:"note"`
}

var dhakaDeliveryCheckpoints = []GPSCheckpoint{
	{
		Step:            1,
		TotalSteps:      6,
		Location:        "Mirpur Botanical Hub & Green Nursery",
		Status:          "Packed & Dispatched",
		Lat:             23.8223,
		Lng:             90.3542,
		SpeedKMH:        0.0,
		ETAMinutes:      28,
		ProgressPercent: 15,
		Note:            "Plant carefully secured in eco-friendly protective wooden frame.",
	},
	{
		Step:            2,
		TotalSteps:      6,
		Location:        "Mirpur-10 Roundabout Checkpoint",
		Status:          "In Transit",
		Lat:             23.8071,
		Lng:             90.3686,
		SpeedKMH:        32.4,
		ETAMinutes:      22,
		ProgressPercent: 35,
		Note:            "Courier eco-van en route along Begum Rokeya Avenue.",
	},
	{
		Step:            3,
		TotalSteps:      6,
		Location:        "Agargaon Link Road",
		Status:          "In Transit",
		Lat:             23.7785,
		Lng:             90.3752,
		SpeedKMH:        28.0,
		ETAMinutes:      16,
		ProgressPercent: 55,
		Note:            "Smooth transit; climate-controlled plant cargo bay active.",
	},
	{
		Step:            4,
		TotalSteps:      6,
		Location:        "Bijoy Sarani Expressway",
		Status:          "In Transit",
		Lat:             23.7650,
		Lng:             90.3880,
		SpeedKMH:        35.8,
		ETAMinutes:      10,
		ProgressPercent: 75,
		Note:            "Passing central avenue, approaching destination district.",
	},
	{
		Step:            5,
		TotalSteps:      6,
		Location:        "Entering Customer Neighborhood",
		Status:          "Out for Delivery",
		Lat:             23.7465,
		Lng:             90.3760,
		SpeedKMH:        18.5,
		ETAMinutes:      4,
		ProgressPercent: 90,
		Note:            "Rider arrived in local area. Contacting customer.",
	},
	{
		Step:            6,
		TotalSteps:      6,
		Location:        "Customer Doorstep (Delivery Complete)",
		Status:          "Delivered",
		Lat:             23.7420,
		Lng:             90.3730,
		SpeedKMH:        0.0,
		ETAMinutes:      0,
		ProgressPercent: 100,
		Note:            "Package handed over successfully. Enjoy your greenery!",
	},
}

// TrackOrderLiveWebSocket streams real-time courier GPS coordinates to connected clients.
// GET /ws/orders/:id/track
func TrackOrderLiveWebSocket(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error for order #%d: %v", orderID, err)
		return
	}
	defer conn.Close()

	log.Printf("[WebSocket] Client connected for live order tracking #%d", orderID)

	// Send initial handshake confirmation
	initMsg := gin.H{
		"event":       "connected",
		"order_id":    orderID,
		"rider_name":  "Mohammad Al-Amin",
		"rider_phone": "+880 1711-234567",
		"vehicle":     "Kather Baksho Green EV #DH-902",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(initMsg); err != nil {
		return
	}

	// Ticker for streaming simulated GPS checkpoints
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	stepIdx := 0

	// Also handle client incoming messages in background
	stopChan := make(chan struct{})
	go func() {
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				close(stopChan)
				return
			}
			if msg["action"] == "ping" {
				_ = conn.WriteJSON(gin.H{"event": "pong", "timestamp": time.Now().UTC().Format(time.RFC3339)})
			}
		}
	}()

	for {
		select {
		case <-stopChan:
			log.Printf("[WebSocket] Client disconnected from order #%d", orderID)
			return
		case <-ticker.C:
			checkpoint := dhakaDeliveryCheckpoints[stepIdx%len(dhakaDeliveryCheckpoints)]
			payload := gin.H{
				"event":            "gps_update",
				"order_id":         orderID,
				"rider_name":       "Mohammad Al-Amin",
				"rider_phone":      "+880 1711-234567",
				"step":             checkpoint.Step,
				"total_steps":      checkpoint.TotalSteps,
				"location":         checkpoint.Location,
				"status":           checkpoint.Status,
				"lat":              checkpoint.Lat,
				"lng":              checkpoint.Lng,
				"speed_kmh":        checkpoint.SpeedKMH,
				"eta_minutes":      checkpoint.ETAMinutes,
				"progress_percent": checkpoint.ProgressPercent,
				"note":             checkpoint.Note,
				"timestamp":        time.Now().UTC().Format(time.RFC3339),
			}

			if err := conn.WriteJSON(payload); err != nil {
				log.Printf("[WebSocket] Write error order #%d: %v", orderID, err)
				return
			}

			stepIdx++
		}
	}
}
