package client

import (
	"context"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Complete sends a simple text completion request
	Complete(ctx context.Context, prompt string, caller string) (string, error)

	// Embed generates embeddings for the given texts
	Embed(ctx context.Context, texts []string, caller string) ([][]float32, error)

	// GetModels retrieves available models
	GetModels(ctx context.Context) ([]core.Model, error)

	// Health checks if the service is healthy
	Health(ctx context.Context) error
}

// Ensure Client implements LLMClient interface
var _ LLMClient = (*Client)(nil)
