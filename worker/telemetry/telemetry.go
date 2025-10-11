package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/pkg/metrics"
)

// Telemetry collects basic metrics and provides structured logging
type Telemetry struct {
	mu sync.RWMutex

	// Prometheus metrics
	metrics *metrics.PrometheusMetrics

	// Internal state for calculations
	totalSolveTime time.Duration
	totalTests     int
	passedTests    int

	logger *slog.Logger
}

// NewTelemetry creates a new telemetry instance
func NewTelemetry() *Telemetry {
	t := &Telemetry{
		metrics: metrics.NewPrometheusMetrics(),
		logger:  slog.Default(),
	}

	return t
}

// LogTaskStart logs the start of a task
func (t *Telemetry) LogTaskStart(ctx context.Context, task core.Task) {
	t.logger.InfoContext(ctx, "task_started",
		"task_id", task.ID,
		"domain", task.Domain,
		"timeout_ms", task.Budget.CPUMillis,
	)

	t.metrics.TasksTotal.WithLabelValues("worker", task.Domain, "started").Inc()
}

// LogTaskEnd logs the end of a task with result
func (t *Telemetry) LogTaskEnd(ctx context.Context, task core.Task, result core.Result, duration time.Duration, iterations int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalSolveTime += duration

	// Record task duration
	t.metrics.TaskDuration.WithLabelValues("worker", task.Domain).Observe(duration.Seconds())

	if result.Success {
		t.metrics.TasksSolved.WithLabelValues("worker", task.Domain).Inc()
		t.logger.InfoContext(ctx, "task_solved",
			"task_id", task.ID,
			"duration_ms", duration.Milliseconds(),
			"iterations", iterations,
			"score", result.Score,
		)
	} else {
		// Extract failure reason from result logs
		failureReason := "unknown"
		if result.Logs != "" {
			failureReason = result.Logs
		}

		t.metrics.TasksFailed.WithLabelValues("worker", task.Domain, failureReason).Inc()
		t.logger.WarnContext(ctx, "task_failed",
			"task_id", task.ID,
			"duration_ms", duration.Milliseconds(),
			"iterations", iterations,
			"reason", failureReason,
		)
	}
}

// LogTestResults logs test execution results
func (t *Telemetry) LogTestResults(ctx context.Context, hypothesis core.Hypothesis, metrics map[string]float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if total, ok := metrics["cases_total"]; ok {
		t.totalTests += int(total)
	}
	if passed, ok := metrics["cases_passed"]; ok {
		t.passedTests += int(passed)
	}

	// Update pass rate
	if t.totalTests > 0 {
		passRate := float64(t.passedTests) / float64(t.totalTests)
		domain := hypothesis.Meta["domain"]
		if domain == "" {
			domain = "unknown"
		}
		t.metrics.SetTestPassRate("worker", domain, passRate)
	}

	t.logger.DebugContext(ctx, "test_results",
		"hypothesis_id", hypothesis.ID,
		"metrics", metrics,
	)
}

// LogIteration logs an evolution iteration
func (t *Telemetry) LogIteration(ctx context.Context, iteration int, bestScore float64, candidates int) {
	t.logger.DebugContext(ctx, "evolution_iteration",
		"iteration", iteration,
		"best_score", bestScore,
		"candidates", candidates,
	)
}

// HealthHandler returns a simple health check
func (t *Telemetry) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"agent-worker"}`))
}

// MetricsHandler returns metrics in Prometheus format
func (t *Telemetry) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Use the default Prometheus handler
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// This would need to be implemented with a proper Prometheus registry
		w.Write([]byte("# Prometheus metrics not yet implemented\n"))
	}).ServeHTTP(w, r)
}

// LogKBHit logs a knowledge base cache hit
func (t *Telemetry) LogKBHit(ctx context.Context, domain string) {
	t.metrics.RecordKBHit("worker", domain)
	t.logger.DebugContext(ctx, "kb_hit", "domain", domain)
}

// LogKBMiss logs a knowledge base cache miss
func (t *Telemetry) LogKBMiss(ctx context.Context, domain string) {
	t.metrics.RecordKBMiss("worker", domain)
	t.logger.DebugContext(ctx, "kb_miss", "domain", domain)
}

// LogWASMExecution logs a WASM execution
func (t *Telemetry) LogWASMExecution(ctx context.Context, domain string, duration time.Duration) {
	t.metrics.RecordWASMExecution("worker", "success")
	t.metrics.RecordWASMExecutionTime("worker", duration)
	t.logger.DebugContext(ctx, "wasm_execution", "domain", domain, "duration_ms", duration.Milliseconds())
}

// LogLLMCall logs an LLM API call
func (t *Telemetry) LogLLMCall(ctx context.Context, provider, model string, tokens int, duration time.Duration) {
	t.metrics.RecordRequest(provider, model, "success")
	t.metrics.RecordLatency(provider, model, duration)
	t.metrics.RecordTokens(provider, model, tokens, 0) // Assuming all tokens are input
	t.logger.DebugContext(ctx, "llm_call", "provider", provider, "model", model, "tokens", tokens, "duration_ms", duration.Milliseconds())
}
