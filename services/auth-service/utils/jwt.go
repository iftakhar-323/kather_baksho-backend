package utils

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecretOnce  sync.Once
	jwtSecretBytes []byte
)

// jwtSecret is resolved lazily (rather than at package-init time) so that a
// JWT_SECRET loaded from .env by godotenv.Load() in main() is picked up.
func jwtSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			if os.Getenv("GIN_MODE") == "release" {
				log.Fatal("JWT_SECRET env var must be set when GIN_MODE=release")
			}
			log.Println("WARNING: JWT_SECRET not set — using an insecure development default. Set JWT_SECRET before deploying.")
			secret = "kather_baksho_secret_key_change_this_later"
		}
		jwtSecretBytes = []byte(secret)
	})
	return jwtSecretBytes
}

func GenerateJWT(userID uint, email string, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// Generate2FAPendingToken creates a short-lived token (10 minutes) for 2FA verification step
func Generate2FAPendingToken(userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    "2fa_pending",
		"exp":     time.Now().Add(time.Minute * 10).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret(), nil
	})
}
