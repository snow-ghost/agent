package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	kbmem "github.com/snow-ghost/agent/kb/memory"
	llmmock "github.com/snow-ghost/agent/llm/mock"
	"github.com/snow-ghost/agent/testkit"
	"github.com/snow-ghost/agent/worker"
	"github.com/snow-ghost/agent/worker/mutate"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Wire components
	kb := kbmem.NewRegistry()
	llm := llmmock.NewMockLLM()
	interp := wasm.NewInterpreter()
	defer interp.Close(context.Background())
	runner := testkit.NewRunner()
	fitness := core.NewWeightedFitness(map[string]float64{"cases_passed": 1.0, "cases_total": 0.0}, 0.0)
	critic := core.NewSimpleCritic()
	mut := mutate.NewSimpleMutator()

	// Create solver with telemetry
	solver := &worker.Solver{KB: kb, LLM: llm, Interp: interp, Tests: runner, Fitness: fitness, Critic: critic, Mut: mut}
	instrumentedSolver := worker.NewInstrumentedSolver(solver)
	ing := worker.NewIngestor(instrumentedSolver.Solve)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/solve", ing)
	mux.Handle("/health", http.HandlerFunc(instrumentedSolver.GetTelemetry().HealthHandler))
	mux.Handle("/metrics", http.HandlerFunc(instrumentedSolver.GetTelemetry().MetricsHandler))

	logger.Info("worker starting", "port", "8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
