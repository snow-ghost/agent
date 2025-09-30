package core

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
