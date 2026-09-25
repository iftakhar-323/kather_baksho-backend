/*
 * Kather Baksho — Real Botanical IoT Planter Monitor
 * Microcontroller: ESP32 Dev Module / ESP32 NodeMCU
 *
 * Sensors:
 * 1. Capacitive Soil Moisture Sensor v1.2 (Connected to ADC1 GPIO 34)
 * 2. DHT22 / DHT11 Digital Temp & Humidity Sensor (Connected to GPIO 4)
 * 3. Light Dependent Resistor (LDR) / Ambient Lux (Connected to ADC1 GPIO 32)
 *
 * Protocol: MQTT over TCP (Broker: Mosquitto on port 1883)
 * Telemetry Topic: kather_baksho/plants/{PLANT_ID}/telemetry
 */

#include <WiFi.h>
#include <PubSubClient.h>
#include "DHT.h"

// ================= USER CONFIGURATION =================
const char* WIFI_SSID       = "YOUR_WIFI_SSID";
const char* WIFI_PASSWORD   = "YOUR_WIFI_PASSWORD";

const char* MQTT_SERVER     = "192.168.1.100"; // IP address of Docker host / KatherBaksho server
const int   MQTT_PORT       = 1883;
const char* MQTT_USER       = "";              // Optional if auth enabled
const char* MQTT_PASS       = "";              // Optional if auth enabled

const uint32_t PLANT_ID     = 1;
const char*    PLANT_NAME   = "Monstera Deliciosa (Smart Planter)";
const char*    LOCATION     = "Living Room Corner";
const char*    DEVICE_ID    = "esp32-kb-botanical-01";

// Pin Definitions (Using ADC1 pins which do NOT conflict with WiFi)
#define SOIL_PIN            34  // Capacitive Soil Moisture (Analog)
#define LDR_PIN             32  // Ambient Light Sensor (Analog)
#define DHT_PIN              4  // DHT22 / DHT11 Data Pin
#define DHT_TYPE         DHT22  // DHT 22 (AM2302), change to DHT11 if using DHT11
#define LED_PIN              2  // Onboard Blue Status LED

// Soil Calibration Values (12-bit ADC: 0 - 4095)
// Calibrate by reading sensor in dry air vs fully dipped in water
const int AIR_VALUE         = 3200; // Sensor in dry air (0% moisture)
const int WATER_VALUE       = 1300; // Sensor submerged in water (100% moisture)

// Telemetry Interval (Milliseconds)
const unsigned long TELEMETRY_INTERVAL_MS = 15000; // 15 seconds

// Deep sleep power saving mode for battery operation
const bool ENABLE_DEEP_SLEEP = false;
const uint64_t SLEEP_DURATION_SEC = 300; // 5 minutes in deep sleep

// ================= SYSTEM OBJECTS =================
WiFiClient espClient;
PubSubClient mqttClient(espClient);
DHT dht(DHT_PIN, DHT_TYPE);

unsigned long lastTelemetryTime = 0;
char topicBuffer[128];

