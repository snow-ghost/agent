package http

import (
	"net"
	"net/http"
	"time"
)

// ConnectionPoolConfig holds configuration for HTTP connection pooling
type ConnectionPoolConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
	IdleConnTimeout       time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
	DialTimeout           time.Duration
	KeepAlive             time.Duration
}

// DefaultConnectionPoolConfig returns default connection pool configuration
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       0, // No limit
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           30 * time.Second,
		KeepAlive:             30 * time.Second,
	}
}

// NewOptimizedHTTPClient creates an HTTP client with optimized connection pooling
func NewOptimizedHTTPClient(config *ConnectionPoolConfig) *http.Client {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	// Create custom transport with optimized settings
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// NewLLMHTTPClient creates an HTTP client optimized for LLM API calls
func NewLLMHTTPClient() *http.Client {
	config := &ConnectionPoolConfig{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       20, // Limit concurrent connections to LLM APIs
		IdleConnTimeout:       2 * time.Minute,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           10 * time.Second,
		KeepAlive:             30 * time.Second,
	}

	return NewOptimizedHTTPClient(config)
}

// NewWorkerHTTPClient creates an HTTP client optimized for worker communication
func NewWorkerHTTPClient() *http.Client {
	config := &ConnectionPoolConfig{
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       10,
		IdleConnTimeout:       1 * time.Minute,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialTimeout:           5 * time.Second,
		KeepAlive:             30 * time.Second,
	}

	return NewOptimizedHTTPClient(config)
}

// HTTPClientManager manages multiple HTTP clients with different configurations
type HTTPClientManager struct {
	clients map[string]*http.Client
}

// NewHTTPClientManager creates a new HTTP client manager
func NewHTTPClientManager() *HTTPClientManager {
	manager := &HTTPClientManager{
		clients: make(map[string]*http.Client),
	}

	// Create default clients
	manager.clients["llm"] = NewLLMHTTPClient()
	manager.clients["worker"] = NewWorkerHTTPClient()
	manager.clients["default"] = NewOptimizedHTTPClient(nil)

	return manager
}

// GetClient gets a client by name
func (m *HTTPClientManager) GetClient(name string) *http.Client {
	if client, exists := m.clients[name]; exists {
		return client
	}
	return m.clients["default"]
}

// AddClient adds a custom client
func (m *HTTPClientManager) AddClient(name string, client *http.Client) {
	m.clients[name] = client
}

// CloseAll closes all clients (useful for cleanup)
func (m *HTTPClientManager) CloseAll() {
	for _, client := range m.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
}

// GetStats returns connection statistics for all clients
func (m *HTTPClientManager) GetStats() map[string]map[string]interface{} {
	stats := make(map[string]map[string]interface{})

	for name, client := range m.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			stats[name] = map[string]interface{}{
				"max_idle_conns":          transport.MaxIdleConns,
				"max_idle_conns_per_host": transport.MaxIdleConnsPerHost,
				"max_conns_per_host":      transport.MaxConnsPerHost,
				"idle_conn_timeout":       transport.IdleConnTimeout,
				"response_header_timeout": transport.ResponseHeaderTimeout,
			}
		}
	}

	return stats
}

// ConnectionPoolMetrics tracks connection pool metrics
type ConnectionPoolMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
}

// NewConnectionPoolMetrics creates new connection pool metrics
func NewConnectionPoolMetrics() *ConnectionPoolMetrics {
	return &ConnectionPoolMetrics{
		MinLatency: time.Hour, // Initialize with high value
	}
}

// RecordRequest records a request metric
func (m *ConnectionPoolMetrics) RecordRequest(success bool, latency time.Duration) {
	m.TotalRequests++

	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
	}

	// Update latency statistics
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
	if latency < m.MinLatency {
		m.MinLatency = latency
	}

	// Update average latency (simple moving average)
	if m.TotalRequests == 1 {
		m.AverageLatency = latency
	} else {
		m.AverageLatency = (m.AverageLatency*time.Duration(m.TotalRequests-1) + latency) / time.Duration(m.TotalRequests)
	}
}

// GetSuccessRate returns the success rate
func (m *ConnectionPoolMetrics) GetSuccessRate() float64 {
	if m.TotalRequests == 0 {
		return 0.0
	}
	return float64(m.SuccessfulRequests) / float64(m.TotalRequests)
}

// GetStats returns all metrics
func (m *ConnectionPoolMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":      m.TotalRequests,
		"successful_requests": m.SuccessfulRequests,
		"failed_requests":     m.FailedRequests,
		"success_rate":        m.GetSuccessRate(),
		"average_latency":     m.AverageLatency.String(),
		"max_latency":         m.MaxLatency.String(),
		"min_latency":         m.MinLatency.String(),
	}
}
