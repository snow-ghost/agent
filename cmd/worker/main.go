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
	// Load configuration
	config := worker.LoadConfig()

	// Setup structured logging
	logLevel := parseLogLevel(config.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Wire components
	kb := kbmem.NewRegistryWithDir(config.HypothesesDir)
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

	logger.Info("worker starting",
		"port", config.WorkerPort,
		"llm_mode", config.LLMMode,
		"hypotheses_dir", config.HypothesesDir,
		"log_level", config.LogLevel)

	log.Fatal(http.ListenAndServe(":"+config.WorkerPort, mux))
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
