package providers

import (
	"context"

	"github.com/snow-ghost/agent/pkg/cost"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
	"github.com/snow-ghost/agent/pkg/tokens"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Chat performs chat completion
	Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error)

	// Embed generates embeddings for input texts
	Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error)

	// GetCostCalculator returns the cost calculator for this provider
	GetCostCalculator() *cost.Calculator
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

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	costCalculator *cost.Calculator
	tokenRegistry  *tokens.EncoderRegistry
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(registry *registry.Registry) *BaseProvider {
	return &BaseProvider{
		costCalculator: cost.NewCalculator(registry),
		tokenRegistry:  tokens.GetDefaultRegistry(),
	}
}

// GetCostCalculator returns the cost calculator
func (b *BaseProvider) GetCostCalculator() *cost.Calculator {
	return b.costCalculator
}

// EstimateUsage estimates token usage when not provided by the provider
func (b *BaseProvider) EstimateUsage(messages []string, responseText string) core.Usage {
	var totalInputTokens int
	for _, msg := range messages {
		if count, err := b.tokenRegistry.CountTokens("", msg); err == nil {
			totalInputTokens += count
		} else {
			// Fallback to simple estimation
			totalInputTokens += len(msg) / 4
		}
	}

	var completionTokens int
	if count, err := b.tokenRegistry.CountTokens("", responseText); err == nil {
		completionTokens = count
	} else {
		// Fallback to simple estimation
		completionTokens = len(responseText) / 4
	}

	if completionTokens < 1 {
		completionTokens = 1
	}

	return core.Usage{
		PromptTokens:     totalInputTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalInputTokens + completionTokens,
	}
}
