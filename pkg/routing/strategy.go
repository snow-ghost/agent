package routing

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
)

// ModelSelector defines the interface for model selection strategies
type ModelSelector interface {
	SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error)
}

// RoundRobinStrategy implements round-robin model selection
type RoundRobinStrategy struct {
	lastIndex int
}

// NewRoundRobinStrategy creates a new round-robin strategy
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{lastIndex: -1}
}

// SelectModel selects the next model in round-robin fashion
func (r *RoundRobinStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	r.lastIndex = (r.lastIndex + 1) % len(models)
	return &models[r.lastIndex], nil
}

// WeightedStrategy implements weighted model selection based on pricing
type WeightedStrategy struct {
	rand *rand.Rand
}

// NewWeightedStrategy creates a new weighted strategy
func NewWeightedStrategy() *WeightedStrategy {
	return &WeightedStrategy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectModel selects a model based on weighted probability (lower cost = higher weight)
func (w *WeightedStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Calculate weights based on inverse cost (lower cost = higher weight)
	weights := make([]float64, len(models))
	totalWeight := 0.0

	for i, model := range models {
		// Use input cost as the primary factor
		cost := model.Pricing.InputPer1K
		if cost == 0 {
			// Free models get highest weight
			weights[i] = 100.0
		} else {
			// Inverse cost weighting
			weights[i] = 1.0 / cost
		}
		totalWeight += weights[i]
	}

	// Select based on weighted probability
	random := w.rand.Float64() * totalWeight
	currentWeight := 0.0

	for i, weight := range weights {
		currentWeight += weight
		if random <= currentWeight {
			return &models[i], nil
		}
	}

	// Fallback to last model
	return &models[len(models)-1], nil
}

// TagBasedStrategy implements model selection based on tags and task domain
type TagBasedStrategy struct {
	fallbackStrategy ModelSelector
}

// NewTagBasedStrategy creates a new tag-based strategy
func NewTagBasedStrategy(fallback ModelSelector) *TagBasedStrategy {
	return &TagBasedStrategy{
		fallbackStrategy: fallback,
	}
}

// SelectModel selects a model based on tags and task domain
func (t *TagBasedStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Get task domain from metadata
	taskDomain := metadata["task_domain"]
	if taskDomain == "" {
		// No specific domain, use fallback strategy
		return t.fallbackStrategy.SelectModel(ctx, models, metadata)
	}

	// Filter models by tags based on task domain
	var candidateModels []registry.ModelConfig

	for _, model := range models {
		if t.modelMatchesDomain(model, taskDomain) {
			candidateModels = append(candidateModels, model)
		}
	}

	// If no models match the domain, use all models
	if len(candidateModels) == 0 {
		candidateModels = models
	}

	// Use fallback strategy on filtered models
	return t.fallbackStrategy.SelectModel(ctx, candidateModels, metadata)
}

// modelMatchesDomain checks if a model matches the given task domain
func (t *TagBasedStrategy) modelMatchesDomain(model registry.ModelConfig, domain string) bool {
	domain = strings.ToLower(domain)

	// Check if any of the model's tags match the domain
	for _, tag := range model.Tags {
		if strings.Contains(strings.ToLower(tag), domain) {
			return true
		}
	}

	// Special domain mappings
	switch domain {
	case "embed", "embedding":
		return model.Kind == "embed"
	case "code", "coding", "programming":
		return containsTag(model.Tags, "code") || containsTag(model.Tags, "programming")
	case "general", "chat", "conversation":
		return containsTag(model.Tags, "general") || containsTag(model.Tags, "chat")
	case "fast", "quick":
		return containsTag(model.Tags, "fast") || containsTag(model.Tags, "quick")
	case "advanced", "complex":
		return containsTag(model.Tags, "advanced") || containsTag(model.Tags, "complex")
	}

	return false
}

// containsTag checks if a slice contains a specific tag
func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
}

// ModelRouter handles model selection with multiple strategies
type ModelRouter struct {
	registry        *registry.Registry
	strategies      map[string]ModelSelector
	defaultStrategy string
	defaultModel    string
}

