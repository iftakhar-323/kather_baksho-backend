# Kather Baksho — Real Botanical IoT Hardware & ESP32 Firmware

This directory contains the production-ready firmware and hardware design to connect physical potted plants, indoor gardens, and botanical planters directly to the **Kather Baksho** cloud platform over MQTT.

---

## 🛠️ 1. Bill of Materials (Approximate Cost: $6 - $10 USD)

| Component | Specification | Purpose | Approx Price (BDT / USD) |
| :--- | :--- | :--- | :--- |
| **ESP32 Dev Module** | ESP-WROOM-32 (WiFi + Bluetooth) | Edge microcontroller | ~450 BDT ($3.80) |
| **Capacitive Soil Sensor** | v1.2 (Corrosion resistant) | Measures volumetric water content | ~120 BDT ($1.00) |
| **DHT22 / AM2302** | Digital Temp & Humidity | Environmental microclimate sensing | ~250 BDT ($2.10) |
| **LDR Sensor Module** | 5528 Light Dependent Resistor | Ambient sunlight / lux monitoring | ~40 BDT ($0.35) |
| **Breadboard & Jumpers** | Female-to-Female & Dupont wires | Interconnecting components | ~80 BDT ($0.70) |
| **Micro USB Cable / Battery** | 5V 1A or 3.7V 18650 Li-Ion | Continuous or solar power | ~150 BDT ($1.25) |

---

## 🔌 2. Wiring & Pinout Diagram

```
                +-------------------+
                |     ESP32 DEV     |
                |                   |
Capacitive Soil |                   |
VCC (3.3V) ---- | 3V3           GND | ---- Soil GND / DHT GND / LDR GND
AOUT (Signal) - | GPIO 34 (ADC1)    |
                |                   |
DHT22 Temp/Hum  |                   |
VCC (3.3V) ---- | 3V3               |
DATA ---------- | GPIO 4            | (Add 10k pullup resistor between DATA & 3V3)
                |                   |
LDR Light       |                   |
VCC (3.3V) ---- | 3V3               |
AO (Signal) --- | GPIO 32 (ADC1)    |
                +-------------------+
```

> **IMPORTANT**: We explicitly utilize **ADC1** pins (`GPIO 34` and `GPIO 32`) because ADC2 channels on the ESP32 cannot be used simultaneously when the WiFi radio is active.

---

## ⚙️ 3. Quick Setup & Flashing Guide

### Option A: Arduino IDE
1. Install **Arduino IDE** (version 2.0+ recommended).
2. Go to **File &rarr; Preferences** and paste this into *Additional Board Manager URLs*:
   ```
   https://raw.githubusercontent.com/espressif/arduino-esp32/gh-pages/package_esp32_index.json
   ```
3. Go to **Tools &rarr; Board &rarr; Boards Manager**, search for `esp32` by Espressif Systems, and click **Install**.
4. Go to **Tools &rarr; Manage Libraries**, install:
   - `PubSubClient` by Nick O'Leary
   - `DHT sensor library` by Adafruit
   - `Adafruit Unified Sensor`
5. Open `firmware/esp32_plant_monitor/esp32_plant_monitor.ino`.
6. Set your WiFi credentials and your KatherBaksho server IP:
   ```cpp
   const char* WIFI_SSID     = "MyHomeWiFi";
   const char* WIFI_PASSWORD = "MySecretPassword";
   const char* MQTT_SERVER   = "192.168.1.50"; // Local IP where docker-compose is running
   ```
7. Select **Board: "ESP32 Dev Module"**, connect via USB, and click **Upload**.

---

## 🧪 4. Soil Moisture Calibration

Different soil mixes (coco peat, loam, perlite) have slightly different dielectric constants. To calibrate:
1. Hold the sensor in open air: note down the Serial monitor reading (e.g. `3200`). Update `AIR_VALUE = 3200`.
2. Dip the sensor into a cup of water up to the white line: note down reading (e.g. `1300`). Update `WATER_VALUE = 1300`.
3. The firmware will now output true `0% to 100%` moisture percentages.

---

## 📡 5. MQTT Topic & Telemetry Schema

The ESP32 publishes to:
`kather_baksho/plants/{plant_id}/telemetry`

**Payload Example:**
```json
{
  "device_id": "esp32-kb-botanical-01",
  "plant_id": 1,
  "plant_name": "Monstera Deliciosa (Smart Planter)",
  "location": "Living Room Corner",
  "soil_moisture_pct": 48.5,
  "ambient_temp_c": 26.2,
  "humidity_pct": 62.0,
  "light_lux": 820.0,
  "battery_pct": 98.0
}
```

The Go backend service (`iot-ai-service`) automatically consumes this topic, checks health thresholds, writes to MongoDB, and updates real-time caches in Redis!
