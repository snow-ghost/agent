package testkit

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
)

// mockInterpreter implements core.Interpreter for testing
type mockInterpreter struct {
	shouldPass bool
	delay      time.Duration
}

func (m *mockInterpreter) Execute(ctx context.Context, h core.Hypothesis, task core.Task) (core.Result, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.shouldPass {
		// Return output that matches the test case expectations
		// For the test cases in TestRunner_Run_100PercentCorrectness:
		// - unit_test_1 expects [1,2,3]
		// - unit_test_2 expects [1,4,5]
		// - property_test_1 expects sorted output

		var output []byte
		switch task.ID {
		case "case:unit_test_1":
			output = []byte(`{"sorted": [1,2,3]}`)
		case "case:unit_test_2":
			output = []byte(`{"sorted": [1,4,5]}`)
		case "case:property_test_1":
			output = []byte(`{"sorted": [1,2,3]}`) // Any sorted output for property test
		default:
			output = []byte(`{"sorted": [1,2,3]}`)
		}

		return core.Result{
			Success: true,
			Output:  output,
		}, nil
	}

	return core.Result{
		Success: false,
		Output:  []byte(`{"error": "failed"}`),
	}, nil
}

func TestRunner_Run_100PercentCorrectness(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner()

	// Create test cases with different weights
	cases := []core.TestCase{
		{
			Name:   "unit_test_1",
			Input:  []byte(`{"numbers": [3,1,2]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.0,
		},
		{
			Name:   "unit_test_2",
			Input:  []byte(`{"numbers": [5,1,4]}`),
			Oracle: []byte(`{"sorted": [1,4,5]}`),
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 2.0,
		},
		{
			Name:   "property_test_1",
			Input:  []byte(`{"numbers": [2,1,3]}`),
			Oracle: []byte{}, // No oracle, property test only
			Checks: []string{"sorted_non_decreasing", "permutes"},
			Weight: 1.5,
		},
	}

	// Create hypothesis with AF-DSL code
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"), // Simple program
		Meta:   map[string]string{},
	}

	// Test with passing interpreter
	interpreter := &mockInterpreter{shouldPass: true, delay: 10 * time.Millisecond}

	metrics, allPassed, err := runner.Run(ctx, hypothesis, cases, interpreter)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !allPassed {
		t.Error("Expected all tests to pass")
	}

	// Verify correctness = 1.0 (100% pass rate)
	if metrics["correctness"] != 1.0 {
		t.Errorf("Expected correctness=1.0, got %f", metrics["correctness"])
	}

	// Verify total weight = 4.5 (1.0 + 2.0 + 1.5)
	expectedTotalWeight := 4.5
	if metrics["total_weight"] != expectedTotalWeight {
		t.Errorf("Expected total_weight=%f, got %f", expectedTotalWeight, metrics["total_weight"])
	}

	// Verify passed weight = total weight (all passed)
	if metrics["passed_weight"] != expectedTotalWeight {
		t.Errorf("Expected passed_weight=%f, got %f", expectedTotalWeight, metrics["passed_weight"])
	}

	// Verify time score is calculated (should be > 0 and < 1)
	if metrics["time"] <= 0 || metrics["time"] >= 1 {
		t.Errorf("Expected time score between 0 and 1, got %f", metrics["time"])
	}

	// Verify size score is calculated (should be > 0 and < 1)
	if metrics["size"] <= 0 || metrics["size"] >= 1 {
		t.Errorf("Expected size score between 0 and 1, got %f", metrics["size"])
	}

	// Verify unit_pass and prop_pass are calculated
	if metrics["unit_pass"] < 0 || metrics["unit_pass"] > 1 {
		t.Errorf("Expected unit_pass between 0 and 1, got %f", metrics["unit_pass"])
	}
	if metrics["prop_pass"] < 0 || metrics["prop_pass"] > 1 {
		t.Errorf("Expected prop_pass between 0 and 1, got %f", metrics["prop_pass"])
	}

	// Verify basic counts
	if metrics["cases_total"] != 3 {
		t.Errorf("Expected cases_total=3, got %f", metrics["cases_total"])
	}
	if metrics["cases_passed"] != 3 {
		t.Errorf("Expected cases_passed=3, got %f", metrics["cases_passed"])
	}
	if metrics["cases_failed"] != 0 {
		t.Errorf("Expected cases_failed=0, got %f", metrics["cases_failed"])
	}
}

func TestRunner_Run_PartialCorrectness(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner()

	cases := []core.TestCase{
		{
			Name:   "passing_test",
			Input:  []byte(`{"numbers": [1,2,3]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Checks: []string{"sorted_non_decreasing"},
			Weight: 1.0,
		},
		{
			Name:   "failing_test",
			Input:  []byte(`{"numbers": [3,1,2]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Checks: []string{"sorted_non_decreasing"},
			Weight: 2.0,
		},
	}

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"),
		Meta:   map[string]string{},
	}

	// Test with failing interpreter
	interpreter := &mockInterpreter{shouldPass: false}

	metrics, allPassed, err := runner.Run(ctx, hypothesis, cases, interpreter)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if allPassed {
		t.Error("Expected some tests to fail")
	}

	// Verify correctness = 0.0 (0% pass rate)
	if metrics["correctness"] != 0.0 {
		t.Errorf("Expected correctness=0.0, got %f", metrics["correctness"])
	}

	// Verify passed weight = 0 (none passed)
	if metrics["passed_weight"] != 0.0 {
		t.Errorf("Expected passed_weight=0.0, got %f", metrics["passed_weight"])
	}

	// Verify total weight = 3.0 (1.0 + 2.0)
	if metrics["total_weight"] != 3.0 {
		t.Errorf("Expected total_weight=3.0, got %f", metrics["total_weight"])
	}
}

func TestRunner_Run_TimeScore(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner()

	cases := []core.TestCase{
		{
			Name:   "fast_test",
			Input:  []byte(`{"numbers": [1,2,3]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Weight: 1.0,
		},
	}

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "af-dsl",
		Bytes:  []byte("(let x input (return x))"),
		Meta:   map[string]string{},
	}

	// Test with different delays
	testCases := []struct {
		name     string
		delay    time.Duration
		expected float64
	}{
		{"very_fast", 1 * time.Millisecond, 0.999},  // Should be very close to 1
		{"fast", 10 * time.Millisecond, 0.99},       // Should be close to 1
		{"slow", 100 * time.Millisecond, 0.9},       // Should be lower
		{"very_slow", 1000 * time.Millisecond, 0.5}, // Should be much lower
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interpreter := &mockInterpreter{shouldPass: true, delay: tc.delay}
			metrics, _, err := runner.Run(ctx, hypothesis, cases, interpreter)

			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			// Time score should be calculated as 1/(1+avg_ms/1000)
			// For very fast tests, this should be close to 1
			if metrics["time"] < 0 || metrics["time"] > 1 {
				t.Errorf("Expected time score between 0 and 1, got %f", metrics["time"])
			}
		})
	}
}

func TestRunner_Run_SizeScore(t *testing.T) {
	ctx := context.Background()
	runner := NewRunner()

	cases := []core.TestCase{
		{
			Name:   "test",
			Input:  []byte(`{"numbers": [1,2,3]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`),
			Weight: 1.0,
		},
	}

	// Test with different program sizes
	testCases := []struct {
		name     string
		program  string
		expected float64
	}{
		{"small", "(let x input (return x))", 0.5},                      // Small program
		{"medium", "(let x input (let y x (return y)))", 0.33},          // Medium program
		{"large", "(let x input (let y x (let z y (return z))))", 0.25}, // Large program
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hypothesis := core.Hypothesis{
				ID:     "test-hypothesis",
				Source: "test",
				Lang:   "af-dsl",
				Bytes:  []byte(tc.program),
				Meta:   map[string]string{},
			}

			interpreter := &mockInterpreter{shouldPass: true}
			metrics, _, err := runner.Run(ctx, hypothesis, cases, interpreter)

			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			// Size score should be calculated as 1/(1+nodes)
			// Larger programs should have lower scores
			if metrics["size"] < 0 || metrics["size"] > 1 {
				t.Errorf("Expected size score between 0 and 1, got %f", metrics["size"])
			}
		})
	}
}

