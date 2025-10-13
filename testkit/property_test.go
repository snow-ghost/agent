package testkit

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewListIntGenerator_ValidSpecs(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		expectedMaxLen int
		expectedMaxVal int
	}{
		{
			name:           "basic spec",
			spec:           "list<int>(n<=100)",
			expectedMaxLen: 100,
			expectedMaxVal: 1000, // default
		},
		{
			name:           "spec with value limit",
			spec:           "list<int>(n<=50, val<=500)",
			expectedMaxLen: 50,
			expectedMaxVal: 500,
		},
		{
			name:           "spec with spaces",
			spec:           "list<int>(n <= 25, val <= 200)",
			expectedMaxLen: 25,
			expectedMaxVal: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewListIntGenerator(tt.spec)
			if err != nil {
				t.Fatalf("NewListIntGenerator failed: %v", err)
			}

			if gen.MaxLength != tt.expectedMaxLen {
				t.Errorf("Expected MaxLength %d, got %d", tt.expectedMaxLen, gen.MaxLength)
			}
			if gen.MaxValue != tt.expectedMaxVal {
				t.Errorf("Expected MaxValue %d, got %d", tt.expectedMaxVal, gen.MaxValue)
			}
			if gen.MinLength != 0 {
				t.Errorf("Expected MinLength 0, got %d", gen.MinLength)
			}
		})
	}
}

func TestNewListIntGenerator_InvalidSpecs(t *testing.T) {
	tests := []string{
		"invalid spec",
		"list<string>(n<=100)",
		"list<int>",
		"list<int>(n=100)",
		"",
	}

	for _, spec := range tests {
		t.Run(spec, func(t *testing.T) {
			_, err := NewListIntGenerator(spec)
			if err == nil {
				t.Errorf("Expected error for spec '%s', got nil", spec)
			}
		})
	}
}

func TestListIntGenerator_Generate(t *testing.T) {
	gen, err := NewListIntGenerator("list<int>(n<=10, val<=100)")
	if err != nil {
		t.Fatalf("NewListIntGenerator failed: %v", err)
	}

	// Generate multiple times to test randomness
	for i := 0; i < 10; i++ {
		result := gen.Generate()

		// Should be valid JSON
		var list []int
		if err := json.Unmarshal(result, &list); err != nil {
			t.Errorf("Generated invalid JSON: %v", err)
			continue
		}

		// Check length constraints
		if len(list) > 10 {
			t.Errorf("Generated list length %d not in range [0, 10]", len(list))
		}

		// Check value constraints
		for _, val := range list {
			if val < -100 || val > 100 {
				t.Errorf("Generated value %d not in range [-100, 100]", val)
			}
		}
	}
}

