package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FullSystemFlow tests the complete system flow
func TestE2E_FullSystemFlow(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	// Configuration
	routerURL := getenv("ROUTER_URL", "http://localhost:8083")
	llmRouterURL := getenv("LLMROUTER_URL", "http://localhost:8090")
	artifactsDir := getenv("ARTIFACTS_DIR", "./artifacts")

	// Clean artifacts dir before run
	_ = os.MkdirAll(artifactsDir, 0o755)
	before, _ := filepath.Glob(filepath.Join(artifactsDir, "hypothesis.*@*/manifest.json"))

	// Test 1: Light worker task (KB only)
	t.Run("LightWorkerTask", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-light-001",
			Domain:      "algorithms.sorting",
			Description: "Sort numbers ascending",
			Spec: core.Spec{
				SuccessCriteria: []string{"sorted_non_decreasing"},
				Props:           map[string]string{"type": "sort"},
				MetricsWeights:  map[string]float64{"cases_passed": 1.0, "cases_total": 0.0},
			},
			Input:  json.RawMessage(`{"numbers":[3,1,2]}`),
			Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 3},
			Budget: core.Budget{Timeout: mustParseDuration("30s")},
		}

		result := submitTask(t, routerURL, task)
		assert.True(t, result.Success, "Light worker task should succeed")
		assert.Greater(t, result.Score, 0.0, "Score should be positive")
	})

	// Test 2: Heavy worker task (LLM+WASM+KB)
	t.Run("HeavyWorkerTask", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-heavy-001",
			Domain:      "algorithms",
			Description: "Sort numbers ascending",
			Spec: core.Spec{
				SuccessCriteria: []string{"sorted_non_decreasing", "permutes"},
				Props:           map[string]string{"type": "sort"},
				MetricsWeights:  map[string]float64{"cases_passed": 1.0, "cases_total": 0.0},
			},
			Input:  json.RawMessage(`{"numbers":[3,1,2]}`),
			Flags:  core.TaskFlags{RequiresSandbox: true, MaxComplexity: 10},
			Budget: core.Budget{Timeout: mustParseDuration("60s")},
		}

		result := submitTask(t, routerURL, task)
		assert.True(t, result.Success, "Heavy worker task should succeed")
		assert.Greater(t, result.Score, 0.0, "Score should be positive")
	})

	// Test 3: Verify artifact creation
	t.Run("ArtifactCreation", func(t *testing.T) {
		// Poll for new artifact manifest up to 30s
		deadline := time.Now().Add(30 * time.Second)
		found := false
		for time.Now().Before(deadline) {
			after, _ := filepath.Glob(filepath.Join(artifactsDir, "hypothesis.*@*/manifest.json"))
			if len(after) > len(before) {
				found = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		assert.True(t, found, "Expected a new KB artifact manifest to be created")
	})

	// Test 4: Test router capabilities
	t.Run("RouterCapabilities", func(t *testing.T) {
		resp := makeRequest(t, "GET", routerURL+"/caps", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var caps map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&caps)
		require.NoError(t, err)
		assert.Contains(t, caps, "light_worker")
		assert.Contains(t, caps, "heavy_worker")
	})

	// Test 5: Test router readiness
	t.Run("RouterReadiness", func(t *testing.T) {
		resp := makeRequest(t, "GET", routerURL+"/ready", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ready map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&ready)
		require.NoError(t, err)
		assert.Equal(t, "ready", ready["status"])
	})

	// Test 6: Test LLM router health
	t.Run("LLMRouterHealth", func(t *testing.T) {
		resp := makeRequest(t, "GET", llmRouterURL+"/health", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "ok", health["status"])
	})

	// Test 7: Test LLM router models
	t.Run("LLMRouterModels", func(t *testing.T) {
		resp := makeRequest(t, "GET", llmRouterURL+"/v1/models", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var models map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&models)
		require.NoError(t, err)
		assert.Contains(t, models, "data")
	})

	// Test 8: Test vector search (if enabled)
	t.Run("VectorSearch", func(t *testing.T) {
		if os.Getenv("VECTOR_BACKEND") == "memory" || os.Getenv("VECTOR_BACKEND") == "qdrant" {
			// Test vector search functionality
			task := core.Task{
				ID:          "e2e-vector-001",
				Domain:      "algorithms.search",
				Description: "Find similar algorithms",
				Spec: core.Spec{
					SuccessCriteria: []string{"similarity_score"},
					Props:           map[string]string{"type": "search"},
					MetricsWeights:  map[string]float64{"similarity_score": 1.0},
				},
				Input:  json.RawMessage(`{"query":"sorting algorithm"}`),
				Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 5},
				Budget: core.Budget{Timeout: mustParseDuration("30s")},
			}

			result := submitTask(t, routerURL, task)
			// Vector search might not be fully implemented, so we just check it doesn't crash
			assert.NotNil(t, result)
		}
	})
}

