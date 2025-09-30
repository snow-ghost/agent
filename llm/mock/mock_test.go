package mock

import (
	"context"
	"os"
	"testing"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLLM_Propose(t *testing.T) {
	// Set mock mode
	os.Setenv("LLM_MODE", "mock")
	defer os.Unsetenv("LLM_MODE")

	llm := NewMockLLM()
	ctx := context.Background()

	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Spec: core.Spec{
			SuccessCriteria: []string{"sort"},
		},
	}

	algo, tests, criteria, err := llm.Propose(ctx, task)
	require.NoError(t, err)
	assert.Equal(t, "wasm-sort-v1", algo)
	assert.Len(t, tests, 4)
	assert.Len(t, criteria, 5)
	assert.Contains(t, criteria, "sorted_non_decreasing")
	assert.Contains(t, criteria, "permutes")
}

func TestMockLLM_GetWASMModule(t *testing.T) {
	llm := NewMockLLM()

	// Test valid algorithm
	wasmBytes, err := llm.GetWASMModule("wasm-sort-v1")
	require.NoError(t, err)
	assert.NotEmpty(t, wasmBytes)

	// Test invalid algorithm
	wasmBytes, err = llm.GetWASMModule("invalid")
	require.NoError(t, err)
	assert.Nil(t, wasmBytes)
}

func TestMockLLM_CreateHypothesis(t *testing.T) {
	llm := NewMockLLM()
	tests := []core.TestCase{
		{Name: "test1", Input: []byte(`{"numbers": [1,2,3]}`)},
	}
	criteria := []string{"sorted_non_decreasing"}

	hypothesis := llm.CreateHypothesis("wasm-sort-v1", tests, criteria)

	assert.Equal(t, "wasm-sort-v1", hypothesis.ID)
	assert.Equal(t, "llm:mock", hypothesis.Source)
	assert.Equal(t, "wasm", hypothesis.Lang)
	assert.NotEmpty(t, hypothesis.Bytes)
	assert.Equal(t, "v1", hypothesis.Meta["version"])
	assert.Equal(t, "mock", hypothesis.Meta["mode"])
}

func TestMockLLM_NonMockMode(t *testing.T) {
	// Set non-mock mode
	os.Setenv("LLM_MODE", "real")
	defer os.Unsetenv("LLM_MODE")

	llm := NewMockLLM()
	ctx := context.Background()

	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
	}

	algo, tests, criteria, err := llm.Propose(ctx, task)
	require.NoError(t, err)
	assert.Empty(t, algo)
	assert.Nil(t, tests)
	assert.Nil(t, criteria)
}
