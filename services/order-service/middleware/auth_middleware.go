package middleware

import (
	"net/http"
	"strings"

	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Extract raw token from "Bearer <token>" header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := utils.ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Inject authenticated user ID into Gin context for downstream controllers
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		if role, ok := claims["role"].(string); ok && role == "2fa_pending" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Two-factor authentication required. Please verify your 2FA code."})
			c.Abort()
			return
		}

		c.Set("user_id", uint(userIDFloat))
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])

		c.Next()
	}
}