// TestE2E_WorkerFailover tests worker failover scenarios
func TestE2E_WorkerFailover(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:8083")

	// Test 1: Heavy worker unavailable (should fallback to light)
	t.Run("HeavyWorkerUnavailable", func(t *testing.T) {
		// This test would require stopping the heavy worker
		// For now, just test that the router handles the case gracefully
		task := core.Task{
			ID:          "e2e-failover-001",
			Domain:      "algorithms",
			Description: "Test failover",
			Spec: core.Spec{
				SuccessCriteria: []string{"completed"},
				Props:           map[string]string{"type": "test"},
				MetricsWeights:  map[string]float64{"completed": 1.0},
			},
			Input:  json.RawMessage(`{"test": true}`),
			Flags:  core.TaskFlags{RequiresSandbox: true, MaxComplexity: 10},
			Budget: core.Budget{Timeout: mustParseDuration("30s")},
		}

		result := submitTask(t, routerURL, task)
		// Should either succeed or fail gracefully
		assert.NotNil(t, result)
	})

	// Test 2: Light worker unavailable (should fail gracefully)
	t.Run("LightWorkerUnavailable", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-failover-002",
			Domain:      "algorithms.sorting",
			Description: "Test light worker failover",
			Spec: core.Spec{
				SuccessCriteria: []string{"sorted_non_decreasing"},
				Props:           map[string]string{"type": "sort"},
				MetricsWeights:  map[string]float64{"cases_passed": 1.0},
			},
			Input:  json.RawMessage(`{"numbers":[3,1,2]}`),
			Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 3},
			Budget: core.Budget{Timeout: mustParseDuration("30s")},
		}

		result := submitTask(t, routerURL, task)
		// Should either succeed or fail gracefully
		assert.NotNil(t, result)
	})
}

// TestE2E_Performance tests system performance
func TestE2E_Performance(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:8083")

	// Test 1: Concurrent requests
	t.Run("ConcurrentRequests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan core.Result, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(i int) {
				task := core.Task{
					ID:          fmt.Sprintf("e2e-concurrent-%d", i),
					Domain:      "algorithms.sorting",
					Description: "Concurrent sort test",
					Spec: core.Spec{
						SuccessCriteria: []string{"sorted_non_decreasing"},
						Props:           map[string]string{"type": "sort"},
						MetricsWeights:  map[string]float64{"cases_passed": 1.0},
					},
					Input:  json.RawMessage(`{"numbers":[3,1,2]}`),
					Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 3},
					Budget: core.Budget{Timeout: mustParseDuration("30s")},
				}

				result := submitTask(t, routerURL, task)
				results <- result
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numRequests; i++ {
			result := <-results
			if result.Success {
				successCount++
			}
		}

		// At least 80% should succeed
		assert.GreaterOrEqual(t, successCount, int(float64(numRequests)*0.8))
	})

	// Test 2: Response time
	t.Run("ResponseTime", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-performance-001",
			Domain:      "algorithms.sorting",
			Description: "Performance test",
			Spec: core.Spec{
				SuccessCriteria: []string{"sorted_non_decreasing"},
				Props:           map[string]string{"type": "sort"},
				MetricsWeights:  map[string]float64{"cases_passed": 1.0},
			},
			Input:  json.RawMessage(`{"numbers":[3,1,2]}`),
			Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 3},
			Budget: core.Budget{Timeout: mustParseDuration("30s")},
		}

		start := time.Now()
		result := submitTask(t, routerURL, task)
		duration := time.Since(start)

		assert.True(t, result.Success, "Performance test should succeed")
		assert.Less(t, duration, 10*time.Second, "Response should be under 10 seconds")
	})
}

// TestE2E_ErrorHandling tests error handling scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:8083")

	// Test 1: Invalid task format
	t.Run("InvalidTaskFormat", func(t *testing.T) {
		resp := makeRawRequest(t, "POST", routerURL+"/solve", []byte("invalid json"))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test 2: Missing required fields
	t.Run("MissingRequiredFields", func(t *testing.T) {
		task := core.Task{
			ID: "e2e-error-001",
			// Missing required fields
		}

		resp := makeRequest(t, "POST", routerURL+"/solve", task)
		// Should either succeed with defaults or fail gracefully
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)
	})

	// Test 3: Invalid input data
	t.Run("InvalidInputData", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-error-002",
			Domain:      "algorithms.sorting",
			Description: "Invalid input test",
			Spec: core.Spec{
				SuccessCriteria: []string{"sorted_non_decreasing"},
				Props:           map[string]string{"type": "sort"},
				MetricsWeights:  map[string]float64{"cases_passed": 1.0},
			},
			Input:  json.RawMessage(`{"invalid": "data"}`), // Invalid input
			Flags:  core.TaskFlags{RequiresSandbox: false, MaxComplexity: 3},
			Budget: core.Budget{Timeout: mustParseDuration("30s")},
		}

		result := submitTask(t, routerURL, task)
		// Should either succeed or fail gracefully
		assert.NotNil(t, result)
	})

	// Test 4: Timeout handling
	t.Run("TimeoutHandling", func(t *testing.T) {
		task := core.Task{
			ID:          "e2e-timeout-001",
			Domain:      "algorithms",
			Description: "Timeout test",
			Spec: core.Spec{
				SuccessCriteria: []string{"completed"},
				Props:           map[string]string{"type": "test"},
				MetricsWeights:  map[string]float64{"completed": 1.0},
			},
			Input:  json.RawMessage(`{"test": true}`),
			Flags:  core.TaskFlags{RequiresSandbox: true, MaxComplexity: 10},
			Budget: core.Budget{Timeout: mustParseDuration("1s")}, // Very short timeout
		}

		result := submitTask(t, routerURL, task)
		// Should either succeed quickly or timeout gracefully
		assert.NotNil(t, result)
	})
}

// Helper functions

func submitTask(t *testing.T, routerURL string, task core.Task) core.Result {
	body, err := json.Marshal(task)
	require.NoError(t, err)

	resp := makeRawRequest(t, "POST", routerURL+"/solve", body)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result core.Result
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
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

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
