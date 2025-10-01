package telemetry

import (
	"context"
	"expvar"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/snow-ghost/agent/core"
)

// Telemetry collects basic metrics and provides structured logging
type Telemetry struct {
	mu sync.RWMutex

	// Metrics
	TasksTotal      *expvar.Int
	TasksSolved     *expvar.Int
	TasksFailed     *expvar.Int
	IterationsTotal *expvar.Int
	TestPassRate    *expvar.Float
	AvgSolveTime    *expvar.Float

	// Internal state for calculations
	totalSolveTime time.Duration
	totalTests     int
	passedTests    int

	logger *slog.Logger
}

// NewTelemetry creates a new telemetry instance
func NewTelemetry() *Telemetry {
	t := &Telemetry{
		TasksTotal:      expvar.NewInt("tasks_total"),
		TasksSolved:     expvar.NewInt("tasks_solved"),
		TasksFailed:     expvar.NewInt("tasks_failed"),
		IterationsTotal: expvar.NewInt("iterations_total"),
		TestPassRate:    expvar.NewFloat("test_pass_rate"),
		AvgSolveTime:    expvar.NewFloat("avg_solve_time_ms"),
		logger:          slog.Default(),
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
}

// LogTaskEnd logs the end of a task with result
func (t *Telemetry) LogTaskEnd(ctx context.Context, task core.Task, result core.Result, duration time.Duration, iterations int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.TasksTotal.Add(1)
	t.IterationsTotal.Add(int64(iterations))
	t.totalSolveTime += duration

	if result.Success {
		t.TasksSolved.Add(1)
		t.logger.InfoContext(ctx, "task_solved",
			"task_id", task.ID,
			"duration_ms", duration.Milliseconds(),
			"iterations", iterations,
			"score", result.Score,
		)
	} else {
		t.TasksFailed.Add(1)
		t.logger.WarnContext(ctx, "task_failed",
			"task_id", task.ID,
			"duration_ms", duration.Milliseconds(),
			"iterations", iterations,
		)
	}

	// Update averages
	if t.TasksTotal.Value() > 0 {
		t.AvgSolveTime.Set(float64(t.totalSolveTime.Milliseconds()) / float64(t.TasksTotal.Value()))
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
		t.TestPassRate.Set(float64(t.passedTests) / float64(t.totalTests))
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

// MetricsHandler returns metrics in expvar format
func (t *Telemetry) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	expvar.Handler().ServeHTTP(w, r)
}
