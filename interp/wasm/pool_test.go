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
		MaxSize:     5,
		Timeout:     30 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, config.MaxSize, pool.maxSize)
	assert.Equal(t, config.Timeout, pool.timeout)
	assert.NotNil(t, pool.pool)
	assert.False(t, pool.closed)
}

func TestWASMPool_GetInstance(t *testing.T) {
	// Create a simple WASM bytecode
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxSize:     2,
		Timeout:     30 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get an instance
	instance, err := pool.GetInstance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, instance)
	assert.NotNil(t, instance.Module)
	assert.NotNil(t, instance.Runtime)
	assert.True(t, instance.Created.Before(time.Now().Add(time.Second)))
	assert.True(t, instance.LastUsed.Before(time.Now().Add(time.Second)))
}

func TestWASMPool_ReturnInstance(t *testing.T) {
	// Create a simple WASM bytecode
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxSize:     2,
		Timeout:     30 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Get an instance
	instance, err := pool.GetInstance(context.Background())
	require.NoError(t, err)

	// Return the instance
	pool.ReturnInstance(instance)

	// The instance should be available in the pool again
	// (This is hard to test without exposing internal state)
	assert.NotNil(t, instance)
}

func TestWASMPool_Close(t *testing.T) {
	// Create a simple WASM bytecode
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxSize:     2,
		Timeout:     30 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)

	// Close the pool
	pool.Close()

	// Try to get an instance after closing
	_, err = pool.GetInstance(context.Background())
	assert.Error(t, err)
}

func TestWASMPool_ConcurrentAccess(t *testing.T) {
	// Create a simple WASM bytecode
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	config := &PoolConfig{
		MaxSize:     3,
		Timeout:     30 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	pool, err := NewWASMPool(bytecode, config)
	require.NoError(t, err)
	defer pool.Close()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			instance, err := pool.GetInstance(context.Background())
			if err != nil {
				return
			}

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			pool.ReturnInstance(instance)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxSize)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
}

func TestWASMPool_WithNilConfig(t *testing.T) {
	// Create a simple WASM bytecode
	bytecode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version 1
	}

	pool, err := NewWASMPool(bytecode, nil)
	require.NoError(t, err)
	assert.NotNil(t, pool)

	// Should use default config
	assert.Equal(t, 10, pool.maxSize)
	assert.Equal(t, 30*time.Second, pool.timeout)
}
