package wasm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMPool manages a pool of WASM runtime instances for better performance
type WASMPool struct {
	pool           chan *WASMInstance
	maxSize        int
	timeout        time.Duration
	compiledModule wazero.CompiledModule
	runtime        wazero.Runtime
	mu             sync.RWMutex
	closed         bool
}

// WASMInstance represents a WASM runtime instance
type WASMInstance struct {
	Module   api.Module
	Runtime  wazero.Runtime
	Created  time.Time
	LastUsed time.Time
}

// PoolConfig holds configuration for the WASM pool
type PoolConfig struct {
	MaxSize     int
	Timeout     time.Duration
	IdleTimeout time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:     10,
		Timeout:     30 * time.Second,
		IdleTimeout: 5 * time.Minute,
	}
}

// NewWASMPool creates a new WASM pool
func NewWASMPool(bytecode []byte, config *PoolConfig) (*WASMPool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	// Create runtime
	runtime := wazero.NewRuntimeWithConfig(context.Background(), wazero.NewRuntimeConfig())

	// Compile module
	compiledModule, err := runtime.CompileModule(context.Background(), bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	pool := &WASMPool{
		pool:           make(chan *WASMInstance, config.MaxSize),
		maxSize:        config.MaxSize,
		timeout:        config.Timeout,
		compiledModule: compiledModule,
		runtime:        runtime,
	}

	// Pre-populate pool with instances
	for i := 0; i < config.MaxSize; i++ {
		instance, err := pool.createInstance()
		if err != nil {
			// If we can't create instances, close the pool
			pool.Close()
			return nil, fmt.Errorf("failed to create WASM instance: %w", err)
		}
		pool.pool <- instance
	}

	// Start cleanup goroutine
	go pool.cleanupIdleInstances(config.IdleTimeout)

	return pool, nil
}

// GetInstance gets a WASM instance from the pool
func (p *WASMPool) GetInstance(ctx context.Context) (*WASMInstance, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	select {
	case instance := <-p.pool:
		// Update last used time
		instance.LastUsed = time.Now()
		return instance, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(p.timeout):
		return nil, fmt.Errorf("timeout waiting for WASM instance")
	}
}

// ReturnInstance returns a WASM instance to the pool
func (p *WASMPool) ReturnInstance(instance *WASMInstance) {
	if instance == nil {
		return
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		// Pool is closed, close the instance
		instance.Close()
		return
	}
	p.mu.RUnlock()

	// Try to return to pool, but don't block
	select {
	case p.pool <- instance:
		// Successfully returned to pool
	default:
		// Pool is full, close the instance
		instance.Close()
	}
}

// createInstance creates a new WASM instance
func (p *WASMPool) createInstance() (*WASMInstance, error) {
	module, err := p.runtime.InstantiateModule(context.Background(), p.compiledModule, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	return &WASMInstance{
		Module:   module,
		Runtime:  p.runtime,
		Created:  time.Now(),
		LastUsed: time.Now(),
	}, nil
}

// cleanupIdleInstances periodically cleans up idle instances
func (p *WASMPool) cleanupIdleInstances(idleTimeout time.Duration) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		// Check for idle instances
		now := time.Now()
		var idleInstances []*WASMInstance

		// Drain pool and check for idle instances
		for {
			select {
			case instance := <-p.pool:
				if now.Sub(instance.LastUsed) > idleTimeout {
					idleInstances = append(idleInstances, instance)
				} else {
					// Return non-idle instance back to pool
					select {
					case p.pool <- instance:
					default:
						// Pool is full, close the instance
						instance.Close()
					}
				}
			default:
				goto done
			}
		}
	done:

		// Close idle instances
		for _, instance := range idleInstances {
			instance.Close()
		}

		// Refill pool if needed
		p.refillPool()
	}
}

// refillPool refills the pool to maintain the desired size
func (p *WASMPool) refillPool() {
	currentSize := len(p.pool)
	needed := p.maxSize - currentSize

	for i := 0; i < needed; i++ {
		instance, err := p.createInstance()
		if err != nil {
			// If we can't create instances, stop trying
			break
		}

		select {
		case p.pool <- instance:
			// Successfully added to pool
		default:
			// Pool is full, close the instance
			instance.Close()
		}
	}
}

// Close closes the WASM pool and all instances
func (p *WASMPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Close all instances in the pool
	for {
		select {
		case instance := <-p.pool:
			instance.Close()
		default:
			goto done
		}
	}
done:

	// Close the runtime
	return p.runtime.Close(context.Background())
}

// GetStats returns pool statistics
func (p *WASMPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"max_size":     p.maxSize,
		"current_size": len(p.pool),
		"closed":       p.closed,
	}
}

// Close closes a WASM instance
func (i *WASMInstance) Close() error {
	if i.Module != nil {
		return i.Module.Close(context.Background())
	}
	return nil
}

// ExecuteFunction executes a function on the WASM instance
func (i *WASMInstance) ExecuteFunction(functionName string, args ...uint64) ([]uint64, error) {
	if i.Module == nil {
		return nil, fmt.Errorf("WASM instance is closed")
	}

	// Get the function
	fn := i.Module.ExportedFunction(functionName)
	if fn == nil {
		return nil, fmt.Errorf("function %s not found", functionName)
	}

	// Execute the function
	results, err := fn.Call(context.Background(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute function %s: %w", functionName, err)
	}

	return results, nil
}

// PoolManager manages multiple WASM pools
type PoolManager struct {
	pools map[string]*WASMPool
	mu    sync.RWMutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]*WASMPool),
	}
}

// GetPool gets a pool by name
func (pm *PoolManager) GetPool(name string) *WASMPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pools[name]
}

// CreatePool creates a new pool
func (pm *PoolManager) CreatePool(name string, bytecode []byte, config *PoolConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.pools[name]; exists {
		return fmt.Errorf("pool %s already exists", name)
	}

	pool, err := NewWASMPool(bytecode, config)
	if err != nil {
		return fmt.Errorf("failed to create pool %s: %w", name, err)
	}

	pm.pools[name] = pool
	return nil
}

// ClosePool closes a pool
func (pm *PoolManager) ClosePool(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pool, exists := pm.pools[name]
	if !exists {
		return fmt.Errorf("pool %s not found", name)
	}

	err := pool.Close()
	delete(pm.pools, name)
	return err
}

// CloseAll closes all pools
func (pm *PoolManager) CloseAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	for name, pool := range pm.pools {
		if err := pool.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close pool %s: %w", name, err)
		}
	}

	pm.pools = make(map[string]*WASMPool)
	return lastErr
}

// GetStats returns statistics for all pools
func (pm *PoolManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, pool := range pm.pools {
		stats[name] = pool.GetStats()
	}

	return stats
}
