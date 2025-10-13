package design

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/snow-ghost/agent/core"
)

// ToHypothesis converts a HypothesisDesign to a core.Hypothesis
func ToHypothesis(hd HypothesisDesign) (core.Hypothesis, error) {
	if hd.Status != "ok" {
		return core.Hypothesis{}, fmt.Errorf("cannot convert non-ok design to hypothesis: status=%s", hd.Status)
	}

	// Convert code source to bytes
	var codeBytes []byte
	var err error

	switch hd.Code.Lang {
	case "af-dsl":
		// For AF-DSL, the source is already text
		codeBytes = []byte(hd.Code.Src)
	case "wasm":
		// For WASM, decode base64
		codeBytes, err = base64.StdEncoding.DecodeString(hd.Code.Src)
		if err != nil {
			return core.Hypothesis{}, fmt.Errorf("failed to decode wasm base64: %w", err)
		}
	default:
		return core.Hypothesis{}, fmt.Errorf("unsupported code language: %s", hd.Code.Lang)
	}

	// Create metadata from design information
	meta := map[string]string{
		"algorithm_name":   hd.Algorithm.Name,
		"algorithm_idea":   hd.Algorithm.Idea,
		"complexity_time":  hd.Algorithm.Complexity.Time,
		"complexity_space": hd.Algorithm.Complexity.Space,
		"fitness_formula":  hd.Evaluation.Fitness,
		"pass_threshold":   fmt.Sprintf("%.3f", hd.Evaluation.PassThreshold),
		"metrics":          strings.Join(hd.Evaluation.Metrics, ","),
		"unit_test_count":  strconv.Itoa(len(hd.Tests.Unit)),
		"property_count":   strconv.Itoa(len(hd.Tests.Property)),
	}

	// Add domain information if available
	if hd.Algorithm.Name != "" {
		meta["domain"] = "algorithms"
	}

	return core.Hypothesis{
		ID:     generateHypothesisID(hd),
		Source: "llm:design",
		Lang:   hd.Code.Lang,
		Bytes:  codeBytes,
		Meta:   meta,
	}, nil
}

// ToTestCases converts unit tests from HypothesisDesign to core.TestCase slice
func ToTestCases(hd HypothesisDesign) []core.TestCase {
	if hd.Status != "ok" {
		return []core.TestCase{}
	}

	var testCases []core.TestCase

	// Convert unit tests
	for _, unit := range hd.Tests.Unit {
		testCase := core.TestCase{
			Name:   unit.Name,
			Input:  []byte(unit.Input),
			Oracle: []byte(unit.Oracle),
			Weight: unit.Weight,
		}
		testCases = append(testCases, testCase)
	}

	return testCases
}

// PropertyPlan represents a property test plan
type PropertyPlan struct {
	Name      string   `json:"name"`
	Generator string   `json:"generator"`
	Checks    []string `json:"checks"`
}

// ToPropertyPlans converts property tests from HypothesisDesign to PropertyPlan slice
func ToPropertyPlans(hd HypothesisDesign) []PropertyPlan {
	if hd.Status != "ok" {
		return []PropertyPlan{}
	}

	var plans []PropertyPlan

	// Convert property tests
	for _, prop := range hd.Tests.Property {
		plan := PropertyPlan{
			Name:      prop.Name,
			Generator: prop.Generator,
			Checks:    prop.Checks,
		}
		plans = append(plans, plan)
	}

	return plans
}

// generateHypothesisID creates a unique ID for the hypothesis
func generateHypothesisID(hd HypothesisDesign) string {
	// Create a simple ID based on algorithm name and complexity
	algoName := strings.ToLower(strings.ReplaceAll(hd.Algorithm.Name, " ", "-"))
	timeComplexity := strings.ReplaceAll(hd.Algorithm.Complexity.Time, "O(", "")
	timeComplexity = strings.ReplaceAll(timeComplexity, ")", "")
	timeComplexity = strings.ReplaceAll(timeComplexity, " ", "")

	return fmt.Sprintf("algo-%s-%s", algoName, timeComplexity)
}
