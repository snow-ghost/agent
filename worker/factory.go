package worker

import (
	"os"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	kbfs "github.com/snow-ghost/agent/kb/fs"
	kbmem "github.com/snow-ghost/agent/kb/memory"
	llmmock "github.com/snow-ghost/agent/llm/mock"
	"github.com/snow-ghost/agent/testkit"
	"github.com/snow-ghost/agent/worker/heavy"
	"github.com/snow-ghost/agent/worker/light"
	"github.com/snow-ghost/agent/worker/mutate"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// NewWorker creates a worker based on the WORKER_TYPE environment variable
func NewWorker(config *Config) (Worker, error) {
	workerType := WorkerType(os.Getenv("WORKER_TYPE"))
	if workerType == "" {
		workerType = WorkerTypeHeavy // default to heavy
	}

	// Create common components
	// Use artifact-based KB if artifacts directory is configured
	var kb core.KnowledgeBase
	if config.ArtifactsDir != "" {
		interp := wasm.NewInterpreter()
		kb = kbfs.NewArtifactKnowledgeBase(config.ArtifactsDir, interp)
	} else {
		// Fallback to memory-based KB
		kb = kbmem.NewRegistryWithDir(config.HypothesesDir)
	}
	telemetry := telemetry.NewTelemetry()

	switch workerType {
	case WorkerTypeLight:
		return light.NewLightWorker(kb, telemetry), nil

	case WorkerTypeHeavy:
		// Create heavy worker components
		llm := llmmock.NewMockLLM()
		interp := wasm.NewInterpreter()
		runner := testkit.NewRunner()
		fitness := core.NewWeightedFitness(map[string]float64{"cases_passed": 1.0, "cases_total": 0.0}, 0.0)
		critic := core.NewSimpleCritic()
		mut := mutate.NewSimpleMutator()

		return heavy.NewHeavyWorker(kb, llm, interp, runner, fitness, critic, mut, telemetry), nil

	default:
		// Default to heavy worker
		llm := llmmock.NewMockLLM()
		interp := wasm.NewInterpreter()
		runner := testkit.NewRunner()
		fitness := core.NewWeightedFitness(map[string]float64{"cases_passed": 1.0, "cases_total": 0.0}, 0.0)
		critic := core.NewSimpleCritic()
		mut := mutate.NewSimpleMutator()

		return heavy.NewHeavyWorker(kb, llm, interp, runner, fitness, critic, mut, telemetry), nil
	}
}
