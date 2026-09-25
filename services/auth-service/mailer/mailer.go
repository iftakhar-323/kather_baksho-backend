// Package mailer is a thin wrapper around the Brevo transactional-email API
// (https://developers.brevo.com/reference/sendtransacemail).
//
// Design choices:
//   - No third-party SDK: a 30-line net/http call is enough and keeps the
//     dependency footprint at zero.
//   - Graceful degradation: when BREVO_API_KEY is empty (local dev), Send()
//     logs to stdout instead of failing — so the rest of the app keeps
//     working without any mail credentials configured.
//   - Synchronous send: keeps error semantics obvious at the call site.
//     Wrap in a goroutine later if latency becomes an issue.
package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Email is the minimal payload we send to Brevo.
type Email struct {
	To      string // recipient address
	Subject string
	HTML    string
	Text    string // plain-text fallback (required by Brevo)
}

const brevoEndpoint = "https://api.brevo.com/v3/smtp/email"

// Send posts the email through Brevo. Returns nil when the key is missing
// (logged-only mode) or when Brevo accepts the message.
func Send(e Email) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	fromEmail := os.Getenv("MAIL_FROM_EMAIL")
	fromName := os.Getenv("MAIL_FROM_NAME")
	if fromEmail == "" {
		fromEmail = "no-reply@katherbox.local"
	}
	if fromName == "" {
		fromName = "KatherBox"
	}

	// Dev fallback: no key configured → log and move on.
	if apiKey == "" {
		log.Printf("[mailer] BREVO_API_KEY missing — would send: %q → %s\n",
			e.Subject, e.To)
		return nil
	}

	payload := map[string]interface{}{
		"sender":      map[string]string{"email": fromEmail, "name": fromName},
		"to":          []map[string]string{{"email": e.To}},
		"subject":     e.Subject,
		"htmlContent": e.HTML,
		"textContent": e.Text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailer: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", brevoEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailer: build request: %w", err)
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mailer: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		// Read up to 1KB of the body for the error message.
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("mailer: brevo status %d: %s", resp.StatusCode, string(buf))
	}
	return nil
}