void setupWiFi() {
  delay(10);
  Serial.println();
  Serial.printf("[WiFi] Connecting to %s", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  int attempts = 0;
  while (WiFi.status() != WL_CONNECTED && attempts < 25) {
    delay(500);
    Serial.print(".");
    digitalWrite(LED_PIN, !digitalRead(LED_PIN)); // Blink LED while connecting
    attempts++;
  }

  if (WiFi.status() == WL_CONNECTED) {
    digitalWrite(LED_PIN, HIGH);
    Serial.println("\n[WiFi] Connected successfully!");
    Serial.printf("[WiFi] IP Address: %s\n", WiFi.localIP().toString().c_str());
  } else {
    Serial.println("\n[WiFi] Connection failed! Retrying in background...");
  }
}

void reconnectMQTT() {
  while (!mqttClient.connected() && WiFi.status() == WL_CONNECTED) {
    Serial.printf("[MQTT] Attempting connection to broker %s:%d...\n", MQTT_SERVER, MQTT_PORT);
    String clientId = "KatherBaksho-ESP32-" + String(DEVICE_ID);

    bool connected = false;
    if (strlen(MQTT_USER) > 0) {
      connected = mqttClient.connect(clientId.c_str(), MQTT_USER, MQTT_PASS);
    } else {
      connected = mqttClient.connect(clientId.c_str());
    }

    if (connected) {
      Serial.println("[MQTT] Connected to Mosquitto broker!");
      // Flash LED twice on success
      digitalWrite(LED_PIN, LOW); delay(100); digitalWrite(LED_PIN, HIGH); delay(100);
      digitalWrite(LED_PIN, LOW); delay(100); digitalWrite(LED_PIN, HIGH);
    } else {
      Serial.printf("[MQTT] Connection failed (rc=%d). Retrying in 5 seconds...\n", mqttClient.state());
      delay(5000);
    }
  }
}

float readSoilMoisture() {
  // Take 5 samples and average to filter out electrical noise
  long sum = 0;
  for (int i = 0; i < 5; i++) {
    sum += analogRead(SOIL_PIN);
    delay(10);
  }
  int raw = sum / 5;

  // Constrain within calibrated bounds
  raw = constrain(raw, WATER_VALUE, AIR_VALUE);

  // Map inverse: AIR_VALUE (dry) -> 0.0%, WATER_VALUE (wet) -> 100.0%
  float moisture = (float)(AIR_VALUE - raw) * 100.0 / (float)(AIR_VALUE - WATER_VALUE);
  return constrain(moisture, 0.0, 100.0);
}

float readLightLux() {
  int raw = analogRead(LDR_PIN);
  // Approximated linear lux conversion for standard 5528 LDR
  float lux = map(raw, 0, 4095, 2500, 10);
  return constrain(lux, 10.0, 2500.0);
}

void sendTelemetry() {
  float soilMoisture = readSoilMoisture();
  float tempC = dht.readTemperature();
  float humidity = dht.readHumidity();
  float lightLux = readLightLux();
  float batteryPct = 98.0; // Simulated or measured via voltage divider on GPIO 35

  // Fallbacks if sensor read fails
  if (isnan(tempC)) tempC = 25.0;
  if (isnan(humidity)) humidity = 60.0;

  // Build JSON Payload without external heavy dependencies
  char payload[512];
  snprintf(payload, sizeof(payload),
    "{"
      "\"device_id\":\"%s\","
      "\"plant_id\":%u,"
      "\"plant_name\":\"%s\","
      "\"location\":\"%s\","
      "\"soil_moisture_pct\":%.2f,"
      "\"ambient_temp_c\":%.2f,"
      "\"humidity_pct\":%.2f,"
      "\"light_lux\":%.1f,"
      "\"battery_pct\":%.1f"
    "}",
    DEVICE_ID, PLANT_ID, PLANT_NAME, LOCATION,
    soilMoisture, tempC, humidity, lightLux, batteryPct
  );

  snprintf(topicBuffer, sizeof(topicBuffer), "kather_baksho/plants/%u/telemetry", PLANT_ID);

  Serial.printf("[Telemetry] Publishing to %s: %s\n", topicBuffer, payload);
  if (mqttClient.publish(topicBuffer, payload)) {
    Serial.println("[Telemetry] Published successfully!");
  } else {
    Serial.println("[Telemetry] Publish failed! MQTT buffer full or disconnected.");
  }
}

void setup() {
  Serial.begin(115200);
  pinMode(LED_PIN, OUTPUT);
  pinMode(SOIL_PIN, INPUT);
  pinMode(LDR_PIN, INPUT);

  analogReadResolution(12); // 0 - 4095 range
  dht.begin();

  Serial.println("\n============================================");
  Serial.println("  Kather Baksho Botanical Smart Planter     ");
  Serial.println("============================================");

  setupWiFi();
  mqttClient.setServer(MQTT_SERVER, MQTT_PORT);
  mqttClient.setBufferSize(1024);

  // Send initial reading immediately
  if (WiFi.status() == WL_CONNECTED) {
    reconnectMQTT();
    sendTelemetry();
  }

  if (ENABLE_DEEP_SLEEP) {
    Serial.printf("[Power] Entering deep sleep for %llu seconds...\n", SLEEP_DURATION_SEC);
    esp_sleep_enable_timer_wakeup(SLEEP_DURATION_SEC * 1000000ULL);
    esp_deep_sleep_start();
  }
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) {
    setupWiFi();
  }

  if (!mqttClient.connected()) {
    reconnectMQTT();
  }
  mqttClient.loop();

  unsigned long now = millis();
  if (now - lastTelemetryTime >= TELEMETRY_INTERVAL_MS) {
    lastTelemetryTime = now;
    sendTelemetry();
  }
}
