package light

import (
	"context"
	"log/slog"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/worker/common"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// LightWorker implements the light worker type with KB-only capabilities
type LightWorker struct {
	*common.BaseWorker
}

// NewLightWorker creates a new light worker
func NewLightWorker(kb core.KnowledgeBase, telemetry *telemetry.Telemetry) *LightWorker {
	baseWorker := common.NewBaseWorker(kb, telemetry, "light")

	return &LightWorker{
		BaseWorker: baseWorker,
	}
}

// Solve processes a task using only the knowledge base
func (l *LightWorker) Solve(ctx context.Context, task core.Task) (core.Result, error) {
	start := time.Now()
	l.LogTaskStart(ctx, task)

	// Light worker only uses KB skills - no LLM or WASM
	slog.InfoContext(ctx, "light worker processing task", "task_id", task.ID, "domain", task.Domain)

	result, err := l.TryKBSkills(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "KB skill execution failed", "error", err, "task_id", task.ID)
		l.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	if result.Success {
		slog.InfoContext(ctx, "task solved by light worker", "task_id", task.ID, "score", result.Score)
	} else {
		slog.InfoContext(ctx, "task not solvable by light worker", "task_id", task.ID)
		// Light worker cannot solve tasks that require LLM/WASM
		result = core.Result{
			Success: false,
			Logs:    "Task requires heavy worker (LLM+WASM) capabilities",
			Metrics: map[string]float64{
				"worker_type":    1.0, // light worker
				"requires_heavy": 1.0,
			},
		}
	}

	l.LogTaskEnd(ctx, task, result, time.Since(start), 0)
	return result, nil
}
