package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetrics_WorkerMetrics tests worker metrics after task execution
func TestMetrics_WorkerMetrics(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:9006")
	metricsURL := getenv("WORKER_METRICS_URL", "http://localhost:9005")

	// Submit a task to generate metrics
	task := core.Task{
		ID:          "metrics-test-001",
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

	// Submit task
	result := submitTask(t, routerURL, task)
	require.True(t, result.Success, "Task should succeed to generate metrics")

	// Wait a bit for metrics to be recorded
	time.Sleep(2 * time.Second)

	// Check worker metrics
	t.Run("WorkerMetrics", func(t *testing.T) {
		metrics := fetchMetrics(t, metricsURL+"/metrics")

		// Check worker task metrics
		assertMetricExists(t, metrics, `worker_task_received_total{worker_type="light"}`, "worker_task_received_total should exist")
		assertMetricExists(t, metrics, `worker_task_completed_total{worker_type="light",status="ok"}`, "worker_task_completed_total should exist")
		assertMetricExists(t, metrics, `worker_task_duration_seconds{worker_type="light"}`, "worker_task_duration_seconds should exist")

		// Check solve stage metrics
		assertMetricExists(t, metrics, `worker_solve_stage_seconds{stage="kb"}`, "solve_stage_seconds should exist")

		// Check KB metrics
		assertMetricExists(t, metrics, `worker_kb_artifacts_loaded`, "kb_artifacts_loaded should exist")
		assertMetricExists(t, metrics, `worker_kb_save_artifact_total`, "kb_save_artifact_total should exist")

		// Check test metrics
		assertMetricExists(t, metrics, `worker_tests_run_total{result="pass"}`, "tests_run_total should exist")
		assertMetricExists(t, metrics, `worker_tests_duration_seconds`, "tests_duration_seconds should exist")

		// Check sandbox metrics
		assertMetricExists(t, metrics, `worker_sandbox_exec_total{result="ok"}`, "sandbox_exec_total should exist")
		assertMetricExists(t, metrics, `worker_sandbox_exec_seconds`, "sandbox_exec_seconds should exist")
	})
}

// TestMetrics_LLMRouterMetrics tests LLM router metrics after requests
func TestMetrics_LLMRouterMetrics(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	llmRouterURL := getenv("LLMROUTER_URL", "http://localhost:9000")
	metricsURL := getenv("LLMROUTER_METRICS_URL", "http://localhost:9001")

	// Make a chat request to generate metrics
	chatReq := map[string]interface{}{
		"model": "mock",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello, world!"},
		},
		"temperature": 0.7,
		"max_tokens":  100,
	}

	// Submit chat request
	resp := makeRequest(t, "POST", llmRouterURL+"/v1/chat", chatReq)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait a bit for metrics to be recorded
	time.Sleep(2 * time.Second)

	// Check LLM router metrics
	t.Run("LLMRouterMetrics", func(t *testing.T) {
		metrics := fetchMetrics(t, metricsURL+"/metrics")

		// Check LLM request metrics
		assertMetricExists(t, metrics, `llm_requests_total{provider="mock",model="mock",status="ok",cache="miss"}`, "llm_requests_total should exist")
		assertMetricExists(t, metrics, `llm_request_duration_seconds{provider="mock",model="mock"}`, "llm_request_duration_seconds should exist")
		assertMetricExists(t, metrics, `llm_tokens_input_total{provider="mock",model="mock"}`, "llm_tokens_input_total should exist")
		assertMetricExists(t, metrics, `llm_tokens_output_total{provider="mock",model="mock"}`, "llm_tokens_output_total should exist")
		assertMetricExists(t, metrics, `llm_cost_total{provider="mock",model="mock",currency="USD"}`, "llm_cost_total should exist")

		// Check HTTP metrics
		assertMetricExists(t, metrics, `http_requests_total{path="/v1/chat",method="POST",code="2xx"}`, "http_requests_total should exist")
		assertMetricExists(t, metrics, `http_request_duration_seconds{path="/v1/chat",method="POST"}`, "http_request_duration_seconds should exist")
	})
}

