package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OpenAPI JSON Specification
const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "Kather Baksho Enterprise API",
    "version": "3.0.0",
    "description": "Production-grade, highly-resilient, event-driven REST and WebSocket API specification for Kather Baksho e-commerce ecosystem with Polyglot persistence (SQLite, Redis, MongoDB), Traefik API gateway, and Goroutine WebSockets.",
    "contact": {
      "name": "Kather Baksho Engineering Team",
      "email": "dev@katherbaksho.local"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8085",
      "description": "Traefik Cloud-Native Ingress API Gateway"
    },
    {
      "url": "http://localhost:8081",
      "description": "Direct Go Core Backend Service"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Enter your JWT token obtained from /api/auth/login"
      }
    },
    "schemas": {
      "StandardResponse": {
        "type": "object",
        "properties": {
          "success": { "type": "boolean" },
          "message": { "type": "string" },
          "data": { "type": "object" }
        }
      },
      "HealthReady": {
        "type": "object",
        "properties": {
          "status": { "type": "string", "example": "ready" },
          "timestamp": { "type": "string", "format": "date-time" },
          "checks": {
            "type": "object",
            "properties": {
              "sqlite": { "type": "string", "example": "connected (WAL mode active)" },
              "redis": { "type": "string", "example": "connected (PONG)" },
              "mongodb": { "type": "string", "example": "connected (MongoDB 7.0 ping ok)" }
            }
          }
        }
      },
      "PlantTelemetry": {
        "type": "object",
        "properties": {
          "plant_id": { "type": "integer", "example": 1 },
          "species": { "type": "string", "example": "Bonsai Ficus Retusa" },
          "moisture_percent": { "type": "number", "example": 42.5 },
          "temperature_c": { "type": "number", "example": 26.8 },
          "humidity_percent": { "type": "number", "example": 68.2 },
          "sunlight_lux": { "type": "number", "example": 850.0 },
          "ph_level": { "type": "number", "example": 6.5 },
          "botanical_status": { "type": "string", "example": "Optimal Health" }
        }
      }
    }
  },
  "paths": {
    "/health/ready": {
      "get": {
        "tags": ["Health & Diagnostics"],
        "summary": "Tri-Database Health Readiness Probe",
        "description": "Validates SQLite WAL, Redis Cache, and MongoDB Polyglot persistence simultaneously.",
        "responses": {
          "200": {
            "description": "All datastores healthy and ready",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/HealthReady" }
              }
            }
          }
        }
      }
    },
    "/metrics": {
      "get": {
        "tags": ["Health & Diagnostics"],
        "summary": "Prometheus Metrics Exporter",
        "description": "Exposes HTTP request durations, status code counts, and active connections.",
        "responses": {
          "200": { "description": "Prometheus metrics in standard text exposition format" }
        }
      }
    },
    "/api/products/": {
      "get": {
        "tags": ["Catalog & Products"],
        "summary": "List Products with Redis Caching",
        "parameters": [
          { "name": "category", "in": "query", "schema": { "type": "string" } },
          { "name": "search", "in": "query", "schema": { "type": "string" } },
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 12 } }
        ],
        "responses": {
          "200": { "description": "Filtered and paginated list of botanical wooden crafts" }
        }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Authentication & Users"],
        "summary": "User & Admin JWT Authentication",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "format": "email", "example": "admin@katherbaksho.com" },
                  "password": { "type": "string", "example": "admin123" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "JWT Bearer token and user profile" }
        }
      }
    },
    "/api/iot/telemetry": {
      "post": {
        "tags": ["IoT Smart Plant Telemetry (MongoDB)"],
        "summary": "Ingest Botanical Sensor Telemetry",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/PlantTelemetry" }
            }
          }
        },
        "responses": {
          "201": { "description": "Telemetry ingested into MongoDB time-series collection" }
        }
      }
    },
    "/api/iot/plants": {
      "get": {
        "tags": ["IoT Smart Plant Telemetry (MongoDB)"],
        "summary": "Get Monitored Botanical Specimens",
        "responses": {
          "200": { "description": "List of all actively monitored plant sensors and their latest metrics" }
        }
      }
    },
    "/api/orders/{id}/invoice/pdf": {
      "get": {
        "tags": ["Microservices & Invoices"],
        "summary": "Generate Vector Invoice PDF via Node.js Microservice",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }
        ],
        "responses": {
          "200": { "description": "Binary PDF document stream" }
        }
      }
    },
    "/ws/orders/{id}/track": {
      "get": {
        "tags": ["Real-Time WebSockets"],
        "summary": "Live Delivery Courier GPS Stream",
        "description": "Upgrades HTTP connection to WebSocket and streams real-time courier coordinates across Dhaka checkpoints.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "integer", "example": 1 } }
        ],
        "responses": {
          "101": { "description": "Switching Protocols to WebSocket" }
        }
      }
    }
  }
}`

// ServeOpenAPISpec returns the raw OpenAPI 3.0 specification JSON
func ServeOpenAPISpec(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, openAPISpecJSON)
}

// ServeSwaggerUI renders an interactive, self-contained Swagger UI
func ServeSwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Kather Baksho API Documentation & Interactive Explorer</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
  <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
  <style>
    body { margin: 0; padding: 0; background: #fafafa; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .topbar { display: none !important; }
    .swagger-ui .info { margin: 30px 0 20px 0; }
    .swagger-ui .info .title { color: #065f46; font-size: 28px; font-weight: 800; }
    .custom-header {
      background: linear-gradient(135deg, #064e3b 0%, #047857 100%);
      color: white;
      padding: 16px 32px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      box-shadow: 0 2px 8px rgba(0,0,0,0.15);
    }
    .custom-header h1 { margin: 0; font-size: 20px; display: flex; align-items: center; gap: 8px; }
    .custom-header .badge { background: #10b981; padding: 4px 10px; border-radius: 9999px; font-size: 12px; font-weight: bold; }
    .custom-header a { color: #a7f3d0; text-decoration: none; font-size: 13px; font-weight: 500; }
    .custom-header a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <div class="custom-header">
    <h1>🌱 Kather Baksho <span>API Explorer</span></h1>
    <div style="display: flex; gap: 16px; align-items: center;">
      <span class="badge">OpenAPI 3.0</span>
      <a href="/api/docs/openapi.json" target="_blank">Download OpenAPI JSON ↗</a>
      <a href="http://localhost:8085" target="_blank">Traefik Ingress Gateway ↗</a>
      <a href="http://localhost:8082" target="_blank">Web Storefront ↗</a>
    </div>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/api/docs/openapi.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
