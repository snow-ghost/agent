package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for models
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

// GetLimiter returns or creates a rate limiter for a model
func (rl *RateLimiter) GetLimiter(modelID string, config registry.ModelConfig) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if limiter already exists
	if limiter, exists := rl.limiters[modelID]; exists {
		return limiter
	}

	// Create new limiter based on model configuration
	// Use the more restrictive limit between RPM and TPM
	rpm := float64(config.MaxRPM)
	tpm := float64(config.MaxTPM)

	// Convert TPM to requests per minute (assuming average 100 tokens per request)
	avgTokensPerRequest := 100.0
	tpmAsRPM := tpm / avgTokensPerRequest

	// Use the more restrictive limit
	var limit float64
	if rpm > 0 && tpmAsRPM > 0 {
		if rpm < tpmAsRPM {
			limit = rpm
		} else {
			limit = tpmAsRPM
		}
	} else if rpm > 0 {
		limit = rpm
	} else if tpmAsRPM > 0 {
		limit = tpmAsRPM
	} else {
		// Default limit if not specified
		limit = 1000.0
	}

	// Create rate limiter (per second)
	limiter := rate.NewLimiter(rate.Limit(limit/60.0), int(limit/10.0)) // Burst = 1/10 of limit
	rl.limiters[modelID] = limiter

	return limiter
}

// Wait waits for the rate limiter to allow the request
func (rl *RateLimiter) Wait(ctx context.Context, modelID string, config registry.ModelConfig) error {
	limiter := rl.GetLimiter(modelID, config)

	// Wait for rate limiter
	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	return nil
}

// Allow checks if the request is allowed without waiting
func (rl *RateLimiter) Allow(modelID string, config registry.ModelConfig) bool {
	limiter := rl.GetLimiter(modelID, config)
	return limiter.Allow()
}

// WaitN waits for N tokens from the rate limiter
func (rl *RateLimiter) WaitN(ctx context.Context, modelID string, config registry.ModelConfig, n int) error {
	limiter := rl.GetLimiter(modelID, config)

	// Wait for rate limiter with N tokens
	if err := limiter.WaitN(ctx, n); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}

	return nil
}

// AllowN checks if N tokens are allowed without waiting
func (rl *RateLimiter) AllowN(modelID string, config registry.ModelConfig, n int) bool {
	limiter := rl.GetLimiter(modelID, config)
	return limiter.AllowN(time.Now(), n)
}

// GetStats returns rate limiter statistics for a model
func (rl *RateLimiter) GetStats(modelID string, config registry.ModelConfig) map[string]interface{} {
	limiter := rl.GetLimiter(modelID, config)

	return map[string]interface{}{
		"model_id": modelID,
		"limit":    limiter.Limit(),
		"burst":    limiter.Burst(),
		"tokens":   limiter.Tokens(),
		"max_rpm":  config.MaxRPM,
		"max_tpm":  config.MaxTPM,
	}
}

// Reset resets the rate limiter for a model
func (rl *RateLimiter) Reset(modelID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limiters, modelID)
}

// ResetAll resets all rate limiters
func (rl *RateLimiter) ResetAll() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limiters = make(map[string]*rate.Limiter)
}
