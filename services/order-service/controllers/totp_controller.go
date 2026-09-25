package controllers

import (
	"net/http"
	"strings"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/services"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type EnableTOTPInput struct {
	Code string `json:"code" binding:"required"`
}

type VerifyTOTPInput struct {
	TempToken    string `json:"temp_token" binding:"required"`
	Code         string `json:"code"`
	RecoveryCode string `json:"recovery_code"`
}

type DisableTOTPInput struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// POST /api/auth/2fa/setup - Start 2FA enrollment: returns secret and otpauth URI
func SetupTOTP(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	secret, err := services.GenerateTOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate 2FA secret"})
		return
	}

	// Persist secret in pending state
	user.TOTPSecret = secret
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store 2FA secret"})
		return
	}

	otpauthURI := services.GenerateTOTPURI(secret, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Scan QR code or enter secret manually into Google Authenticator or Authy",
		"secret":      secret,
		"otpauth_uri": otpauthURI,
		"email":       user.Email,
		"issuer":      "KatherBaksho",
	})
}

// POST /api/auth/2fa/enable - Complete 2FA enrollment by verifying first OTP code
func EnableTOTP(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	var input EnableTOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA setup has not been initiated. Call /setup first."})
		return
	}

	if !services.VerifyTOTPCode(user.TOTPSecret, input.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 6-digit verification code. Please check your authenticator clock."})
		return
	}

	recoveryCodes := services.GenerateRecoveryCodes(8)
	user.TOTPEnabled = true
	user.TOTPRecoveryCodes = strings.Join(recoveryCodes, ",")

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Two-factor authentication successfully enabled",
		"totp_enabled":   true,
		"recovery_codes": recoveryCodes,
	})
}

// POST /api/auth/2fa/verify - Complete login challenge with 6-digit TOTP code or recovery code
func VerifyTOTP(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	var input VerifyTOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate temp token
	token, err := utils.ValidateJWT(input.TempToken)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired 2FA session token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	if role, ok := claims["role"].(string); !ok || role != "2fa_pending" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not a pending 2FA token"})
		return
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uint(userIDFloat)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled for this user"})
		return
	}

	verified := false

	// Attempt TOTP verification
	if input.Code != "" {
		if services.VerifyTOTPCode(user.TOTPSecret, input.Code) {
			verified = true
		}
	}

	// Attempt Recovery Code verification
	if !verified && input.RecoveryCode != "" {
		cleanRecovery := strings.TrimSpace(strings.ToUpper(input.RecoveryCode))
		existingCodes := strings.Split(user.TOTPRecoveryCodes, ",")
		newCodes := make([]string, 0, len(existingCodes))
		for _, rc := range existingCodes {
			trimmed := strings.TrimSpace(rc)
			if trimmed != "" && strings.ToUpper(trimmed) == cleanRecovery && !verified {
				verified = true // Consumed code
			} else if trimmed != "" {
				newCodes = append(newCodes, trimmed)
			}
		}

		if verified {
			user.TOTPRecoveryCodes = strings.Join(newCodes, ",")
			_ = database.DB.Save(&user)
		}
	}

	if !verified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code or recovery code"})
		return
	}

	// Issue standard JWT
	authToken, _ := utils.GenerateJWT(user.ID, user.Email, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"message": "Two-factor authentication successful",
		"token":   authToken,
		"user": gin.H{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"role":         user.Role,
			"totp_enabled": true,
		},
	})
}

// POST /api/auth/2fa/disable - Disable 2FA with password or valid TOTP code
func DisableTOTP(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	var input DisableTOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	authorized := false
	if input.Password != "" && utils.CheckPasswordHash(input.Password, user.Password) {
		authorized = true
	} else if input.Code != "" && services.VerifyTOTPCode(user.TOTPSecret, input.Code) {
		authorized = true
	}

	if !authorized {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password or OTP code to disable 2FA"})
		return
	}

	user.TOTPEnabled = false
	user.TOTPSecret = ""
	user.TOTPRecoveryCodes = ""

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Two-factor authentication disabled successfully",
		"totp_enabled": false,
	})
}

// GET /api/auth/2fa/status - Get current user 2FA status
func StatusTOTP(c *gin.Context) {
	if utils.ProxyToService(c, "AUTH_SERVICE_URL") {
		return
	}

	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totp_enabled": user.TOTPEnabled,
	})
}
