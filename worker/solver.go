// worker/solver.go
package worker

import (
	"context"
	"time"

	"github.com/snow-ghost/agent/core"
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
	// 1) Try KB first
	if skills := s.KB.Find(task); len(skills) > 0 {
		for _, sk := range skills {
			res, err := sk.Execute(ctx, task)
			if err == nil && res.Success {
				return res, nil
			}
		}
	}
	// 2) Request LLM (algorithm, tests, criteria)
	algo, tests, _, err := s.LLM.Propose(ctx, task)
	if err != nil {
		return core.Result{Success: false}, err
	}
	h := core.Hypothesis{ID: "llm-0", Source: "llm", Lang: "wasm-ir", Bytes: []byte(algo)}

	// 3) Evolutionary mini-cycle
	best := h
	bestScore := -1.0
	deadline := time.Now().Add(task.Budget.Timeout)

	for iter := 0; time.Now().Before(deadline); iter++ {
		candidates := append([]core.Hypothesis{h}, s.Mut.Mutate(best)...)
		for _, c := range candidates {
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
	return core.Result{Success: false}, nil
}
