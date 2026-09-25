package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"kather_baksho/models"
)

type MockHMACProvider struct {
	secret string
}

func NewMockHMACProvider() *MockHMACProvider {
	sec := os.Getenv("PAYMENT_SECRET")
	if sec == "" {
		sec = "katherbox_default_secure_payment_hmac_secret_2026"
	}
	return &MockHMACProvider{secret: sec}
}

func (m *MockHMACProvider) Name() string {
	return "mock"
}

func (m *MockHMACProvider) computeHMAC(data string) string {
	h := hmac.New(sha256.New, []byte(m.secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (m *MockHMACProvider) Initiate(order *models.Order, returnURL string) (*PaymentSession, error) {
	sessionID := fmt.Sprintf("PAY-SES-%d-%d", order.ID, time.Now().UnixNano())
	payload := fmt.Sprintf("%d|%.2f|%s", order.ID, order.TotalPrice, sessionID)
	signature := m.computeHMAC(payload)

	return &PaymentSession{
		SessionID:     sessionID,
		OrderID:       order.ID,
		Amount:        order.TotalPrice,
		Currency:      "BDT",
		PaymentMethod: "mock",
		GatewayURL:    fmt.Sprintf("/api/payments/simulate-gateway?session_id=%s", sessionID),
		Signature:     signature,
	}, nil
}

func (m *MockHMACProvider) Verify(payload map[string]interface{}) (*PaymentResult, error) {
	orderIDVal, ok := payload["order_id"]
	if !ok {
		return nil, fmt.Errorf("missing order_id in mock verification")
	}
	sessionID, _ := payload["session_id"].(string)
	sig, _ := payload["signature"].(string)
	status, _ := payload["status"].(string)
	amountVal, _ := payload["amount"].(float64)

	var orderID uint
	switch v := orderIDVal.(type) {
	case float64:
		orderID = uint(v)
	case int:
		orderID = uint(v)
	case uint:
		orderID = v
	}

	dataPayload := fmt.Sprintf("%d|%.2f|%s", orderID, amountVal, sessionID)
	expectedSig := m.computeHMAC(dataPayload)

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid payment signature")
	}

	return &PaymentResult{
		OrderID:       orderID,
		SessionID:     sessionID,
		TransactionID: fmt.Sprintf("TXN-%s", sessionID),
		Amount:        amountVal,
		Status:        status,
		PaymentMethod: "mock",
	}, nil
}

func (m *MockHMACProvider) Query(paymentID string) (*PaymentResult, error) {
	return &PaymentResult{
		SessionID:     paymentID,
		TransactionID: fmt.Sprintf("TXN-%s", paymentID),
		Status:        "SUCCESS",
		PaymentMethod: "mock",
	}, nil
}

func (m *MockHMACProvider) Refund(paymentID string, amount float64, reason string) error {
	return nil
}
