package worker

import (
	"context"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// Worker interface defines the contract for all worker types
type Worker interface {
	// Solve processes a task and returns a result
	Solve(ctx context.Context, task core.Task) (core.Result, error)

	// Type returns the worker type (e.g., "heavy", "light")
	Type() string
}

// WorkerType represents the type of worker
type WorkerType string

const (
	WorkerTypeHeavy WorkerType = "heavy"
	WorkerTypeLight WorkerType = "light"
)

// WorkerConfig holds configuration for worker creation
type WorkerConfig struct {
	Type      WorkerType
	KB        core.KnowledgeBase
	LLM       core.LLMClient
	Interp    core.Interpreter
	Tests     core.TestRunner
	Fitness   core.FitnessEvaluator
	Critic    core.Critic
	Mut       core.Mutator
	Policy    core.PolicyGuard
	Telemetry *telemetry.Telemetry
}
