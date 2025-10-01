package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/snow-ghost/agent/core"
	kbmem "github.com/snow-ghost/agent/kb/memory"
)

func main() {
	// Create a temporary directory for hypotheses
	tempDir := "./test_hypotheses"
	os.RemoveAll(tempDir) // Clean up

	// Create registry
	kb := kbmem.NewRegistryWithDir(tempDir)

	// Create a test hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis-1",
		Source: "llm",
		Lang:   "wasm",
		Bytes:  []byte("test wasm bytecode"),
		Meta:   map[string]string{"test": "true"},
	}

	fmt.Println("=== Testing SaveHypothesis ===")
	err := kb.SaveHypothesis(context.Background(), hypothesis, 0.95)
	if err != nil {
		fmt.Printf("Error saving hypothesis: %v\n", err)
		return
	}
	fmt.Println("Hypothesis saved successfully!")

	// Check if files were created
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

	// Test if the saved hypothesis can solve a task
	fmt.Println("\n=== Testing saved hypothesis execution ===")
	task := core.Task{
		ID:     "test-task",
		Domain: "general",
		Spec: core.Spec{
			Props: map[string]string{"type": "numbers"},
		},
		Input:     []byte(`[3,1,2]`),
		Budget:    core.Budget{CPUMillis: 1000, Timeout: time.Second},
		CreatedAt: time.Now(),
	}

	// Find skills for this task
	foundSkills := kb.Find(task)
	fmt.Printf("Found %d skills for task:\n", len(foundSkills))
	for _, skill := range foundSkills {
		fmt.Printf("  - %s\n", skill.Name())

		// Test if skill can solve the task
		canSolve, confidence := skill.CanSolve(task)
		fmt.Printf("    Can solve: %v (confidence: %f)\n", canSolve, confidence)

		if canSolve {
			// Try to execute the skill
			result, err := skill.Execute(context.Background(), task)
			if err != nil {
				fmt.Printf("    Execution error: %v\n", err)
			} else {
				fmt.Printf("    Execution result: Success=%v, Score=%f\n", result.Success, result.Score)
				fmt.Printf("    Logs: %s\n", result.Logs)
			}
		}
	}

	// Clean up
	os.RemoveAll(tempDir)
	fmt.Println("\n=== Test completed ===")
}