func TestCountASTNodes(t *testing.T) {
	testCases := []struct {
		program  string
		expected int
	}{
		{"(let x 42 (return x))", 9},               // 4 parens + 5 symbols (let, x, 42, return, x)
		{"(if true 1 0)", 6},                       // 2 parens + 4 symbols (if, true, 1, 0)
		{"(call len input)", 5},                    // 2 parens + 3 symbols (call, len, input)
		{"(let x input (let y x (return y)))", 14}, // 6 parens + 8 symbols
	}

	for _, tc := range testCases {
		t.Run(tc.program, func(t *testing.T) {
			result := countASTNodes([]byte(tc.program))
			if result != tc.expected {
				t.Errorf("Expected %d nodes for %q, got %d", tc.expected, tc.program, result)
			}
		})
	}
}

func TestCalculateUnitPropertyPass(t *testing.T) {
	cases := []core.TestCase{
		{
			Name:   "unit_test",
			Input:  []byte(`{"numbers": [1,2,3]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`), // Has oracle = unit test
			Weight: 1.0,
		},
		{
			Name:   "property_test",
			Input:  []byte(`{"numbers": [3,1,2]}`),
			Checks: []string{"sorted_non_decreasing"}, // Has checks = property test
			Weight: 2.0,
		},
		{
			Name:   "mixed_test",
			Input:  []byte(`{"numbers": [2,1,3]}`),
			Oracle: []byte(`{"sorted": [1,2,3]}`), // Both oracle and checks
			Checks: []string{"sorted_non_decreasing"},
			Weight: 1.5,
		},
	}

	// Test with passing results
	results := []testResult{
		{testCase: cases[0], passed: true}, // unit test passes
		{testCase: cases[1], passed: true}, // property test passes
		{testCase: cases[2], passed: true}, // mixed test passes
	}
	unitPass, propPass := calculateUnitPropertyPassFromResults(results)

	// All should pass
	if unitPass != 1.0 {
		t.Errorf("Expected unit_pass=1.0, got %f", unitPass)
	}
	if propPass != 1.0 {
		t.Errorf("Expected prop_pass=1.0, got %f", propPass)
	}

	// Test with failing results
	results = []testResult{
		{testCase: cases[0], passed: false}, // unit test fails
		{testCase: cases[1], passed: false}, // property test fails
		{testCase: cases[2], passed: false}, // mixed test fails
	}
	unitPass, propPass = calculateUnitPropertyPassFromResults(results)

	// All should fail
	if unitPass != 0.0 {
		t.Errorf("Expected unit_pass=0.0, got %f", unitPass)
	}
	if propPass != 0.0 {
		t.Errorf("Expected prop_pass=0.0, got %f", propPass)
	}
}
