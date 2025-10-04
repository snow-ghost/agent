package mock

import (
	"context"
	"os"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
)

// MockLLM implements core.LLMClient interface
type MockLLM struct {
	mode string
}

// NewMockLLM creates a new mock LLM client
func NewMockLLM() *MockLLM {
	mode := os.Getenv("LLM_MODE")
	if mode == "" {
		mode = "mock" // default to mock mode
	}
	return &MockLLM{mode: mode}
}

// Propose returns a pre-prepared WASM module for sorting with basic tests and criteria
func (m *MockLLM) Propose(ctx context.Context, task core.Task) (algo string, tests []core.TestCase, criteria []string, err error) {
	// Check if we should use mock mode
	if m.mode != "mock" {
		return "", nil, nil, nil
	}

	// Return pre-prepared WASM module for sorting
	algo = "wasm-sort-v1"

	// Basic test cases for sorting
	tests = []core.TestCase{
		{
			Name:   "sort_basic",
			Input:  []byte(`{"numbers": [3, 1, 4, 1, 5]}`),
			Oracle: []byte(`{"sorted": [1, 1, 3, 4, 5]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
		{
			Name:   "sort_empty",
			Input:  []byte(`{"numbers": []}`),
			Oracle: []byte(`{"sorted": []}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
		{
			Name:   "sort_single",
			Input:  []byte(`{"numbers": [42]}`),
			Oracle: []byte(`{"sorted": [42]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
		{
			Name:   "sort_negative",
			Input:  []byte(`{"numbers": [-1, 0, -5, 2]}`),
			Oracle: []byte(`{"sorted": [-5, -1, 0, 2]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
	}

	// Success criteria
	criteria = []string{
		"sorted_non_decreasing",
		"permutes",
		"handles_empty_input",
		"handles_single_element",
		"handles_negative_numbers",
	}

	return algo, tests, criteria, nil
}

// ProposeWithCaller returns a pre-prepared WASM module for sorting with basic tests and criteria
func (m *MockLLM) ProposeWithCaller(ctx context.Context, task core.Task, caller string) (algo string, tests []core.TestCase, criteria []string, err error) {
	// For mock, just call the regular Propose method
	// In a real implementation, caller would be used for cost tracking
	return m.Propose(ctx, task)
}

// GetWASMModule returns the pre-prepared WASM module for the proposed algorithm
func (m *MockLLM) GetWASMModule(algo string) ([]byte, error) {
	if algo == "wasm-sort-v1" {
		return wasm.GetTestModule(), nil
	}
	return nil, nil
}

// CreateHypothesis creates a hypothesis from the proposed algorithm
func (m *MockLLM) CreateHypothesis(algo string, tests []core.TestCase, criteria []string) core.Hypothesis {
	wasmBytes, _ := m.GetWASMModule(algo)

	return core.Hypothesis{
		ID:     algo,
		Source: "llm:mock",
		Lang:   "wasm",
		Bytes:  wasmBytes,
		Meta: map[string]string{
			"version": "v1",
			"mode":    m.mode,
		},
	}
}
