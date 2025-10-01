package common

import (
	"context"
	"log/slog"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// BaseWorker provides common functionality for all worker types
type BaseWorker struct {
	kb         core.KnowledgeBase
	telemetry  *telemetry.Telemetry
	workerType string
}

// NewBaseWorker creates a new base worker with common functionality
func NewBaseWorker(kb core.KnowledgeBase, telemetry *telemetry.Telemetry, workerType string) *BaseWorker {
	return &BaseWorker{
		kb:         kb,
		telemetry:  telemetry,
		workerType: workerType,
	}
}

// Type returns the worker type
func (b *BaseWorker) Type() string {
	return b.workerType
}

// TryKBSkills attempts to solve the task using knowledge base skills
func (b *BaseWorker) TryKBSkills(ctx context.Context, task core.Task) (core.Result, error) {
	slog.InfoContext(ctx, "trying KB skills", "task_id", task.ID, "domain", task.Domain)

	skills := b.kb.Find(task)
	if len(skills) == 0 {
		slog.DebugContext(ctx, "no KB skills found", "task_id", task.ID)
		return core.Result{Success: false}, nil
	}

	slog.InfoContext(ctx, "found KB skills", "count", len(skills), "task_id", task.ID)

	for _, skill := range skills {
		result, err := skill.Execute(ctx, task)
		if err != nil {
			slog.WarnContext(ctx, "skill execution failed",
				"skill_id", skill.Name(), "error", err, "task_id", task.ID)
			continue
		}

		if result.Success {
			slog.InfoContext(ctx, "task solved by KB skill",
				"skill_id", skill.Name(), "score", result.Score, "task_id", task.ID)
			return result, nil
		}
	}

	slog.DebugContext(ctx, "no KB skills could solve task", "task_id", task.ID)
	return core.Result{Success: false}, nil
}

// LogTaskStart logs the start of task processing
func (b *BaseWorker) LogTaskStart(ctx context.Context, task core.Task) {
	b.telemetry.LogTaskStart(ctx, task)
}

// LogTaskEnd logs the end of task processing
func (b *BaseWorker) LogTaskEnd(ctx context.Context, task core.Task, result core.Result, duration time.Duration, iterations int) {
	b.telemetry.LogTaskEnd(ctx, task, result, duration, iterations)
}

// GetTelemetry returns the telemetry instance
func (b *BaseWorker) GetTelemetry() *telemetry.Telemetry {
	return b.telemetry
}

// GetKB returns the knowledge base instance
func (b *BaseWorker) GetKB() core.KnowledgeBase {
	return b.kb
}
