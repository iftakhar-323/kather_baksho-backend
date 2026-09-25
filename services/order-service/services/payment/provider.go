package payment

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"kather_baksho/models"
)

// PaymentSession represents an initialized payment session across any provider.
type PaymentSession struct {
	SessionID     string  `json:"session_id"`
	OrderID       uint    `json:"order_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"payment_method"`
	GatewayURL    string  `json:"gateway_url"`
	Signature     string  `json:"signature,omitempty"`
	ProviderData  any     `json:"provider_data,omitempty"`
}

// PaymentResult represents the verified outcome of a payment attempt.
type PaymentResult struct {
	OrderID         uint    `json:"order_id"`
	SessionID       string  `json:"session_id"`
	TransactionID   string  `json:"transaction_id"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"` // SUCCESS, FAILED, CANCELLED
	PaymentMethod   string  `json:"payment_method"`
	GatewayResponse string  `json:"gateway_response,omitempty"`
}

// PaymentProvider defines the common interface for payment gateways.
type PaymentProvider interface {
	Name() string
	Initiate(order *models.Order, returnURL string) (*PaymentSession, error)
	Verify(payload map[string]interface{}) (*PaymentResult, error)
	Query(paymentID string) (*PaymentResult, error)
	Refund(paymentID string, amount float64, reason string) error
}

// GetProvider returns the appropriate payment provider by name.
func GetProvider(method string) (PaymentProvider, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	switch method {
	case "bkash":
		return NewBKashProvider(), nil
	case "sslcommerz", "nagad", "card":
		return NewSSLCommerzProvider(), nil
	case "mock", "hmac", "":
		return NewMockHMACProvider(), nil
	default:
		// Fallback to mock HMAC if unknown or dev mode
		if os.Getenv("PAYMENT_ENV") != "production" {
			return NewMockHMACProvider(), nil
		}
		return nil, fmt.Errorf("unsupported payment method: %s", method)
	}
}

var ErrPaymentFailed = errors.New("payment failed")
