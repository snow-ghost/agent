// worker/solver.go
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/snow-ghost/agent/core"
	llmmock "github.com/snow-ghost/agent/llm/mock"
)

type Solver struct {
	KB      core.KnowledgeBase
	LLM     core.LLMClient
	Interp  core.Interpreter
	Tests   core.TestRunner
	Fitness core.FitnessEvaluator
	Critic  core.Critic
	Mut     core.Mutator
	Policy  core.PolicyGuard
}

func (s *Solver) Solve(ctx context.Context, task core.Task) (core.Result, error) {
	slog.InfoContext(ctx, "solving task", "task_id", task.ID, "domain", task.Domain)

	// 1) Try KB first
	if skills := s.KB.Find(task); len(skills) > 0 {
		slog.InfoContext(ctx, "found KB skills", "count", len(skills))
		for _, sk := range skills {
			res, err := sk.Execute(ctx, task)
			if err == nil && res.Success {
				slog.InfoContext(ctx, "task solved by KB skill", "skill_id", sk.Name())
				return res, nil
			}
		}
	}
	// 2) Request LLM (algorithm, tests, criteria)
	slog.InfoContext(ctx, "requesting LLM proposal")
	algo, tests, criteria, err := s.LLM.Propose(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "LLM proposal failed", "error", err)
		return core.Result{Success: false}, err
	}

	// Convert algorithm string to WASM bytecode
	var wasmBytes []byte
	if mockLLM, ok := s.LLM.(*llmmock.MockLLM); ok {
		wasmBytes, err = mockLLM.GetWASMModule(algo)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get WASM module", "error", err)
			return core.Result{Success: false}, err
		}
	} else {
		// For other LLM implementations, assume algo is already bytecode
		wasmBytes = []byte(algo)
	}

	h := core.Hypothesis{ID: "llm-0", Source: "llm", Lang: "wasm", Bytes: wasmBytes, Meta: map[string]string{"criteria": "set"}}
	slog.InfoContext(ctx, "LLM proposal received", "tests_count", len(tests), "wasm_size", len(wasmBytes))

	// 3) Evolutionary mini-cycle
	best := h
	bestScore := -1.0
	deadline := time.Now().Add(task.Budget.Timeout)
	slog.InfoContext(ctx, "starting evolution", "deadline", deadline)

	for iter := 0; time.Now().Before(deadline); iter++ {
		candidates := append([]core.Hypothesis{h}, s.Mut.Mutate(best)...)
		for _, c := range candidates {
			// attach criteria to task spec for checks
			task.Spec.SuccessCriteria = criteria
			metrics, pass, _ := s.Tests.Run(ctx, c, tests, s.Interp)
			score := s.Fitness.Score(task, metrics, len(c.Bytes))
			if pass && score > bestScore {
				best, bestScore = c, score
			}
			ok, _ := s.Critic.Accept(task, metrics)
			if ok {
				res, err := s.Interp.Execute(ctx, c, task)
				if err == nil && res.Success {
					_ = s.KB.SaveHypothesis(ctx, c, score)
					return res, nil
				}
			}
		}
	}

	// If we found a good hypothesis, try to execute it and save it
	if bestScore > 0 {
		res, err := s.Interp.Execute(ctx, best, task)
		if err == nil && res.Success {
			_ = s.KB.SaveHypothesis(ctx, best, bestScore)
			return res, nil
		}
	}

	return core.Result{Success: false}, nil
}
