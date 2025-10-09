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

	// Worker metrics
	TasksTotal   *prometheus.CounterVec
	TasksSolved  *prometheus.CounterVec
	TasksFailed  *prometheus.CounterVec
	TaskDuration *prometheus.HistogramVec
	TestPassRate *prometheus.GaugeVec
	WorkerHealth *prometheus.GaugeVec

	// KB metrics
	KBHitsTotal      *prometheus.CounterVec
	KBMissesTotal    *prometheus.CounterVec
	KBArtifactsTotal *prometheus.GaugeVec
	KBIndexDuration  *prometheus.HistogramVec

	// WASM metrics
	WASMExecutionsTotal *prometheus.CounterVec
	WASMExecutionTime   *prometheus.HistogramVec
	WASMMemoryUsage     *prometheus.GaugeVec

	// Router metrics
	RouterRequestsTotal *prometheus.CounterVec
	RouterLatency       *prometheus.HistogramVec
	WorkerRouting       *prometheus.CounterVec

	// Vector search metrics
	VectorSearchTotal   *prometheus.CounterVec
	VectorSearchLatency *prometheus.HistogramVec
	VectorIndexSize     *prometheus.GaugeVec

	// System metrics
	MemoryUsage    *prometheus.GaugeVec
	CPUUsage       *prometheus.GaugeVec
	GoroutineCount prometheus.Gauge
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

		// Worker metrics
		TasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_tasks_total",
				Help: "Total number of tasks processed",
			},
			[]string{"worker_type", "domain", "status"},
		),

		TasksSolved: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_tasks_solved_total",
				Help: "Total number of tasks successfully solved",
			},
			[]string{"worker_type", "domain"},
		),

		TasksFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_tasks_failed_total",
				Help: "Total number of tasks that failed",
			},
			[]string{"worker_type", "domain", "reason"},
		),

		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_task_duration_seconds",
				Help:    "Task execution duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
			},
			[]string{"worker_type", "domain"},
		),

		TestPassRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_test_pass_rate",
				Help: "Test pass rate for tasks",
			},
			[]string{"worker_type", "domain"},
		),

		WorkerHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_worker_health",
				Help: "Worker health status (1=healthy, 0=unhealthy)",
			},
			[]string{"worker_type", "worker_id"},
		),

		// KB metrics
		KBHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_kb_hits_total",
				Help: "Total number of KB hits",
			},
			[]string{"worker_type", "domain"},
		),

		KBMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_kb_misses_total",
				Help: "Total number of KB misses",
			},
			[]string{"worker_type", "domain"},
		),

		KBArtifactsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_kb_artifacts_total",
				Help: "Total number of artifacts in KB",
			},
			[]string{"worker_type"},
		),

		KBIndexDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_kb_index_duration_seconds",
				Help:    "KB indexing duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"worker_type"},
		),

		// WASM metrics
		WASMExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_wasm_executions_total",
				Help: "Total number of WASM executions",
			},
			[]string{"worker_type", "status"},
		),

		WASMExecutionTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_wasm_execution_time_seconds",
				Help:    "WASM execution time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"worker_type"},
		),

		WASMMemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_wasm_memory_usage_bytes",
				Help: "WASM memory usage in bytes",
			},
			[]string{"worker_type"},
		),

		// Router metrics
		RouterRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_router_requests_total",
				Help: "Total number of router requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		RouterLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_router_latency_seconds",
				Help:    "Router request latency in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"method", "endpoint"},
		),

		WorkerRouting: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_worker_routing_total",
				Help: "Total number of worker routing decisions",
			},
			[]string{"worker_type", "reason"},
		),

		// Vector search metrics
		VectorSearchTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_vector_search_total",
				Help: "Total number of vector searches",
			},
			[]string{"backend", "status"},
		),

		VectorSearchLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_vector_search_latency_seconds",
				Help:    "Vector search latency in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"backend"},
		),

		VectorIndexSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_vector_index_size",
				Help: "Number of vectors in the index",
			},
			[]string{"backend"},
		),

		// System metrics
		MemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"component"},
		),

		CPUUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
			[]string{"component"},
		),

		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_goroutines_total",
				Help: "Total number of goroutines",
			},
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

