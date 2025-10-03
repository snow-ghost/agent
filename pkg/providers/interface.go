package providers

import (
	"context"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Chat performs chat completion
	Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error)

	// Embed generates embeddings for input texts
	Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error)
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	CreateProvider(providerType string) (Provider, error)
}

// UsageEstimator estimates token usage when providers don't return it
type UsageEstimator interface {
	EstimateTokens(text string) (promptTokens, completionTokens int)
}

// MockUsageEstimator provides a simple token estimation
type MockUsageEstimator struct{}

// EstimateTokens provides a simple character-based token estimation
func (m *MockUsageEstimator) EstimateTokens(text string) (promptTokens, completionTokens int) {
	// Simple estimation: ~4 characters per token
	estimatedTokens := len(text) / 4
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}
	return estimatedTokens, 0
}

// EstimateCompletionTokens estimates completion tokens
func (m *MockUsageEstimator) EstimateCompletionTokens(text string) int {
	estimatedTokens := len(text) / 4
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}
	return estimatedTokens
}
