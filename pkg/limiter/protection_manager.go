package limiter

import (
	"context"
	"fmt"

	"github.com/snow-ghost/agent/pkg/registry"
)

// ProtectionManager integrates rate limiting, retries, and circuit breaker
type ProtectionManager struct {
	rateLimiter    *RateLimiter
	retryManager   *RetryManager
	circuitBreaker *CircuitBreakerManager
	registry       *registry.Registry
}

// NewProtectionManager creates a new protection manager
func NewProtectionManager(registry *registry.Registry) *ProtectionManager {
	return &ProtectionManager{
		rateLimiter:    NewRateLimiter(),
		retryManager:   NewRetryManager(DefaultRetryConfig()),
		circuitBreaker: NewCircuitBreakerManager(),
		registry:       registry,
	}
}

// ExecuteWithProtection executes a function with all protection mechanisms
func (pm *ProtectionManager) ExecuteWithProtection(
	ctx context.Context,
	modelID string,
	fn func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	// Get model configuration
	modelConfig := pm.registry.FindModel(modelID)
	if modelConfig == nil {
		return nil, fmt.Errorf("model %s not found in registry", modelID)
	}

	// Check if circuit breaker is open
	if pm.circuitBreaker.IsOpen(modelID, *modelConfig) {
		return nil, fmt.Errorf("circuit breaker is open for model %s", modelID)
	}

	// Apply rate limiting
	if err := pm.rateLimiter.Wait(ctx, modelID, *modelConfig); err != nil {
		return nil, fmt.Errorf("rate limiting failed: %w", err)
	}

	// Execute with retry and circuit breaker protection
	result, err := pm.circuitBreaker.Execute(ctx, modelID, *modelConfig, func() (interface{}, error) {
		return pm.retryManager.Execute(ctx, fn)
	})

	if err != nil {
		return nil, fmt.Errorf("protected execution failed: %w", err)
	}

	return result, nil
}

// ExecuteWithCustomRetry executes with custom retry configuration
func (pm *ProtectionManager) ExecuteWithCustomRetry(
	ctx context.Context,
	modelID string,
	retryConfig *RetryConfig,
	fn func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	// Get model configuration
	modelConfig := pm.registry.FindModel(modelID)
	if modelConfig == nil {
		return nil, fmt.Errorf("model %s not found in registry", modelID)
	}

	// Check if circuit breaker is open
	if pm.circuitBreaker.IsOpen(modelID, *modelConfig) {
		return nil, fmt.Errorf("circuit breaker is open for model %s", modelID)
	}

	// Apply rate limiting
	if err := pm.rateLimiter.Wait(ctx, modelID, *modelConfig); err != nil {
		return nil, fmt.Errorf("rate limiting failed: %w", err)
	}

	// Create custom retry manager
	customRetryManager := NewRetryManager(retryConfig)

	// Execute with retry and circuit breaker protection
	result, err := pm.circuitBreaker.Execute(ctx, modelID, *modelConfig, func() (interface{}, error) {
		return customRetryManager.Execute(ctx, fn)
	})

	if err != nil {
		return nil, fmt.Errorf("protected execution failed: %w", err)
	}

	return result, nil
}

// GetStats returns comprehensive statistics for all protection mechanisms
func (pm *ProtectionManager) GetStats(modelID string) map[string]interface{} {
	modelConfig := pm.registry.FindModel(modelID)
	if modelConfig == nil {
		return map[string]interface{}{
			"error": "model not found",
		}
	}

	rateLimiterStats := pm.rateLimiter.GetStats(modelID, *modelConfig)
	circuitBreakerStats := pm.circuitBreaker.GetStats(modelID, *modelConfig)

	return map[string]interface{}{
		"model_id":        modelID,
		"rate_limiter":    rateLimiterStats,
		"circuit_breaker": circuitBreakerStats,
		"retry_config": map[string]interface{}{
			"max_retries":      pm.retryManager.config.MaxRetries,
			"base_delay":       pm.retryManager.config.BaseDelay.String(),
			"max_delay":        pm.retryManager.config.MaxDelay.String(),
			"backoff_factor":   pm.retryManager.config.BackoffFactor,
			"jitter":           pm.retryManager.config.Jitter,
			"retryable_errors": pm.retryManager.config.RetryableErrors,
		},
	}
}

// GetAllStats returns statistics for all models
func (pm *ProtectionManager) GetAllStats() map[string]interface{} {
	allStats := make(map[string]interface{})

	for _, model := range pm.registry.Models {
		allStats[model.ID] = pm.GetStats(model.ID)
	}

	return allStats
}

// ResetModel resets all protection mechanisms for a specific model
func (pm *ProtectionManager) ResetModel(modelID string) {
	pm.rateLimiter.Reset(modelID)
	pm.circuitBreaker.Reset(modelID)
}

// ResetAll resets all protection mechanisms
func (pm *ProtectionManager) ResetAll() {
	pm.rateLimiter.ResetAll()
	pm.circuitBreaker.ResetAll()
}

// IsModelAvailable checks if a model is available (not rate limited or circuit broken)
func (pm *ProtectionManager) IsModelAvailable(modelID string) bool {
	modelConfig := pm.registry.FindModel(modelID)
	if modelConfig == nil {
		return false
	}

	// Check if circuit breaker is open
	if pm.circuitBreaker.IsOpen(modelID, *modelConfig) {
		return false
	}

	// Check if rate limiter allows the request
	return pm.rateLimiter.Allow(modelID, *modelConfig)
}

// GetAvailableModels returns a list of available models
func (pm *ProtectionManager) GetAvailableModels() []string {
	var available []string

	for _, model := range pm.registry.Models {
		if pm.IsModelAvailable(model.ID) {
			available = append(available, model.ID)
		}
	}

	return available
}

// SimulateError simulates an error for testing purposes
func (pm *ProtectionManager) SimulateError(modelID string, errorType string) error {
	modelConfig := pm.registry.FindModel(modelID)
	if modelConfig == nil {
		return fmt.Errorf("model %s not found", modelID)
	}

	switch errorType {
	case "429":
		return NewHTTPError(429, "Rate limit exceeded", "Too many requests")
	case "500":
		return NewHTTPError(500, "Internal server error", "Server error")
	case "503":
		return NewHTTPError(503, "Service unavailable", "Service temporarily unavailable")
	default:
		return fmt.Errorf("simulated error: %s", errorType)
	}
}
