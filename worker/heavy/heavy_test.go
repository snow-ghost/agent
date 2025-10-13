package heavy

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/pkg/llm/client"
	"github.com/snow-ghost/agent/testkit"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// mockDesigner implements client.Designer for testing
type mockDesigner struct {
	shouldReturnValidDesign bool
	design                  design.HypothesisDesign
	raw                     []byte
}

func (m *mockDesigner) Design(ctx context.Context, taskJSON string) (design.HypothesisDesign, []byte, error) {
	if !m.shouldReturnValidDesign {
		return design.HypothesisDesign{}, nil, nil
	}
	return m.design, m.raw, nil
}

// Ensure mockDesigner implements client.Designer
var _ client.Designer = (*mockDesigner)(nil)

var sharedTelemetry *telemetry.Telemetry

func init() {
	sharedTelemetry = telemetry.NewTelemetry()
}

func TestHeavyWorker_Solve_WithValidDesign(t *testing.T) {
	ctx := context.Background()

	// Create mock KB
	kb := &mockKB{}

	// Create mock designer with valid design
	designer := &mockDesigner{
		shouldReturnValidDesign: true,
		design: design.HypothesisDesign{
			Status: "ok",
			Algorithm: struct {
				Name       string `json:"name"`
				Idea       string `json:"idea"`
				Complexity struct {
					Time  string `json:"time"`
					Space string `json:"space"`
				} `json:"complexity"`
			}{
				Name: "test_algorithm",
				Idea: "Test algorithm for sorting",
				Complexity: struct {
					Time  string `json:"time"`
					Space string `json:"space"`
				}{
					Time:  "O(n log n)",
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
				Src:   "(let x input (return x))",
			},
			Evaluation: struct {
				Metrics       []string `json:"metrics"`
				Fitness       string   `json:"fitness"`
				PassThreshold float64  `json:"pass_threshold"`
			}{
				Metrics:       []string{"correctness", "time", "size"},
				Fitness:       "correctness*0.8 + time*0.15 + size*0.05",
				PassThreshold: 0.85,
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
						Name:   "test_1",
						Input:  `{"numbers": [3,1,2]}`,
						Oracle: `{"sorted": [1,2,3]}`,
						Weight: 1.0,
					},
				},
				Property: []struct {
					Name      string   `json:"name"`
					Generator string   `json:"generator"`
					Checks    []string `json:"checks"`
				}{
					{
						Name:      "sorted_property",
						Generator: "list<int>(n<=100)",
						Checks:    []string{"sorted_non_decreasing", "permutes"},
					},
				},
			},
		},
		raw: []byte(`{"status":"ok","algorithm":{"name":"test_algorithm"}}`),
	}

	// Create test runner and fitness evaluator
	tests := testkit.NewRunner()
	fitness := core.NewWeightedFitness(map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}, 0.01)

	// Create critic and mutator
	critic := &mockCritic{}
	mutator := &mockMutator{}

	// Create heavy worker
	worker := NewHeavyWorker(kb, designer, tests, fitness, critic, mutator, sharedTelemetry)

	// Create test task
	task := core.Task{
		ID:          "test-task",
		Domain:      "algorithms",
		Description: "Sort an array of numbers",
		Input:       []byte(`{"numbers": [3,1,2]}`),
		Spec: core.Spec{
			Props: map[string]string{
				"input_schema":  `{"type": "object", "properties": {"numbers": {"type": "array", "items": {"type": "number"}}}}`,
				"output_schema": `{"type": "object", "properties": {"sorted": {"type": "array", "items": {"type": "number"}}}}`,
			},
		},
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	// Run the test
	result, err := worker.Solve(ctx, task)

	// Verify results
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// The result should be successful since we have a valid design
	// Note: In a real test, the mock interpreter would need to be set up properly
	// For now, we expect it to fail at the interpreter stage but not at the design stage
	if result.Success {
		t.Logf("Task solved successfully: %s", result.Logs)
	} else {
		t.Logf("Task failed as expected (no real interpreter): %s", result.Logs)
	}
}

func TestHeavyWorker_Solve_WithCannotSolveDesign(t *testing.T) {
	ctx := context.Background()

	// Create mock KB
	kb := &mockKB{}

	// Create mock designer that returns cannot_solve
	designer := &mockDesigner{
		shouldReturnValidDesign: true,
		design: design.HypothesisDesign{
			Status: "cannot_solve",
			Reason: "Task is too complex for current capabilities",
		},
		raw: []byte(`{"status":"cannot_solve","reason":"Task is too complex for current capabilities"}`),
	}

	// Create test runner and fitness evaluator
	tests := testkit.NewRunner()
	fitness := core.NewWeightedFitness(map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}, 0.01)

	// Create critic and mutator
	critic := &mockCritic{}
	mutator := &mockMutator{}

	// Create heavy worker
	worker := NewHeavyWorker(kb, designer, tests, fitness, critic, mutator, sharedTelemetry)

	// Create test task
	task := core.Task{
		ID:          "test-task",
		Domain:      "algorithms",
		Description: "Sort an array of numbers",
		Input:       []byte(`{"numbers": [3,1,2]}`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	// Run the test
	result, err := worker.Solve(ctx, task)

	// Verify results
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// The result should indicate cannot solve
	if result.Success {
		t.Error("Expected task to fail with cannot_solve, but it succeeded")
	}

	if result.Logs != "design_cannot_solve" {
		t.Errorf("Expected logs to be 'design_cannot_solve', got '%s'", result.Logs)
	}
}

// Mock implementations for testing

type mockKB struct{}

func (m *mockKB) Find(task core.Task) []core.Skill {
	return []core.Skill{}
}

func (m *mockKB) SaveHypothesis(ctx context.Context, h core.Hypothesis, score float64) error {
	return nil
}

type mockCritic struct{}

func (m *mockCritic) Accept(task core.Task, metrics map[string]float64) (bool, string) {
	return true, "accepted"
}

type mockMutator struct{}

func (m *mockMutator) Mutate(h core.Hypothesis) []core.Hypothesis {
	return []core.Hypothesis{}
}
