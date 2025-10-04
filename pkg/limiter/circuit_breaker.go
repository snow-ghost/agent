package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name        string                             `json:"name"`
	MaxRequests uint32                             `json:"max_requests"`
	Interval    time.Duration                      `json:"interval"`
	Timeout     time.Duration                      `json:"timeout"`
	ReadyToTrip func(counts gobreaker.Counts) bool `json:"-"`
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if failure rate is > 50% and we have at least 5 requests
			return counts.Requests >= 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	}
}

// CircuitBreakerManager manages circuit breakers for models
type CircuitBreakerManager struct {
	breakers map[string]*gobreaker.CircuitBreaker
	configs  map[string]*CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		configs:  make(map[string]*CircuitBreakerConfig),
	}
}

// GetBreaker returns or creates a circuit breaker for a model
func (cbm *CircuitBreakerManager) GetBreaker(modelID string, config registry.ModelConfig) *gobreaker.CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Check if breaker already exists
	if breaker, exists := cbm.breakers[modelID]; exists {
		return breaker
	}

	// Create circuit breaker configuration
	cbConfig := cbm.getConfigForModel(modelID, config)

	// Create circuit breaker
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        cbConfig.Name,
		MaxRequests: cbConfig.MaxRequests,
		Interval:    cbConfig.Interval,
		Timeout:     cbConfig.Timeout,
		ReadyToTrip: cbConfig.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state changes (in a real implementation, you'd use a logger)
			fmt.Printf("Circuit breaker %s changed from %s to %s\n", name, from, to)
		},
	})

	cbm.breakers[modelID] = breaker
	cbm.configs[modelID] = cbConfig

	return breaker
}

// getConfigForModel returns circuit breaker configuration for a model
func (cbm *CircuitBreakerManager) getConfigForModel(modelID string, config registry.ModelConfig) *CircuitBreakerConfig {
	// Use model-specific configuration if available
	if cbConfig, exists := cbm.configs[modelID]; exists {
		return cbConfig
	}

	// Create default configuration based on model characteristics
	cbConfig := DefaultCircuitBreakerConfig(fmt.Sprintf("model-%s", modelID))

	// Adjust configuration based on model reliability
	// Models with higher RPM/TPM might be more reliable
	if config.MaxRPM > 5000 || config.MaxTPM > 100000 {
		// More reliable models get more lenient settings
		cbConfig.MaxRequests = 5
		cbConfig.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6
		}
	} else {
		// Less reliable models get stricter settings
		cbConfig.MaxRequests = 2
		cbConfig.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.Requests >= 3 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.4
		}
	}

	return cbConfig
}

// Execute executes a function through the circuit breaker
func (cbm *CircuitBreakerManager) Execute(ctx context.Context, modelID string, config registry.ModelConfig, fn func() (interface{}, error)) (interface{}, error) {
	breaker := cbm.GetBreaker(modelID, config)

	// Execute through circuit breaker
	result, err := breaker.Execute(func() (interface{}, error) {
		return fn()
	})

	if err != nil {
		return nil, fmt.Errorf("circuit breaker execution failed: %w", err)
	}

	return result, nil
}

// GetState returns the current state of a circuit breaker
func (cbm *CircuitBreakerManager) GetState(modelID string, config registry.ModelConfig) gobreaker.State {
	breaker := cbm.GetBreaker(modelID, config)
	return breaker.State()
}

// GetStats returns circuit breaker statistics for a model
func (cbm *CircuitBreakerManager) GetStats(modelID string, config registry.ModelConfig) map[string]interface{} {
	breaker := cbm.GetBreaker(modelID, config)
	counts := breaker.Counts()

	return map[string]interface{}{
		"model_id":             modelID,
		"state":                breaker.State().String(),
		"requests":             counts.Requests,
		"total_success":        counts.TotalSuccesses,
		"total_failures":       counts.TotalFailures,
		"consecutive_success":  counts.ConsecutiveSuccesses,
		"consecutive_failures": counts.ConsecutiveFailures,
	}
}

// Reset resets the circuit breaker for a model
func (cbm *CircuitBreakerManager) Reset(modelID string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	delete(cbm.breakers, modelID)
	delete(cbm.configs, modelID)
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	cbm.breakers = make(map[string]*gobreaker.CircuitBreaker)
	cbm.configs = make(map[string]*CircuitBreakerConfig)
}

// IsOpen checks if the circuit breaker is open for a model
func (cbm *CircuitBreakerManager) IsOpen(modelID string, config registry.ModelConfig) bool {
	breaker := cbm.GetBreaker(modelID, config)
	return breaker.State() == gobreaker.StateOpen
}

// IsHalfOpen checks if the circuit breaker is half-open for a model
func (cbm *CircuitBreakerManager) IsHalfOpen(modelID string, config registry.ModelConfig) bool {
	breaker := cbm.GetBreaker(modelID, config)
	return breaker.State() == gobreaker.StateHalfOpen
}

// IsClosed checks if the circuit breaker is closed for a model
func (cbm *CircuitBreakerManager) IsClosed(modelID string, config registry.ModelConfig) bool {
	breaker := cbm.GetBreaker(modelID, config)
	return breaker.State() == gobreaker.StateClosed
}
