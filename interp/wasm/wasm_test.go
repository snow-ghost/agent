package wasm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpreter_Execute(t *testing.T) {
	interpreter := NewInterpreter()
	defer interpreter.Close(context.Background())

	ctx := context.Background()

	t.Run("execute simple wasm module", func(t *testing.T) {
		// Create a test hypothesis with WASM module
		hypothesis := core.Hypothesis{
			ID:     "test-wasm",
			Source: "test",
			Lang:   "wasm",
			Bytes:  GetTestModule(),
			Meta: map[string]string{
				"version": "v1",
			},
		}

		// Create a test task
		task := core.Task{
			ID:     "task1",
			Domain: "test",
			Spec: core.Spec{
				SuccessCriteria: []string{"output_valid"},
				Props:           map[string]string{"type": "test"},
			},
			Input: json.RawMessage(`{"test": "data"}`),
			Budget: core.Budget{
				CPUMillis: 5000, // 5 seconds
				MemMB:     64,   // 64MB
				Timeout:   time.Second * 10,
			},
			CreatedAt: time.Now(),
		}

		// Execute the WASM module
		result, err := interpreter.Execute(ctx, hypothesis, task)

		// The simple test module doesn't do much, so we expect it to succeed
		// but with minimal output
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1.0, result.Score)
		assert.Contains(t, result.Logs, "WASM module test-wasm executed successfully")
	})

	t.Run("test timeout", func(t *testing.T) {
		// Create a test hypothesis with WASM module
		hypothesis := core.Hypothesis{
			ID:     "test-timeout",
			Source: "test",
			Lang:   "wasm",
			Bytes:  GetTestModule(),
			Meta: map[string]string{
				"version": "v1",
			},
		}

		// Create a test task with very short timeout
		task := core.Task{
			ID:     "task2",
			Domain: "test",
			Spec: core.Spec{
				SuccessCriteria: []string{"output_valid"},
				Props:           map[string]string{"type": "test"},
			},
			Input: json.RawMessage(`{"test": "data"}`),
			Budget: core.Budget{
				CPUMillis: 1, // 1ms - very short timeout
				MemMB:     64,
				Timeout:   time.Millisecond,
			},
			CreatedAt: time.Now(),
		}

		// Execute the WASM module
		result, err := interpreter.Execute(ctx, hypothesis, task)

		// Should either succeed quickly or timeout
		if err != nil {
			assert.Contains(t, err.Error(), "context deadline exceeded")
		} else {
			assert.True(t, result.Success)
		}
	})

	t.Run("test invalid wasm module", func(t *testing.T) {
		// Create a test hypothesis with invalid WASM
		hypothesis := core.Hypothesis{
			ID:     "test-invalid",
			Source: "test",
			Lang:   "wasm",
			Bytes:  []byte("invalid wasm"),
			Meta: map[string]string{
				"version": "v1",
			},
		}

		// Create a test task
		task := core.Task{
			ID:     "task3",
			Domain: "test",
			Spec: core.Spec{
				SuccessCriteria: []string{"output_valid"},
				Props:           map[string]string{"type": "test"},
			},
			Input: json.RawMessage(`{"test": "data"}`),
			Budget: core.Budget{
				CPUMillis: 5000,
				MemMB:     64,
				Timeout:   time.Second * 10,
			},
			CreatedAt: time.Now(),
		}

		// Execute the WASM module
		_, err := interpreter.Execute(ctx, hypothesis, task)

		// Should fail with compilation error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compile")
	})
}

func TestInterpreter_Cache(t *testing.T) {
	interpreter := NewInterpreter()
	defer interpreter.Close(context.Background())

	ctx := context.Background()

	// Create a test hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-cache",
		Source: "test",
		Lang:   "wasm",
		Bytes:  GetTestModule(),
		Meta: map[string]string{
			"version": "v1",
		},
	}

	// Create a test task
	task := core.Task{
		ID:     "task1",
		Domain: "test",
		Spec: core.Spec{
			SuccessCriteria: []string{"output_valid"},
			Props:           map[string]string{"type": "test"},
		},
		Input: json.RawMessage(`{"test": "data"}`),
		Budget: core.Budget{
			CPUMillis: 5000,
			MemMB:     64,
			Timeout:   time.Second * 10,
		},
		CreatedAt: time.Now(),
	}

	// Execute the same module twice
	result1, err1 := interpreter.Execute(ctx, hypothesis, task)
	require.NoError(t, err1)
	assert.True(t, result1.Success)

	result2, err2 := interpreter.Execute(ctx, hypothesis, task)
	require.NoError(t, err2)
	assert.True(t, result2.Success)

	// Both should succeed (cache should work)
	assert.Equal(t, result1.Success, result2.Success)
}

func TestInterpreter_MemoryLimits(t *testing.T) {
	interpreter := NewInterpreter()
	defer interpreter.Close(context.Background())

	ctx := context.Background()

	// Create a test hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-memory",
		Source: "test",
		Lang:   "wasm",
		Bytes:  GetTestModule(),
		Meta: map[string]string{
			"version": "v1",
		},
	}

	// Create a test task with very small memory limit
	task := core.Task{
		ID:     "task1",
		Domain: "test",
		Spec: core.Spec{
			SuccessCriteria: []string{"output_valid"},
			Props:           map[string]string{"type": "test"},
		},
		Input: json.RawMessage(`{"test": "data"}`),
		Budget: core.Budget{
			CPUMillis: 5000,
			MemMB:     1, // Very small memory limit
			Timeout:   time.Second * 10,
		},
		CreatedAt: time.Now(),
	}

	// Execute the WASM module
	result, err := interpreter.Execute(ctx, hypothesis, task)

	// Should either succeed or fail due to memory limits
	if err != nil {
		// If it fails, it should be due to memory or other resource limits
		assert.True(t,
			err.Error() == "context deadline exceeded" ||
				err.Error() == "not enough memory" ||
				err.Error() == "failed to instantiate module",
		)
	} else {
		assert.True(t, result.Success)
	}
}
