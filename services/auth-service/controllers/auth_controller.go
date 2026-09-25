package controllers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"kather_baksho/auth_service/database"
	"kather_baksho/auth_service/mailer"
	"kather_baksho/auth_service/models"
	"kather_baksho/auth_service/utils"

	"github.com/gin-gonic/gin"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email is already registered
	var existingUser models.User
	if err := database.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: hashedPassword,
		Role:     "customer",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	token, _ := utils.GenerateJWT(user.ID, user.Email, user.Role)

	// Issue a 6-digit verification code (15-min TTL) and best-effort send
	// the welcome email. The send call is safe to ignore if no Brevo key is
	// configured — it just logs the would-be subject line.
	if code, err := issueVerificationCode(user.ID); err == nil {
		_ = mailer.Send(mailer.VerifyEmail(user.Email, code))
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"token":   token,
		"user": gin.H{
			"id":             user.ID,
			"name":           user.Name,
			"email":          user.Email,
			"role":           user.Role,
			"email_verified": user.EmailVerified,
		},
	})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if user.TOTPEnabled {
		tempToken, err := utils.Generate2FAPendingToken(user.ID, user.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue 2FA challenge"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"requires_2fa": true,
			"temp_token":   tempToken,
			"message":      "Two-factor authentication required",
		})
		return
	}

	token, _ := utils.GenerateJWT(user.ID, user.Email, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"role":         user.Role,
			"totp_enabled": user.TOTPEnabled,
		},
	})
}

// GET /api/auth/me - Current user profile info (fetched from DB to prevent stale JWT claims)
func Me(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"phone":          user.Phone,
		"address":        user.Address,
		"avatar_url":     user.AvatarURL,
		"role":           user.Role,
		"points":         user.Points,
		"email_verified": user.EmailVerified,
		"totp_enabled":   user.TOTPEnabled,
	})
}

// =====================================================================
// Sprint A — Profile, Change Password, Delete Account, Address Book
// =====================================================================

type UpdateProfileInput struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	AvatarURL string `json:"avatar_url"`
}

