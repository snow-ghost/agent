package embeddings

import (
	"context"
)

// Embedder defines the interface for text embedding generation
type Embedder interface {
	// EmbedText converts text to a vector representation
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// EmbeddingConfig holds configuration for embedders
type EmbeddingConfig struct {
	Model     string `json:"model"`
	Dimension int    `json:"dimension"`
	MaxTokens int    `json:"max_tokens"`
	BatchSize int    `json:"batch_size"`
}

// DefaultConfig returns default embedding configuration
func DefaultConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Model:     "text-embedding-3-small",
		Dimension: 1536,
		MaxTokens: 8192,
		BatchSize: 100,
	}
}
