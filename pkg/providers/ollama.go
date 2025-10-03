package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// OllamaProvider implements the Provider interface for Ollama API
type OllamaProvider struct {
	*BaseProvider
	client  *http.Client
	baseURL string
}

// OllamaMessage represents a message in Ollama format
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaRequest represents the request format for Ollama API
type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse represents the response format from Ollama API
type OllamaResponse struct {
	Model     string        `json:"model"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
	CreatedAt string        `json:"created_at"`
}

// OllamaEmbedRequest represents the request format for Ollama embeddings
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse represents the response format from Ollama embeddings
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL string) *OllamaProvider {
	// Create a default registry for cost calculation
	registry := registry.GetDefaultRegistry()

	return &OllamaProvider{
		BaseProvider: NewBaseProvider(registry),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
	}
}

// Chat performs chat completion using Ollama API
func (p *OllamaProvider) Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error) {
	// Convert messages
	messages := make([]OllamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Build request
	ollamaReq := OllamaRequest{
		Model:    mc.ID,
		Messages: messages,
		Stream:   false, // We'll handle streaming separately
		Options: map[string]interface{}{
			"temperature": req.Temperature,
			"top_p":       req.TopP,
			"num_predict": req.MaxTokens,
		},
	}

	// Marshal request
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(reqBody))
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("ollama API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return core.ChatResponse{}, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	// Parse response
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	// Estimate usage (Ollama doesn't provide token counts)
	estimator := &MockUsageEstimator{}
	promptTokens := 0
	for _, msg := range req.Messages {
		tokens, _ := estimator.EstimateTokens(msg.Content)
		promptTokens += tokens
	}
	completionTokens := estimator.EstimateCompletionTokens(ollamaResp.Message.Content)

	// Convert response
	chatResp := core.ChatResponse{
		Text: ollamaResp.Message.Content,
		Usage: core.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Model:        mc.ID,
		Provider:     mc.Provider,
		FinishReason: "stop", // Ollama doesn't provide detailed finish reasons
	}

	return chatResp, nil
}

// Embed generates embeddings using Ollama API
func (p *OllamaProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error) {
	embeddings := make([][]float32, len(input))
	totalTokens := 0

	for i, text := range input {
		// Create request for this input
		ollamaReq := OllamaEmbedRequest{
			Model:  mc.ID,
			Prompt: text,
		}

		// Marshal request
		reqBody, err := json.Marshal(ollamaReq)
		if err != nil {
			return nil, core.Usage{}, fmt.Errorf("failed to marshal ollama embed request: %w", err)
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/embeddings", bytes.NewReader(reqBody))
		if err != nil {
			return nil, core.Usage{}, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")

		// Make request
		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, core.Usage{}, fmt.Errorf("ollama embed API request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, core.Usage{}, fmt.Errorf("ollama embed API returned status %d", resp.StatusCode)
		}

		// Parse response
		var ollamaResp OllamaEmbedResponse
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return nil, core.Usage{}, fmt.Errorf("failed to decode ollama embed response: %w", err)
		}

		embeddings[i] = ollamaResp.Embedding

		// Estimate tokens
		estimator := &MockUsageEstimator{}
		tokens, _ := estimator.EstimateTokens(text)
		totalTokens += tokens
	}

	usage := core.Usage{
		PromptTokens:     totalTokens,
		CompletionTokens: 0, // Embeddings don't have completion tokens
		TotalTokens:      totalTokens,
	}

	return embeddings, usage, nil
}

// CreateOllamaProviderFromConfig creates an Ollama provider from model config
func CreateOllamaProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) *OllamaProvider {
	return &OllamaProvider{
		BaseProvider: NewBaseProvider(registry),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: mc.BaseURL,
	}
}
