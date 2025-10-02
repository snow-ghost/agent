package embeddings

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI's API
type OpenAIEmbedder struct {
	client *openai.Client
	config *EmbeddingConfig
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(config *EmbeddingConfig) (*OpenAIEmbedder, error) {
	if config == nil {
		config = DefaultConfig()
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbedder{
		client: client,
		config: config,
	}, nil
}

// EmbedText converts text to a vector using OpenAI's embedding API
func (o *OpenAIEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Truncate text if it's too long
	if len(text) > o.config.MaxTokens {
		text = o.truncateText(text, o.config.MaxTokens)
	}

	// Create embedding request
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	}

	// Override model if specified in config
	if o.config.Model != "" {
		req.Model = openai.EmbeddingModel(o.config.Model)
	}

	// Make API call
	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert to float32 slice
	embedding := resp.Data[0].Embedding
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

// EmbedTexts embeds multiple texts in a single batch
func (o *OpenAIEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	// Truncate texts if needed
	truncatedTexts := make([]string, len(texts))
	for i, text := range texts {
		if len(text) > o.config.MaxTokens {
			truncatedTexts[i] = o.truncateText(text, o.config.MaxTokens)
		} else {
			truncatedTexts[i] = text
		}
	}

	// Create embedding request
	req := openai.EmbeddingRequest{
		Input: truncatedTexts,
		Model: openai.SmallEmbedding3,
	}

	// Override model if specified in config
	if o.config.Model != "" {
		req.Model = openai.EmbeddingModel(o.config.Model)
	}

	// Make API call
	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Convert to float32 slices
	result := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := data.Embedding
		result[i] = make([]float32, len(embedding))
		for j, v := range embedding {
			result[i][j] = float32(v)
		}
	}

	return result, nil
}

// truncateText truncates text to approximately maxTokens characters
func (o *OpenAIEmbedder) truncateText(text string, maxTokens int) string {
	// Rough approximation: 4 characters per token
	maxChars := maxTokens * 4

	if len(text) <= maxChars {
		return text
	}

	// Find the last complete word within the limit
	truncated := text[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")

	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// GetConfig returns the embedder configuration
func (o *OpenAIEmbedder) GetConfig() *EmbeddingConfig {
	return o.config
}

// SetModel updates the embedding model
func (o *OpenAIEmbedder) SetModel(model string) {
	o.config.Model = model
}

// ValidateAPIKey checks if the OpenAI API key is valid
func (o *OpenAIEmbedder) ValidateAPIKey(ctx context.Context) error {
	// Try to create a simple embedding to validate the API key
	_, err := o.EmbedText(ctx, "test")
	return err
}

// NewOpenAIEmbedderFromEnv creates an OpenAI embedder using environment variables
func NewOpenAIEmbedderFromEnv() (*OpenAIEmbedder, error) {
	config := &EmbeddingConfig{
		Model:     getEnv("EMBEDDINGS_MODEL", "text-embedding-3-small"),
		Dimension: getEnvInt("EMBEDDINGS_DIMENSION", 1536),
		MaxTokens: getEnvInt("EMBEDDINGS_MAX_TOKENS", 8192),
		BatchSize: getEnvInt("EMBEDDINGS_BATCH_SIZE", 100),
	}

	return NewOpenAIEmbedder(config)
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