// NewModelRouter creates a new model router
func NewModelRouter(registry *registry.Registry) *ModelRouter {
	// Create default strategies
	roundRobin := NewRoundRobinStrategy()
	weighted := NewWeightedStrategy()
	tagBased := NewTagBasedStrategy(weighted)
	costAware := NewCostAwareStrategy(0.01, 0.6, 0.4) // 60% quality, 40% cost
	latencyBased := NewLatencyBasedStrategy(100)      // Keep 100 recent measurements
	loadBalancing := NewLoadBalancingStrategy()
	abTesting := NewABTestingStrategy()

	router := &ModelRouter{
		registry: registry,
		strategies: map[string]ModelSelector{
			"round-robin":    roundRobin,
			"weighted":       weighted,
			"tag-based":      tagBased,
			"cost-aware":     costAware,
			"latency-based":  latencyBased,
			"load-balancing": loadBalancing,
			"ab-testing":     abTesting,
		},
		defaultStrategy: "cost-aware",
		defaultModel:    os.Getenv("DEFAULT_MODEL"),
	}

	return router
}

// SelectModel selects a model using the specified strategy
func (r *ModelRouter) SelectModel(ctx context.Context, strategy string, metadata map[string]string) (*registry.ModelConfig, error) {
	// Get available models
	models := r.registry.Models
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available in registry")
	}

	// Use specified strategy or default
	selector, exists := r.strategies[strategy]
	if !exists {
		selector = r.strategies[r.defaultStrategy]
	}

	// Select model
	selectedModel, err := selector.SelectModel(ctx, models, metadata)
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	// If no model was selected and we have a default model, try to find it
	if selectedModel == nil && r.defaultModel != "" {
		for _, model := range models {
			if model.ID == r.defaultModel {
				return &model, nil
			}
		}
	}

	if selectedModel == nil {
		return nil, fmt.Errorf("no suitable model found")
	}

	return selectedModel, nil
}

// GetAvailableStrategies returns the list of available strategies
func (r *ModelRouter) GetAvailableStrategies() []string {
	strategies := make([]string, 0, len(r.strategies))
	for strategy := range r.strategies {
		strategies = append(strategies, strategy)
	}
	return strategies
}

