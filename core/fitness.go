package core

import (
	"regexp"
	"strconv"
)

// WeightedFitness evaluates fitness as weighted sum of metrics minus size penalty.
type WeightedFitness struct {
	MetricWeights    map[string]float64
	SizePenaltyPerKB float64
}

func NewWeightedFitness(weights map[string]float64, sizePenaltyPerKB float64) *WeightedFitness {
	return &WeightedFitness{MetricWeights: weights, SizePenaltyPerKB: sizePenaltyPerKB}
}

func (w *WeightedFitness) Score(task Task, metrics map[string]float64, sizeBytes int) float64 {
	score := 0.0
	for k, weight := range w.MetricWeights {
		if v, ok := metrics[k]; ok {
			score += weight * v
		}
	}
	// penalty grows with size in KB
	kb := float64(sizeBytes) / 1024.0
	score -= w.SizePenaltyPerKB * kb
	return score
}

func (w *WeightedFitness) Passed(score float64, threshold float64) bool {
	return score >= threshold
}

// ParseFitnessFormula parses a fitness formula string and extracts weights
// Format: "correctness*0.8 + time*0.15 + size*0.05" or similar
// Returns weights map and pass threshold
func ParseFitnessFormula(fitnessStr string, passThreshold float64) (map[string]float64, float64) {
	// Default weights if parsing fails
	defaultWeights := map[string]float64{
		"correctness": 0.8,
		"time":        0.15,
		"size":        0.05,
	}

	if fitnessStr == "" {
		return defaultWeights, passThreshold
	}

	weights := make(map[string]float64)

	// Regex to match patterns like "correctness*0.8", "time*0.15", etc.
	re := regexp.MustCompile(`(\w+)\*([0-9.]+)`)
	matches := re.FindAllStringSubmatch(fitnessStr, -1)

	for _, match := range matches {
		if len(match) == 3 {
			metric := match[1]
			weightStr := match[2]
			if weight, err := strconv.ParseFloat(weightStr, 64); err == nil {
				weights[metric] = weight
			}
		}
	}

	// If no weights were parsed, use defaults
	if len(weights) == 0 {
		return defaultWeights, passThreshold
	}

	return weights, passThreshold
}

// NewFitnessFromDesign creates a WeightedFitness from design metadata
func NewFitnessFromDesign(meta map[string]string) *WeightedFitness {
	fitnessStr := meta["fitness_formula"]
	passThresholdStr := meta["pass_threshold"]

	passThreshold := 0.8 // default
	if threshold, err := strconv.ParseFloat(passThresholdStr, 64); err == nil {
		passThreshold = threshold
	}

	weights, _ := ParseFitnessFormula(fitnessStr, passThreshold)

	// Default size penalty
	sizePenalty := 0.01 // 0.01 per KB

	return NewWeightedFitness(weights, sizePenalty)
}