func TestSortedChecker_Check(t *testing.T) {
	checker := &SortedChecker{}

	tests := []struct {
		name     string
		input    []byte
		output   []byte
		expected bool
	}{
		{
			name:     "sorted array",
			input:    []byte("[3,1,4]"),
			output:   []byte("[1,3,4]"),
			expected: true,
		},
		{
			name:     "unsorted array",
			input:    []byte("[3,1,4]"),
			output:   []byte("[3,1,4]"),
			expected: false,
		},
		{
			name:     "empty array",
			input:    []byte("[]"),
			output:   []byte("[]"),
			expected: true,
		},
		{
			name:     "single element",
			input:    []byte("[5]"),
			output:   []byte("[5]"),
			expected: true,
		},
		{
			name:     "duplicate elements",
			input:    []byte("[3,1,1,4]"),
			output:   []byte("[1,1,3,4]"),
			expected: true,
		},
		{
			name:     "invalid JSON",
			input:    []byte("[3,1,4]"),
			output:   []byte("invalid"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check(tt.input, tt.output)
			if result != tt.expected {
				t.Errorf("Check() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPermutesChecker_Check(t *testing.T) {
	checker := &PermutesChecker{}

	tests := []struct {
		name     string
		input    []byte
		output   []byte
		expected bool
	}{
		{
			name:     "valid permutation",
			input:    []byte("[3,1,4]"),
			output:   []byte("[1,3,4]"),
			expected: true,
		},
		{
			name:     "same order",
			input:    []byte("[1,2,3]"),
			output:   []byte("[1,2,3]"),
			expected: true,
		},
		{
			name:     "not a permutation - different length",
			input:    []byte("[3,1,4]"),
			output:   []byte("[1,3]"),
			expected: false,
		},
		{
			name:     "not a permutation - different values",
			input:    []byte("[3,1,4]"),
			output:   []byte("[1,2,3]"),
			expected: false,
		},
		{
			name:     "duplicate elements",
			input:    []byte("[3,1,1,4]"),
			output:   []byte("[1,1,3,4]"),
			expected: true,
		},
		{
			name:     "empty arrays",
			input:    []byte("[]"),
			output:   []byte("[]"),
			expected: true,
		},
		{
			name:     "invalid JSON",
			input:    []byte("[3,1,4]"),
			output:   []byte("invalid"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check(tt.input, tt.output)
			if result != tt.expected {
				t.Errorf("Check() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStableChecker_Check(t *testing.T) {
	checker := &StableChecker{}

	tests := []struct {
		name     string
		input    []byte
		output   []byte
		expected bool
	}{
		{
			name:     "stable sort with duplicates",
			input:    []byte("[3,1,3,2]"),
			output:   []byte("[1,2,3,3]"),
			expected: true,
		},
		{
			name:     "unsorted output",
			input:    []byte("[3,1,3,2]"),
			output:   []byte("[1,3,2,3]"), // Not sorted
			expected: false,
		},
		{
			name:     "no duplicates - always stable",
			input:    []byte("[3,1,4,2]"),
			output:   []byte("[1,2,3,4]"),
			expected: true,
		},
		{
			name:     "empty arrays",
			input:    []byte("[]"),
			output:   []byte("[]"),
			expected: true,
		},
		{
			name:     "single element",
			input:    []byte("[5]"),
			output:   []byte("[5]"),
			expected: true,
		},
		{
			name:     "different lengths",
			input:    []byte("[3,1,4]"),
			output:   []byte("[1,3]"),
			expected: false,
		},
		{
			name:     "invalid JSON",
			input:    []byte("[3,1,4]"),
			output:   []byte("invalid"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check(tt.input, tt.output)
			if result != tt.expected {
				t.Errorf("Check() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateChecker(t *testing.T) {
	tests := []struct {
		name      string
		checkName string
		expected  bool // whether checker should be created
	}{
		{"sorted checker", "sorted?", true},
		{"permutes checker", "permutes?", true},
		{"stable checker", "stable?", true},
		{"unknown checker", "unknown?", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := CreateChecker(tt.checkName)
			if (checker != nil) != tt.expected {
				t.Errorf("CreateChecker(%s) = %v, want checker != nil to be %v",
					tt.checkName, checker != nil, tt.expected)
			}
		})
	}
}

func TestMakePropertyCases(t *testing.T) {
	plan := PropertyPlan{
		Name:      "test_prop",
		Generator: "list<int>(n<=5, val<=10)",
		Checks:    []string{"sorted?", "permutes?"},
	}

	cases := MakePropertyCases(plan, 3)

	if len(cases) != 3 {
		t.Errorf("Expected 3 test cases, got %d", len(cases))
	}

	for i, tc := range cases {
		// Check name
		expectedName := fmt.Sprintf("test_prop_prop_%d", i+1)
		if tc.Name != expectedName {
			t.Errorf("Expected name %s, got %s", expectedName, tc.Name)
		}

		// Check input is valid JSON
		var input []int
		if err := json.Unmarshal(tc.Input, &input); err != nil {
			t.Errorf("Invalid input JSON: %v", err)
		}

		// Check checks
		if len(tc.Checks) != 2 {
			t.Errorf("Expected 2 checks, got %d", len(tc.Checks))
		}
		if tc.Checks[0] != "sorted?" || tc.Checks[1] != "permutes?" {
			t.Errorf("Expected checks ['sorted?', 'permutes?'], got %v", tc.Checks)
		}

		// Check weight
		if tc.Weight != 1.0 {
			t.Errorf("Expected weight 1.0, got %f", tc.Weight)
		}

		// Check oracle is nil
		if tc.Oracle != nil {
			t.Errorf("Expected nil oracle, got %v", tc.Oracle)
		}
	}
}

func TestMakePropertyCases_InvalidGenerator(t *testing.T) {
	plan := PropertyPlan{
		Name:      "test_prop",
		Generator: "invalid generator spec",
		Checks:    []string{"sorted?"},
	}

	cases := MakePropertyCases(plan, 3)

	if len(cases) != 0 {
		t.Errorf("Expected 0 test cases for invalid generator, got %d", len(cases))
	}
}

func TestValidatePropertyTest(t *testing.T) {
	plan := PropertyPlan{
		Name:      "test_prop",
		Generator: "list<int>(n<=5)",
		Checks:    []string{"sorted?", "permutes?"},
	}

	// Test with valid sorted and permuted output
	input := []byte("[3,1,4]")
	output := []byte("[1,3,4]")

	passed, results := ValidatePropertyTest(input, output, plan)

	if !passed {
		t.Errorf("Expected test to pass, got false")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that both checks passed
	for _, result := range results {
		if !contains(result, ": true") {
			t.Errorf("Expected all checks to pass, got result: %s", result)
		}
	}
}

func TestValidatePropertyTest_InvalidOutput(t *testing.T) {
	plan := PropertyPlan{
		Name:      "test_prop",
		Generator: "list<int>(n<=5)",
		Checks:    []string{"sorted?", "permutes?"},
	}

	// Test with invalid output (not sorted)
	input := []byte("[3,1,4]")
	output := []byte("[3,1,4]") // Not sorted

	passed, results := ValidatePropertyTest(input, output, plan)

	if passed {
		t.Errorf("Expected test to fail, got true")
	}

	// Should have at least one failure
	hasFailure := false
	for _, result := range results {
		if contains(result, ": false") {
			hasFailure = true
			break
		}
	}
	if !hasFailure {
		t.Errorf("Expected at least one check to fail, got results: %v", results)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr))))
}
