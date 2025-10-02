package heavy

import (
	"context"
	"log/slog"
	"time"

	"github.com/snow-ghost/agent/core"
	llmmock "github.com/snow-ghost/agent/llm/mock"
	"github.com/snow-ghost/agent/worker/capabilities"
	"github.com/snow-ghost/agent/worker/common"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// HeavyWorker implements the heavy worker type with LLM+WASM capabilities
type HeavyWorker struct {
	*common.BaseWorker
	llm     core.LLMClient
	interp  core.Interpreter
	tests   core.TestRunner
	fitness core.FitnessEvaluator
	critic  core.Critic
	mut     core.Mutator
}

// NewHeavyWorker creates a new heavy worker
func NewHeavyWorker(kb core.KnowledgeBase, llm core.LLMClient, interp core.Interpreter,
	tests core.TestRunner, fitness core.FitnessEvaluator, critic core.Critic,
	mut core.Mutator, telemetry *telemetry.Telemetry) *HeavyWorker {

	baseWorker := common.NewBaseWorker(kb, telemetry, "heavy")

	return &HeavyWorker{
		BaseWorker: baseWorker,
		llm:        llm,
		interp:     interp,
		tests:      tests,
		fitness:    fitness,
		critic:     critic,
		mut:        mut,
	}
}

// Caps returns the capabilities of the heavy worker
func (h *HeavyWorker) Caps() capabilities.Capabilities {
	return capabilities.DefaultCapabilities("heavy")
}

// Solve processes a task using the full heavy worker pipeline
func (h *HeavyWorker) Solve(ctx context.Context, task core.Task) (core.Result, error) {
	start := time.Now()
	h.LogTaskStart(ctx, task)

	// 1) Try KB first
	result, err := h.TryKBSkills(ctx, task)
	if err != nil {
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}
	if result.Success {
		h.LogTaskEnd(ctx, task, result, time.Since(start), 0)
		return result, nil
	}

	// 2) Request LLM (algorithm, tests, criteria)
	slog.InfoContext(ctx, "requesting LLM proposal", "task_id", task.ID)
	algo, tests, criteria, err := h.llm.Propose(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "LLM proposal failed", "error", err, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	// Convert algorithm string to WASM bytecode
	var wasmBytes []byte
	if mockLLM, ok := h.llm.(*llmmock.MockLLM); ok {
		wasmBytes, err = mockLLM.GetWASMModule(algo)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get WASM module", "error", err, "task_id", task.ID)
			h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
			return core.Result{Success: false}, err
		}
	} else {
		// For other LLM implementations, assume algo is already bytecode
		wasmBytes = []byte(algo)
	}

	hypothesis := core.Hypothesis{ID: "llm-0", Source: "llm", Lang: "wasm", Bytes: wasmBytes, Meta: map[string]string{"criteria": "set"}}
	slog.InfoContext(ctx, "LLM proposal received", "tests_count", len(tests), "wasm_size", len(wasmBytes), "task_id", task.ID)

	// 3) Evolutionary mini-cycle
	best := hypothesis
	bestScore := -1.0
	deadline := time.Now().Add(task.Budget.Timeout)
	slog.InfoContext(ctx, "starting evolution", "deadline", deadline, "task_id", task.ID)

	iterations := 0
	for time.Now().Before(deadline) {
		iterations++
		candidates := append([]core.Hypothesis{hypothesis}, h.mut.Mutate(best)...)
		for _, c := range candidates {
			// attach criteria to task spec for checks
			task.Spec.SuccessCriteria = criteria
			metrics, pass, _ := h.tests.Run(ctx, c, tests, h.interp)
			score := h.fitness.Score(task, metrics, len(c.Bytes))
			if pass && score > bestScore {
				best, bestScore = c, score
			}
			ok, _ := h.critic.Accept(task, metrics)
			if ok {
				res, err := h.interp.Execute(ctx, c, task)
				if err == nil && res.Success {
					_ = h.GetKB().SaveHypothesis(ctx, c, score)
					h.LogTaskEnd(ctx, task, res, time.Since(start), iterations)
					return res, nil
				}
			}
		}
	}

	// If we found a good hypothesis, try to execute it and save it
	if bestScore > 0 {
		res, err := h.interp.Execute(ctx, best, task)
		if err == nil && res.Success {
			_ = h.GetKB().SaveHypothesis(ctx, best, bestScore)
			h.LogTaskEnd(ctx, task, res, time.Since(start), iterations)
			return res, nil
		}
	}

	h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), iterations)
	return core.Result{Success: false}, nil
}
