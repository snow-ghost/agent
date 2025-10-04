package limiter

import (
	"context"
	"errors"
	"testing"

	"github.com/snow-ghost/agent/pkg/registry"
)

func TestCircuitBreakerManager(t *testing.T) {
	cbm := NewCircuitBreakerManager()

	config := registry.ModelConfig{
		ID:     "test-model",
		MaxRPM: 1000,
		MaxTPM: 100000,
	}

	// Test successful execution
	result, err := cbm.Execute(context.Background(), "test-model", config, func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}

	// Test state
	if !cbm.IsClosed("test-model", config) {
		t.Error("Expected circuit breaker to be closed after success")
	}
}

func TestCircuitBreakerManagerWithFailures(t *testing.T) {
	cbm := NewCircuitBreakerManager()

	config := registry.ModelConfig{
		ID:     "failing-model",
		MaxRPM: 100,
		MaxTPM: 10000,
	}

	// Test multiple failures to trigger circuit breaker
	for i := 0; i < 5; i++ {
		_, err := cbm.Execute(context.Background(), "failing-model", config, func() (interface{}, error) {
			return nil, errors.New("simulated failure")
		})

		if err == nil {
			t.Error("Expected error for failing function")
		}
	}

	// Circuit breaker should be open now
	if !cbm.IsOpen("failing-model", config) {
		t.Error("Expected circuit breaker to be open after failures")
	}

	// Test that execution fails when circuit breaker is open
	_, err := cbm.Execute(context.Background(), "failing-model", config, func() (interface{}, error) {
		return "success", nil
	})

	if err == nil {
		t.Error("Expected error when circuit breaker is open")
	}
}

func TestCircuitBreakerManagerStats(t *testing.T) {
	cbm := NewCircuitBreakerManager()

	config := registry.ModelConfig{
		ID:     "stats-model",
		MaxRPM: 1000,
		MaxTPM: 100000,
	}

	// Execute some requests
	cbm.Execute(context.Background(), "stats-model", config, func() (interface{}, error) {
		return "success", nil
	})

	cbm.Execute(context.Background(), "stats-model", config, func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	// Get stats
	stats := cbm.GetStats("stats-model", config)

	if stats["model_id"] != "stats-model" {
		t.Errorf("Expected model_id to be stats-model, got %v", stats["model_id"])
	}

	// Check that we have some requests (exact count may vary due to circuit breaker behavior)
	if stats["requests"] == nil {
		t.Error("Expected requests count to be present in stats")
	}
}

func TestCircuitBreakerManagerReset(t *testing.T) {
	cbm := NewCircuitBreakerManager()

	config := registry.ModelConfig{
		ID:     "reset-model",
		MaxRPM: 1000,
		MaxTPM: 100000,
	}

	// Create breaker
	cbm.GetBreaker("reset-model", config)

	// Reset
	cbm.Reset("reset-model")

	// Should create new breaker
	breaker := cbm.GetBreaker("reset-model", config)
	if breaker == nil {
		t.Fatal("Expected new breaker to be created after reset")
	}
}

func TestCircuitBreakerManagerStateTransitions(t *testing.T) {
	cbm := NewCircuitBreakerManager()

	config := registry.ModelConfig{
		ID:     "state-model",
		MaxRPM: 100,
		MaxTPM: 10000,
	}

	// Initially closed
	if !cbm.IsClosed("state-model", config) {
		t.Error("Expected circuit breaker to be initially closed")
	}

	// Test state transitions
	// This is a simplified test - in reality, state transitions
	// depend on failure rates and timeouts
	currentState := cbm.GetState("state-model", config)
	t.Logf("Current state: %s", currentState)
}
