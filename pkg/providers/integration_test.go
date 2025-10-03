package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// MockOpenAIServer creates a mock OpenAI-compatible server
func MockOpenAIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Debug: log the request
		// fmt.Printf("Mock server received request: %s %s\n", r.Method, r.URL.Path)

		if r.URL.Path == "/chat/completions" {
			// Mock chat completion response
			response := map[string]interface{}{
				"id":      "chatcmpl-test",
				"object":  "chat.completion",
				"created": 1234567890,
				"model":   "gpt-3.5-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "This is a mock response from OpenAI",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 15,
					"total_tokens":      25,
				},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/embeddings" {
			// Mock embeddings response
			response := map[string]interface{}{
				"object": "list",
				"data": []map[string]interface{}{
					{
						"object":    "embedding",
						"index":     0,
						"embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5},
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens": 5,
					"total_tokens":  5,
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// Debug: log unexpected requests
			// fmt.Printf("Mock server: unexpected request %s %s\n", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

// MockAnthropicServer creates a mock Anthropic server
func MockAnthropicServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/v1/messages" {
			// Mock Anthropic messages response
			response := map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "This is a mock response from Anthropic",
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  10,
					"output_tokens": 15,
				},
				"stop_reason": "end_turn",
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
}

// MockOllamaServer creates a mock Ollama server
func MockOllamaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/chat" {
			// Mock Ollama chat response
			response := map[string]interface{}{
				"model": "llama3.2",
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "This is a mock response from Ollama",
				},
				"done":       true,
				"created_at": "2023-01-01T00:00:00Z",
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/api/embeddings" {
			// Mock Ollama embeddings response
			response := map[string]interface{}{
				"embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
}

func TestOpenAIProvider(t *testing.T) {
	server := MockOpenAIServer()
	defer server.Close()

	// Create provider with mock server
	provider := NewOpenAIProvider(server.URL, "test-key")

	// Test chat
	mc := registry.ModelConfig{
		ID:       "gpt-3.5-turbo",
		Provider: "openai",
		BaseURL:  server.URL,
	}

	req := core.ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Chat(context.Background(), mc, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Text != "This is a mock response from OpenAI" {
		t.Errorf("Expected mock response, got: %s", resp.Text)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got: %d", resp.Usage.PromptTokens)
	}

	// Test embeddings
	embeddings, _, err := provider.Embed(context.Background(), mc, []string{"test"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != 1 {
		t.Errorf("Expected 1 embedding, got: %d", len(embeddings))
	}
}

func TestAnthropicProvider(t *testing.T) {
	server := MockAnthropicServer()
	defer server.Close()

	// Create provider with mock server
	provider := NewAnthropicProvider(server.URL, "test-key")

	// Test chat
	mc := registry.ModelConfig{
		ID:       "claude-3-sonnet",
		Provider: "anthropic",
		BaseURL:  server.URL,
	}

	req := core.ChatRequest{
		Model: "claude-3-sonnet",
		Messages: []core.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Chat(context.Background(), mc, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Text != "This is a mock response from Anthropic" {
		t.Errorf("Expected mock response, got: %s", resp.Text)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got: %d", resp.Usage.PromptTokens)
	}

	// Test embeddings (should fail)
	_, _, err = provider.Embed(context.Background(), mc, []string{"test"})
	if err == nil {
		t.Error("Expected embeddings to fail for Anthropic")
	}
}

func TestOllamaProvider(t *testing.T) {
	server := MockOllamaServer()
	defer server.Close()

	// Create provider with mock server
	provider := NewOllamaProvider(server.URL)

	// Test chat
	mc := registry.ModelConfig{
		ID:       "llama3.2",
		Provider: "ollama",
		BaseURL:  server.URL,
	}

	req := core.ChatRequest{
		Model: "llama3.2",
		Messages: []core.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := provider.Chat(context.Background(), mc, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Text != "This is a mock response from Ollama" {
		t.Errorf("Expected mock response, got: %s", resp.Text)
	}

	// Test embeddings
	embeddings, _, err := provider.Embed(context.Background(), mc, []string{"test"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != 1 {
		t.Errorf("Expected 1 embedding, got: %d", len(embeddings))
	}

	if len(embeddings[0]) != 5 {
		t.Errorf("Expected 5-dimensional embedding, got: %d", len(embeddings[0]))
	}
}

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory()

	// Test supported providers
	supported := factory.GetSupportedProviders()
	expected := []string{"openai", "anthropic", "ollama", "vllm", "lmstudio", "openrouter"}

	if len(supported) != len(expected) {
		t.Errorf("Expected %d supported providers, got: %d", len(expected), len(supported))
	}

	// Test creating providers
	for _, providerType := range expected {
		provider, err := factory.CreateProvider(providerType)
		if err != nil {
			t.Errorf("Failed to create provider %s: %v", providerType, err)
		}
		if provider == nil {
			t.Errorf("Provider %s is nil", providerType)
		}
	}

	// Test unsupported provider
	_, err := factory.CreateProvider("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

func TestUsageEstimator(t *testing.T) {
	estimator := &MockUsageEstimator{}

	// Test token estimation
	text := "This is a test message with some content."
	promptTokens, completionTokens := estimator.EstimateTokens(text)

	if promptTokens == 0 {
		t.Error("Expected non-zero prompt tokens")
	}

	if completionTokens != 0 {
		t.Error("Expected zero completion tokens for prompt estimation")
	}

	// Test completion token estimation
	completionTokens = estimator.EstimateCompletionTokens("This is a completion.")
	if completionTokens == 0 {
		t.Error("Expected non-zero completion tokens")
	}
}
