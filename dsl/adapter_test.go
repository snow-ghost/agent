package dsl

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
)

func TestAFDSLInterpreter_Execute(t *testing.T) {
	// Create a simple AF-DSL program
	program := "(let result (call sorted? input) (return result))"

	// Create hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	// Create task
	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[1, 2, 3, 4, 5]`),
		Spec: core.Spec{
			Props: map[string]string{
				"type": "sort",
			},
		},
	}

	// Create interpreter
	interpreter := NewAFDSLInterpreter(nil)

	// Execute
	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err != nil {
		t.Errorf("Execute failed: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Logs)
		return
	}

	// Parse result
	var outputValue interface{}
	if err := json.Unmarshal(result.Output, &outputValue); err != nil {
		t.Errorf("Failed to parse output: %v", err)
		return
	}

	// Should be true (sorted array)
	if outputValue != true {
		t.Errorf("Expected true, got %v", outputValue)
	}
}

func TestAFDSLInterpreter_Execute_UnsupportedLanguage(t *testing.T) {
	// Create hypothesis with unsupported language
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "python",
		Bytes:  []byte("print('hello')"),
		Meta:   map[string]string{},
	}

	// Create task
	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[1, 2, 3]`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	// Create interpreter
	interpreter := NewAFDSLInterpreter(nil)

	// Execute
	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err == nil {
		t.Error("Expected error for unsupported language, got nil")
		return
	}

	if result.Success {
		t.Error("Expected failure for unsupported language, got success")
	}
}

func TestAFDSLInterpreter_Execute_InvalidJSON(t *testing.T) {
	// Create a simple AF-DSL program
	program := "(return input)"

	// Create hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	// Create task with invalid JSON
	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`invalid json`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	// Create interpreter
	interpreter := NewAFDSLInterpreter(nil)

	// Execute
	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
		return
	}

	if result.Success {
		t.Error("Expected failure for invalid JSON, got success")
	}
}

func TestAFDSLInterpreter_Execute_Timeout(t *testing.T) {
	// Create a program that runs forever
	program := "(loop true (let x 1))"

	// Create hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	// Create task
	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[]`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	// Create interpreter
	interpreter := NewAFDSLInterpreter(nil)

	// Execute with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err == nil {
		t.Error("Expected timeout error, got nil")
		return
	}

	if result.Success {
		t.Error("Expected failure due to timeout, got success")
	}
}

func TestAFDSLInterpreter_Execute_SortingProgram(t *testing.T) {
	// Create a simple sorting program using merge sort
	program := `
		(let merge-sort
			(if (call len input)
				(if (= (call len input) 1)
					input
					(let split-result (call split input)
						(let left (call merge-sort (get split-result "left")))
							(let right (call merge-sort (get split-result "right")))
								(call merge left right))))))
		(call merge-sort input)
	`

	// This is a simplified test - the actual program would need more functions
	// For now, just test that it parses and fails gracefully
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[3, 1, 4, 1, 5]`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	interpreter := NewAFDSLInterpreter(nil)
	ctx := context.Background()

	result, err := interpreter.Execute(ctx, hypothesis, task)

	// We expect this to fail because the program is incomplete
	// but it should fail gracefully, not crash
	if err == nil {
		t.Error("Expected error for incomplete program, got nil")
	}

	// The result should indicate failure
	if result.Success {
		t.Error("Expected failure for incomplete program, got success")
	}
}

func TestAFDSLInterpreter_Execute_SimpleSort(t *testing.T) {
	// Test a very simple sorting program that just checks if input is sorted
	program := "(call sorted? input)"

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	// Test with sorted input
	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[1, 2, 3, 4, 5]`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	interpreter := NewAFDSLInterpreter(nil)
	ctx := context.Background()

	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err != nil {
		t.Errorf("Execute failed: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Logs)
		return
	}

	// Parse result
	var outputValue interface{}
	if err := json.Unmarshal(result.Output, &outputValue); err != nil {
		t.Errorf("Failed to parse output: %v", err)
		return
	}

	// Should be true (sorted array)
	if outputValue != true {
		t.Errorf("Expected true, got %v", outputValue)
	}
}

// Mock policy guard for testing
type mockPolicyGuard struct{}

func (m *mockPolicyGuard) Wrap(ctx context.Context, budget core.Budget, run func(ctx context.Context) error) error {
	return run(ctx)
}

func (m *mockPolicyGuard) AllowTool(name string) bool {
	return true
}

func TestAFDSLInterpreter_Execute_WithPolicyGuard(t *testing.T) {
	// Test with a mock policy guard
	program := "(return 42)"

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte(program),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "algorithms",
		Input:  []byte(`[]`),
		Spec: core.Spec{
			Props: map[string]string{},
		},
	}

	// Create interpreter with policy guard
	policy := &mockPolicyGuard{}
	interpreter := NewAFDSLInterpreter(policy)

	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err != nil {
		t.Errorf("Execute failed: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Logs)
		return
	}

	// Parse result
	var outputValue interface{}
	if err := json.Unmarshal(result.Output, &outputValue); err != nil {
		t.Errorf("Failed to parse output: %v", err)
		return
	}

	// Should be 42
	if outputValue != 42.0 {
		t.Errorf("Expected 42, got %v", outputValue)
	}
}
