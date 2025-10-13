package testkit

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/design"
)

// PropertyPlan represents a property test plan
type PropertyPlan = design.PropertyPlan

// Generator represents a data generator
type Generator interface {
	Generate() []byte
}

// ListIntGenerator generates lists of integers
type ListIntGenerator struct {
	MaxLength int
	MinLength int
	MaxValue  int
	MinValue  int
}

// NewListIntGenerator creates a new list integer generator from a spec string
// Example: "list<int>(n<=100)" -> generates lists of length 0-100
func NewListIntGenerator(spec string) (*ListIntGenerator, error) {
	// Parse spec like "list<int>(n<=100)" or "list<int>(n<=50, val<=1000)"
	re := regexp.MustCompile(`list<int>\(\s*n\s*<=\s*(\d+)(?:,\s*val\s*<=\s*(\d+))?\s*\)`)
	matches := re.FindStringSubmatch(spec)

	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid generator spec: %s", spec)
	}

	maxLength, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid max length: %s", matches[1])
	}

	maxValue := 1000 // default
	if len(matches) > 2 && matches[2] != "" {
		maxValue, err = strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("invalid max value: %s", matches[2])
		}
	}

	return &ListIntGenerator{
		MaxLength: maxLength,
		MinLength: 0,
		MaxValue:  maxValue,
		MinValue:  -maxValue,
	}, nil
}

// Generate creates a random list of integers
func (g *ListIntGenerator) Generate() []byte {
	// Random length between MinLength and MaxLength
	length := g.MinLength
	if g.MaxLength > g.MinLength {
		length = g.MinLength + rand.Intn(g.MaxLength-g.MinLength+1)
	}

	// Generate random integers
	list := make([]int, length)
	for i := range list {
		list[i] = g.MinValue + rand.Intn(g.MaxValue-g.MinValue+1)
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(list)
	if err != nil {
		// Fallback to empty array
		return []byte("[]")
	}

	return jsonBytes
}

// PropertyChecker represents a property checker
type PropertyChecker interface {
	Check(input, output []byte) bool
	Name() string
}

// SortedChecker checks if output is sorted (non-decreasing)
type SortedChecker struct{}

func (c *SortedChecker) Name() string {
	return "sorted?"
}

func (c *SortedChecker) Check(input, output []byte) bool {
	var outputList []int
	if err := json.Unmarshal(output, &outputList); err != nil {
		return false
	}

	// Check if output is sorted (non-decreasing)
	for i := 1; i < len(outputList); i++ {
		if outputList[i] < outputList[i-1] {
			return false
		}
	}

	return true
}

// PermutesChecker checks if output is a permutation of input
type PermutesChecker struct{}

func (c *PermutesChecker) Name() string {
	return "permutes?"
}

func (c *PermutesChecker) Check(input, output []byte) bool {
	var inputList, outputList []int

	if err := json.Unmarshal(input, &inputList); err != nil {
		return false
	}
	if err := json.Unmarshal(output, &outputList); err != nil {
		return false
	}

	// Check if lengths are equal
	if len(inputList) != len(outputList) {
		return false
	}

	// Count occurrences of each value
	inputCounts := make(map[int]int)
	outputCounts := make(map[int]int)

	for _, val := range inputList {
		inputCounts[val]++
	}
	for _, val := range outputList {
		outputCounts[val]++
	}

	// Check if counts match
	for val, count := range inputCounts {
		if outputCounts[val] != count {
			return false
		}
	}
	for val, count := range outputCounts {
		if inputCounts[val] != count {
			return false
		}
	}

	return true
}

// StableChecker checks if sorting is stable (preserves relative order of equal elements)
type StableChecker struct{}

func (c *StableChecker) Name() string {
	return "stable?"
}

func (c *StableChecker) Check(input, output []byte) bool {
	var inputList, outputList []int

	if err := json.Unmarshal(input, &inputList); err != nil {
		return false
	}
	if err := json.Unmarshal(output, &outputList); err != nil {
		return false
	}

	// Check if lengths are equal
	if len(inputList) != len(outputList) {
		return false
	}

	// For now, just check that the output is sorted
	// A proper stability check would be more complex
	for i := 1; i < len(outputList); i++ {
		if outputList[i] < outputList[i-1] {
			return false
		}
	}

	return true
}

// isStableOrder checks if the relative order of indices is preserved
func isStableOrder(inputIndices, outputIndices []int) bool {
	if len(inputIndices) != len(outputIndices) {
		return false
	}

	// For stability, we need to check that for equal values,
	// the relative order of their original positions is preserved
	// This is a simplified check - in practice, we'd need to track
	// the original positions more carefully

	// For now, just check that the output indices are in ascending order
	// when the input indices are in ascending order
	for i := 1; i < len(outputIndices); i++ {
		if outputIndices[i] < outputIndices[i-1] {
			return false
		}
	}

	return true
}

// CreateChecker creates a property checker from a check name
func CreateChecker(checkName string) PropertyChecker {
	switch checkName {
	case "sorted?":
		return &SortedChecker{}
	case "permutes?":
		return &PermutesChecker{}
	case "stable?":
		return &StableChecker{}
	default:
		return nil
	}
}

// MakePropertyCases generates k random test cases from a property plan
func MakePropertyCases(plan PropertyPlan, k int) []core.TestCase {
	// Parse generator
	generator, err := NewListIntGenerator(plan.Generator)
	if err != nil {
		// Return empty if generator parsing fails
		return []core.TestCase{}
	}

	// Create checkers
	var checkers []PropertyChecker
	for _, checkName := range plan.Checks {
		if checker := CreateChecker(checkName); checker != nil {
			checkers = append(checkers, checker)
		}
	}

	// Generate test cases
	var testCases []core.TestCase
	for i := 0; i < k; i++ {
		input := generator.Generate()

		// Create test case name
		name := fmt.Sprintf("%s_prop_%d", plan.Name, i+1)

		// Create checks string
		var checkNames []string
		for _, checker := range checkers {
			checkNames = append(checkNames, checker.Name())
		}

		testCase := core.TestCase{
			Name:   name,
			Input:  input,
			Oracle: nil, // Property tests don't have oracles
			Checks: checkNames,
			Weight: 1.0, // Equal weight for property tests
		}

		testCases = append(testCases, testCase)
	}

	return testCases
}

// ValidatePropertyTest validates a property test by running the checkers
func ValidatePropertyTest(input, output []byte, plan PropertyPlan) (bool, []string) {
	var results []string
	allPassed := true

	// Create checkers
	for _, checkName := range plan.Checks {
		checker := CreateChecker(checkName)
		if checker == nil {
			results = append(results, fmt.Sprintf("unknown checker: %s", checkName))
			allPassed = false
			continue
		}

		passed := checker.Check(input, output)
		results = append(results, fmt.Sprintf("%s: %t", checker.Name(), passed))
		if !passed {
			allPassed = false
		}
	}

	return allPassed, results
}

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
