package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/snow-ghost/agent/pkg/cost"
	"github.com/snow-ghost/agent/pkg/limiter"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
	"github.com/snow-ghost/agent/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelPriorityInRequest tests that when a model is explicitly specified in the request,
// it should be used regardless of the routing strategy
func TestModelPriorityInRequest(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Create test registry with multiple models
	testRegistry := createTestRegistry()

	// Create model router with test registry
	modelRouter := routing.NewModelRouter(testRegistry)

	// Create cost calculator
	costCalculator := cost.NewCalculator(testRegistry)

	// Create protection manager
	protectionManager := limiter.NewProtectionManager(testRegistry)

	// Create server with test registry
	server := &Server{
		logger:            logger,
		registry:          testRegistry,
		modelRouter:       modelRouter,
		costCalculator:    costCalculator,
		protectionManager: protectionManager,
	}

	tests := []struct {
		name           string
		requestModel   string
		strategy       string
		expectedModel  string
		shouldUseModel bool // true if the request model should be used, false if strategy should be used
	}{
		{
			name:           "explicit_model_with_round_robin_strategy",
			requestModel:   "gpt-4",
			strategy:       "round-robin",
			expectedModel:  "gpt-4",
			shouldUseModel: true,
		},
		{
			name:           "explicit_model_with_weighted_strategy",
			requestModel:   "claude-3-sonnet",
			strategy:       "weighted",
			expectedModel:  "claude-3-sonnet",
			shouldUseModel: true,
		},
		{
			name:           "explicit_model_with_cost_aware_strategy",
			requestModel:   "gpt-3.5-turbo",
			strategy:       "cost-aware",
			expectedModel:  "gpt-3.5-turbo",
			shouldUseModel: true,
		},
		{
			name:           "empty_model_uses_strategy",
			requestModel:   "",
			strategy:       "round-robin",
			expectedModel:  "", // Will be determined by strategy
			shouldUseModel: false,
		},
		{
			name:           "no_model_field_uses_strategy",
			requestModel:   "", // Simulate missing model field
			strategy:       "weighted",
			expectedModel:  "", // Will be determined by strategy
			shouldUseModel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create chat request
			req := core.ChatRequest{
				Model: tt.requestModel,
				Messages: []core.Message{
					{Role: "user", Content: "Hello, world!"},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			}

			// Create HTTP request
			reqBody, err := json.Marshal(req)
			require.NoError(t, err)

			httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
			httpReq.Header.Set("Content-Type", "application/json")

			// Add strategy as query parameter
			if tt.strategy != "" {
				httpReq.URL.RawQuery = fmt.Sprintf("strategy=%s", tt.strategy)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			server.handleChat(w, httpReq)

			// Check response status
			assert.Equal(t, http.StatusOK, w.Code, "Expected successful response")

			// Parse response
			var response core.ChatResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response")

			if tt.shouldUseModel {
				// When model is explicitly specified, it should be used
				assert.Equal(t, tt.expectedModel, response.Model,
					"Expected response model to match request model when explicitly specified")
			} else {
				// When model is not specified, strategy should be used
				// The response model should be one of the available models from registry
				availableModels := getModelIDs(testRegistry)
				assert.Contains(t, availableModels, response.Model,
					"Expected response model to be one of the available models from strategy selection")
			}

			// Verify that the response contains the expected model information
			assert.NotEmpty(t, response.Model, "Response model should not be empty")
			assert.NotEmpty(t, response.Provider, "Response provider should not be empty")
		})
	}
}

// TestModelSelectionWithStrategy tests that when no model is specified,
// the routing strategy is used to select a model
func TestModelSelectionWithStrategy(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create test registry
	testRegistry := createTestRegistry()

	// Create model router
	modelRouter := routing.NewModelRouter(testRegistry)

	// Create cost calculator
	costCalculator := cost.NewCalculator(testRegistry)

	// Create protection manager
	protectionManager := limiter.NewProtectionManager(testRegistry)

	// Create server
	server := &Server{
		logger:            logger,
		registry:          testRegistry,
		modelRouter:       modelRouter,
		costCalculator:    costCalculator,
		protectionManager: protectionManager,
	}

	strategies := []string{"round-robin", "weighted", "cost-aware", "tag-based"}
	availableModels := getModelIDs(testRegistry)

	for _, strategy := range strategies {
		t.Run(fmt.Sprintf("strategy_%s", strategy), func(t *testing.T) {
			// Create request without model
			req := core.ChatRequest{
				Messages: []core.Message{
					{Role: "user", Content: "Hello, world!"},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			}

			// Create HTTP request
			reqBody, err := json.Marshal(req)
			require.NoError(t, err)

			httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.URL.RawQuery = fmt.Sprintf("strategy=%s", strategy)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			server.handleChat(w, httpReq)

			// Check response status
			assert.Equal(t, http.StatusOK, w.Code, "Expected successful response for strategy %s", strategy)

			// Parse response
			var response core.ChatResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response for strategy %s", strategy)

			// Verify that a model was selected by the strategy
			assert.NotEmpty(t, response.Model, "Strategy %s should select a model", strategy)
			assert.Contains(t, availableModels, response.Model,
				"Strategy %s selected model %s should be from available models", strategy, response.Model)
		})
	}
}

// TestModelValidation tests that the selected model is valid and exists in the registry
func TestModelValidation(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create test registry
	testRegistry := createTestRegistry()

	// Create model router
	modelRouter := routing.NewModelRouter(testRegistry)

	// Create cost calculator
	costCalculator := cost.NewCalculator(testRegistry)

	// Create protection manager
	protectionManager := limiter.NewProtectionManager(testRegistry)

	// Create server
	server := &Server{
		logger:            logger,
		registry:          testRegistry,
		modelRouter:       modelRouter,
		costCalculator:    costCalculator,
		protectionManager: protectionManager,
	}

	t.Run("valid_explicit_model", func(t *testing.T) {
		// Test with a valid model that exists in registry
		// Current implementation ignores explicit model and uses strategy
		req := core.ChatRequest{
			Model: "gpt-4",
			Messages: []core.Message{
				{Role: "user", Content: "Hello, world!"},
			},
		}

		reqBody, err := json.Marshal(req)
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.handleChat(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Valid model should succeed")

		var response core.ChatResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Should use the explicitly specified model
		assert.Equal(t, "gpt-4", response.Model, "Should use the explicitly specified model")
	})

	t.Run("invalid_explicit_model", func(t *testing.T) {
		// Test with an invalid model that doesn't exist in registry
		req := core.ChatRequest{
			Model: "non-existent-model",
			Messages: []core.Message{
				{Role: "user", Content: "Hello, world!"},
			},
		}

		reqBody, err := json.Marshal(req)
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.handleChat(w, httpReq)

		// Should still succeed because the current implementation uses strategy when model selection fails
		// This might need to be adjusted based on actual business requirements
		assert.Equal(t, http.StatusOK, w.Code, "Invalid model should fallback to strategy")

		var response core.ChatResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Should have selected a model via strategy
		assert.NotEmpty(t, response.Model, "Should have selected a model via strategy")
		availableModels := getModelIDs(testRegistry)
		assert.Contains(t, availableModels, response.Model, "Selected model should be from available models")
	})
}

// Helper function to create a test registry with multiple models
func createTestRegistry() *registry.Registry {
	models := []registry.ModelConfig{
		{
			ID:       "gpt-4",
			Provider: "openai",
			Kind:     "chat",
			BaseURL:  "https://api.openai.com/v1",
			Pricing: registry.Pricing{
				InputPer1K:  0.03,
				OutputPer1K: 0.06,
				Currency:    "USD",
			},
			MaxRPM: 500,
			MaxTPM: 150000,
			Tags:   []string{"premium", "latest"},
		},
		{
			ID:       "gpt-3.5-turbo",
			Provider: "openai",
			Kind:     "chat",
			BaseURL:  "https://api.openai.com/v1",
			Pricing: registry.Pricing{
				InputPer1K:  0.0015,
				OutputPer1K: 0.002,
				Currency:    "USD",
			},
			MaxRPM: 1000,
			MaxTPM: 200000,
			Tags:   []string{"cost-effective", "fast"},
		},
		{
			ID:       "claude-3-sonnet",
			Provider: "anthropic",
			Kind:     "chat",
			BaseURL:  "https://api.anthropic.com/v1",
			Pricing: registry.Pricing{
				InputPer1K:  0.003,
				OutputPer1K: 0.015,
				Currency:    "USD",
			},
			MaxRPM: 200,
			MaxTPM: 100000,
			Tags:   []string{"premium", "reasoning"},
		},
		{
			ID:       "claude-3-haiku",
			Provider: "anthropic",
			Kind:     "chat",
			BaseURL:  "https://api.anthropic.com/v1",
			Pricing: registry.Pricing{
				InputPer1K:  0.00025,
				OutputPer1K: 0.00125,
				Currency:    "USD",
			},
			MaxRPM: 500,
			MaxTPM: 200000,
			Tags:   []string{"fast", "cost-effective"},
		},
	}

	return &registry.Registry{
		Models: models,
	}
}

// Helper function to get model IDs from registry
func getModelIDs(reg *registry.Registry) []string {
	ids := make([]string, len(reg.Models))
	for i, model := range reg.Models {
		ids[i] = model.ID
	}
	return ids
}

// TestModelPriorityDesiredBehavior tests the desired behavior where explicit model takes priority over strategy
// This test demonstrates what the implementation should do when model priority is implemented
func TestModelPriorityDesiredBehavior(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create test registry
	testRegistry := createTestRegistry()

	// Create model router
	modelRouter := routing.NewModelRouter(testRegistry)

	// Create cost calculator
	costCalculator := cost.NewCalculator(testRegistry)

	// Create protection manager
	protectionManager := limiter.NewProtectionManager(testRegistry)

	// Create server with model priority support
	server := &Server{
		logger:            logger,
		registry:          testRegistry,
		modelRouter:       modelRouter,
		costCalculator:    costCalculator,
		protectionManager: protectionManager,
	}

	t.Run("explicit_model_should_override_strategy", func(t *testing.T) {
		// This test documents the desired behavior
		// When model priority is implemented, this test should pass

		req := core.ChatRequest{
			Model: "gpt-4", // Explicitly specify model
			Messages: []core.Message{
				{Role: "user", Content: "Hello, world!"},
			},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		reqBody, err := json.Marshal(req)
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.URL.RawQuery = "strategy=round-robin" // Use different strategy

		w := httptest.NewRecorder()
		server.handleChat(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Request should succeed")

		var response core.ChatResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Explicit model should override strategy
		assert.Equal(t, "gpt-4", response.Model, "Explicit model should override strategy")
	})

	t.Run("empty_model_should_use_strategy", func(t *testing.T) {
		req := core.ChatRequest{
			// No model specified
			Messages: []core.Message{
				{Role: "user", Content: "Hello, world!"},
			},
			Temperature: 0.7,
			MaxTokens:   100,
		}

		reqBody, err := json.Marshal(req)
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.URL.RawQuery = "strategy=round-robin"

		w := httptest.NewRecorder()
		server.handleChat(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Request should succeed")

		var response core.ChatResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Should use strategy when no model is specified
		availableModels := getModelIDs(testRegistry)
		assert.Contains(t, availableModels, response.Model, "Should select a model via strategy")
	})
}
