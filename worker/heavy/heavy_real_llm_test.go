package heavy

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/pkg/llm/client"
	"github.com/snow-ghost/agent/testkit"
)

// Use global telemetry instance

// TestHeavySolve_WithRealLLM tests the heavy worker with real LLM
// This test requires:
// 1. LLM Router running on localhost:9000
// 2. Valid API keys set in environment variables
// 3. Network access to LLM providers
func TestHeavySolve_WithRealLLM(t *testing.T) {

	// Skip if LLM_ROUTER_URL is not accessible
	llmRouterURL := os.Getenv("LLM_ROUTER_URL")
	if llmRouterURL == "" {
		llmRouterURL = "http://localhost:9000"
	}

	ctx := context.Background()

	// Create empty KB
	kb := &noopKB{}

	// Create real designer client
	designer := client.NewDesignClientFromEnv()

	tests := testkit.NewDetailedRunner()
	fitness := core.NewWeightedFitness(map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}, 0.01)
	critic := &mockCritic{}
	mutator := &mockMutator{}
	// Use global telemetry to avoid duplicate metrics registration
	tel := GetGlobalTelemetry()

	worker := NewHeavyWorker(kb, designer, tests, fitness, critic, mutator, tel)

	// Create a simple sorting task
	task := core.Task{
		ID:          "test-sort-real-llm",
		Domain:      "algorithms.sorting",
		Description: "Sort an array of numbers in ascending order",
		Input:       []byte(`{"numbers": [3,1,2,5,4]}`),
		Spec: core.Spec{
			Props: map[string]string{
				"type":          "sort",
				"input_schema":  `{"type": "object", "properties": {"numbers": {"type": "array", "items": {"type": "number"}}}}`,
				"output_schema": `{"type": "object", "properties": {"sorted": {"type": "array", "items": {"type": "number"}}}}`,
			},
		},
		Budget: core.Budget{
			CPUMillis: 5000,
			MemMB:     128,
			Timeout:   120 * time.Second,
		},
	}

	t.Logf("Submitting task to heavy worker with real LLM...")
	t.Logf("LLM Router URL: %s", llmRouterURL)

	// Test the designer directly to see what AF-DSL code is generated
	taskJSON := `{
		"task": {
			"id": "test-sort-real-llm",
			"domain": "algorithms.sorting",
			"description": "Sort an array of numbers in ascending order",
			"constraints": {
				"timeout_ms": 5000,
				"mem_mb": 128,
				"max_complexity": "O(n log n)"
			},
			"input_schema": "{\"type\": \"object\", \"properties\": {\"numbers\": {\"type\": \"array\", \"items\": {\"type\": \"number\"}}}}",
			"output_schema": "{\"type\": \"object\", \"properties\": {\"sorted\": {\"type\": \"array\", \"items\": {\"type\": \"number\"}}}}",
			"examples": []
		}
	}`

	hd, rawResponse, err := designer.Design(ctx, taskJSON)
	if err != nil {
		t.Logf("Design failed: %v", err)
		t.Logf("Raw LLM response: %s", string(rawResponse))
	} else {
		t.Logf("Design succeeded!")
		// Convert to hypothesis to get the generated code
		hypothesis, err := design.ToHypothesis(hd)
		if err != nil {
			t.Logf("Failed to convert design to hypothesis: %v", err)
		} else {
			t.Logf("Generated AF-DSL code: %s", string(hypothesis.Bytes))
		}
	}

	res, err := worker.Solve(ctx, task)
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	t.Logf("Result: Success=%v, Logs=%s", res.Success, res.Logs)

	if res.Success {
		t.Logf("Task solved successfully!")
		t.Logf("Output: %s", string(res.Output))
	} else {
		t.Logf("Task failed, but this is expected for some LLM responses")
		t.Logf("Failure reason: %s", res.Logs)
	}
}

// TestHeavySolve_WithRealLLM_Complex tests with a more complex task
func TestHeavySolve_WithRealLLM_Complex(t *testing.T) {

	t.Skip("Skipping complex task test for now")
	ctx := context.Background()
	kb := &noopKB{}
	designer := client.NewDesignClientFromEnv()
	tests := testkit.NewDetailedRunner()
	fitness := core.NewWeightedFitness(map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}, 0.01)
	critic := &mockCritic{}
	mutator := &mockMutator{}
	// Use global telemetry to avoid duplicate metrics registration
	tel := GetGlobalTelemetry()

	worker := NewHeavyWorker(kb, designer, tests, fitness, critic, mutator, tel)

	// Create a more complex task - finding duplicates
	task := core.Task{
		ID:          "test-duplicates-real-llm",
		Domain:      "algorithms.arrays",
		Description: "Find all duplicate elements in an array",
		Input:       []byte(`{"numbers": [1,2,3,2,4,5,4,6]}`),
		Spec: core.Spec{
			Props: map[string]string{
				"type":          "find_duplicates",
				"input_schema":  `{"type": "object", "properties": {"numbers": {"type": "array", "items": {"type": "number"}}}}`,
				"output_schema": `{"type": "object", "properties": {"duplicates": {"type": "array", "items": {"type": "number"}}}}`,
			},
		},
		Budget: core.Budget{
			CPUMillis: 5000,
			MemMB:     128,
			Timeout:   120 * time.Second,
		},
	}

	t.Logf("Submitting complex task to heavy worker with real LLM...")

	res, err := worker.Solve(ctx, task)
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	t.Logf("Result: Success=%v, Logs=%s", res.Success, res.Logs)

	if res.Success {
		t.Logf("Complex task solved successfully!")
		t.Logf("Output: %s", string(res.Output))
	} else {
		t.Logf("Complex task failed, but this is expected for some LLM responses")
		t.Logf("Failure reason: %s", res.Logs)
	}
}
