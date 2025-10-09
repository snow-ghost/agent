//go:build ignore
// +build ignore

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouterWorkerIntegration tests the full router -> worker -> LLM router flow
func TestRouterWorkerIntegration(t *testing.T) {
	// Skip if no API keys are available
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("No API keys available, skipping integration test")
	}

	// Create test server
	logger := createTestLogger()
	server := NewServer("0", logger) // Use port 0 for dynamic port assignment

	// Start server
	err := server.StartWithGracefulShutdown()
	require.NoError(t, err)

	// Get server address
	addr := server.httpServer.Addr
	if addr == "" {
		addr = ":0" // Fallback
	}

	// Test health endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		resp := makeRequest(t, "GET", fmt.Sprintf("http://localhost%s/health", addr), nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "ok", health["status"])
	})

	// Test chat completion endpoint
	t.Run("ChatCompletion", func(t *testing.T) {
		req := ChatRequest{
			Model:       "gpt-3.5-turbo",
			Messages:    []core.Message{{Role: "user", Content: "Hello, world!"}},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var chatResp ChatResponse
		err := json.NewDecoder(resp.Body).Decode(&chatResp)
		require.NoError(t, err)
		assert.NotEmpty(t, chatResp.Choices)
		assert.NotEmpty(t, chatResp.Choices[0].Message.Content)
	})

	// Test streaming endpoint
	t.Run("Streaming", func(t *testing.T) {
		req := ChatRequest{
			Model:       "gpt-3.5-turbo",
			Messages:    []core.Message{{Role: "user", Content: "Count from 1 to 3"}},
			Temperature: 0.7,
			MaxTokens:   100,
			Stream:      true,
		}

		resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify streaming response
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	})

	// Test embeddings endpoint
	t.Run("Embeddings", func(t *testing.T) {
		req := EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: "Hello, world!",
		}

		resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/embeddings", addr), req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var embedResp EmbeddingResponse
		err := json.NewDecoder(resp.Body).Decode(&embedResp)
		require.NoError(t, err)
		assert.NotEmpty(t, embedResp.Data)
		assert.NotEmpty(t, embedResp.Data[0].Embedding)
	})

	// Test models endpoint
	t.Run("Models", func(t *testing.T) {
		resp := makeRequest(t, "GET", fmt.Sprintf("http://localhost%s/v1/models", addr), nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var modelsResp ModelsResponse
		err := json.NewDecoder(resp.Body).Decode(&modelsResp)
		require.NoError(t, err)
		assert.NotEmpty(t, modelsResp.Data)
	})

	// Test metrics endpoint
	t.Run("Metrics", func(t *testing.T) {
		resp := makeRequest(t, "GET", fmt.Sprintf("http://localhost%s/metrics", addr), nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify Prometheus metrics format
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		metrics := string(body[:n])
		assert.Contains(t, metrics, "llm_requests_total")
	})

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestRouterErrorHandlingIntegration tests error handling
func TestRouterErrorHandlingIntegration(t *testing.T) {
	logger := createTestLogger()
	server := NewServer("0", logger)

	err := server.StartWithGracefulShutdown()
	require.NoError(t, err)

	addr := server.httpServer.Addr
	if addr == "" {
		addr = ":0"
	}

	// Test invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		resp := makeRawRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr),
			[]byte("invalid json"))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test invalid model
	t.Run("InvalidModel", func(t *testing.T) {
		req := ChatRequest{
			Model:       "invalid-model",
			Messages:    []core.Message{{Role: "user", Content: "Hello!"}},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test missing messages
	t.Run("MissingMessages", func(t *testing.T) {
		req := ChatRequest{
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			MaxTokens:   100,
		}

		resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestRouterRateLimitingIntegration tests rate limiting
func TestRouterRateLimitingIntegration(t *testing.T) {
	logger := createTestLogger()
	server := NewServer("0", logger)

	err := server.StartWithGracefulShutdown()
	require.NoError(t, err)

	addr := server.httpServer.Addr
	if addr == "" {
		addr = ":0"
	}

	// Test rate limiting (if enabled)
	t.Run("RateLimiting", func(t *testing.T) {
		req := ChatRequest{
			Model:       "gpt-3.5-turbo",
			Messages:    []core.Message{{Role: "user", Content: "Hello!"}},
			Temperature: 0.7,
			MaxTokens:   10,
		}

		// Make multiple requests quickly
		for i := 0; i < 10; i++ {
			resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
			// Should not get rate limited immediately in test environment
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests)
		}
	})

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// TestRouterCircuitBreakerIntegration tests circuit breaker
func TestRouterCircuitBreakerIntegration(t *testing.T) {
	logger := createTestLogger()
	server := NewServer("0", logger)

	err := server.StartWithGracefulShutdown()
	require.NoError(t, err)

	addr := server.httpServer.Addr
	if addr == "" {
		addr = ":0"
	}

	// Test circuit breaker (if enabled)
	t.Run("CircuitBreaker", func(t *testing.T) {
		req := ChatRequest{
			Model:       "gpt-3.5-turbo",
			Messages:    []core.Message{{Role: "user", Content: "Hello!"}},
			Temperature: 0.7,
			MaxTokens:   10,
		}

		// Make requests that might trigger circuit breaker
		for i := 0; i < 5; i++ {
			resp := makeRequest(t, "POST", fmt.Sprintf("http://localhost%s/v1/chat/completions", addr), req)
			// Should not get circuit breaker error immediately in test environment
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)
		}
	})

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// Helper functions

func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))
}

func makeRequest(t *testing.T, method, url string, body interface{}) *http.Response {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	return makeRawRequest(t, method, url, reqBody)
}

func makeRawRequest(t *testing.T, method, url string, body []byte) *http.Response {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// Test data structures (simplified versions for testing)

type ChatRequest struct {
	Model       string         `json:"model"`
	Messages    []core.Message `json:"messages"`
	Temperature float32        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type ModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
