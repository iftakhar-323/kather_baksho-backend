package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Message struct {
	To       string `json:"to"`
	Content  string `json:"content"`
	Language string `json:"language"` // "bn" or "en"
}

type Provider interface {
	Name() string
	Send(msg Message) error
}

type Service struct {
	provider Provider
	client   *http.Client
}

var globalService *Service

func init() {
	globalService = NewService()
}

func NewService() *Service {
	client := &http.Client{Timeout: 10 * time.Second}
	providerName := strings.ToLower(os.Getenv("SMS_PROVIDER"))

	switch providerName {
	case "greenweb":
		return &Service{
			provider: &GreenwebProvider{
				token:  os.Getenv("GREENWEB_SMS_TOKEN"),
				client: client,
			},
			client: client,
		}
	case "twilio":
		return &Service{
			provider: &TwilioProvider{
				accountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
				authToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
				fromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
				client:     client,
			},
			client: client,
		}
	default:
		return &Service{
			provider: &LoggerProvider{},
			client:   client,
		}
	}
}

// Send dispatches the SMS synchronously.
func Send(msg Message) error {
	return globalService.Send(msg)
}

// SendAsync dispatches the SMS non-blockingly in a background goroutine.
func SendAsync(msg Message) {
	go func() {
		if err := Send(msg); err != nil {
			log.Printf("[SMS] Error sending to %s: %v", sanitizePhone(msg.To), err)
		}
	}()
}

func (s *Service) Send(msg Message) error {
	if strings.TrimSpace(msg.To) == "" || strings.TrimSpace(msg.Content) == "" {
		return fmt.Errorf("recipient phone number and message content cannot be empty")
	}
	return s.provider.Send(msg)
}

// LoggerProvider logs SMS messages to stdout (standard for dev / staging).
type LoggerProvider struct{}

func (l *LoggerProvider) Name() string { return "logger" }
func (l *LoggerProvider) Send(msg Message) error {
	log.Printf("[SMS:LOGGER] To: %s | Lang: %s | Message: %q", sanitizePhone(msg.To), msg.Language, msg.Content)
	return nil
}

// GreenwebProvider sends SMS via Greenweb Bangladesh SMS Gateway.
type GreenwebProvider struct {
	token  string
	client *http.Client
}

func (g *GreenwebProvider) Name() string { return "greenweb" }
func (g *GreenwebProvider) Send(msg Message) error {
	if g.token == "" {
		log.Printf("[SMS:Greenweb] No token configured; mock logged: To: %s | Message: %q", sanitizePhone(msg.To), msg.Content)
		return nil
	}

	form := url.Values{}
	form.Set("token", g.token)
	form.Set("to", formatBDNumber(msg.To))
	form.Set("message", msg.Content)

	resp, err := g.client.PostForm("http://api.greenweb.com.bd/api.php", form)
	if err != nil {
		return fmt.Errorf("greenweb request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	respStr := string(bodyBytes)
	if !strings.Contains(respStr, "Ok") && !strings.Contains(respStr, "OK") {
		return fmt.Errorf("greenweb gateway response error: %s", respStr)
	}

	log.Printf("[SMS:Greenweb] Successfully dispatched to %s", sanitizePhone(msg.To))
	return nil
}

// TwilioProvider sends SMS via Twilio REST API.
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *http.Client
}

func (t *TwilioProvider) Name() string { return "twilio" }
func (t *TwilioProvider) Send(msg Message) error {
	if t.accountSID == "" || t.authToken == "" {
		log.Printf("[SMS:Twilio] No credentials configured; mock logged: To: %s | Message: %q", sanitizePhone(msg.To), msg.Content)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	data := url.Values{}
	data.Set("To", msg.To)
	data.Set("From", t.fromNumber)
	data.Set("Body", msg.Content)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errRes map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errRes)
		return fmt.Errorf("twilio api error (status %d): %v", resp.StatusCode, errRes)
	}

	log.Printf("[SMS:Twilio] Successfully dispatched to %s", sanitizePhone(msg.To))
	return nil
}

func sanitizePhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-2:]
}

func formatBDNumber(phone string) string {
	p := strings.ReplaceAll(phone, " ", "")
	p = strings.ReplaceAll(p, "-", "")
	if strings.HasPrefix(p, "+88") {
		return p[3:]
	}
	if strings.HasPrefix(p, "88") {
		return p[2:]
	}
	return p
}
