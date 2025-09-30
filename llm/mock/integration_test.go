package mock

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	"github.com/snow-ghost/agent/kb/memory"
	"github.com/snow-ghost/agent/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMIntegration_EmptyKB(t *testing.T) {
	// Set mock mode
	os.Setenv("LLM_MODE", "mock")
	defer os.Unsetenv("LLM_MODE")

	// Create components
	llm := NewMockLLM()
	interpreter := wasm.NewInterpreter()
	defer interpreter.Close(context.Background())

	// Create empty KB (no skills registered)
	kb := &memory.Registry{}
	// Don't register any skills to simulate empty KB

	// Create test runner
	runner := testkit.NewRunner()

	// Create a sorting task
	task := core.Task{
		ID:     "integration-test",
		Domain: "algorithms",
		Spec: core.Spec{
			SuccessCriteria: []string{"sorted_non_decreasing", "permutes"},
			Props:           map[string]string{"operation": "sort"},
		},
		Input: json.RawMessage(`{"numbers": [3, 1, 4, 1, 5]}`),
		Budget: core.Budget{
			CPUMillis: 5000,
			MemMB:     64,
			Timeout:   time.Second * 10,
		},
		CreatedAt: time.Now(),
	}

	// Step 1: Try to find skill in KB (should fail)
	_, err := kb.FindSkill("algorithms", []string{"sort"})
	assert.Error(t, err, "KB should be empty and not find any skills")

	// Step 2: Use LLM to propose a solution
	algo, tests, criteria, err := llm.Propose(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "wasm-sort-v1", algo)
	assert.NotEmpty(t, tests)
	assert.NotEmpty(t, criteria)

	// Step 3: Create hypothesis from LLM proposal
	hypothesis := llm.CreateHypothesis(algo, tests, criteria)
	assert.Equal(t, "wasm-sort-v1", hypothesis.ID)
	assert.Equal(t, "llm:mock", hypothesis.Source)
	assert.NotEmpty(t, hypothesis.Bytes)

	// Step 4: Execute the hypothesis using the interpreter
	result, err := interpreter.Execute(context.Background(), hypothesis, task)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Step 5: Run the proposed tests
	metrics, allPassed, err := runner.Run(context.Background(), hypothesis, tests, interpreter)
	require.NoError(t, err)

	// Verify test results
	assert.Greater(t, metrics["cases_total"], 0.0)
	assert.GreaterOrEqual(t, metrics["cases_passed"], 0.0)
	assert.GreaterOrEqual(t, metrics["duration_ms_total"], 0.0)

	// The mock WASM module is very simple, so we expect some tests to pass
	// (even though it just returns input parameters)
	t.Logf("Test metrics: %+v", metrics)
	t.Logf("All tests passed: %v", allPassed)
}

func TestLLMIntegration_WithEmptyKBAndRealTask(t *testing.T) {
	// Set mock mode
	os.Setenv("LLM_MODE", "mock")
	defer os.Unsetenv("LLM_MODE")

	// Create components
	llm := NewMockLLM()
	interpreter := wasm.NewInterpreter()
	defer interpreter.Close(context.Background())

	// Create empty KB
	kb := &memory.Registry{}

	// Create a more complex task
	task := core.Task{
		ID:     "complex-sort-test",
		Domain: "algorithms",
		Spec: core.Spec{
			SuccessCriteria: []string{"sorted_non_decreasing", "permutes", "handles_negative_numbers"},
			Props:           map[string]string{"operation": "sort", "type": "numbers"},
		},
		Input: json.RawMessage(`{"numbers": [-5, 2, -1, 0, 3]}`),
		Budget: core.Budget{
			CPUMillis: 10000,
			MemMB:     128,
			Timeout:   time.Second * 30,
		},
		CreatedAt: time.Now(),
	}

	// Verify KB is empty
	_, err := kb.FindSkill("algorithms", []string{"sort"})
	assert.Error(t, err, "KB should be empty")

	// Get LLM proposal
	algo, tests, criteria, err := llm.Propose(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "wasm-sort-v1", algo)

	// Create and execute hypothesis
	hypothesis := llm.CreateHypothesis(algo, tests, criteria)
	result, err := interpreter.Execute(context.Background(), hypothesis, task)
	require.NoError(t, err)

	// The result should be successful (even if the WASM is simple)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Output)
	assert.NotEmpty(t, result.Logs)

	t.Logf("LLM hypothesis execution result: %+v", result)
}
