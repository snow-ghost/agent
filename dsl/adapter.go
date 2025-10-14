package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/snow-ghost/agent/core"
)

// AFDSLInterpreter implements core.Interpreter for AF-DSL programs
type AFDSLInterpreter struct {
	Policy core.PolicyGuard
}

// Runtime limits for safety
const (
	DefaultMaxExecutionSteps = 100000
	DefaultMaxCallDepth      = 128
	DefaultMaxMemoryMB       = 64
)

func getMaxSteps() int {
	if v := os.Getenv("DSL_MAX_STEPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxExecutionSteps
}

func getMaxDepth() int {
	if v := os.Getenv("DSL_MAX_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxCallDepth
}

func getMaxMemoryMB() int {
	if v := os.Getenv("DSL_MAX_MEMORY_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxMemoryMB
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
			MemMB:     getMaxMemoryMB(),
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

// executeProgram executes the AF-DSL program with runtime protection
func (i *AFDSLInterpreter) executeProgram(ctx context.Context, h core.Hypothesis, inputValue interface{}) (core.Result, error) {
	// Execute with panic recovery and runtime limits
	return i.executeWithProtection(ctx, h, inputValue)
}

// executeWithProtection executes the program with safety measures
func (i *AFDSLInterpreter) executeWithProtection(ctx context.Context, h core.Hypothesis, inputValue interface{}) (core.Result, error) {
	// Set up runtime monitoring
	var result core.Result
	var err error

	// Channel to receive result from goroutine
	resultChan := make(chan struct {
		result core.Result
		err    error
	}, 1)

	// Start execution in goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Convert panic to error
				err = fmt.Errorf("runtime panic: %v", r)
				resultChan <- struct {
					result core.Result
					err    error
				}{core.Result{Success: false}, err}
			}
		}()

		// Execute the actual program
		result, err = i.executeAFDSL(ctx, h, inputValue)
		resultChan <- struct {
			result core.Result
			err    error
		}{result, err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-ctx.Done():
		return core.Result{Success: false}, fmt.Errorf("execution timeout: %w", ctx.Err())
	}
}

// executeAFDSL executes the actual AF-DSL program with step counting
func (i *AFDSLInterpreter) executeAFDSL(ctx context.Context, h core.Hypothesis, inputValue interface{}) (core.Result, error) {
	source := string(h.Bytes)

	// Set up step counting and memory monitoring
	stepCount := 0
	callDepth := 0

	// Check memory usage before execution
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startMem := m.Alloc

	// Create execution context with step limits
	execCtx := context.WithValue(ctx, "stepCount", &stepCount)
	execCtx = context.WithValue(execCtx, "callDepth", &callDepth)
	execCtx = context.WithValue(execCtx, "maxSteps", getMaxSteps())
	execCtx = context.WithValue(execCtx, "maxDepth", getMaxDepth())

	// Execute the program
	outputValue, err := ExecuteProgram(execCtx, source, inputValue)
	if err != nil {
		return core.Result{
			Success: false,
			Logs:    fmt.Sprintf("Execution error: %v", err),
		}, err
	}

	// Check final memory usage
	runtime.ReadMemStats(&m)
	endMem := m.Alloc
	memoryUsed := endMem - startMem
	memoryUsedMB := float64(memoryUsed) / (1024 * 1024)

	// Check memory limit
	if memoryUsedMB > float64(getMaxMemoryMB()) {
		return core.Result{
			Success: false,
			Logs:    fmt.Sprintf("Memory limit exceeded: %.2f MB (max %d MB)", memoryUsedMB, getMaxMemoryMB()),
		}, fmt.Errorf("memory limit exceeded: %.2f MB", memoryUsedMB)
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
			"execution_steps": float64(stepCount),
			"call_depth":      float64(callDepth),
			"memory_usage_mb": memoryUsedMB,
		},
	}

	return result, nil
}

// Ensure AFDSLInterpreter implements core.Interpreter
var _ core.Interpreter = (*AFDSLInterpreter)(nil)