// GetModelsByKind returns models filtered by kind (chat, embed, complete)
func (r *ModelRouter) GetModelsByKind(kind string) []registry.ModelConfig {
	var filtered []registry.ModelConfig
	for _, model := range r.registry.Models {
		if model.Kind == kind {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

// CostAwareStrategy implements cost-aware model selection
type CostAwareStrategy struct {
	costThreshold float64
	qualityWeight float64
	costWeight    float64
}

// NewCostAwareStrategy creates a new cost-aware strategy
func NewCostAwareStrategy(costThreshold, qualityWeight, costWeight float64) *CostAwareStrategy {
	return &CostAwareStrategy{
		costThreshold: costThreshold,
		qualityWeight: qualityWeight,
		costWeight:    costWeight,
	}
}

// SelectModel selects a model based on cost and quality balance
func (c *CostAwareStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Calculate scores for each model
	type ModelScore struct {
		model registry.ModelConfig
		score float64
	}

	scores := make([]ModelScore, 0, len(models))
	for _, model := range models {
		score := c.calculateScore(model, metadata)
		scores = append(scores, ModelScore{model: model, score: score})
	}

	// Sort by score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return the highest scoring model
	return &scores[0].model, nil
}

// calculateScore calculates a score for a model based on cost and quality
func (c *CostAwareStrategy) calculateScore(model registry.ModelConfig, metadata map[string]string) float64 {
	// Get cost per 1K tokens (input + output)
	inputCost := model.Pricing.InputPer1K
	outputCost := model.Pricing.OutputPer1K
	totalCost := inputCost + outputCost

	// Calculate quality score based on model capabilities
	qualityScore := c.calculateQualityScore(model)

	// Calculate cost score (lower cost = higher score)
	costScore := 0.0
	if totalCost == 0 {
		costScore = 1.0 // Free models get highest cost score
	} else {
		costScore = 1.0 / (1.0 + totalCost) // Inverse cost scoring
	}

	// Weighted combination
	return c.qualityWeight*qualityScore + c.costWeight*costScore
}

// calculateQualityScore calculates quality score based on model capabilities
func (c *CostAwareStrategy) calculateQualityScore(model registry.ModelConfig) float64 {
	score := 0.5 // Base score

	// Boost for advanced models
	if containsTag(model.Tags, "advanced") || containsTag(model.Tags, "complex") {
		score += 0.3
	}

	// Boost for fast models
	if containsTag(model.Tags, "fast") || containsTag(model.Tags, "quick") {
		score += 0.2
	}

	// Boost for code-specific models
	if containsTag(model.Tags, "code") || containsTag(model.Tags, "programming") {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// LatencyBasedStrategy implements latency-based model selection
type LatencyBasedStrategy struct {
	latencyHistory map[string][]time.Duration
	mu             sync.RWMutex
	maxHistory     int
}

// NewLatencyBasedStrategy creates a new latency-based strategy
func NewLatencyBasedStrategy(maxHistory int) *LatencyBasedStrategy {
	return &LatencyBasedStrategy{
		latencyHistory: make(map[string][]time.Duration),
		maxHistory:     maxHistory,
	}
}

// SelectModel selects a model based on historical latency
func (l *LatencyBasedStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Calculate average latency for each model
	type ModelLatency struct {
		model   registry.ModelConfig
		latency time.Duration
	}

	latencies := make([]ModelLatency, 0, len(models))
	for _, model := range models {
		avgLatency := l.getAverageLatency(model.ID)
		latencies = append(latencies, ModelLatency{model: model, latency: avgLatency})
	}

	// Sort by latency (ascending - faster first)
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].latency < latencies[j].latency
	})

	// Return the fastest model
	return &latencies[0].model, nil
}

// RecordLatency records latency for a model
func (l *LatencyBasedStrategy) RecordLatency(modelID string, latency time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	history := l.latencyHistory[modelID]
	history = append(history, latency)

	// Keep only the most recent measurements
	if len(history) > l.maxHistory {
		history = history[len(history)-l.maxHistory:]
	}

	l.latencyHistory[modelID] = history
}

// getAverageLatency returns the average latency for a model
func (l *LatencyBasedStrategy) getAverageLatency(modelID string) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	history := l.latencyHistory[modelID]
	if len(history) == 0 {
		return time.Minute // Default high latency for unknown models
	}

	var total time.Duration
	for _, latency := range history {
		total += latency
	}

	return total / time.Duration(len(history))
}

// LoadBalancingStrategy implements load balancing with health checks
type LoadBalancingStrategy struct {
	modelHealth    map[string]bool
	modelLoad      map[string]int
	mu             sync.RWMutex
	healthCheckers map[string]HealthChecker
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	IsHealthy(modelID string) bool
}

// NewLoadBalancingStrategy creates a new load balancing strategy
func NewLoadBalancingStrategy() *LoadBalancingStrategy {
	return &LoadBalancingStrategy{
		modelHealth:    make(map[string]bool),
		modelLoad:      make(map[string]int),
		healthCheckers: make(map[string]HealthChecker),
	}
}

// SelectModel selects a model based on load and health
func (lb *LoadBalancingStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Filter healthy models
	var healthyModels []registry.ModelConfig
	for _, model := range models {
		if lb.isHealthy(model.ID) {
			healthyModels = append(healthyModels, model)
		}
	}

	// If no healthy models, use all models
	if len(healthyModels) == 0 {
		healthyModels = models
	}

	// Select model with least load
	selectedModel := &healthyModels[0]
	minLoad := lb.getLoad(selectedModel.ID)

	for i := 1; i < len(healthyModels); i++ {
		model := &healthyModels[i]
		load := lb.getLoad(model.ID)
		if load < minLoad {
			selectedModel = model
			minLoad = load
		}
	}

	// Increment load for selected model
	lb.incrementLoad(selectedModel.ID)

	return selectedModel, nil
}

// isHealthy checks if a model is healthy
func (lb *LoadBalancingStrategy) isHealthy(modelID string) bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Check if we have a health checker for this model
	if checker, exists := lb.healthCheckers[modelID]; exists {
		return checker.IsHealthy(modelID)
	}

	// Default to healthy if no checker
	return lb.modelHealth[modelID] || !lb.modelHealth[modelID] // Default to true
}

