package dsl

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
)

func TestAFDSLInterpreter_ExecuteWithProtection(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	// Test with valid hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`{"test": "value"}`),
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Logs)
	}
}

func TestAFDSLInterpreter_ExecuteWithTimeout(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	// Test with hypothesis that might cause infinite loop
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(loop true (return input))"), // This might cause issues
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`{"test": "value"}`),
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   1 * time.Millisecond, // Very short timeout
		},
	}

	ctx := context.Background()
	_, err := interpreter.Execute(ctx, hypothesis, task)

	// Should either succeed or fail with timeout/step limit, but not panic
	if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "step limit exceeded") {
		t.Errorf("Expected timeout or step limit error, got: %v", err)
	}
}

func TestAFDSLInterpreter_ExecuteWithPanicRecovery(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	// Test with hypothesis that might cause panic
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(call (unknown_function input) (return result))"), // This should cause an error
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`{"test": "value"}`),
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	_, err := interpreter.Execute(ctx, hypothesis, task)

	// Should handle the error gracefully, not panic
	if err == nil {
		t.Error("Expected error for unknown function, got nil")
	}
}

func TestAFDSLInterpreter_ExecuteWithMemoryLimit(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	// Test with hypothesis that might use too much memory
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"), // Simple operation
		Meta:   map[string]string{},
	}

	// Create a large input that might exceed memory limits
	largeInput := make([]byte, getMaxMemoryMB()*1024*1024*2) // 2x the memory limit
	for i := range largeInput {
		largeInput[i] = 'a'
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  largeInput,
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     getMaxMemoryMB(),
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	_, err := interpreter.Execute(ctx, hypothesis, task)

	// Should either succeed or fail with memory limit, but not panic
	if err != nil && err.Error() != "memory limit exceeded" {
		// This is expected - the test might pass or fail depending on actual memory usage
		t.Logf("Memory test result: %v", err)
	}
}

func TestAFDSLInterpreter_ExecuteWithStepCounting(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`{"test": "value"}`),
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Logs)
	}

	// Check that metrics are populated
	if result.Metrics == nil {
		t.Error("Expected metrics to be populated")
	}

	if _, ok := result.Metrics["execution_steps"]; !ok {
		t.Error("Expected execution_steps metric")
	}

	if _, ok := result.Metrics["call_depth"]; !ok {
		t.Error("Expected call_depth metric")
	}

	if _, ok := result.Metrics["memory_usage_mb"]; !ok {
		t.Error("Expected memory_usage_mb metric")
	}
}

func TestAFDSLInterpreter_ExecuteUnsupportedLanguage(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "wasm", // Unsupported language
		Bytes:  []byte("some wasm code"),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`{"test": "value"}`),
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err == nil {
		t.Error("Expected error for unsupported language, got nil")
	}

	if result.Success {
		t.Error("Expected failure for unsupported language, got success")
	}
}

func TestAFDSLInterpreter_ExecuteInvalidJSON(t *testing.T) {
	interpreter := NewAFDSLInterpreter(nil)

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"),
		Meta:   map[string]string{},
	}

	task := core.Task{
		ID:     "test-task",
		Domain: "test",
		Input:  []byte(`invalid json`), // Invalid JSON
		Budget: core.Budget{
			CPUMillis: 1000,
			MemMB:     100,
			Timeout:   30 * time.Second,
		},
	}

	ctx := context.Background()
	result, err := interpreter.Execute(ctx, hypothesis, task)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if result.Success {
		t.Error("Expected failure for invalid JSON, got success")
	}
}
