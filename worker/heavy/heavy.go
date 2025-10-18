package heavy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/dsl"
	"github.com/snow-ghost/agent/pkg/llm/client"
	"github.com/snow-ghost/agent/prompts"
	"github.com/snow-ghost/agent/testkit"
	"github.com/snow-ghost/agent/worker/capabilities"
	"github.com/snow-ghost/agent/worker/common"
	"github.com/snow-ghost/agent/worker/metrics"
	"github.com/snow-ghost/agent/worker/telemetry"
)

// HeavyWorker implements the heavy worker type with design-based capabilities
type HeavyWorker struct {
	*common.BaseWorker
	designer client.Designer
	tests    core.TestRunner
	fitness  core.FitnessEvaluator
	critic   core.Critic
	mut      core.Mutator
}

// NewHeavyWorker creates a new heavy worker
func NewHeavyWorker(kb core.KnowledgeBase, designer client.Designer,
	tests core.TestRunner, fitness core.FitnessEvaluator, critic core.Critic,
	mut core.Mutator, telemetry *telemetry.Telemetry) *HeavyWorker {

	baseWorker := common.NewBaseWorker(kb, telemetry, "heavy")

	return &HeavyWorker{
		BaseWorker: baseWorker,
		designer:   designer,
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

// Solve processes a task using the design-based heavy worker pipeline
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

	// 2) Build user JSON and request design
	slog.InfoContext(ctx, "building user task JSON", "task_id", task.ID, "stage", "llm_design")

	// Build user JSON from task
	userJSON, err := h.buildUserTaskJSON(task)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build user task JSON", "error", err, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	// Request design from Designer
	slog.InfoContext(ctx, "requesting design", "task_id", task.ID, "stage", "llm_design")
	var hd design.HypothesisDesign
	workerType := "heavy"
	// measure llm_design stage
	metrics.WithLabeledStage(ctx, "llm_design", workerType, task.Domain, func(ctx context.Context) {
		var derr error
		hd, _, derr = h.designer.Design(ctx, userJSON)
		if derr != nil {
			err = derr
		}
	})
	if err != nil {
		metrics.IncDesignFail(ctx, workerType, task.Domain, "llm")
		slog.ErrorContext(ctx, "design request failed", "error", err, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	// 3) Validate design
	slog.InfoContext(ctx, "validating design", "task_id", task.ID, "stage", "dsl_parse")
	var vErr error
	metrics.WithLabeledStage(ctx, "dsl_parse", workerType, task.Domain, func(ctx context.Context) {
		vErr = design.Validate(hd)
	})
	if vErr != nil {
		metrics.IncDesignFail(ctx, workerType, task.Domain, "validation")
		slog.ErrorContext(ctx, "design validation failed", "error", vErr, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, fmt.Errorf("design validation failed: %w", vErr)
	}

	// Check if design indicates cannot_solve
	if hd.Status == "cannot_solve" {
		slog.InfoContext(ctx, "design indicates cannot solve", "task_id", task.ID, "reason", hd.Status)
		h.LogTaskEnd(ctx, task, core.Result{Success: false, Logs: "design_cannot_solve"}, time.Since(start), 0)
		return core.Result{Success: false, Logs: "design_cannot_solve"}, nil
	}

	// 4) Convert to hypothesis and test cases
	hypothesis, err := design.ToHypothesis(hd)
	if err != nil {
		slog.ErrorContext(ctx, "failed to convert design to hypothesis", "error", err, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	unitTests := design.ToTestCases(hd)
	slog.InfoContext(ctx, "converted design", "task_id", task.ID, "lang", hypothesis.Lang, "unit_tests", len(unitTests))

	// Log the generated AF-DSL code for debugging
	if hypothesis.Lang == "af-dsl" {
		slog.InfoContext(ctx, "generated AF-DSL code", "task_id", task.ID, "code", string(hypothesis.Bytes))
	}

	// 5) Generate property test cases if property plan exists
	var allTests []core.TestCase
	allTests = append(allTests, unitTests...)

	if len(hd.Tests.Property) > 0 {
		slog.InfoContext(ctx, "generating property test cases", "task_id", task.ID, "stage", "tests")
		propK := h.getPropertyTestCount()
		propTests := h.generatePropertyTests(hd.Tests.Property, propK)
		allTests = append(allTests, propTests...)
		slog.InfoContext(ctx, "generated property tests", "task_id", task.ID, "count", len(propTests))
	}

	// 6) Get appropriate interpreter based on language
	interpreter, err := h.getInterpreter(hypothesis.Lang)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get interpreter", "error", err, "task_id", task.ID, "lang", hypothesis.Lang)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	// 7) Run tests and evaluate fitness
	slog.InfoContext(ctx, "running tests", "task_id", task.ID, "stage", "tests")
	var testMetrics map[string]float64
	var pass bool
	metrics.WithLabeledStage(ctx, "tests", workerType, task.Domain, func(ctx context.Context) {
		testMetrics, pass, err = h.tests.Run(ctx, hypothesis, allTests, interpreter)
	})
	if err != nil {
		slog.ErrorContext(ctx, "test run failed", "error", err, "task_id", task.ID)
		h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
		return core.Result{Success: false}, err
	}

	// Log detailed test results if tests failed
	if !pass {
		if detailedRunner, ok := h.tests.(*testkit.DetailedRunner); ok {
			failedDetails := detailedRunner.GetFailedTestDetails()
			if len(failedDetails) > 0 {
				slog.WarnContext(ctx, "test failures details", "task_id", task.ID, "failed_tests", failedDetails)
			}
		}
	}

	// 8) Evaluate fitness and compare with PassThreshold
	slog.InfoContext(ctx, "evaluating fitness", "task_id", task.ID, "stage", "fitness")
	var score float64
	fitness := core.NewFitnessFromDesign(hypothesis.Meta)
	metrics.WithLabeledStage(ctx, "fitness", workerType, task.Domain, func(ctx context.Context) {
		score = fitness.Score(task, testMetrics, len(hypothesis.Bytes))
	})
	passThreshold, _ := strconv.ParseFloat(hypothesis.Meta["pass_threshold"], 64)
	passed := fitness.Passed(score, passThreshold)

	slog.InfoContext(ctx, "fitness evaluation complete", "task_id", task.ID,
		"score", score, "threshold", passThreshold, "passed", passed)

	// 9) If passed, execute and save as artifact
	if passed {
		slog.InfoContext(ctx, "fitness threshold passed, executing solution", "task_id", task.ID)
		res, err := interpreter.Execute(ctx, hypothesis, task)
		if err != nil {
			slog.ErrorContext(ctx, "solution execution failed", "error", err, "task_id", task.ID)
			h.LogTaskEnd(ctx, task, core.Result{Success: false}, time.Since(start), 0)
			return core.Result{Success: false}, err
		}

		if res.Success {
			// Save as artifact in KB
			_ = h.GetKB().SaveHypothesis(ctx, hypothesis, score)
			slog.InfoContext(ctx, "solution saved as artifact", "task_id", task.ID, "score", score)
			h.LogTaskEnd(ctx, task, res, time.Since(start), 1)
			return res, nil
		}
	}

	// 10) If not passed or execution failed, return failure
	failureReason := "fitness_threshold_not_met"
	if !pass {
		failureReason = "tests_failed"
	}

	slog.InfoContext(ctx, "task failed", "task_id", task.ID, "reason", failureReason, "score", score)
	h.LogTaskEnd(ctx, task, core.Result{Success: false, Logs: failureReason}, time.Since(start), 1)
	return core.Result{Success: false, Logs: failureReason}, nil
}

// buildUserTaskJSON builds user task JSON from core.Task
func (h *HeavyWorker) buildUserTaskJSON(task core.Task) (string, error) {
	// Extract input/output schemas from task spec props
	inSchema := "{}"
	outSchema := "{}"

	if inputSchema, ok := task.Spec.Props["input_schema"]; ok {
		inSchema = inputSchema
	}
	if outputSchema, ok := task.Spec.Props["output_schema"]; ok {
		outSchema = outputSchema
	}

	// Build examples from task spec props if available
	var examples []map[string]string
	if examplesStr, ok := task.Spec.Props["examples"]; ok {
		// Parse examples from JSON string
		var exampleList []map[string]interface{}
		if err := json.Unmarshal([]byte(examplesStr), &exampleList); err == nil {
			for _, example := range exampleList {
				exampleMap := make(map[string]string)
				for k, v := range example {
					exampleMap[k] = fmt.Sprintf("%v", v)
				}
				examples = append(examples, exampleMap)
			}
		}
	}

	// Build options
	opts := prompts.BuildOpts{
		TimeoutMS:     int(task.Budget.Timeout.Milliseconds()),
		MemMB:         task.Budget.MemMB,
		MaxComplexity: "medium",
	}

	return prompts.BuildUserTaskJSON(
		task.ID,
		task.Domain,
		task.Description,
		inSchema,
		outSchema,
		examples,
		opts,
	), nil
}

// getPropertyTestCount gets the number of property tests to generate from environment
func (h *HeavyWorker) getPropertyTestCount() int {
	if propK := os.Getenv("PROP_K"); propK != "" {
		if k, err := strconv.Atoi(propK); err == nil && k > 0 {
			return k
		}
	}
	return 64 // default
}

// generatePropertyTests generates property test cases from property plans
func (h *HeavyWorker) generatePropertyTests(propertyPlans []struct {
	Name      string   `json:"name"`
	Generator string   `json:"generator"`
	Checks    []string `json:"checks"`
}, count int) []core.TestCase {
	var testCases []core.TestCase

	for _, plan := range propertyPlans {
		// Generate test cases using the property test framework
		// This is a simplified implementation - in practice you'd use the full property test framework
		for i := 0; i < count/len(propertyPlans); i++ {
			// Generate random test data based on the generator
			var inputData []byte
			switch plan.Generator {
			case "list<int>(n<=100)":
				// Generate a random array of integers
				inputData = h.generateRandomIntArray(i, 100)
			default:
				// Fallback to a simple array
				inputData = []byte(fmt.Sprintf(`{"numbers": [%d, %d, %d]}`, i, i+1, i+2))
			}

			testCase := core.TestCase{
				Name:   fmt.Sprintf("%s_prop_%d", plan.Name, i),
				Input:  inputData,
				Checks: plan.Checks,
				Weight: 1.0,
			}
			testCases = append(testCases, testCase)
		}
	}

	return testCases
}

// generateRandomIntArray generates a random array of integers for property testing
func (h *HeavyWorker) generateRandomIntArray(seed, maxLen int) []byte {
	// Simple deterministic "random" generation based on seed
	length := (seed % maxLen) + 1
	if length == 0 {
		length = 1
	}

	var numbers []int
	for i := 0; i < length; i++ {
		// Generate numbers based on seed and position
		num := (seed*7 + i*11) % 100
		if num < 0 {
			num = -num
		}
		numbers = append(numbers, num)
	}

	// Convert to JSON
	jsonStr := fmt.Sprintf(`{"numbers": [%d`, numbers[0])
	for i := 1; i < len(numbers); i++ {
		jsonStr += fmt.Sprintf(`, %d`, numbers[i])
	}
	jsonStr += "]}"

	return []byte(jsonStr)
}

// getInterpreter returns the appropriate interpreter based on language
func (h *HeavyWorker) getInterpreter(lang string) (core.Interpreter, error) {
	switch lang {
	case "af-dsl":
		return dsl.NewAFDSLInterpreter(nil), nil
	case "wasm":
		// TODO: Return actual WASM interpreter when available
		return nil, fmt.Errorf("WASM interpreter not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}
