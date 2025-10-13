package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestDesignClient_Design_Success(t *testing.T) {
	// Mock response
	mockResponse := design.HypothesisDesign{
		Status: "ok",
		Algorithm: struct {
			Name       string `json:"name"`
			Idea       string `json:"idea"`
			Complexity struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			} `json:"complexity"`
		}{
			Name: "QuickSort",
			Idea: "Divide and conquer sorting",
			Complexity: struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			}{
				Time:  "O(n log n)",
				Space: "O(log n)",
			},
		},
		Code: struct {
			Lang  string `json:"lang"`
			Entry string `json:"entry"`
			Src   string `json:"src"`
		}{
			Lang:  "af-dsl",
			Entry: "program",
			Src:   "(program (let x 5) (return x))",
		},
		Evaluation: struct {
			Metrics       []string `json:"metrics"`
			Fitness       string   `json:"fitness"`
			PassThreshold float64  `json:"pass_threshold"`
		}{
			Metrics:       []string{"correctness", "time", "size"},
			Fitness:       "score = 0.8*correctness + 0.15*time + 0.05*size",
			PassThreshold: 0.95,
		},
		Tests: struct {
			Unit []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			} `json:"unit"`
			Property []struct {
				Name      string   `json:"name"`
				Generator string   `json:"generator"`
				Checks    []string `json:"checks"`
			} `json:"property"`
		}{
			Unit: []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			}{
				{
					Name:   "test1",
					Input:  "[3,1,4]",
					Oracle: "[1,3,4]",
					Weight: 1.0,
				},
			},
			Property: []struct {
				Name      string   `json:"name"`
				Generator string   `json:"generator"`
				Checks    []string `json:"checks"`
			}{
				{
					Name:      "prop1",
					Generator: "list<int>(n<=100)",
					Checks:    []string{"sorted?", "permutes?"},
				},
			},
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat" {
			t.Errorf("Expected /v1/chat path, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Caller") != "test-caller" {
			t.Errorf("Expected X-Caller: test-caller, got %s", r.Header.Get("X-Caller"))
		}

		// Parse and verify request body
		var req DesignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request structure
		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got %s", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("Expected first message role 'system', got %s", req.Messages[0].Role)
		}
		if req.Messages[1].Role != "user" {
			t.Errorf("Expected second message role 'user', got %s", req.Messages[1].Role)
		}
		if req.ResponseFormat["type"] != "json" {
			t.Errorf("Expected response_format type 'json', got %s", req.ResponseFormat["type"])
		}

		// Return mock response
		response := DesignResponse{
			Text:         jsonResponse(mockResponse),
			Usage:        core.Usage{PromptTokens: 100, CompletionTokens: 200, TotalTokens: 300},
			Model:        "test-model",
			Provider:     "test-provider",
			FinishReason: "stop",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewDesignClient(server.URL, "test-model", "test-caller", &http.Client{Timeout: 5 * time.Second})

	// Test design request
	taskJSON := `{"task": {"id": "test-001", "domain": "algorithms", "description": "Sort array"}}`
	ctx := context.Background()

	result, rawResponse, err := client.Design(ctx, taskJSON)
	if err != nil {
		t.Fatalf("Design failed: %v", err)
	}

	// Verify result
	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", result.Status)
	}
	if result.Algorithm.Name != "QuickSort" {
		t.Errorf("Expected algorithm name 'QuickSort', got %s", result.Algorithm.Name)
	}
	if result.Code.Lang != "af-dsl" {
		t.Errorf("Expected code lang 'af-dsl', got %s", result.Code.Lang)
	}

	// Verify raw response is not empty
	if len(rawResponse) == 0 {
		t.Error("Expected non-empty raw response")
	}
}

func TestDesignClient_Design_WithGarbage(t *testing.T) {
	// Mock response with garbage before and after JSON
	mockResponse := design.HypothesisDesign{
		Status: "ok",
		Algorithm: struct {
			Name       string `json:"name"`
			Idea       string `json:"idea"`
			Complexity struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			} `json:"complexity"`
		}{
			Name: "TestAlgorithm",
			Idea: "Test idea",
			Complexity: struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			}{
				Time:  "O(n)",
				Space: "O(1)",
			},
		},
		Code: struct {
			Lang  string `json:"lang"`
			Entry string `json:"entry"`
			Src   string `json:"src"`
		}{
			Lang:  "af-dsl",
			Entry: "program",
			Src:   "(program (return 42))",
		},
		Evaluation: struct {
			Metrics       []string `json:"metrics"`
			Fitness       string   `json:"fitness"`
			PassThreshold float64  `json:"pass_threshold"`
		}{
			Metrics:       []string{"correctness", "time"},
			Fitness:       "score = 0.8*correctness + 0.2*time",
			PassThreshold: 0.9,
		},
		Tests: struct {
			Unit []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			} `json:"unit"`
			Property []struct {
				Name      string   `json:"name"`
				Generator string   `json:"generator"`
				Checks    []string `json:"checks"`
			} `json:"property"`
		}{
			Unit: []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			}{
				{
					Name:   "test1",
					Input:  "[1,2,3]",
					Oracle: "[1,2,3]",
					Weight: 1.0,
				},
			},
		},
	}

	// Create test server that returns response with garbage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with garbage before and after JSON
		jsonStr := jsonResponse(mockResponse)
		garbageResponse := "Here's some garbage before the JSON:\n" + jsonStr + "\nAnd some more garbage after the JSON."

		response := DesignResponse{
			Text:         garbageResponse,
			Usage:        core.Usage{PromptTokens: 50, CompletionTokens: 100, TotalTokens: 150},
			Model:        "test-model",
			Provider:     "test-provider",
			FinishReason: "stop",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewDesignClient(server.URL, "test-model", "test-caller", &http.Client{Timeout: 5 * time.Second})

	// Test design request
	taskJSON := `{"task": {"id": "test-002", "domain": "algorithms", "description": "Test with garbage"}}`
	ctx := context.Background()

	result, _, err := client.Design(ctx, taskJSON)
	if err != nil {
		t.Fatalf("Design failed: %v", err)
	}

	// Verify result (should extract clean JSON despite garbage)
	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", result.Status)
	}
	if result.Algorithm.Name != "TestAlgorithm" {
		t.Errorf("Expected algorithm name 'TestAlgorithm', got %s", result.Algorithm.Name)
	}
}

