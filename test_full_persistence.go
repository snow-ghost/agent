package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	kbmem "github.com/snow-ghost/agent/kb/memory"
	llmmock "github.com/snow-ghost/agent/llm/mock"
	"github.com/snow-ghost/agent/testkit"
	"github.com/snow-ghost/agent/worker"
	"github.com/snow-ghost/agent/worker/mutate"
)

func main() {
	// Create a temporary directory for hypotheses
	tempDir := "./test_hypotheses"
	os.RemoveAll(tempDir) // Clean up

	// Wire components
	kb := kbmem.NewRegistryWithDir(tempDir)
	llm := llmmock.NewMockLLM()
	interp := wasm.NewInterpreter()
	defer interp.Close(context.Background())
	runner := testkit.NewRunner()
	fitness := core.NewWeightedFitness(map[string]float64{"cases_passed": 1.0, "cases_total": 0.0}, 0.0)
	critic := core.NewSimpleCritic()
	mut := mutate.NewSimpleMutator()

	// Create solver
	solver := &worker.Solver{KB: kb, LLM: llm, Interp: interp, Tests: runner, Fitness: fitness, Critic: critic, Mut: mut}

	// Create test task that won't match built-in skills
	task := core.Task{
		ID:     "test-persistence-1",
		Domain: "custom",
		Spec: core.Spec{
			SuccessCriteria: []string{"sorted_non_decreasing"},
			Props:           map[string]string{"type": "numbers"},
			MetricsWeights:  map[string]float64{"cases_passed": 1.0, "cases_total": 0.0},
		},
		Input:     []byte(`{"numbers": [3,1,2]}`),
		Budget:    core.Budget{CPUMillis: 1000, Timeout: time.Second},
		CreatedAt: time.Now(),
	}

	fmt.Println("=== First task (should use LLM and fail) ===")
	result1, err := solver.Solve(context.Background(), task)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result 1: Success=%v, Score=%f\n", result1.Success, result1.Score)
	}

	// Manually save a hypothesis to simulate a successful LLM result
	fmt.Println("\n=== Manually saving a hypothesis ===")
	hypothesis := core.Hypothesis{
		ID:     "manual-hypothesis-1",
		Source: "llm",
		Lang:   "wasm",
		Bytes:  []byte("test wasm bytecode"),
		Meta:   map[string]string{"test": "true", "domain": "custom"},
	}

	err = kb.SaveHypothesis(context.Background(), hypothesis, 0.9)
	if err != nil {
		fmt.Printf("Error saving hypothesis: %v\n", err)
		return
	}
	fmt.Println("Hypothesis saved successfully!")

	// Check saved files
	fmt.Println("\n=== Checking saved files ===")
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
	} else {
		fmt.Printf("Found %d files:\n", len(entries))
		for _, entry := range entries {
			fmt.Printf("  - %s\n", entry.Name())
		}
	}

	// List skills in KB
	fmt.Println("\n=== Skills in KB ===")
	skills := kb.ListSkills()
	for _, skill := range skills {
		fmt.Printf("  - %s (domain: %s)\n", skill.Name(), skill.Domain())
	}

	// Test second similar task
	task2 := core.Task{
		ID:     "test-persistence-2",
		Domain: "custom",
		Spec: core.Spec{
			SuccessCriteria: []string{"sorted_non_decreasing"},
			Props:           map[string]string{"type": "numbers"},
			MetricsWeights:  map[string]float64{"cases_passed": 1.0, "cases_total": 0.0},
		},
		Input:     []byte(`{"numbers": [5,2,8,1]}`),
		Budget:    core.Budget{CPUMillis: 1000, Timeout: time.Second},
		CreatedAt: time.Now(),
	}

	fmt.Println("\n=== Second task (should use saved hypothesis) ===")
	result2, err := solver.Solve(context.Background(), task2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result 2: Success=%v, Score=%f\n", result2.Success, result2.Score)
	}

	// Clean up
	os.RemoveAll(tempDir)
	fmt.Println("\n=== Test completed ===")
}