// getLoad returns the current load for a model
func (lb *LoadBalancingStrategy) getLoad(modelID string) int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.modelLoad[modelID]
}

// incrementLoad increments the load for a model
func (lb *LoadBalancingStrategy) incrementLoad(modelID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.modelLoad[modelID]++
}

// decrementLoad decrements the load for a model
func (lb *LoadBalancingStrategy) decrementLoad(modelID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lb.modelLoad[modelID] > 0 {
		lb.modelLoad[modelID]--
	}
}

// SetHealthChecker sets a health checker for a model
func (lb *LoadBalancingStrategy) SetHealthChecker(modelID string, checker HealthChecker) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.healthCheckers[modelID] = checker
}

// SetHealth sets the health status for a model
func (lb *LoadBalancingStrategy) SetHealth(modelID string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.modelHealth[modelID] = healthy
}

// A/B Testing Strategy
type ABTestingStrategy struct {
	experiments map[string]*ABExperiment
	mu          sync.RWMutex
}

// ABExperiment represents an A/B test experiment
type ABExperiment struct {
	Name         string
	Variants     []ABVariant
	TrafficSplit []float64
	StartTime    time.Time
	EndTime      time.Time
	Active       bool
}

// ABVariant represents a variant in an A/B test
type ABVariant struct {
	Name   string
	Model  string
	Weight float64
}

// NewABTestingStrategy creates a new A/B testing strategy
func NewABTestingStrategy() *ABTestingStrategy {
	return &ABTestingStrategy{
		experiments: make(map[string]*ABExperiment),
	}
}

// SelectModel selects a model based on A/B testing
func (ab *ABTestingStrategy) SelectModel(ctx context.Context, models []registry.ModelConfig, metadata map[string]string) (*registry.ModelConfig, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Get experiment name from metadata
	experimentName := metadata["experiment"]
	if experimentName == "" {
		// No experiment, use first model
		return &models[0], nil
	}

	ab.mu.RLock()
	experiment, exists := ab.experiments[experimentName]
	ab.mu.RUnlock()

	if !exists || !experiment.Active {
		// No active experiment, use first model
		return &models[0], nil
	}

	// Check if experiment is still active
	now := time.Now()
	if now.Before(experiment.StartTime) || now.After(experiment.EndTime) {
		return &models[0], nil
	}

	// Select variant based on traffic split
	selectedVariant := ab.selectVariant(experiment)
	if selectedVariant == nil {
		return &models[0], nil
	}

	// Find the model for the selected variant
	for _, model := range models {
		if model.ID == selectedVariant.Model {
			return &model, nil
		}
	}

	// Fallback to first model
	return &models[0], nil
}

// selectVariant selects a variant based on traffic split
func (ab *ABTestingStrategy) selectVariant(experiment *ABExperiment) *ABVariant {
	if len(experiment.Variants) == 0 {
		return nil
	}

	// Generate random number between 0 and 1
	random := rand.Float64()

	// Find which variant to select based on cumulative weights
	cumulativeWeight := 0.0
	for _, variant := range experiment.Variants {
		cumulativeWeight += variant.Weight
		if random <= cumulativeWeight {
			return &variant
		}
	}

	// Fallback to last variant
	return &experiment.Variants[len(experiment.Variants)-1]
}

// AddExperiment adds an A/B test experiment
func (ab *ABTestingStrategy) AddExperiment(experiment *ABExperiment) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.experiments[experiment.Name] = experiment
}

// RemoveExperiment removes an A/B test experiment
func (ab *ABTestingStrategy) RemoveExperiment(name string) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	delete(ab.experiments, name)
}

// GetExperiments returns all experiments
func (ab *ABTestingStrategy) GetExperiments() map[string]*ABExperiment {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*ABExperiment)
	for k, v := range ab.experiments {
		result[k] = v
	}
	return result
}
