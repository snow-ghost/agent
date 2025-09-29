package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Interpreter implements core.Interpreter interface using wazero WASM runtime
type Interpreter struct {
	runtime wazero.Runtime
	cache   map[string]wazero.CompiledModule
}

// NewInterpreter creates a new WASM interpreter with default configuration
func NewInterpreter() *Interpreter {
	// Create runtime with memory and timeout limits
	config := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(64). // 64 pages = 4MB
		WithCloseOnContextDone(true)

	runtime := wazero.NewRuntimeWithConfig(context.Background(), config)

	// Enable WASI for basic functionality
	wasi_snapshot_preview1.MustInstantiate(context.Background(), runtime)

	return &Interpreter{
		runtime: runtime,
		cache:   make(map[string]wazero.CompiledModule),
	}
}

// Execute runs a WASM module with the given hypothesis and task
func (i *Interpreter) Execute(ctx context.Context, h core.Hypothesis, task core.Task) (core.Result, error) {
	// Create context with timeout from task budget
	timeout := time.Duration(task.Budget.CPUMillis) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get or compile the module
	module, err := i.getOrCompileModule(execCtx, h)
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to compile module: %w", err)
	}

	// Create module instance
	instance, err := i.runtime.InstantiateModule(execCtx, module, wazero.NewModuleConfig().
		WithName(h.ID))
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer instance.Close(execCtx)

	// Prepare input data
	inputData := map[string]any{
		"input": json.RawMessage(task.Input),
		"spec":  task.Spec,
	}
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Call the solve function
	solveFunc := instance.ExportedFunction("solve")
	if solveFunc == nil {
		return core.Result{}, fmt.Errorf("module does not export 'solve' function")
	}

	// Allocate memory for input
	inputPtr, inputSize, err := i.allocateString(instance, execCtx, string(inputJSON))
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to allocate input: %w", err)
	}

	// Call solve function
	results, err := solveFunc.Call(execCtx, uint64(inputPtr), uint64(inputSize))
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to call solve function: %w", err)
	}

	if len(results) != 2 {
		return core.Result{}, fmt.Errorf("solve function should return (ptr, size), got %d results", len(results))
	}

	outputPtr := uint32(results[0])
	outputSize := uint32(results[1])

	// Read output from memory
	outputBytes, err := i.readString(instance, execCtx, outputPtr, outputSize)
	if err != nil {
		return core.Result{}, fmt.Errorf("failed to read output: %w", err)
	}

	// Parse output as JSON
	var output map[string]any
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return core.Result{}, fmt.Errorf("failed to parse output JSON: %w", err)
	}

	// Create result
	outputJSON, _ := json.Marshal(output)
	return core.Result{
		Success: true,
		Score:   1.0,
		Output:  outputJSON,
		Logs:    fmt.Sprintf("WASM module %s executed successfully", h.ID),
		Metrics: map[string]float64{
			"execution_time_ms": float64(time.Since(time.Now()).Milliseconds()),
			"output_size":       float64(len(outputBytes)),
		},
	}, nil
}

// getOrCompileModule returns a compiled module, using cache if available
func (i *Interpreter) getOrCompileModule(ctx context.Context, h core.Hypothesis) (wazero.CompiledModule, error) {
	// Check cache first
	if module, exists := i.cache[h.ID]; exists {
		return module, nil
	}

	// Compile the module
	module, err := i.runtime.CompileModule(ctx, h.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	// Cache the compiled module
	i.cache[h.ID] = module
	return module, nil
}

// allocateString allocates memory and writes a string to it
func (i *Interpreter) allocateString(instance api.Module, ctx context.Context, str string) (uint32, uint32, error) {
	// Get memory
	mem := instance.Memory()
	if mem == nil {
		return 0, 0, fmt.Errorf("module has no memory")
	}

	// Allocate memory (simple linear allocator)
	// In a real implementation, you'd want a proper allocator
	strBytes := []byte(str)
	strLen := uint32(len(strBytes))

	// For simplicity, write at offset 0
	// In a real implementation, you'd want proper memory management
	offset := uint32(0)

	// Check if we have enough space
	if uint64(offset)+uint64(strLen) > uint64(mem.Size()) {
		return 0, 0, fmt.Errorf("not enough memory: need %d bytes, have %d", strLen, mem.Size())
	}

	// Write string to memory
	if !mem.Write(offset, strBytes) {
		return 0, 0, fmt.Errorf("failed to write to memory")
	}

	return offset, strLen, nil
}

// readString reads a string from memory
func (i *Interpreter) readString(instance api.Module, ctx context.Context, ptr, size uint32) ([]byte, error) {
	mem := instance.Memory()
	if mem == nil {
		return nil, fmt.Errorf("module has no memory")
	}

	// Read from memory
	data, ok := mem.Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("failed to read from memory")
	}

	return data, nil
}

// Close closes the interpreter and cleans up resources
func (i *Interpreter) Close(ctx context.Context) error {
	return i.runtime.Close(ctx)
}
