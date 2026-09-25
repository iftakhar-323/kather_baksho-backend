package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// GenerateTOTPSecret generates a random 160-bit base32 encoded secret key
func GenerateTOTPSecret() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}

// GenerateTOTPCode calculates the 6-digit TOTP code for a given timestamp
func GenerateTOTPCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		// Try with padding if unpadded decode fails
		key, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return "", fmt.Errorf("invalid base32 secret: %w", err)
		}
	}

	counter := uint64(t.Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	code := truncated % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// VerifyTOTPCode checks if the given code matches the secret with +-1 interval clock skew tolerance
func VerifyTOTPCode(secret, code string) bool {
	cleanCode := strings.TrimSpace(code)
	if len(cleanCode) != 6 {
		return false
	}

	now := time.Now()
	// Test current, -30s, and +30s time windows
	for _, offset := range []int{0, -30, 30} {
		testTime := now.Add(time.Duration(offset) * time.Second)
		expectedCode, err := GenerateTOTPCode(secret, testTime)
		if err == nil && expectedCode == cleanCode {
			return true
		}
	}
	return false
}

// GenerateRecoveryCodes produces 8 human-friendly one-time recovery codes
func GenerateRecoveryCodes(count int) []string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		part1 := make([]byte, 4)
		part2 := make([]byte, 4)
		for j := 0; j < 4; j++ {
			n1, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			n2, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			part1[j] = chars[n1.Int64()]
			part2[j] = chars[n2.Int64()]
		}
		codes[i] = fmt.Sprintf("%s-%s", string(part1), string(part2))
	}
	return codes
}

// GenerateTOTPURI formats the standard otpauth URL for authenticator apps
func GenerateTOTPURI(secret, accountEmail string) string {
	return fmt.Sprintf("otpauth://totp/KatherBaksho:%s?secret=%s&issuer=KatherBaksho&algorithm=SHA1&digits=6&period=30",
		accountEmail, secret)
}
