package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics holds all Prometheus metrics
type PrometheusMetrics struct {
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	LatencyHistogram *prometheus.HistogramVec

	// Token metrics
	TokensInputTotal  *prometheus.CounterVec
	TokensOutputTotal *prometheus.CounterVec

	// Cost metrics
	CostTotal *prometheus.CounterVec

	// Cache metrics
	CacheHitsTotal   prometheus.Counter
	CacheMissesTotal prometheus.Counter

	// Retry metrics
	RetriesTotal *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitOpenTotal     *prometheus.CounterVec
	CircuitClosedTotal   *prometheus.CounterVec
	CircuitHalfOpenTotal *prometheus.CounterVec
}

// NewPrometheusMetrics creates a new Prometheus metrics instance
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		// Request metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_requests_total",
				Help: "Total number of LLM requests",
			},
			[]string{"provider", "model", "status"},
		),

		LatencyHistogram: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "llm_latency_seconds",
				Help:    "LLM request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "model"},
		),

		// Token metrics
		TokensInputTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_tokens_input_total",
				Help: "Total number of input tokens processed",
			},
			[]string{"provider", "model"},
		),

		TokensOutputTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_tokens_output_total",
				Help: "Total number of output tokens generated",
			},
			[]string{"provider", "model"},
		),

		// Cost metrics
		CostTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_cost_total",
				Help: "Total cost of LLM requests",
			},
			[]string{"provider", "model", "currency"},
		),

		// Cache metrics
		CacheHitsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "llm_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),

		CacheMissesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "llm_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),

		// Retry metrics
		RetriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_retries_total",
				Help: "Total number of retries",
			},
			[]string{"provider", "model", "reason"},
		),

		// Circuit breaker metrics
		CircuitOpenTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_circuit_open_total",
				Help: "Total number of circuit breaker opens",
			},
			[]string{"provider", "model"},
		),

		CircuitClosedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_circuit_closed_total",
				Help: "Total number of circuit breaker closes",
			},
			[]string{"provider", "model"},
		),

		CircuitHalfOpenTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_circuit_half_open_total",
				Help: "Total number of circuit breaker half-opens",
			},
			[]string{"provider", "model"},
		),
	}
}

// RecordRequest records a request metric
func (m *PrometheusMetrics) RecordRequest(provider, model, status string) {
	m.RequestsTotal.WithLabelValues(provider, model, status).Inc()
}

// RecordLatency records a latency metric
func (m *PrometheusMetrics) RecordLatency(provider, model string, duration time.Duration) {
	m.LatencyHistogram.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordTokens records token metrics
func (m *PrometheusMetrics) RecordTokens(provider, model string, inputTokens, outputTokens int) {
	if inputTokens > 0 {
		m.TokensInputTotal.WithLabelValues(provider, model).Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		m.TokensOutputTotal.WithLabelValues(provider, model).Add(float64(outputTokens))
	}
}

// RecordCost records a cost metric
func (m *PrometheusMetrics) RecordCost(provider, model, currency string, cost float64) {
	m.CostTotal.WithLabelValues(provider, model, currency).Add(cost)
}

// RecordCacheHit records a cache hit
func (m *PrometheusMetrics) RecordCacheHit() {
	m.CacheHitsTotal.Inc()
}

// RecordCacheMiss records a cache miss
func (m *PrometheusMetrics) RecordCacheMiss() {
	m.CacheMissesTotal.Inc()
}

// RecordRetry records a retry
func (m *PrometheusMetrics) RecordRetry(provider, model, reason string) {
	m.RetriesTotal.WithLabelValues(provider, model, reason).Inc()
}

// RecordCircuitOpen records a circuit breaker open
func (m *PrometheusMetrics) RecordCircuitOpen(provider, model string) {
	m.CircuitOpenTotal.WithLabelValues(provider, model).Inc()
}

// RecordCircuitClosed records a circuit breaker close
func (m *PrometheusMetrics) RecordCircuitClosed(provider, model string) {
	m.CircuitClosedTotal.WithLabelValues(provider, model).Inc()
}

// RecordCircuitHalfOpen records a circuit breaker half-open
func (m *PrometheusMetrics) RecordCircuitHalfOpen(provider, model string) {
	m.CircuitHalfOpenTotal.WithLabelValues(provider, model).Inc()
}

// GetCacheHitRate returns the current cache hit rate
func (m *PrometheusMetrics) GetCacheHitRate() float64 {
	// Note: In a real implementation, you would need to track these values
	// or use a different approach to get the current values
	// For now, return 0 as we can't easily get the current counter values
	return 0
}
