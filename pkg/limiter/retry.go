package limiter

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []int         `json:"retryable_errors"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		Jitter:          true,
		RetryableErrors: []int{429, 500, 502, 503, 504},
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func(ctx context.Context) (interface{}, error)

// RetryManager manages retry logic
type RetryManager struct {
	config *RetryConfig
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *RetryConfig) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryManager{config: config}
}

// Execute executes a function with retry logic
func (rm *RetryManager) Execute(ctx context.Context, fn RetryableFunc) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= rm.config.MaxRetries; attempt++ {
		// Execute the function
		result, err := fn(ctx)

		// If no error, return success
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if this is the last attempt
		if attempt == rm.config.MaxRetries {
			break
		}

		// Check if error is retryable
		if !rm.isRetryableError(err) {
			return nil, err
		}

		// Calculate delay
		delay := rm.calculateDelay(attempt)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError checks if an error is retryable
func (rm *RetryManager) isRetryableError(err error) bool {
	// Check if it's an HTTP error
	if httpErr, ok := err.(*HTTPError); ok {
		for _, retryableCode := range rm.config.RetryableErrors {
			if httpErr.StatusCode == retryableCode {
				return true
			}
		}
	}

	// Check for context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Default to non-retryable for unknown errors
	return false
}

// calculateDelay calculates the delay for the given attempt
func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * (backoffFactor ^ attempt)
	delay := float64(rm.config.BaseDelay) * math.Pow(rm.config.BackoffFactor, float64(attempt))

	// Cap at max delay
	if delay > float64(rm.config.MaxDelay) {
		delay = float64(rm.config.MaxDelay)
	}

	// Add jitter if enabled
	if rm.config.Jitter {
		// Add ±25% jitter
		jitter := rand.Float64()*0.5 - 0.25 // -0.25 to +0.25
		delay = delay * (1 + jitter)
	}

	return time.Duration(delay)
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
	Body       string
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message, body string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}

// IsRetryableHTTPError checks if an HTTP status code is retryable
func IsRetryableHTTPError(statusCode int) bool {
	retryableCodes := []int{429, 500, 502, 503, 504}
	for _, code := range retryableCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}
