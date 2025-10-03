package providers

import (
	"testing"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestProviderCostCalculation(t *testing.T) {

	// Test OpenAI provider
	t.Run("OpenAI Provider", func(t *testing.T) {
		provider := NewOpenAIProvider("https://api.openai.com/v1", "test-key")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("OpenAI provider should have a cost calculator")
		}

		usage := core.Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		}

		costResult, err := calculator.CalcCostForModel("openai:gpt-4o-mini", usage)
		if err != nil {
			t.Fatalf("OpenAI cost calculation failed: %v", err)
		}

		if costResult.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", costResult.Currency)
		}

		if costResult.TotalCost <= 0 {
			t.Error("Expected positive total cost")
		}
	})

	// Test Anthropic provider
	t.Run("Anthropic Provider", func(t *testing.T) {
		provider := NewAnthropicProvider("https://api.anthropic.com", "test-key")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("Anthropic provider should have a cost calculator")
		}

		usage := core.Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		}

		costResult, err := calculator.CalcCostForModel("anthropic:claude-3-5-sonnet-20241022", usage)
		if err != nil {
			t.Fatalf("Anthropic cost calculation failed: %v", err)
		}

		if costResult.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", costResult.Currency)
		}

		if costResult.TotalCost <= 0 {
			t.Error("Expected positive total cost")
		}
	})

	// Test Ollama provider
	t.Run("Ollama Provider", func(t *testing.T) {
		provider := NewOllamaProvider("http://localhost:11434")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("Ollama provider should have a cost calculator")
		}

		usage := core.Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		}

		costResult, err := calculator.CalcCostForModel("ollama:llama3.2", usage)
		if err != nil {
			t.Fatalf("Ollama cost calculation failed: %v", err)
		}

		if costResult.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", costResult.Currency)
		}

		// Ollama should have zero cost (local model)
		if costResult.TotalCost != 0 {
			t.Errorf("Expected zero cost for local model, got %f", costResult.TotalCost)
		}
	})

	// Test vLLM provider
	t.Run("vLLM Provider", func(t *testing.T) {
		provider := NewVLLMProvider("http://localhost:8000", "dummy-key")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("vLLM provider should have a cost calculator")
		}
	})

	// Test LM Studio provider
	t.Run("LM Studio Provider", func(t *testing.T) {
		provider := NewLMStudioProvider("http://localhost:1234", "dummy-key")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("LM Studio provider should have a cost calculator")
		}
	})

	// Test OpenRouter provider
	t.Run("OpenRouter Provider", func(t *testing.T) {
		provider := NewOpenRouterProvider("https://openrouter.ai/api/v1", "test-key")
		calculator := provider.GetCostCalculator()

		if calculator == nil {
			t.Fatal("OpenRouter provider should have a cost calculator")
		}
	})
}

func TestProviderFactoryWithRegistry(t *testing.T) {
	// Create a test registry
	reg := &registry.Registry{
		Models: []registry.ModelConfig{
			{
				ID:        "ollama:llama3.2",
				Provider:  "ollama",
				BaseURL:   "http://localhost:11434",
				APIKeyEnv: "", // No API key required
				Pricing: registry.Pricing{
					Currency:    "USD",
					InputPer1K:  0.0,
					OutputPer1K: 0.0,
				},
			},
		},
	}

	factory := &DefaultProviderFactory{}

	// Test creating provider from config with registry
	modelConfig := registry.ModelConfig{
		ID:        "ollama:llama3.2",
		Provider:  "ollama",
		BaseURL:   "http://localhost:11434",
		APIKeyEnv: "", // No API key required
		Pricing: registry.Pricing{
			Currency:    "USD",
			InputPer1K:  0.0,
			OutputPer1K: 0.0,
		},
	}

	provider, err := factory.CreateProviderFromConfig(modelConfig, reg)
	if err != nil {
		t.Fatalf("Failed to create provider from config: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider should not be nil")
	}

	// Test that the provider has cost calculation capability
	calculator := provider.GetCostCalculator()
	if calculator == nil {
		t.Fatal("Provider should have a cost calculator")
	}
}
