package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectionPool(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	pool := NewConnectionPool(config)
	require.NotNil(t, pool)

	// Verify the transport is configured correctly
	transport := pool.Transport.(*http.Transport)
	assert.Equal(t, config.MaxIdleConns, transport.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
}

func TestNewConnectionPool_DefaultConfig(t *testing.T) {
	pool := NewConnectionPool(nil)
	require.NotNil(t, pool)

	// Verify default values are used
	transport := pool.Transport.(*http.Transport)
	assert.Equal(t, 100, transport.MaxIdleConns)
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
}

func TestNewConnectionPool_CustomTimeouts(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
	}

	pool := NewConnectionPool(config)
	require.NotNil(t, pool)

	transport := pool.Transport.(*http.Transport)
	assert.Equal(t, 50, transport.MaxIdleConns)
	assert.Equal(t, 5, transport.MaxIdleConnsPerHost)
}

func TestConnectionPool_GetClient(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)
	client := pool.GetClient()

	require.NotNil(t, client)
	assert.Equal(t, pool.Transport, client.Transport)
	assert.Equal(t, 30*time.Second, client.Timeout)
}

func TestConnectionPool_GetClient_MultipleCalls(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)

	// Get multiple clients
	client1 := pool.GetClient()
	client2 := pool.GetClient()

	require.NotNil(t, client1)
	require.NotNil(t, client2)

	// Clients should be different instances but share the same transport
	assert.NotEqual(t, client1, client2)
	assert.Equal(t, client1.Transport, client2.Transport)
}

func TestConnectionPool_Close(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)

	// Close should not panic
	assert.NotPanics(t, func() {
		pool.Close()
	})
}

func TestConnectionPool_CloseIdleConnections(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)

	// CloseIdleConnections should not panic
	assert.NotPanics(t, func() {
		pool.CloseIdleConnections()
	})
}

func TestConnectionPool_Stats(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	pool := NewConnectionPool(config)
	stats := pool.Stats()

	assert.Equal(t, 100, stats.MaxIdleConns)
	assert.Equal(t, 10, stats.MaxIdleConnsPerHost)
	assert.Equal(t, 0, stats.IdleConns) // Initially 0
}

func TestConnectionPool_Stats_AfterUsage(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)

	// Get a client and make a request to create connections
	client := pool.GetClient()

	// Create a test server
	server := &http.Server{
		Addr: "localhost:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	// Start server
	go server.ListenAndServe()
	defer server.Close()

	// Make a request to create connections
	req, _ := http.NewRequest("GET", "http://localhost:8080/test", nil)
	client.Do(req) // This will fail but will create connections

	// Check stats
	stats := pool.Stats()
	assert.Equal(t, 10, stats.MaxIdleConns)
	assert.Equal(t, 2, stats.MaxIdleConnsPerHost)
}

func TestDefaultConnectionPoolConfig(t *testing.T) {
	config := DefaultConnectionPoolConfig()

	assert.Equal(t, 100, config.MaxIdleConns)
	assert.Equal(t, 10, config.MaxIdleConnsPerHost)
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 5,
	}

	pool := NewConnectionPool(config)

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			client := pool.GetClient()
			assert.NotNil(t, client)
			time.Sleep(10 * time.Millisecond)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify pool is still functional
	client := pool.GetClient()
	assert.NotNil(t, client)
}

func TestConnectionPool_TransportConfiguration(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
	}

	pool := NewConnectionPool(config)
	transport := pool.Transport.(*http.Transport)

	// Verify transport configuration
	assert.Equal(t, 50, transport.MaxIdleConns)
	assert.Equal(t, 5, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 30*time.Second, transport.IdleConnTimeout)
	assert.Equal(t, 90*time.Second, transport.ResponseHeaderTimeout)
	assert.True(t, transport.DisableKeepAlives)
}

func TestConnectionPool_ClientTimeout(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}

	pool := NewConnectionPool(config)
	client := pool.GetClient()

	// Verify client timeout
	assert.Equal(t, 30*time.Second, client.Timeout)
}

func TestConnectionPool_ZeroValues(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 0,
	}

	pool := NewConnectionPool(config)
	transport := pool.Transport.(*http.Transport)

	// Zero values should be handled gracefully
	assert.Equal(t, 0, transport.MaxIdleConns)
	assert.Equal(t, 0, transport.MaxIdleConnsPerHost)
}
