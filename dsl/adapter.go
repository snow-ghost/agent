package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/snow-ghost/agent/core"
)

// AFDSLInterpreter implements core.Interpreter for AF-DSL programs
type AFDSLInterpreter struct {
	Policy core.PolicyGuard
}

// NewAFDSLInterpreter creates a new AF-DSL interpreter
func NewAFDSLInterpreter(policy core.PolicyGuard) *AFDSLInterpreter {
	return &AFDSLInterpreter{
		Policy: policy,
	}
}

// Execute executes an AF-DSL hypothesis on a task
func (i *AFDSLInterpreter) Execute(ctx context.Context, h core.Hypothesis, t core.Task) (core.Result, error) {
	// Check if this is an AF-DSL hypothesis
	if h.Lang != "af-dsl" {
		return core.Result{Success: false}, fmt.Errorf("unsupported language: %s", h.Lang)
	}

	// Parse input JSON
	var inputValue interface{}
	if err := json.Unmarshal(t.Input, &inputValue); err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Wrap execution with policy guard if available
	var result core.Result
	var err error

	if i.Policy != nil {
		// Create a budget (simplified)
		budget := core.Budget{
			CPUMillis: 1000000,
			MemMB:     128,
			Timeout:   30 * time.Second,
		}

		err = i.Policy.Wrap(execCtx, budget, func(ctx context.Context) error {
			result, err = i.executeProgram(ctx, h, inputValue)
			return err
		})
	} else {
		result, err = i.executeProgram(execCtx, h, inputValue)
	}

	if err != nil {
		return core.Result{Success: false}, err
	}

	return result, nil
}

// executeProgram executes the AF-DSL program
func (i *AFDSLInterpreter) executeProgram(ctx context.Context, h core.Hypothesis, inputValue interface{}) (core.Result, error) {
	// Convert hypothesis bytes to string
	source := string(h.Bytes)

	// Execute the program
	outputValue, err := ExecuteProgram(ctx, source, inputValue)
	if err != nil {
		return core.Result{
			Success: false,
			Logs:    fmt.Sprintf("Execution error: %v", err),
		}, err
	}

	// Convert output to JSON
	outputJSON, err := json.Marshal(outputValue)
	if err != nil {
		return core.Result{
			Success: false,
			Logs:    fmt.Sprintf("Failed to marshal output: %v", err),
		}, err
	}

	// Create result
	result := core.Result{
		Success: true,
		Output:  outputJSON,
		Logs:    fmt.Sprintf("AF-DSL program executed successfully"),
		Metrics: map[string]float64{
			"execution_time_ms": 0, // Would be measured in real implementation
			"memory_usage_kb":   0, // Would be measured in real implementation
		},
	}

	return result, nil
}

// Ensure AFDSLInterpreter implements core.Interpreter
var _ core.Interpreter = (*AFDSLInterpreter)(nil)