// Worker metrics methods
func (m *PrometheusMetrics) RecordTask(workerType, domain, status string) {
	m.TasksTotal.WithLabelValues(workerType, domain, status).Inc()
}

func (m *PrometheusMetrics) RecordTaskSolved(workerType, domain string) {
	m.TasksSolved.WithLabelValues(workerType, domain).Inc()
}

func (m *PrometheusMetrics) RecordTaskFailed(workerType, domain, reason string) {
	m.TasksFailed.WithLabelValues(workerType, domain, reason).Inc()
}

func (m *PrometheusMetrics) RecordTaskDuration(workerType, domain string, duration time.Duration) {
	m.TaskDuration.WithLabelValues(workerType, domain).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) SetTestPassRate(workerType, domain string, rate float64) {
	m.TestPassRate.WithLabelValues(workerType, domain).Set(rate)
}

func (m *PrometheusMetrics) SetWorkerHealth(workerType, workerID string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.WorkerHealth.WithLabelValues(workerType, workerID).Set(value)
}

// KB metrics methods
func (m *PrometheusMetrics) RecordKBHit(workerType, domain string) {
	m.KBHitsTotal.WithLabelValues(workerType, domain).Inc()
}

func (m *PrometheusMetrics) RecordKBMiss(workerType, domain string) {
	m.KBMissesTotal.WithLabelValues(workerType, domain).Inc()
}

func (m *PrometheusMetrics) SetKBArtifactsCount(workerType string, count int) {
	m.KBArtifactsTotal.WithLabelValues(workerType).Set(float64(count))
}

func (m *PrometheusMetrics) RecordKBIndexDuration(workerType string, duration time.Duration) {
	m.KBIndexDuration.WithLabelValues(workerType).Observe(duration.Seconds())
}

// WASM metrics methods
func (m *PrometheusMetrics) RecordWASMExecution(workerType, status string) {
	m.WASMExecutionsTotal.WithLabelValues(workerType, status).Inc()
}

func (m *PrometheusMetrics) RecordWASMExecutionTime(workerType string, duration time.Duration) {
	m.WASMExecutionTime.WithLabelValues(workerType).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) SetWASMMemoryUsage(workerType string, bytes int64) {
	m.WASMMemoryUsage.WithLabelValues(workerType).Set(float64(bytes))
}

// Router metrics methods
func (m *PrometheusMetrics) RecordRouterRequest(method, endpoint, status string) {
	m.RouterRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

func (m *PrometheusMetrics) RecordRouterLatency(method, endpoint string, duration time.Duration) {
	m.RouterLatency.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordWorkerRouting(workerType, reason string) {
	m.WorkerRouting.WithLabelValues(workerType, reason).Inc()
}

// Vector search metrics methods
func (m *PrometheusMetrics) RecordVectorSearch(backend, status string) {
	m.VectorSearchTotal.WithLabelValues(backend, status).Inc()
}

func (m *PrometheusMetrics) RecordVectorSearchLatency(backend string, duration time.Duration) {
	m.VectorSearchLatency.WithLabelValues(backend).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) SetVectorIndexSize(backend string, size int) {
	m.VectorIndexSize.WithLabelValues(backend).Set(float64(size))
}

// System metrics methods
func (m *PrometheusMetrics) SetMemoryUsage(component string, bytes int64) {
	m.MemoryUsage.WithLabelValues(component).Set(float64(bytes))
}

func (m *PrometheusMetrics) SetCPUUsage(component string, percent float64) {
	m.CPUUsage.WithLabelValues(component).Set(percent)
}

func (m *PrometheusMetrics) SetGoroutineCount(count int) {
	m.GoroutineCount.Set(float64(count))
}
