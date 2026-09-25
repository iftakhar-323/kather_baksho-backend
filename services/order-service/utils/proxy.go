package utils

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

// ProxyToService forwards an incoming HTTP request to a dedicated microservice.
// Returns true if the request was proxied, or false if the microservice is unconfigured.
func ProxyToService(c *gin.Context, envVarName string) bool {
	targetBase := os.Getenv(envVarName)
	if targetBase == "" {
		return false
	}

	targetURL, err := url.Parse(targetBase)
	if err != nil {
		return false
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		if corrID := c.GetString("correlation_id"); corrID != "" {
			req.Header.Set("X-Correlation-ID", corrID)
		}
		if reqID := c.GetString("request_id"); reqID != "" {
			req.Header.Set("X-Request-ID", reqID)
		}
	}

	proxy.ServeHTTP(c.Writer, c.Request)
	return true
}
