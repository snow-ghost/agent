package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptimizedHTTPClient(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	client := NewOptimizedHTTPClient(config)
	require.NotNil(t, client)

	// Verify the transport is configured correctly
	transport := client.Transport.(*http.Transport)
	assert.Equal(t, config.MaxIdleConns, transport.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
}

func TestNewOptimizedHTTPClient_DefaultConfig(t *testing.T) {
	client := NewOptimizedHTTPClient(nil)
	require.NotNil(t, client)

	// Should use default configuration
	transport := client.Transport.(*http.Transport)
	assert.Equal(t, 100, transport.MaxIdleConns)
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
}

func TestNewLLMHTTPClient(t *testing.T) {
	client := NewLLMHTTPClient()
	require.NotNil(t, client)

	// Check that it's configured for LLM usage
	transport := client.Transport.(*http.Transport)
	assert.Equal(t, 50, transport.MaxIdleConns)
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 20, transport.MaxConnsPerHost)
}

func TestNewWorkerHTTPClient(t *testing.T) {
	client := NewWorkerHTTPClient()
	require.NotNil(t, client)

	// Check that it's configured for worker usage
	transport := client.Transport.(*http.Transport)
	assert.Equal(t, 20, transport.MaxIdleConns)
	assert.Equal(t, 5, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 10, transport.MaxConnsPerHost)
}

func TestHTTPClientManager(t *testing.T) {
	manager := NewHTTPClientManager()
	require.NotNil(t, manager)

	// Test getting default clients
	llmClient := manager.GetClient("llm")
	workerClient := manager.GetClient("worker")
	defaultClient := manager.GetClient("default")

	require.NotNil(t, llmClient)
	require.NotNil(t, workerClient)
	require.NotNil(t, defaultClient)

	// Test getting non-existent client (should return default)
	unknownClient := manager.GetClient("unknown")
	assert.Equal(t, defaultClient, unknownClient)
}

func TestHTTPClientManager_AddClient(t *testing.T) {
	manager := NewHTTPClientManager()
	customClient := &http.Client{Timeout: 30 * time.Second}

	manager.AddClient("custom", customClient)
	retrievedClient := manager.GetClient("custom")

	assert.Equal(t, customClient, retrievedClient)
}

func TestHTTPClientManager_CloseAll(t *testing.T) {
	manager := NewHTTPClientManager()

	// Should not panic
	assert.NotPanics(t, func() {
		manager.CloseAll()
	})
}

func TestHTTPClientManager_GetStats(t *testing.T) {
	manager := NewHTTPClientManager()
	stats := manager.GetStats()

	require.NotNil(t, stats)
	assert.Contains(t, stats, "llm")
	assert.Contains(t, stats, "worker")
	assert.Contains(t, stats, "default")
}

func TestConnectionPoolMetrics(t *testing.T) {
	metrics := NewConnectionPoolMetrics()
	require.NotNil(t, metrics)

	// Test initial state
	assert.Equal(t, int64(0), metrics.TotalRequests)
	assert.Equal(t, int64(0), metrics.SuccessfulRequests)
	assert.Equal(t, int64(0), metrics.FailedRequests)
	assert.Equal(t, 0.0, metrics.GetSuccessRate())

	// Test recording successful request
	metrics.RecordRequest(true, 100*time.Millisecond)
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.Equal(t, int64(0), metrics.FailedRequests)
	assert.Equal(t, 1.0, metrics.GetSuccessRate())

	// Test recording failed request
	metrics.RecordRequest(false, 200*time.Millisecond)
	assert.Equal(t, int64(2), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.Equal(t, int64(1), metrics.FailedRequests)
	assert.Equal(t, 0.5, metrics.GetSuccessRate())

	// Test latency tracking
	assert.Equal(t, 100*time.Millisecond, metrics.MinLatency)
	assert.Equal(t, 200*time.Millisecond, metrics.MaxLatency)
}

func TestConnectionPoolMetrics_GetStats(t *testing.T) {
	metrics := NewConnectionPoolMetrics()
	metrics.RecordRequest(true, 100*time.Millisecond)
	metrics.RecordRequest(false, 200*time.Millisecond)

	stats := metrics.GetStats()
	require.NotNil(t, stats)

	assert.Equal(t, int64(2), stats["total_requests"])
	assert.Equal(t, int64(1), stats["successful_requests"])
	assert.Equal(t, int64(1), stats["failed_requests"])
	assert.Equal(t, 0.5, stats["success_rate"])
}
