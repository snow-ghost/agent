package routing

import (
	"context"
	"os"
	"testing"

	"github.com/snow-ghost/agent/pkg/registry"
)

func TestRoundRobinStrategy(t *testing.T) {
	strategy := NewRoundRobinStrategy()

	models := []registry.ModelConfig{
		{ID: "model1", Provider: "test"},
		{ID: "model2", Provider: "test"},
		{ID: "model3", Provider: "test"},
	}

	// Test round-robin selection
	selected1, err := strategy.SelectModel(context.Background(), models, map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected1.ID != "model1" {
		t.Errorf("Expected model1, got %s", selected1.ID)
	}

	selected2, err := strategy.SelectModel(context.Background(), models, map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected2.ID != "model2" {
		t.Errorf("Expected model2, got %s", selected2.ID)
	}

	selected3, err := strategy.SelectModel(context.Background(), models, map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected3.ID != "model3" {
		t.Errorf("Expected model3, got %s", selected3.ID)
	}

	// Should wrap around
	selected4, err := strategy.SelectModel(context.Background(), models, map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected4.ID != "model1" {
		t.Errorf("Expected model1 (wrapped), got %s", selected4.ID)
	}
}

func TestWeightedStrategy(t *testing.T) {
	strategy := NewWeightedStrategy()

	models := []registry.ModelConfig{
		{ID: "cheap", Provider: "test", Pricing: registry.Pricing{InputPer1K: 0.001}},
		{ID: "expensive", Provider: "test", Pricing: registry.Pricing{InputPer1K: 0.01}},
		{ID: "free", Provider: "test", Pricing: registry.Pricing{InputPer1K: 0.0}},
	}

	// Test that we can select a model
	selected, err := strategy.SelectModel(context.Background(), models, map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected == nil {
		t.Fatal("Expected a selected model")
	}

	// Test multiple selections to see if cheaper models are selected more often
	selections := make(map[string]int)
	for i := 0; i < 1000; i++ {
		selected, err := strategy.SelectModel(context.Background(), models, map[string]string{})
		if err != nil {
			t.Fatalf("SelectModel failed: %v", err)
		}
		selections[selected.ID]++
	}

	// All models should be selected at least once
	if selections["free"] == 0 || selections["cheap"] == 0 || selections["expensive"] == 0 {
		t.Error("All models should be selected at least once")
	}
}

func TestTagBasedStrategy(t *testing.T) {

	models := []registry.ModelConfig{
		{ID: "general-model", Provider: "test", Kind: "chat", Tags: []string{"general"}},
		{ID: "code-model", Provider: "test", Kind: "chat", Tags: []string{"code"}},
		{ID: "embed-model", Provider: "test", Kind: "embed", Tags: []string{"embed"}},
		{ID: "fast-model", Provider: "test", Kind: "chat", Tags: []string{"fast"}},
	}

	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{"code domain", "code", "code-model"},
		{"embedding domain", "embed", "embed-model"},
		{"general domain", "general", "general-model"},
		{"fast domain", "fast", "fast-model"},
		{"unknown domain", "unknown", "general-model"}, // Should fallback to first model (round-robin starts with index 0)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh strategies for each test to avoid state pollution
			fallback := NewRoundRobinStrategy()
			strategy := NewTagBasedStrategy(fallback)

			selected, err := strategy.SelectModel(context.Background(), models, map[string]string{
				"task_domain": tt.domain,
			})
			if err != nil {
				t.Fatalf("SelectModel failed: %v", err)
			}
			if selected.ID != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, selected.ID)
			}
		})
	}
}

func TestModelRouter(t *testing.T) {
	registry := &registry.Registry{
		Models: []registry.ModelConfig{
			{ID: "model1", Provider: "test", Kind: "chat", Tags: []string{"general"}},
			{ID: "model2", Provider: "test", Kind: "chat", Tags: []string{"code"}},
			{ID: "model3", Provider: "test", Kind: "embed", Tags: []string{"embed"}},
		},
	}

	router := NewModelRouter(registry)

	// Test available strategies
	strategies := router.GetAvailableStrategies()
	expectedStrategies := []string{"round-robin", "weighted", "tag-based"}

	if len(strategies) != len(expectedStrategies) {
		t.Errorf("Expected %d strategies, got %d", len(expectedStrategies), len(strategies))
	}

	// Test model selection
	selected, err := router.SelectModel(context.Background(), "round-robin", map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected == nil {
		t.Fatal("Expected a selected model")
	}

	// Test models by kind
	chatModels := router.GetModelsByKind("chat")
	if len(chatModels) != 2 {
		t.Errorf("Expected 2 chat models, got %d", len(chatModels))
	}

	embedModels := router.GetModelsByKind("embed")
	if len(embedModels) != 1 {
		t.Errorf("Expected 1 embed model, got %d", len(embedModels))
	}
}

func TestModelRouterWithDefaultModel(t *testing.T) {
	registry := &registry.Registry{
		Models: []registry.ModelConfig{
			{ID: "default-model", Provider: "test", Kind: "chat"},
		},
	}

	// Set default model via environment (in test we'll simulate this)
	originalDefault := os.Getenv("DEFAULT_MODEL")
	defer func() {
		if originalDefault != "" {
			os.Setenv("DEFAULT_MODEL", originalDefault)
		} else {
			os.Unsetenv("DEFAULT_MODEL")
		}
	}()

	os.Setenv("DEFAULT_MODEL", "default-model")
	router := NewModelRouter(registry)

	// Test that default model is used when no strategy works
	selected, err := router.SelectModel(context.Background(), "round-robin", map[string]string{})
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if selected.ID != "default-model" {
		t.Errorf("Expected default-model, got %s", selected.ID)
	}
}

func TestModelRouterEmptyRegistry(t *testing.T) {
	registry := &registry.Registry{Models: []registry.ModelConfig{}}
	router := NewModelRouter(registry)

	_, err := router.SelectModel(context.Background(), "round-robin", map[string]string{})
	if err == nil {
		t.Error("Expected error for empty registry")
	}
}
