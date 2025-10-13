package core

import (
	"math"
	"testing"
)

// floatEquals checks if two floats are equal within a tolerance
func floatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestParseFitnessFormula(t *testing.T) {
	testCases := []struct {
		name              string
		formula           string
		passThreshold     float64
		expectedWeights   map[string]float64
		expectedThreshold float64
	}{
		{
			name:          "valid_formula",
			formula:       "correctness*0.8 + time*0.15 + size*0.05",
			passThreshold: 0.85,
			expectedWeights: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
			expectedThreshold: 0.85,
		},
		{
			name:          "simple_formula",
			formula:       "correctness*0.9",
			passThreshold: 0.7,
			expectedWeights: map[string]float64{
				"correctness": 0.9,
			},
			expectedThreshold: 0.7,
		},
		{
			name:          "empty_formula",
			formula:       "",
			passThreshold: 0.8,
			expectedWeights: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
			expectedThreshold: 0.8,
		},
		{
			name:          "invalid_formula",
			formula:       "invalid*formula*here",
			passThreshold: 0.6,
			expectedWeights: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
			expectedThreshold: 0.6,
		},
		{
			name:          "complex_formula",
			formula:       "correctness*0.7 + time*0.2 + size*0.1 + unit_pass*0.05",
			passThreshold: 0.9,
			expectedWeights: map[string]float64{
				"correctness": 0.7,
				"time":        0.2,
				"size":        0.1,
				"unit_pass":   0.05,
			},
			expectedThreshold: 0.9,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			weights, threshold := ParseFitnessFormula(tc.formula, tc.passThreshold)

			if threshold != tc.expectedThreshold {
				t.Errorf("Expected threshold %f, got %f", tc.expectedThreshold, threshold)
			}

			if len(weights) != len(tc.expectedWeights) {
				t.Errorf("Expected %d weights, got %d", len(tc.expectedWeights), len(weights))
			}

			for metric, expectedWeight := range tc.expectedWeights {
				if actualWeight, exists := weights[metric]; !exists {
					t.Errorf("Missing weight for metric %s", metric)
				} else if actualWeight != expectedWeight {
					t.Errorf("Expected weight %f for metric %s, got %f", expectedWeight, metric, actualWeight)
				}
			}
		})
	}
}

func TestNewFitnessFromDesign(t *testing.T) {
	testCases := []struct {
		name     string
		meta     map[string]string
		expected map[string]float64
	}{
		{
			name: "valid_design",
			meta: map[string]string{
				"fitness_formula": "correctness*0.8 + time*0.15 + size*0.05",
				"pass_threshold":  "0.85",
			},
			expected: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
		},
		{
			name: "missing_fitness_formula",
			meta: map[string]string{
				"pass_threshold": "0.7",
			},
			expected: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
		},
		{
			name: "missing_pass_threshold",
			meta: map[string]string{
				"fitness_formula": "correctness*0.9",
			},
			expected: map[string]float64{
				"correctness": 0.9,
			},
		},
		{
			name: "empty_meta",
			meta: map[string]string{},
			expected: map[string]float64{
				"correctness": 0.8,
				"time":        0.15,
				"size":        0.05,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fitness := NewFitnessFromDesign(tc.meta)

			if len(fitness.MetricWeights) != len(tc.expected) {
				t.Errorf("Expected %d weights, got %d", len(tc.expected), len(fitness.MetricWeights))
			}

			for metric, expectedWeight := range tc.expected {
				if actualWeight, exists := fitness.MetricWeights[metric]; !exists {
					t.Errorf("Missing weight for metric %s", metric)
				} else if actualWeight != expectedWeight {
					t.Errorf("Expected weight %f for metric %s, got %f", expectedWeight, metric, actualWeight)
				}
			}

			// Check default size penalty
			if fitness.SizePenaltyPerKB != 0.01 {
				t.Errorf("Expected size penalty 0.01, got %f", fitness.SizePenaltyPerKB)
			}
		})
	}
}

func TestWeightedFitness_Score(t *testing.T) {
	weights := map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}
	fitness := NewWeightedFitness(weights, 0.01)

	task := Task{
		ID:     "test-task",
		Domain: "algorithms",
		Spec:   Spec{},
	}

	metrics := map[string]float64{
		"correctness": 1.0,
		"time":        0.9,
		"size":        0.8,
	}

	// Test with small size (should have minimal penalty)
	score := fitness.Score(task, metrics, 1024) // 1KB
	expected := 0.8*1.0 + 0.15*0.9 + 0.05*0.8 - 0.01*1.0
	if !floatEquals(score, expected, 0.0001) {
		t.Errorf("Expected score %f, got %f", expected, score)
	}

	// Test with large size (should have significant penalty)
	score = fitness.Score(task, metrics, 10240) // 10KB
	expected = 0.8*1.0 + 0.15*0.9 + 0.05*0.8 - 0.01*10.0
	if !floatEquals(score, expected, 0.0001) {
		t.Errorf("Expected score %f, got %f", expected, score)
	}
}

func TestWeightedFitness_Passed(t *testing.T) {
	weights := map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}
	fitness := NewWeightedFitness(weights, 0.01)

	testCases := []struct {
		score     float64
		threshold float64
		expected  bool
	}{
		{0.9, 0.8, true},
		{0.7, 0.8, false},
		{0.8, 0.8, true},
		{0.85, 0.8, true},
		{0.75, 0.8, false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			result := fitness.Passed(tc.score, tc.threshold)
			if result != tc.expected {
				t.Errorf("Expected Passed(%f, %f) = %v, got %v", tc.score, tc.threshold, tc.expected, result)
			}
		})
	}
}
