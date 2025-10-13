package design

import (
	"encoding/base64"
	"testing"
)

func TestToHypothesis_ValidAFDSL(t *testing.T) {
	hd := createValidDesign()
	hd.Code.Lang = "af-dsl"
	hd.Code.Src = "(program (let x 5) (return x))"

	hypothesis, err := ToHypothesis(hd)
	if err != nil {
		t.Fatalf("ToHypothesis failed: %v", err)
	}

	// Check basic fields
	if hypothesis.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if hypothesis.Source != "llm:design" {
		t.Errorf("Expected source 'llm:design', got %s", hypothesis.Source)
	}
	if hypothesis.Lang != "af-dsl" {
		t.Errorf("Expected lang 'af-dsl', got %s", hypothesis.Lang)
	}

	// Check code bytes
	expectedCode := "(program (let x 5) (return x))"
	if string(hypothesis.Bytes) != expectedCode {
		t.Errorf("Expected code %s, got %s", expectedCode, string(hypothesis.Bytes))
	}

	// Check metadata
	meta := hypothesis.Meta
	if meta["algorithm_name"] != "TestAlgorithm" {
		t.Errorf("Expected algorithm_name 'TestAlgorithm', got %s", meta["algorithm_name"])
	}
	if meta["algorithm_idea"] != "Test idea" {
		t.Errorf("Expected algorithm_idea 'Test idea', got %s", meta["algorithm_idea"])
	}
	if meta["complexity_time"] != "O(n)" {
		t.Errorf("Expected complexity_time 'O(n)', got %s", meta["complexity_time"])
	}
	if meta["complexity_space"] != "O(1)" {
		t.Errorf("Expected complexity_space 'O(1)', got %s", meta["complexity_space"])
	}
	if meta["domain"] != "algorithms" {
		t.Errorf("Expected domain 'algorithms', got %s", meta["domain"])
	}
}

func TestToHypothesis_ValidWASM(t *testing.T) {
	hd := createValidDesign()
	hd.Code.Lang = "wasm"
	hd.Code.Src = base64.StdEncoding.EncodeToString([]byte("dummy wasm binary"))

	hypothesis, err := ToHypothesis(hd)
	if err != nil {
		t.Fatalf("ToHypothesis failed: %v", err)
	}

	// Check basic fields
	if hypothesis.Lang != "wasm" {
		t.Errorf("Expected lang 'wasm', got %s", hypothesis.Lang)
	}

	// Check that bytes are decoded correctly
	expectedBytes := []byte("dummy wasm binary")
	if string(hypothesis.Bytes) != string(expectedBytes) {
		t.Errorf("Expected decoded wasm bytes, got %s", string(hypothesis.Bytes))
	}
}

func TestToHypothesis_InvalidStatus(t *testing.T) {
	hd := createValidDesign()
	hd.Status = "cannot_solve"

	_, err := ToHypothesis(hd)
	if err == nil {
		t.Fatal("Expected error for non-ok status, got nil")
	}

	if !contains(err.Error(), "cannot convert non-ok design") {
		t.Errorf("Expected error about non-ok design, got: %v", err)
	}
}

func TestToHypothesis_InvalidWASM(t *testing.T) {
	hd := createValidDesign()
	hd.Code.Lang = "wasm"
	hd.Code.Src = "invalid base64!"

	_, err := ToHypothesis(hd)
	if err == nil {
		t.Fatal("Expected error for invalid base64, got nil")
	}

	if !contains(err.Error(), "failed to decode wasm base64") {
		t.Errorf("Expected error about base64 decode, got: %v", err)
	}
}

func TestToHypothesis_UnsupportedLang(t *testing.T) {
	hd := createValidDesign()
	hd.Code.Lang = "python"

	_, err := ToHypothesis(hd)
	if err == nil {
		t.Fatal("Expected error for unsupported language, got nil")
	}

	if !contains(err.Error(), "unsupported code language") {
		t.Errorf("Expected error about unsupported language, got: %v", err)
	}
}

