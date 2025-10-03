package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude API
type AnthropicProvider struct {
	*BaseProvider
	client  *http.Client
	baseURL string
	apiKey  string
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents the request format for Anthropic API
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature float32            `json:"temperature,omitempty"`
	TopP        float32            `json:"top_p,omitempty"`
}

// AnthropicResponse represents the response format from Anthropic API
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	StopReason string `json:"stop_reason"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(baseURL, apiKey string) *AnthropicProvider {
	// Create a default registry for cost calculation
	registry := registry.GetDefaultRegistry()

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(registry),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// Chat performs chat completion using Anthropic API
func (p *AnthropicProvider) Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error) {
	// Convert messages (Anthropic uses different format)
	messages := make([]AnthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		// Skip system messages as Anthropic handles them differently
		if msg.Role == "system" {
			continue
		}
		messages = append(messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Build request
	anthropicReq := AnthropicRequest{
		Model:       mc.ID,
		MaxTokens:   req.MaxTokens,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Marshal request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("anthropic API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return core.ChatResponse{}, fmt.Errorf("anthropic API returned status %d", resp.StatusCode)
	}

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to decode anthropic response: %w", err)
	}

	// Extract text content
	var text string
	for _, content := range anthropicResp.Content {
		if content.Type == "text" {
			text += content.Text
		}
	}

	// Convert response
	chatResp := core.ChatResponse{
		Text: text,
		Usage: core.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
		Model:        mc.ID,
		Provider:     mc.Provider,
		FinishReason: anthropicResp.StopReason,
	}

	return chatResp, nil
}

// Embed generates embeddings (Anthropic doesn't have embeddings API, return error)
func (p *AnthropicProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error) {
	return nil, core.Usage{}, fmt.Errorf("anthropic does not support embeddings")
}

// CreateAnthropicProviderFromConfig creates an Anthropic provider from model config
func CreateAnthropicProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) (*AnthropicProvider, error) {
	apiKey := os.Getenv(mc.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variable %s", mc.APIKeyEnv)
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(registry),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: mc.BaseURL,
		apiKey:  apiKey,
	}, nil
}
