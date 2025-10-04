package routing

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
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

	router := &ModelRouter{
		registry: registry,
		strategies: map[string]ModelSelector{
			"round-robin": roundRobin,
			"weighted":    weighted,
			"tag-based":   tagBased,
		},
		defaultStrategy: "tag-based",
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
