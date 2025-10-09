package wasm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWASMPool(t *testing.T) {
	// Create a simple WASM bytecode (this is a minimal valid WASM module)
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 5,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, config, pool.config)
	assert.Equal(t, bytecode, pool.bytecode)
	assert.NotNil(t, pool.instances)
	assert.Equal(t, 0, pool.currentInstances)
}

func TestNewWASMPool_DefaultConfig(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	pool, err := NewWASMPool(bytecode, nil)
	require.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, DefaultPoolConfig(), pool.config)
}

func TestNewWASMPool_InvalidBytecode(t *testing.T) {
	invalidBytecode := []byte{0x00, 0x00, 0x00, 0x00} // Invalid WASM

	_, err := NewWASMPool(invalidBytecode, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile WASM module")
}

func TestWASMPool_GetInstance(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 2,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get first instance
	instance1, err := pool.GetInstance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, instance1)
	assert.Equal(t, 1, pool.currentInstances)

	// Get second instance
	instance2, err := pool.GetInstance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, instance2)
	assert.Equal(t, 2, pool.currentInstances)

	// Verify instances are different
	assert.NotEqual(t, instance1, instance2)
}

func TestWASMPool_GetInstance_MaxInstances(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 1,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get first instance
	_, err = pool.GetInstance(context.Background())
	require.NoError(t, err)

	// Try to get second instance (should fail)
	_, err = pool.GetInstance(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pool exhausted")
}

func TestWASMPool_ReturnInstance(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 2,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get an instance
	instance, err := pool.GetInstance(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, pool.currentInstances)

	// Return the instance
	pool.ReturnInstance(instance)
	assert.Equal(t, 0, pool.currentInstances)
}

func TestWASMPool_ReturnInstance_InvalidInstance(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	pool, err := NewWASMPool(bytecode, nil)
	require.NoError(t, err)

	// Try to return an invalid instance
	pool.ReturnInstance(nil)
	// Should not panic or error
}

func TestWASMPool_Close(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	pool, err := NewWASMPool(bytecode, nil)
	require.NoError(t, err)

	// Get some instances
	instance1, err := pool.GetInstance(context.Background())
	require.NoError(t, err)
	instance2, err := pool.GetInstance(context.Background())
	require.NoError(t, err)

	// Close the pool
	err = pool.Close()
	require.NoError(t, err)

	// Verify instances are closed
	assert.Nil(t, instance1)
	assert.Nil(t, instance2)
}

func TestWASMPool_Stats(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 3,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Initial stats
	stats := pool.Stats()
	assert.Equal(t, 0, stats.ActiveInstances)
	assert.Equal(t, 0, stats.IdleInstances)
	assert.Equal(t, 3, stats.MaxInstances)

	// Get an instance
	_, err = pool.GetInstance(context.Background())
	require.NoError(t, err)

	// Check stats after getting instance
	stats = pool.Stats()
	assert.Equal(t, 1, stats.ActiveInstances)
	assert.Equal(t, 0, stats.IdleInstances)
	assert.Equal(t, 3, stats.MaxInstances)
}

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	assert.Equal(t, 10, config.MaxInstances)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
}

func TestWASMPool_GetInstance_ContextCancellation(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 1,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get first instance
	_, err = pool.GetInstance(context.Background())
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to get instance with cancelled context
	_, err = pool.GetInstance(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestWASMPool_ConcurrentAccess(t *testing.T) {
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxInstances: 5,
		IdleTimeout:  30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			instance, err := pool.GetInstance(context.Background())
			if err == nil {
				time.Sleep(10 * time.Millisecond)
				pool.ReturnInstance(instance)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify pool is in a consistent state
	stats := pool.Stats()
	assert.True(t, stats.ActiveInstances <= config.MaxInstances)
}
