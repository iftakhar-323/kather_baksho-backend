package utils

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test_service",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
	})

	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed, got %v", cb.State())
	}

	simulatedErr := errors.New("remote service failure")

	// 2 failures: still closed
	_ = cb.Execute(func() error { return simulatedErr })
	_ = cb.Execute(func() error { return simulatedErr })
	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed after 2 failures, got %v", cb.State())
	}

	// 3rd failure: trips to Open
	_ = cb.Execute(func() error { return simulatedErr })
	if cb.State() != StateOpen {
		t.Fatalf("expected state Open after 3 failures, got %v", cb.State())
	}

	// In Open state, requests fail fast with ErrCircuitBreakerOpen
	err := cb.Execute(func() error { return nil })
	if err != ErrCircuitBreakerOpen {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}

	// Wait for recovery timeout to expire
	time.Sleep(60 * time.Millisecond)

	// State should now become Half-Open
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state HalfOpen after recovery timeout, got %v", cb.State())
	}

	// First success in Half-Open
	err = cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Second success in Half-Open: should reset to Closed
	err = cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed after 2 successes, got %v", cb.State())
	}
}

func TestCircuitBreakerFallback(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test_fallback",
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
	})

	_ = cb.Execute(func() error { return errors.New("fatal") })

	fallbackInvoked := false
	err := cb.ExecuteWithFallback(
		func() error { return nil },
		func(err error) error {
			fallbackInvoked = true
			return nil
		},
	)

	if err != nil {
		t.Fatalf("expected nil error from fallback, got %v", err)
	}
	if !fallbackInvoked {
		t.Fatalf("expected fallback to be executed when circuit is open")
	}
}
