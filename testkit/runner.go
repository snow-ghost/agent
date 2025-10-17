package testkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/snow-ghost/agent/core"
	workermetrics "github.com/snow-ghost/agent/worker/metrics"
)

// GenerateSortCasesFixed returns a fixed set of sorting property test cases.
// Input format: {"numbers": [..]}
// Oracle format: {"sorted": [..]}
func GenerateSortCasesFixed() []core.TestCase {
	cases := []core.TestCase{
		{
			Name:   "sorted_small",
			Input:  []byte(`{"numbers": [3,1,2]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
		{
			Name:   "sorted_with_dupes",
			Input:  []byte(`{"numbers": [5,1,1,4]}`),
			Oracle: []byte(`{"sorted": [1,1,4,5]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
	}
	return cases
}

// Runner implements core.TestRunner
type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

// DetailedRunner extends Runner with detailed test failure information
type DetailedRunner struct {
	*Runner
	testResults []testResult
}

func NewDetailedRunner() *DetailedRunner {
	return &DetailedRunner{
		Runner: NewRunner(),
	}
}

// Run executes tests and stores detailed results
func (r *DetailedRunner) Run(ctx context.Context, h core.Hypothesis, cases []core.TestCase, exec core.Interpreter) (map[string]float64, bool, error) {
	// Call the parent Run method but store test results
	metrics := map[string]float64{
		"cases_total":       0,
		"cases_passed":      0,
		"cases_failed":      0,
		"duration_ms_total": 0,
		"passed_weight":     0,
		"total_weight":      0,
		"unit_pass":         0,
		"prop_pass":         0,
	}

	allPassed := true
	var durations []float64
	r.testResults = []testResult{}

	for _, tc := range cases {
		start := time.Now()

		task := core.Task{
			ID:     "case:" + tc.Name,
			Domain: "algorithms",
			Spec:   core.Spec{SuccessCriteria: tc.Checks},
			Input:  json.RawMessage(tc.Input),
		}

		res, err := exec.Execute(ctx, h, task)
		durMs := float64(time.Since(start).Milliseconds())
		metrics["duration_ms_total"] += durMs
		metrics["cases_total"] += 1
		metrics["total_weight"] += tc.Weight

		passed := false
		var errorMsg string
		if err == nil {
			passed, errorMsg = evaluateCaseWithDetails(tc, task, res)
		} else {
			errorMsg = fmt.Sprintf("execution error: %v", err)
		}

		// Store detailed test results
		r.testResults = append(r.testResults, testResult{
			testCase: tc,
			passed:   passed,
			errorMsg: errorMsg,
		})

		if passed {
			metrics["cases_passed"] += 1
			metrics["passed_weight"] += tc.Weight
			workermetrics.ObserveTest(ctx, "pass", float64(time.Since(start).Seconds()))
		} else {
			metrics["cases_failed"] += 1
			allPassed = false
			if err != nil {
				workermetrics.ObserveTest(ctx, "error", float64(time.Since(start).Seconds()))
			} else {
				workermetrics.ObserveTest(ctx, "fail", float64(time.Since(start).Seconds()))
			}
		}

		durations = append(durations, durMs)
	}

	// Calculate correctness = passed_weight / total_weight
	if metrics["total_weight"] > 0 {
		metrics["correctness"] = metrics["passed_weight"] / metrics["total_weight"]
	} else {
		metrics["correctness"] = 0
	}

	// Calculate time score = 1/(1+avg_ms/ceil)
	avgMs := metrics["duration_ms_total"] / float64(len(cases))
	ceil := 1000.0 // 1 second ceiling
	metrics["time"] = 1.0 / (1.0 + avgMs/ceil)

	// Calculate size score = 1/(1+nodes) where nodes = AST node count
	astNodes := countASTNodes(h.Bytes)
	metrics["size"] = 1.0 / (1.0 + float64(astNodes))

	// Calculate unit vs property pass rates
	unitPass, propPass := calculateUnitPropertyPassFromResults(r.testResults)
	metrics["unit_pass"] = unitPass
	metrics["prop_pass"] = propPass

	return metrics, allPassed, nil
}

// GetFailedTestDetails returns detailed information about failed tests
func (r *DetailedRunner) GetFailedTestDetails() []string {
	return GetFailedTestDetails(r.testResults)
}

// Run executes each test case via the provided interpreter and aggregates metrics.
func (r *Runner) Run(ctx context.Context, h core.Hypothesis, cases []core.TestCase, exec core.Interpreter) (map[string]float64, bool, error) {
	metrics := map[string]float64{
		"cases_total":       0,
		"cases_passed":      0,
		"cases_failed":      0,
		"duration_ms_total": 0,
		"passed_weight":     0,
		"total_weight":      0,
		"unit_pass":         0,
		"prop_pass":         0,
	}

	allPassed := true
	var durations []float64
	var testResults []testResult

	for _, tc := range cases {
		start := time.Now()

		task := core.Task{
			ID:     "case:" + tc.Name,
			Domain: "algorithms",
			Spec:   core.Spec{SuccessCriteria: tc.Checks},
			Input:  json.RawMessage(tc.Input),
		}

		res, err := exec.Execute(ctx, h, task)
		durMs := float64(time.Since(start).Milliseconds())
		metrics["duration_ms_total"] += durMs
		metrics["cases_total"] += 1
		metrics["total_weight"] += tc.Weight

		passed := false
		var errorMsg string
		if err == nil {
			passed, errorMsg = evaluateCaseWithDetails(tc, task, res)
		} else {
			errorMsg = fmt.Sprintf("execution error: %v", err)
		}

		// Track individual test results for unit/property calculation
		testResults = append(testResults, testResult{
			testCase: tc,
			passed:   passed,
			errorMsg: errorMsg,
		})

		if passed {
			metrics["cases_passed"] += 1
			metrics["passed_weight"] += tc.Weight
			workermetrics.ObserveTest(ctx, "pass", float64(time.Since(start).Seconds()))
		} else {
			metrics["cases_failed"] += 1
			allPassed = false
			// If there was an execution error, mark as error, else fail
			if err != nil {
				workermetrics.ObserveTest(ctx, "error", float64(time.Since(start).Seconds()))
			} else {
				workermetrics.ObserveTest(ctx, "fail", float64(time.Since(start).Seconds()))
			}
		}

		durations = append(durations, durMs)
	}

	// Calculate correctness = passed_weight / total_weight
	if metrics["total_weight"] > 0 {
		metrics["correctness"] = metrics["passed_weight"] / metrics["total_weight"]
	} else {
		metrics["correctness"] = 0
	}

	// Calculate time score = 1/(1+avg_ms/ceil)
	avgMs := metrics["duration_ms_total"] / float64(len(cases))
	ceil := 1000.0 // 1 second ceiling
	metrics["time"] = 1.0 / (1.0 + avgMs/ceil)

	// Calculate size score = 1/(1+nodes) where nodes = AST node count
	astNodes := countASTNodes(h.Bytes)
	metrics["size"] = 1.0 / (1.0 + float64(astNodes))

	// Calculate unit vs property pass rates
	unitPass, propPass := calculateUnitPropertyPassFromResults(testResults)
	metrics["unit_pass"] = unitPass
	metrics["prop_pass"] = propPass

	return metrics, allPassed, nil
}

// GetFailedTestDetails returns detailed information about failed tests
func GetFailedTestDetails(testResults []testResult) []string {
	var details []string
	for _, result := range testResults {
		if !result.passed {
			detail := fmt.Sprintf("Test '%s' failed: %s", result.testCase.Name, result.errorMsg)
			details = append(details, detail)
		}
	}
	return details
}

type testResult struct {
	testCase core.TestCase
	passed   bool
	errorMsg string
}

// evaluateCase validates the output against the oracle and checks.
func evaluateCase(tc core.TestCase, task core.Task, res core.Result) bool {
	// If Oracle is provided, require exact JSON equality (semantic)
	if len(tc.Oracle) > 0 {
		var want, got any
		if err := json.Unmarshal(tc.Oracle, &want); err != nil {
			return false
		}
		if err := json.Unmarshal(res.Output, &got); err != nil {
			return false
		}
		if !deepEqualJSON(want, got) {
			return false
		}
	}

	// Property checks
	for _, check := range tc.Checks {
		switch check {
		case "sorted_non_decreasing":
			if !checkSorted(res.Output) {
				return false
			}
		case "permutes":
			if !checkPermutes(tc.Input, res.Output) {
				return false
			}
		}
	}
	return true
}

// evaluateCaseWithDetails validates the output and returns detailed error information
func evaluateCaseWithDetails(tc core.TestCase, task core.Task, res core.Result) (bool, string) {
	var errors []string

	// If Oracle is provided, require exact JSON equality (semantic)
	if len(tc.Oracle) > 0 {
		var want, got any
		if err := json.Unmarshal(tc.Oracle, &want); err != nil {
			errors = append(errors, fmt.Sprintf("oracle JSON parse error: %v", err))
		} else if err := json.Unmarshal(res.Output, &got); err != nil {
			errors = append(errors, fmt.Sprintf("output JSON parse error: %v", err))
		} else if !deepEqualJSON(want, got) {
			wantStr, _ := json.MarshalIndent(want, "", "  ")
			gotStr, _ := json.MarshalIndent(got, "", "  ")
			errors = append(errors, fmt.Sprintf("oracle mismatch:\nExpected: %s\nGot: %s", string(wantStr), string(gotStr)))
		}
	}

	// Property checks
	for _, check := range tc.Checks {
		switch check {
		case "sorted_non_decreasing":
			if !checkSorted(res.Output) {
				errors = append(errors, "sorted_non_decreasing check failed")
			}
		case "permutes":
			if !checkPermutes(tc.Input, res.Output) {
				errors = append(errors, "permutes check failed")
			}
		}
	}

	if len(errors) > 0 {
		return false, strings.Join(errors, "; ")
	}
	return true, ""
}

func deepEqualJSON(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func extractNumbers(data []byte, field string) ([]float64, bool) {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, false
	}
	v, ok := obj[field]
	if !ok {
		return nil, false
	}
	switch arr := v.(type) {
	case []any:
		nums := make([]float64, 0, len(arr))
		for _, it := range arr {
			switch n := it.(type) {
			case float64:
				nums = append(nums, n)
			case int:
				nums = append(nums, float64(n))
			default:
				return nil, false
			}
		}
		return nums, true
	default:
		return nil, false
	}
}

func checkSorted(output []byte) bool {
	nums, ok := extractNumbers(output, "sorted")
	if !ok {
		return false
	}
	for i := 1; i < len(nums); i++ {
		if nums[i] < nums[i-1] {
			return false
		}
	}
	return true
}

func checkPermutes(input, output []byte) bool {
	in, ok1 := extractNumbers(input, "numbers")
	out, ok2 := extractNumbers(output, "sorted")
	if !ok1 || !ok2 || len(in) != len(out) {
		return false
	}
	count := map[float64]int{}
	for _, n := range in {
		count[n]++
	}
	for _, n := range out {
		count[n]--
	}
	for _, v := range count {
		if v != 0 {
			return false
		}
	}
	return true
}

// countASTNodes estimates the number of AST nodes in the hypothesis bytecode
func countASTNodes(bytes []byte) int {
	// Simple heuristic: count parentheses and symbols in AF-DSL
	content := string(bytes)
	parens := strings.Count(content, "(") + strings.Count(content, ")")

	// Split by spaces and parentheses to get individual tokens
	// Replace parentheses with spaces, then split
	normalized := strings.ReplaceAll(content, "(", " ")
	normalized = strings.ReplaceAll(normalized, ")", " ")
	words := strings.Fields(normalized)

	return parens + len(words)
}

// calculateUnitPropertyPassFromResults calculates pass rates for unit vs property tests
func calculateUnitPropertyPassFromResults(results []testResult) (float64, float64) {
	unitPassed := 0.0
	unitTotal := 0.0
	propPassed := 0.0
	propTotal := 0.0

	for _, result := range results {
		tc := result.testCase
		// Determine if this is a unit test (has oracle) or property test (has checks)
		isUnit := len(tc.Oracle) > 0
		isProperty := len(tc.Checks) > 0

		if isUnit {
			unitTotal += tc.Weight
			if result.passed {
				unitPassed += tc.Weight
			}
		}

		if isProperty {
			propTotal += tc.Weight
			if result.passed {
				propPassed += tc.Weight
			}
		}
	}

	unitPass := 0.0
	if unitTotal > 0 {
		unitPass = unitPassed / unitTotal
	}

	propPass := 0.0
	if propTotal > 0 {
		propPass = propPassed / propTotal
	}

	return unitPass, propPass
}
