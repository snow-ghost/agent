//go:build ignore
// +build ignore

package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/pkg/registry"
	routercore "github.com/snow-ghost/agent/pkg/router/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAIProviderIntegration tests OpenAI provider integration
func TestOpenAIProviderIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	// Create OpenAI provider
	provider := NewOpenAIProvider("", "")

	// Create test request
	req := routercore.ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Get model config
	reg := registry.GetDefaultRegistry()
	modelConfig := reg.FindModel("openai:gpt-3.5-turbo")
	require.NotNil(t, modelConfig)

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, *modelConfig, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
	assert.NotEmpty(t, resp.Usage)
	assert.Greater(t, resp.Usage.TotalTokens, 0)
}

// TestAnthropicProviderIntegration tests Anthropic provider integration
func TestAnthropicProviderIntegration(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	// Create Anthropic provider
	provider := NewAnthropicProvider("", "")

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "claude-3-haiku-20240307",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
	assert.NotEmpty(t, resp.Usage)
	assert.Greater(t, resp.Usage.TotalTokens, 0)
}

// TestOllamaProviderIntegration tests Ollama provider integration
func TestOllamaProviderIntegration(t *testing.T) {
	// Check if Ollama is running
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Create Ollama provider
	provider := NewOllamaProvider(baseURL)

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "llama2",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
}

// TestVLLMProviderIntegration tests vLLM provider integration
func TestVLLMProviderIntegration(t *testing.T) {
	// Check if vLLM is running
	baseURL := os.Getenv("VLLM_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	// Create vLLM provider
	provider := NewVLLMProvider(baseURL)

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "llama-3.1-8b",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Skipf("vLLM not available: %v", err)
	}

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
}

// TestLMStudioProviderIntegration tests LMStudio provider integration
func TestLMStudioProviderIntegration(t *testing.T) {
	// Check if LMStudio is running
	baseURL := os.Getenv("LMSTUDIO_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	// Create LMStudio provider
	provider := NewLMStudioProvider(baseURL)

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "llama-3.1-8b",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Skipf("LMStudio not available: %v", err)
	}

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
}

// TestOpenRouterProviderIntegration tests OpenRouter provider integration
func TestOpenRouterProviderIntegration(t *testing.T) {
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		t.Skip("OPENROUTER_API_KEY not set, skipping integration test")
	}

	// Create OpenRouter provider
	provider := NewOpenRouterProvider()

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "meta-llama/llama-3.1-8b-instruct",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test chat completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Text)
	assert.NotEmpty(t, resp.Usage)
	assert.Greater(t, resp.Usage.TotalTokens, 0)
}

// TestProviderStreamingIntegration tests streaming functionality
func TestProviderStreamingIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping streaming test")
	}

	// Create OpenAI provider
	provider := NewOpenAIProvider("", "")

	// Create test request
	req := &routercore.ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    []core.Message{{Role: "user", Content: "Count from 1 to 5"}},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      true,
	}

	// Test streaming
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chunks := make([]StreamChunk, 0)
	handler := &TestStreamHandler{chunks: &chunks}

	err := provider.Stream(ctx, req, handler)
	require.NoError(t, err)

	// Verify we received chunks
	assert.Greater(t, len(chunks), 0)

	// Verify chunks have content
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.Content)
	}
}

// TestStreamHandler is a test implementation of StreamHandler
type TestStreamHandler struct {
	chunks *[]StreamChunk
}

func (h *TestStreamHandler) HandleChunk(chunk StreamChunk) error {
	*h.chunks = append(*h.chunks, chunk)
	return nil
}

func (h *TestStreamHandler) HandleDone(response StreamResponse) error {
	return nil
}

func (h *TestStreamHandler) HandleError(err error) error {
	return err
}

// TestProviderFactoryIntegration tests provider factory
func TestProviderFactoryIntegration(t *testing.T) {
	// Test OpenAI provider creation
	provider, err := CreateProvider("openai", "")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.IsType(t, &OpenAIProvider{}, provider)

	// Test Anthropic provider creation
	provider, err = CreateProvider("anthropic", "")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.IsType(t, &AnthropicProvider{}, provider)

	// Test Ollama provider creation
	provider, err = CreateProvider("ollama", "http://localhost:11434")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.IsType(t, &OllamaProvider{}, provider)

	// Test invalid provider
	provider, err = CreateProvider("invalid", "")
	require.Error(t, err)
	assert.Nil(t, provider)
}

// TestProviderWithRegistryIntegration tests provider with registry
func TestProviderWithRegistryIntegration(t *testing.T) {
	// Create a test registry
	reg := registry.GetDefaultRegistry()

	// Test getting provider for a model
	modelConfig := reg.FindModel("openai:gpt-3.5-turbo")
	require.NotNil(t, modelConfig)

	provider, err := CreateProvider(modelConfig.Provider, "")
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

// TestProviderErrorHandlingIntegration tests error handling
func TestProviderErrorHandlingIntegration(t *testing.T) {
	// Test with invalid API key
	provider := NewOpenAIProvider()

	req := &routercore.ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Temporarily set invalid API key
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := provider.Chat(ctx, req)
	assert.Error(t, err)
}

// TestProviderTimeoutIntegration tests timeout handling
func TestProviderTimeoutIntegration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping timeout test")
	}

	provider := NewOpenAIProvider()

	req := &routercore.ChatRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    []routercore.Message{{Role: "user", Content: "Hello, world!"}},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := provider.Chat(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
