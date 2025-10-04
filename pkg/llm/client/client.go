package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// Client represents an HTTP client for LLM router
type Client struct {
	baseURL      string
	httpClient   *http.Client
	defaultModel string
	modelTag     string
	retryCount   int
	retryDelay   time.Duration
}

// Config holds client configuration
type Config struct {
	BaseURL      string
	DefaultModel string
	ModelTag     string
	Timeout      time.Duration
	RetryCount   int
	RetryDelay   time.Duration
}

// NewClient creates a new LLM router client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	return &Client{
		baseURL:      config.BaseURL,
		defaultModel: config.DefaultModel,
		modelTag:     config.ModelTag,
		retryCount:   config.RetryCount,
		retryDelay:   config.RetryDelay,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []core.Message    `json:"messages"`
	Tools       []core.Tool       `json:"tools,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Caller      string            `json:"caller,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Text         string          `json:"text"`
	ToolCalls    []core.ToolCall `json:"tool_calls,omitempty"`
	Usage        core.Usage      `json:"usage"`
	Model        string          `json:"model"`
	Provider     string          `json:"provider"`
	FinishReason string          `json:"finish_reason"`
}

// Chat sends a chat completion request to the LLM router
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Set default model if not specified
	if req.Model == "" {
		req.Model = c.defaultModel
	}

	// Add model tag to metadata if specified
	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}
	if c.modelTag != "" {
		req.Metadata["task_domain"] = c.modelTag
	}

	// Marshal request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat", bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Caller", req.Caller)

	// Send request with retry
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		resp, lastErr = c.httpClient.Do(httpReq)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt == c.retryCount {
			return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryCount+1, lastErr)
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM router returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// Complete sends a text completion request (simplified chat)
func (c *Client) Complete(ctx context.Context, prompt string, caller string) (string, error) {
	req := ChatRequest{
		Messages: []core.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Caller: caller,
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}

// Embed sends an embedding request
func (c *Client) Embed(ctx context.Context, texts []string, caller string) ([][]float32, error) {
	// Marshal request
	reqData, err := json.Marshal(map[string]interface{}{
		"input":  texts,
		"caller": caller,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/embed", bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Caller", caller)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM router returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var embedResp struct {
		Data [][]float32 `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Data, nil
}

// GetModels retrieves available models from the LLM router
func (c *Client) GetModels(ctx context.Context) ([]core.Model, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM router returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var modelsResp struct {
		Models []core.Model `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return modelsResp.Models, nil
}

// Health checks if the LLM router is healthy
func (c *Client) Health(ctx context.Context) error {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM router health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