// PUT /api/auth/profile
// Updates editable profile fields. Email + role are intentionally NOT editable here.
func UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	// Address may legitimately be empty (clearing it), so always set it
	updates["address"] = input.Address
	// Avatar URL may also be cleared back to "" to fall back to the generated one
	updates["avatar_url"] = input.AvatarURL

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	var user models.User
	database.DB.First(&user, userID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated",
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"phone":      user.Phone,
			"address":    user.Address,
			"avatar_url": user.AvatarURL,
		},
	})
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// PUT /api/auth/change-password
// Requires the current password as proof-of-identity. No "forgot password" link.
func ChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !utils.CheckPasswordHash(input.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	hashed, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	database.DB.Model(&user).Update("password", hashed)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DELETE /api/auth/account
// Hard-deletes the user + cascades to cart/wishlist/orders via GORM hooks.
// Also requires password confirmation so someone with a stolen JWT can't nuke the account.
func DeleteAccount(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password is incorrect"})
		return
	}

	// cascade delete related rows
	database.DB.Exec("DELETE FROM addresses WHERE user_id = ?", userID)
	database.DB.Exec("DELETE FROM carts WHERE user_id = ?", userID)
	database.DB.Exec("DELETE FROM wishlist_items WHERE user_id = ?", userID)
	database.DB.Exec("DELETE FROM notifications WHERE user_id = ?", userID)
	database.DB.Exec("DELETE FROM care_reminders WHERE user_id = ?", userID)
	database.DB.Exec("DELETE FROM orders WHERE user_id = ?", userID)

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

// =====================================================================
// Address book CRUD
// =====================================================================

type AddressInput struct {
	Label      string `json:"label"`
	Recipient  string `json:"recipient" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" binding:"required"`
	Region     string `json:"region"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	IsDefault  bool   `json:"is_default"`
}

// GET /api/auth/addresses
func ListAddresses(c *gin.Context) {
	userID := c.GetUint("user_id")
	var addresses []models.Address
	if err := database.DB.Where("user_id = ?", userID).Order("is_default DESC, id ASC").Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": addresses})
}

// POST /api/auth/addresses
func CreateAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input AddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Country == "" {
		input.Country = "Bangladesh"
	}
	if input.Label == "" {
		input.Label = "Home"
	}

	// If this address is marked default, unset all other defaults first.
	if input.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	} else {
		// If the user has zero addresses, force this one to be default
		var count int64
		database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Count(&count)
		if count == 0 {
			input.IsDefault = true
		}
	}

	addr := models.Address{
		UserID:     userID,
		Label:      input.Label,
		Recipient:  input.Recipient,
		Phone:      input.Phone,
		Line1:      input.Line1,
		Line2:      input.Line2,
		City:       input.City,
		Region:     input.Region,
		PostalCode: input.PostalCode,
		Country:    input.Country,
		IsDefault:  input.IsDefault,
	}
	database.DB.Create(&addr)
	c.JSON(http.StatusCreated, gin.H{"message": "Address added", "address": addr})
}

// PUT /api/auth/addresses/:id
func UpdateAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var input AddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var addr models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&addr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	if input.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ? AND id <> ?", userID, addr.ID).Update("is_default", false)
	}

	addr.Label = input.Label
	addr.Recipient = input.Recipient
	addr.Phone = input.Phone
	addr.Line1 = input.Line1
	addr.Line2 = input.Line2
	addr.City = input.City
	addr.Region = input.Region
	addr.PostalCode = input.PostalCode
	addr.Country = input.Country
	if input.Country == "" {
		addr.Country = "Bangladesh"
	}
	if input.Label == "" {
		addr.Label = "Home"
	}
	addr.IsDefault = input.IsDefault

	database.DB.Save(&addr)
	c.JSON(http.StatusOK, gin.H{"message": "Address updated", "address": addr})
}

// DELETE /api/auth/addresses/:id
func DeleteAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var addr models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&addr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}
	wasDefault := addr.IsDefault
	database.DB.Delete(&addr)

	// If we deleted the default address, promote the most recent remaining one to default.
	if wasDefault {
		var next models.Address
		if err := database.DB.Where("user_id = ?", userID).Order("id DESC").First(&next).Error; err == nil {
			database.DB.Model(&next).Update("is_default", true)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Address deleted"})
}

// PUT /api/auth/addresses/:id/default — quick "make default" toggle
func SetDefaultAddress(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var addr models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&addr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}
	database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	database.DB.Model(&addr).Update("is_default", true)
	c.JSON(http.StatusOK, gin.H{"message": "Default address updated"})
}

// =====================================================================
// Email verification (Sprint F1 — Brevo mailer)
// =====================================================================

const verifyTTL = 15 * time.Minute

type VerifyEmailInput struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// issueVerificationCode writes a fresh row for the user and returns the
// 6-digit code as a string. Any previous unconsumed codes are left in the
// table — VerifyEmail picks the newest matching one.
func issueVerificationCode(userID uint) (string, error) {
	code, err := randomCode(6)
	if err != nil {
		return "", err
	}
	row := models.EmailVerification{
		UserID:    userID,
		Code:      code,
		ExpiresAt: time.Now().Add(verifyTTL),
	}
	if err := database.DB.Create(&row).Error; err != nil {
		return "", err
	}
	return code, nil
}

// randomCode returns an n-digit decimal string using crypto/rand.
func randomCode(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("randomCode: length must be positive")
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		// bias is negligible at n=6
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0') + byte(n.Int64())
	}
	return string(out), nil
}

// POST /api/auth/verify — body: { email, code }
// Public (no JWT) so freshly-registered users can verify before login.
func VerifyEmail(c *gin.Context) {
	var input VerifyEmailInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or code"})
		return
	}
	if user.EmailVerified {
		c.JSON(http.StatusOK, gin.H{"message": "Email already verified"})
		return
	}

	var row models.EmailVerification
	err := database.DB.
		Where("user_id = ? AND code = ? AND consumed_at IS NULL", user.ID, input.Code).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
		return
	}
	if time.Now().After(row.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code expired — request a new one"})
		return
	}

	now := time.Now()
	database.DB.Model(&row).Update("consumed_at", &now)
	database.DB.Model(&user).Update("email_verified", true)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Email verified",
		"email_verified": true,
	})
}

// POST /api/auth/resend-verification — body: { email }
// Public: lets the user re-trigger an email without being logged in.
// Rate-limiting is intentionally out of scope (handled in reverse-proxy in
// production); for dev, 1 row per call is fine.
func ResendVerification(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Don't leak whether the email exists
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a new code was sent"})
		return
	}
	if user.EmailVerified {
		c.JSON(http.StatusOK, gin.H{"message": "Email already verified"})
		return
	}

	code, err := issueVerificationCode(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue code"})
		return
	}
	_ = mailer.Send(mailer.VerifyEmail(user.Email, code))
	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a new code was sent"})
}