// TestMetrics_MonotonicGrowth tests that counters grow monotonically
func TestMetrics_MonotonicGrowth(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:9006")
	workerMetricsURL := getenv("WORKER_METRICS_URL", "http://localhost:9005")
	llmRouterURL := getenv("LLMROUTER_URL", "http://localhost:9000")
	llmMetricsURL := getenv("LLMROUTER_METRICS_URL", "http://localhost:9001")

	// Get initial metrics
	initialWorkerMetrics := fetchMetrics(t, workerMetricsURL+"/metrics")
	initialLLMMetrics := fetchMetrics(t, llmMetricsURL+"/metrics")

	// Submit multiple tasks
	for i := 0; i < 3; i++ {
		task := core.Task{
			ID:          fmt.Sprintf("metrics-growth-test-%d", i),
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

		submitTask(t, routerURL, task)
	}

	// Make multiple LLM requests
	for i := 0; i < 2; i++ {
		chatReq := map[string]interface{}{
			"model": "mock",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Test message %d", i)},
			},
			"temperature": 0.7,
			"max_tokens":  100,
		}

		makeRequest(t, "POST", llmRouterURL+"/v1/chat", chatReq)
	}

	// Wait for metrics to be recorded
	time.Sleep(3 * time.Second)

	// Get final metrics
	finalWorkerMetrics := fetchMetrics(t, workerMetricsURL+"/metrics")
	finalLLMMetrics := fetchMetrics(t, llmMetricsURL+"/metrics")

	// Check monotonic growth
	t.Run("WorkerCounterGrowth", func(t *testing.T) {
		initial := getMetricValue(t, initialWorkerMetrics, `worker_task_received_total{worker_type="light"}`)
		final := getMetricValue(t, finalWorkerMetrics, `worker_task_received_total{worker_type="light"}`)

		assert.Greater(t, final, initial, "worker_task_received_total should grow")
		assert.GreaterOrEqual(t, final-initial, float64(3), "Should have at least 3 new tasks")
	})

	t.Run("LLMCounterGrowth", func(t *testing.T) {
		initial := getMetricValue(t, initialLLMMetrics, `llm_requests_total{provider="mock",model="mock",status="ok",cache="miss"}`)
		final := getMetricValue(t, finalLLMMetrics, `llm_requests_total{provider="mock",model="mock",status="ok",cache="miss"}`)

		assert.Greater(t, final, initial, "llm_requests_total should grow")
		assert.GreaterOrEqual(t, final-initial, float64(2), "Should have at least 2 new requests")
	})
}

// TestMetrics_HealthEndpoints tests that health endpoints work
func TestMetrics_HealthEndpoints(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	workerMetricsURL := getenv("WORKER_METRICS_URL", "http://localhost:9005")
	llmMetricsURL := getenv("LLMROUTER_METRICS_URL", "http://localhost:9001")

	t.Run("WorkerHealth", func(t *testing.T) {
		resp := makeRequest(t, "GET", workerMetricsURL+"/healthz", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "ok", health["status"])
	})

	t.Run("LLMRouterHealth", func(t *testing.T) {
		resp := makeRequest(t, "GET", llmMetricsURL+"/healthz", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "ok", health["status"])
	})
}

// Helper functions

func fetchMetrics(t *testing.T, url string) string {
	resp := makeRequest(t, "GET", url, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var buf bytes.Buffer
	_, err := buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return buf.String()
}

func assertMetricExists(t *testing.T, metrics, pattern, message string) {
	matched, err := regexp.MatchString(pattern, metrics)
	require.NoError(t, err)
	assert.True(t, matched, "%s: pattern %s not found in metrics", message, pattern)
}

func getMetricValue(t *testing.T, metrics, pattern string) float64 {
	re := regexp.MustCompile(pattern + `\s+(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(metrics)

	if len(matches) < 2 {
		return 0.0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	require.NoError(t, err)
	return value
}

// Helper functions are defined in e2e_test.go
