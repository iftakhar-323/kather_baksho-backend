package payment

import (
	"testing"

	"kather_baksho/models"
)

func TestMockHMACProvider(t *testing.T) {
	prov := NewMockHMACProvider()
	order := &models.Order{
		TotalPrice: 1500.0,
	}
	order.ID = 42

	sess, err := prov.Initiate(order, "/api/payments/callback")
	if err != nil {
		t.Fatalf("Initiate failed: %v", err)
	}
	if sess.Signature == "" {
		t.Errorf("Expected signature, got empty")
	}

	payload := map[string]interface{}{
		"order_id":   order.ID,
		"session_id": sess.SessionID,
		"signature":  sess.Signature,
		"amount":     1500.0,
		"status":     "SUCCESS",
	}

	res, err := prov.Verify(payload)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", res.Status)
	}
}

func TestBKashProviderSimulation(t *testing.T) {
	prov := NewBKashProvider()
	order := &models.Order{
		TotalPrice: 2200.0,
	}
	order.ID = 101

	sess, err := prov.Initiate(order, "/api/payments/callback")
	if err != nil {
		t.Fatalf("Initiate failed: %v", err)
	}
	if sess.PaymentMethod != "bkash" {
		t.Errorf("Expected bkash, got %s", sess.PaymentMethod)
	}

	res, err := prov.Verify(map[string]interface{}{
		"paymentID": sess.SessionID,
		"status":    "SUCCESS",
	})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", res.Status)
	}
}

func TestSSLCommerzProviderSimulation(t *testing.T) {
	prov := NewSSLCommerzProvider()
	order := &models.Order{
		TotalPrice: 3400.0,
	}
	order.ID = 202

	sess, err := prov.Initiate(order, "/api/payments/callback")
	if err != nil {
		t.Fatalf("Initiate failed: %v", err)
	}
	if sess.PaymentMethod != "sslcommerz" {
		t.Errorf("Expected sslcommerz, got %s", sess.PaymentMethod)
	}

	res, err := prov.Verify(map[string]interface{}{
		"tran_id": sess.SessionID,
		"status":  "SUCCESS",
	})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", res.Status)
	}
}

func TestGetProviderFactory(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"bkash", "bkash"},
		{"sslcommerz", "sslcommerz"},
		{"card", "sslcommerz"},
		{"mock", "mock"},
		{"", "mock"},
	}

	for _, tc := range cases {
		p, err := GetProvider(tc.input)
		if err != nil {
			t.Errorf("Unexpected error for %s: %v", tc.input, err)
		}
		if p.Name() != tc.expected {
			t.Errorf("For input %s expected provider %s, got %s", tc.input, tc.expected, p.Name())
		}
	}
}