func TestToTestCases_ValidDesign(t *testing.T) {
	hd := createValidDesign()
	hd.Tests.Unit = []struct {
		Name   string  `json:"name"`
		Input  string  `json:"input"`
		Oracle string  `json:"oracle"`
		Weight float64 `json:"weight"`
	}{
		{
			Name:   "test1",
			Input:  "[3,1,4]",
			Oracle: "[1,3,4]",
			Weight: 1.0,
		},
		{
			Name:   "test2",
			Input:  "[5,2,8]",
			Oracle: "[2,5,8]",
			Weight: 0.5,
		},
	}

	testCases := ToTestCases(hd)

	if len(testCases) != 2 {
		t.Errorf("Expected 2 test cases, got %d", len(testCases))
	}

	// Check first test case
	tc1 := testCases[0]
	if tc1.Name != "test1" {
		t.Errorf("Expected name 'test1', got %s", tc1.Name)
	}
	if string(tc1.Input) != "[3,1,4]" {
		t.Errorf("Expected input '[3,1,4]', got %s", string(tc1.Input))
	}
	if string(tc1.Oracle) != "[1,3,4]" {
		t.Errorf("Expected oracle '[1,3,4]', got %s", string(tc1.Oracle))
	}
	if tc1.Weight != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", tc1.Weight)
	}

	// Check second test case
	tc2 := testCases[1]
	if tc2.Name != "test2" {
		t.Errorf("Expected name 'test2', got %s", tc2.Name)
	}
	if tc2.Weight != 0.5 {
		t.Errorf("Expected weight 0.5, got %f", tc2.Weight)
	}
}

func TestToTestCases_InvalidStatus(t *testing.T) {
	hd := createValidDesign()
	hd.Status = "cannot_solve"

	testCases := ToTestCases(hd)

	if len(testCases) != 0 {
		t.Errorf("Expected 0 test cases for invalid status, got %d", len(testCases))
	}
}

func TestToPropertyPlans_ValidDesign(t *testing.T) {
	hd := createValidDesign()
	hd.Tests.Property = []struct {
		Name      string   `json:"name"`
		Generator string   `json:"generator"`
		Checks    []string `json:"checks"`
	}{
		{
			Name:      "prop1",
			Generator: "list<int>(n<=100)",
			Checks:    []string{"sorted?", "permutes?"},
		},
		{
			Name:      "prop2",
			Generator: "list<int>(n<=50, val<=1000)",
			Checks:    []string{"stable?"},
		},
	}

	plans := ToPropertyPlans(hd)

	if len(plans) != 2 {
		t.Errorf("Expected 2 property plans, got %d", len(plans))
	}

	// Check first plan
	plan1 := plans[0]
	if plan1.Name != "prop1" {
		t.Errorf("Expected name 'prop1', got %s", plan1.Name)
	}
	if plan1.Generator != "list<int>(n<=100)" {
		t.Errorf("Expected generator 'list<int>(n<=100)', got %s", plan1.Generator)
	}
	if len(plan1.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(plan1.Checks))
	}
	if plan1.Checks[0] != "sorted?" || plan1.Checks[1] != "permutes?" {
		t.Errorf("Expected checks ['sorted?', 'permutes?'], got %v", plan1.Checks)
	}
}

func TestToPropertyPlans_InvalidStatus(t *testing.T) {
	hd := createValidDesign()
	hd.Status = "cannot_solve"

	plans := ToPropertyPlans(hd)

	if len(plans) != 0 {
		t.Errorf("Expected 0 property plans for invalid status, got %d", len(plans))
	}
}

func TestGenerateHypothesisID(t *testing.T) {
	hd := createValidDesign()
	hd.Algorithm.Name = "Quick Sort"
	hd.Algorithm.Complexity.Time = "O(n log n)"

	id := generateHypothesisID(hd)

	expectedPrefix := "algo-quick-sort-nlogn"
	if !contains(id, expectedPrefix) {
		t.Errorf("Expected ID to contain %s, got %s", expectedPrefix, id)
	}
}
