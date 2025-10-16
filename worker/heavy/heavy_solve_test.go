package heavy

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/dsl"
	"github.com/snow-ghost/agent/pkg/llm/client"
	"github.com/snow-ghost/agent/testkit"
)

// Use global telemetry instance

// simple in-memory KB based on artifact FS with a temp dir
type noopKB struct{}

func (n *noopKB) Find(task core.Task) []core.Skill { return nil }
func (n *noopKB) SaveHypothesis(ctx context.Context, h core.Hypothesis, score float64) error {
	return nil
}

func TestHeavySolve_WithMockDesigner(t *testing.T) {
	ctx := context.Background()

	kb := &noopKB{}
	designer := &client.MockDesigner{FixturePath: "../../testdata/design_sort.json"}
	tests := testkit.NewRunner()
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

	task := core.Task{
		ID:          "test-sort",
		Domain:      "algorithms.sorting",
		Description: "Sort numbers",
		Input:       []byte(`{"numbers": [3,1,2]}`),
		Spec:        core.Spec{Props: map[string]string{}},
		Budget:      core.Budget{CPUMillis: 1000, MemMB: 64, Timeout: 5 * time.Second},
	}

	res, err := worker.Solve(ctx, task)
	if err != nil {
		t.Fatalf("Solve returned error: %v", err)
	}

	// We expect the pipeline to run tests and compute correctness
	if res.Success {
		// Great, solved and executed
	}

	// Validate metrics via a separate run through the test runner,
	// since HeavyWorker returns core.Result; correctness is part of metrics map
	// We'll rebuild hypothesis and run TestRunner directly
	hd, _, err := designer.Design(ctx, "{}")
	if err != nil {
		t.Fatalf("design: %v", err)
	}
	h, err := design.ToHypothesis(hd)
	if err != nil {
		t.Fatalf("to hypothesis: %v", err)
	}
	cases := design.ToTestCases(hd)

	// choose AF-DSL interpreter
	interp := dsl.NewAFDSLInterpreter(nil)
	metrics, pass, err := tests.Run(ctx, h, cases, interp)
	if err != nil {
		t.Fatalf("tests run: %v", err)
	}
	if !pass {
		t.Fatalf("expected tests to pass")
	}
	if c, ok := metrics["correctness"]; !ok || c < 0.9999 {
		t.Fatalf("expected correctness=1, got %v", c)
	}
}