func TestDesignClient_Design_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := DesignResponse{
			Text:         "This is not valid JSON at all!",
			Usage:        core.Usage{PromptTokens: 50, CompletionTokens: 100, TotalTokens: 150},
			Model:        "test-model",
			Provider:     "test-provider",
			FinishReason: "stop",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewDesignClient(server.URL, "test-model", "test-caller", &http.Client{Timeout: 5 * time.Second})

	// Test design request
	taskJSON := `{"task": {"id": "test-003", "domain": "algorithms", "description": "Test invalid JSON"}}`
	ctx := context.Background()

	_, _, err := client.Design(ctx, taskJSON)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "no JSON object found") {
		t.Errorf("Expected error about no JSON object found, got: %v", err)
	}
}

func TestDesignClient_Design_ValidationError(t *testing.T) {
	// Mock response with invalid design (missing required fields)
	invalidResponse := `{
		"status": "ok",
		"algorithm": {
			"name": "",
			"idea": "Test idea",
			"complexity": {
				"time": "O(n)",
				"space": "O(1)"
			}
		},
		"code": {
			"lang": "af-dsl",
			"entry": "program",
			"src": "(program (return 42))"
		},
		"evaluation": {
			"metrics": ["correctness", "time"],
			"fitness": "score = 0.8*correctness + 0.2*time",
			"pass_threshold": 0.9
		},
		"tests": {
			"unit": [
				{
					"name": "test1",
					"input": "[1,2,3]",
					"oracle": "[1,2,3]",
					"weight": 1.0
				}
			],
			"property": []
		}
	}`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := DesignResponse{
			Text:         invalidResponse,
			Usage:        core.Usage{PromptTokens: 50, CompletionTokens: 100, TotalTokens: 150},
			Model:        "test-model",
			Provider:     "test-provider",
			FinishReason: "stop",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewDesignClient(server.URL, "test-model", "test-caller", &http.Client{Timeout: 5 * time.Second})

	// Test design request
	taskJSON := `{"task": {"id": "test-004", "domain": "algorithms", "description": "Test validation error"}}`
	ctx := context.Background()

	_, _, err := client.Design(ctx, taskJSON)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if !strings.Contains(err.Error(), "design validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestDesignClient_Design_ServerError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create client
	client := NewDesignClient(server.URL, "test-model", "test-caller", &http.Client{Timeout: 5 * time.Second})

	// Test design request
	taskJSON := `{"task": {"id": "test-005", "domain": "algorithms", "description": "Test server error"}}`
	ctx := context.Background()

	_, _, err := client.Design(ctx, taskJSON)
	if err == nil {
		t.Fatal("Expected server error, got nil")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Expected server error, got: %v", err)
	}
}

func TestSanitizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "clean JSON",
			input:    `{"status": "ok", "algorithm": {"name": "test"}}`,
			expected: `{"status": "ok", "algorithm": {"name": "test"}}`,
			wantErr:  false,
		},
		{
			name:     "JSON with garbage before",
			input:    `garbage before {"status": "ok", "algorithm": {"name": "test"}}`,
			expected: `{"status": "ok", "algorithm": {"name": "test"}}`,
			wantErr:  false,
		},
		{
			name:     "JSON with garbage after",
			input:    `{"status": "ok", "algorithm": {"name": "test"}} garbage after`,
			expected: `{"status": "ok", "algorithm": {"name": "test"}}`,
			wantErr:  false,
		},
		{
			name:     "JSON with garbage before and after",
			input:    `before {"status": "ok", "algorithm": {"name": "test"}} after`,
			expected: `{"status": "ok", "algorithm": {"name": "test"}}`,
			wantErr:  false,
		},
		{
			name:     "nested JSON",
			input:    `text {"outer": {"inner": {"value": 42}}} more text`,
			expected: `{"outer": {"inner": {"value": 42}}}`,
			wantErr:  false,
		},
		{
			name:    "no JSON object",
			input:   `just plain text with no JSON`,
			wantErr: true,
		},
		{
			name:    "unclosed JSON",
			input:   `{"status": "ok", "algorithm": {"name": "test"`,
			wantErr: true,
		},
		{
			name:     "invalid JSON structure",
			input:    `{"status": "ok", "algorithm": {"name": "test"}} invalid`,
			expected: `{"status": "ok", "algorithm": {"name": "test"}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("sanitizeJSON() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("sanitizeJSON() error = %v, want nil", err)
				return
			}

			if result != tt.expected {
				t.Errorf("sanitizeJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to convert design to JSON string
func jsonResponse(d design.HypothesisDesign) string {
	bytes, _ := json.Marshal(d)
	return string(bytes)
}
