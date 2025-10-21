package embeddings

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/snow-ghost/agent/pkg/providers"
)

// LMStudioEmbedder implements the Embedder interface using LMStudio
type LMStudioEmbedder struct {
	client *openai.Client
	model  string
}

// NewLMStudioEmbedder creates a new LMStudio embedder
func NewLMStudioEmbedder(baseURL, model string) (*LMStudioEmbedder, error) {
	// Create OpenAI client configured for LMStudio
	config := openai.DefaultConfig("dummy-key") // LMStudio often doesn't require auth
	config.BaseURL = baseURL
	if !strings.HasSuffix(config.BaseURL, "/v1") {
		config.BaseURL = config.BaseURL + "/v1"
	}

	client := openai.NewClientWithConfig(config)

	return &LMStudioEmbedder{
		client: client,
		model:  model,
	}, nil
}

// NewLMStudioEmbedderFromEnv creates an LMStudio embedder from environment variables
func NewLMStudioEmbedderFromEnv() (*LMStudioEmbedder, error) {
	baseURL := os.Getenv("LMSTUDIO_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	model := os.Getenv("EMBEDDINGS_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	return NewLMStudioEmbedder(baseURL, model)
}

// EmbedText converts text to a vector representation using LMStudio
func (e *LMStudioEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Use the OpenAI embeddings API through LMStudio
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(e.model),
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LMStudio embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert to float32
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// EmbedTexts converts multiple texts to vector representations
func (e *LMStudioEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Use the OpenAI embeddings API through LMStudio
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(e.model),
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LMStudio batch embedding failed: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	// Convert to float32
	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float32(v)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (e *LMStudioEmbedder) GetDimension() int {
	// Default to 1536 for text-embedding-3-small
	// This could be made configurable based on the model
	return 1536
}

// Health checks if the LMStudio service is available
func (e *LMStudioEmbedder) Health(ctx context.Context) error {
	// Try to create a simple embedding to test connectivity
	_, err := e.EmbedText(ctx, "test")
	if err != nil {
		return fmt.Errorf("LMStudio health check failed: %w", err)
	}
	return nil
}

// CreateLMStudioEmbedderFromProvider creates an embedder using the existing LMStudio provider
func CreateLMStudioEmbedderFromProvider(provider *providers.LMStudioProvider, model string) *LMStudioEmbedder {
	// Extract the client from the provider
	// This is a bit of a hack since the provider doesn't expose the client
	// In a real implementation, we might want to refactor this

	// For now, create a new client with the same configuration
	baseURL := os.Getenv("LMSTUDIO_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	config := openai.DefaultConfig("dummy-key")
	config.BaseURL = baseURL
	if !strings.HasSuffix(config.BaseURL, "/v1") {
		config.BaseURL = config.BaseURL + "/v1"
	}

	client := openai.NewClientWithConfig(config)

	return &LMStudioEmbedder{
		client: client,
		model:  model,
	}
}

// Ensure LMStudioEmbedder implements Embedder interface
var _ Embedder = (*LMStudioEmbedder)(nil)
