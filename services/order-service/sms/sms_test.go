package sms

import (
	"strings"
	"testing"
)

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+8801712345678", "+88****78"},
		{"01712345678", "017****78"},
		{"123", "****"},
		{"", "****"},
	}

	for _, tc := range tests {
		got := sanitizePhone(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizePhone(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestFormatBDNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+8801712345678", "01712345678"},
		{"8801712345678", "01712345678"},
		{"01712-345678", "01712345678"},
		{"01712 345 678", "01712345678"},
	}

	for _, tc := range tests {
		got := formatBDNumber(tc.input)
		if got != tc.expected {
			t.Errorf("formatBDNumber(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestLoggerProvider(t *testing.T) {
	p := &LoggerProvider{}
	if p.Name() != "logger" {
		t.Errorf("expected provider name 'logger', got '%s'", p.Name())
	}

	err := p.Send(Message{
		To:      "+8801711223344",
		Content: "Test message",
	})
	if err != nil {
		t.Fatalf("unexpected error from LoggerProvider: %v", err)
	}
}

func TestValidation(t *testing.T) {
	s := NewService()

	err := s.Send(Message{To: "", Content: "Hello"})
	if err == nil {
		t.Error("expected error for empty recipient")
	}

	err = s.Send(Message{To: "01711223344", Content: ""})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestBilingualTemplates(t *testing.T) {
	// Bengali test
	msgBn := BuildOrderMessage("+8801711223344", "ORD-1234", "1500.00", EventOrderPlaced, "bn")
	if !strings.Contains(msgBn.Content, "অর্ডার #ORD-1234 নিশ্চিত হয়েছে") {
		t.Errorf("expected Bengali confirmation text, got: %s", msgBn.Content)
	}
	if !strings.Contains(msgBn.Content, "৳1500.00") {
		t.Errorf("expected Bengali amount, got: %s", msgBn.Content)
	}

	// English test
	msgEn := BuildOrderMessage("+8801711223344", "ORD-1234", "1500.00", EventOrderPlaced, "en")
	if !strings.Contains(msgEn.Content, "order #ORD-1234 is confirmed") {
		t.Errorf("expected English confirmation text, got: %s", msgEn.Content)
	}
	if !strings.Contains(msgEn.Content, "BDT 1500.00") {
		t.Errorf("expected English amount, got: %s", msgEn.Content)
	}

	// Delivery event
	msgDeliv := BuildOrderMessage("+8801711223344", "ORD-1234", "", EventOutForDelivery, "bn")
	if !strings.Contains(msgDeliv.Content, "ডেলিভারির উদ্দেশ্যে পাঠানো হয়েছে") {
		t.Errorf("expected out for delivery Bengali text, got: %s", msgDeliv.Content)
	}
}
