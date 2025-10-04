package limiter

import (
	"context"
	"testing"
	"time"
)

func TestRetryManager(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 2
	config.BaseDelay = 10 * time.Millisecond

	rm := NewRetryManager(config)

	// Test successful execution
	attempts := 0
	result, err := rm.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		attempts++
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryManagerWithRetries(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.BaseDelay = 10 * time.Millisecond

	rm := NewRetryManager(config)

	// Test retryable error
	attempts := 0
	result, err := rm.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		attempts++
		if attempts < 3 {
			return nil, NewHTTPError(429, "Rate limited", "")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryManagerMaxRetriesExceeded(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 2
	config.BaseDelay = 10 * time.Millisecond

	rm := NewRetryManager(config)

	// Test max retries exceeded
	attempts := 0
	result, err := rm.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		attempts++
		return nil, NewHTTPError(429, "Rate limited", "")
	})

	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if attempts != 3 { // MaxRetries + 1
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryManagerNonRetryableError(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.BaseDelay = 10 * time.Millisecond

	rm := NewRetryManager(config)

	// Test non-retryable error
	attempts := 0
	result, err := rm.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		attempts++
		return nil, NewHTTPError(400, "Bad request", "")
	})

	if err == nil {
		t.Error("Expected error for non-retryable error")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryManagerContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.BaseDelay = 100 * time.Millisecond

	rm := NewRetryManager(config)

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	result, err := rm.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		attempts++
		return nil, NewHTTPError(429, "Rate limited", "")
	})

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt before cancellation, got %d", attempts)
	}
}

func TestHTTPError(t *testing.T) {
	err := NewHTTPError(429, "Rate limited", "Too many requests")

	if err.StatusCode != 429 {
		t.Errorf("Expected status code 429, got %d", err.StatusCode)
	}
	if err.Message != "Rate limited" {
		t.Errorf("Expected message 'Rate limited', got %s", err.Message)
	}
	if err.Error() != "HTTP 429: Rate limited" {
		t.Errorf("Expected error string 'HTTP 429: Rate limited', got %s", err.Error())
	}
}

func TestIsRetryableHTTPError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, false},
		{400, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		result := IsRetryableHTTPError(tt.statusCode)
		if result != tt.expected {
			t.Errorf("IsRetryableHTTPError(%d) = %v, want %v", tt.statusCode, result, tt.expected)
		}
	}
}
