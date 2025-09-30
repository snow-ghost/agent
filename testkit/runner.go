package testkit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/snow-ghost/agent/core"
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

// Run executes each test case via the provided interpreter and aggregates metrics.
func (r *Runner) Run(ctx context.Context, h core.Hypothesis, cases []core.TestCase, exec core.Interpreter) (map[string]float64, bool, error) {
	metrics := map[string]float64{
		"cases_total":       0,
		"cases_passed":      0,
		"cases_failed":      0,
		"duration_ms_total": 0,
	}

	allPassed := true

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

		passed := false
		if err == nil {
			passed = evaluateCase(tc, task, res)
		}

		if passed {
			metrics["cases_passed"] += 1
		} else {
			metrics["cases_failed"] += 1
			allPassed = false
		}
	}

	return metrics, allPassed, nil
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
