package utils

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateHalfOpen
	StateOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF-OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open: request blocked")
)

// CircuitBreaker manages fault tolerance by preventing repeated execution of failing dependencies.
type CircuitBreaker struct {
	name             string
	failureThreshold int
	successThreshold int
	recoveryTimeout  time.Duration

	mu                   sync.RWMutex
	state                CircuitState
	consecutiveFailures  int
	consecutiveSuccesses int
	lastStateChange      time.Time
}

// CircuitBreakerConfig holds settings for initializing a breaker.
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	RecoveryTimeout  time.Duration
}

// Global registry of circuit breakers for monitoring.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]*CircuitBreaker)
)

// NewCircuitBreaker creates and registers a named circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = 10 * time.Second
	}

	cb := &CircuitBreaker{
		name:             cfg.Name,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		recoveryTimeout:  cfg.RecoveryTimeout,
		state:            StateClosed,
		lastStateChange:  time.Now(),
	}

	registryMu.Lock()
	registry[cfg.Name] = cb
	registryMu.Unlock()

	return cb
}

// GetCircuitBreaker returns a registered breaker by name.
func GetCircuitBreaker(name string) *CircuitBreaker {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// GetAllCircuitBreakers returns a snapshot of all registered breakers and their states.
func GetAllCircuitBreakers() map[string]string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	res := make(map[string]string)
	for name, cb := range registry {
		res[name] = cb.State().String()
	}
	return res
}

// ResetAllCircuitBreakers resets all registered circuit breakers back to Closed state
func ResetAllCircuitBreakers() {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, cb := range registry {
		cb.Reset()
	}
}

// Reset clears failure counts and transitions back to StateClosed
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.lastStateChange = time.Now()
}

// State returns the active state, transitioning from Open to Half-Open if timeout has expired.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.lastStateChange) >= cb.recoveryTimeout {
		cb.state = StateHalfOpen
		cb.consecutiveSuccesses = 0
		cb.consecutiveFailures = 0
		cb.lastStateChange = time.Now()
	}
	return cb.state
}

// Execute wraps a protected function call with circuit breaker logic.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	state := cb.State()
	if state == StateOpen {
		return ErrCircuitBreakerOpen
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.consecutiveFailures++
		cb.consecutiveSuccesses = 0
		if cb.consecutiveFailures >= cb.failureThreshold || cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
		return err
	}

	// Success
	cb.consecutiveFailures = 0
	if cb.state == StateHalfOpen {
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.successThreshold {
			cb.state = StateClosed
			cb.consecutiveFailures = 0
			cb.lastStateChange = time.Now()
		}
	}
	return nil
}

// ExecuteWithFallback executes fn, and executes fallback if the breaker is open or fn errors.
func (cb *CircuitBreaker) ExecuteWithFallback(fn func() error, fallback func(err error) error) error {
	err := cb.Execute(fn)
	if err != nil {
		if fallback != nil {
			return fallback(err)
		}
		return err
	}
	return nil
}
