package services

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPGenerationAndVerification(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret failed: %v", err)
	}

	if len(secret) < 16 {
		t.Fatalf("Secret too short: %s", secret)
	}

	now := time.Now()
	code, err := GenerateTOTPCode(secret, now)
	if err != nil {
		t.Fatalf("GenerateTOTPCode failed: %v", err)
	}

	if len(code) != 6 {
		t.Fatalf("Expected 6-digit code, got %s", code)
	}

	// Verify valid code
	if !VerifyTOTPCode(secret, code) {
		t.Fatalf("VerifyTOTPCode rejected valid code %s", code)
	}

	// Verify invalid code
	if VerifyTOTPCode(secret, "000000") && code != "000000" {
		t.Fatalf("VerifyTOTPCode accepted false code")
	}

	// Verify recovery codes
	codes := GenerateRecoveryCodes(8)
	if len(codes) != 8 {
		t.Fatalf("Expected 8 recovery codes, got %d", len(codes))
	}
	for _, rc := range codes {
		if !strings.Contains(rc, "-") || len(rc) != 9 {
			t.Errorf("Invalid recovery code format: %s", rc)
		}
	}

	// Verify URI
	uri := GenerateTOTPURI(secret, "admin@katherbaksho.com")
	if !strings.HasPrefix(uri, "otpauth://totp/KatherBaksho:") {
		t.Errorf("Invalid URI format: %s", uri)
	}
}
