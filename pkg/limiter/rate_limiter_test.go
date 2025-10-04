package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()

	config := registry.ModelConfig{
		ID:     "test-model",
		MaxRPM: 100,
		MaxTPM: 10000,
	}

	// Test basic functionality
	limiter := rl.GetLimiter("test-model", config)
	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}

	// Test Allow
	if !rl.Allow("test-model", config) {
		t.Error("Expected first request to be allowed")
	}

	// Test Wait
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx, "test-model", config)
	if err != nil {
		t.Errorf("Expected wait to succeed, got error: %v", err)
	}

	// Test stats
	stats := rl.GetStats("test-model", config)
	if stats["model_id"] != "test-model" {
		t.Errorf("Expected model_id to be test-model, got %v", stats["model_id"])
	}
}

func TestRateLimiterWithHighLoad(t *testing.T) {
	rl := NewRateLimiter()

	config := registry.ModelConfig{
		ID:     "high-load-model",
		MaxRPM: 10, // Very low rate limit
		MaxTPM: 1000,
	}

	// Test that we can make some requests but not too many
	allowedCount := 0
	for i := 0; i < 20; i++ {
		if rl.Allow("high-load-model", config) {
			allowedCount++
		}
		time.Sleep(10 * time.Millisecond) // Small delay
	}

	// Should allow some requests but not all
	if allowedCount == 0 {
		t.Error("Expected at least some requests to be allowed")
	}
	if allowedCount >= 20 {
		t.Error("Expected rate limiting to prevent all requests")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter()

	config := registry.ModelConfig{
		ID:     "reset-model",
		MaxRPM: 100,
		MaxTPM: 10000,
	}

	// Create limiter
	rl.GetLimiter("reset-model", config)

	// Reset
	rl.Reset("reset-model")

	// Should create new limiter
	limiter := rl.GetLimiter("reset-model", config)
	if limiter == nil {
		t.Fatal("Expected new limiter to be created after reset")
	}
}
