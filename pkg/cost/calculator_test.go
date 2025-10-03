package cost

import (
	"testing"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestCalcCost(t *testing.T) {
	tests := []struct {
		name     string
		usage    core.Usage
		pricing  registry.Pricing
		expected struct {
			inputCost  float64
			outputCost float64
			totalCost  float64
		}
	}{
		{
			name: "basic calculation",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
				TotalTokens:      1500,
			},
			pricing: registry.Pricing{
				Currency:    "USD",
				InputPer1K:  0.0015,
				OutputPer1K: 0.006,
			},
			expected: struct {
				inputCost  float64
				outputCost float64
				totalCost  float64
			}{
				inputCost:  0.0015,
				outputCost: 0.003,
				totalCost:  0.0045,
			},
		},
		{
			name: "zero tokens",
			usage: core.Usage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			},
			pricing: registry.Pricing{
				Currency:    "USD",
				InputPer1K:  0.0015,
				OutputPer1K: 0.006,
			},
			expected: struct {
				inputCost  float64
				outputCost float64
				totalCost  float64
			}{
				inputCost:  0.0,
				outputCost: 0.0,
				totalCost:  0.0,
			},
		},
		{
			name: "high precision",
			usage: core.Usage{
				PromptTokens:     1,
				CompletionTokens: 1,
				TotalTokens:      2,
			},
			pricing: registry.Pricing{
				Currency:    "USD",
				InputPer1K:  0.00015,
				OutputPer1K: 0.0006,
			},
			expected: struct {
				inputCost  float64
				outputCost float64
				totalCost  float64
			}{
				inputCost:  0.0,      // Rounded to 0 due to precision
				outputCost: 0.000001, // Rounded to 6 decimal places
				totalCost:  0.000001, // Rounded to 6 decimal places
			},
		},
		{
			name: "euro currency",
			usage: core.Usage{
				PromptTokens:     2000,
				CompletionTokens: 1000,
				TotalTokens:      3000,
			},
			pricing: registry.Pricing{
				Currency:    "EUR",
				InputPer1K:  0.002,
				OutputPer1K: 0.008,
			},
			expected: struct {
				inputCost  float64
				outputCost float64
				totalCost  float64
			}{
				inputCost:  0.004,
				outputCost: 0.008,
				totalCost:  0.012,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCost, outputCost, totalCost := CalcCost(tt.usage, tt.pricing)

			if inputCost != tt.expected.inputCost {
				t.Errorf("CalcCost() inputCost = %v, want %v", inputCost, tt.expected.inputCost)
			}
			if outputCost != tt.expected.outputCost {
				t.Errorf("CalcCost() outputCost = %v, want %v", outputCost, tt.expected.outputCost)
			}
			if totalCost != tt.expected.totalCost {
				t.Errorf("CalcCost() totalCost = %v, want %v", totalCost, tt.expected.totalCost)
			}
		})
	}
}

func TestFormatCostHeader(t *testing.T) {
	tests := []struct {
		name     string
		cost     *CostResult
		expected string
	}{
		{
			name: "USD cost",
			cost: &CostResult{
				TotalCost: 0.0045,
				Currency:  "USD",
			},
			expected: "X-Cost-Total=0.004500;currency=USD",
		},
		{
			name: "EUR cost",
			cost: &CostResult{
				TotalCost: 0.012,
				Currency:  "EUR",
			},
			expected: "X-Cost-Total=0.012000;currency=EUR",
		},
		{
			name: "zero cost",
			cost: &CostResult{
				TotalCost: 0.0,
				Currency:  "USD",
			},
			expected: "X-Cost-Total=0.000000;currency=USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCostHeader(tt.cost)
			if result != tt.expected {
				t.Errorf("FormatCostHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatCostHeaders(t *testing.T) {
	tests := []struct {
		name     string
		costs    []*CostResult
		expected map[string]string
	}{
		{
			name:     "no costs",
			costs:    []*CostResult{},
			expected: map[string]string{},
		},
		{
			name: "single cost",
			costs: []*CostResult{
				{TotalCost: 0.0045, Currency: "USD"},
			},
			expected: map[string]string{
				"X-Cost-Total": "0.004500;currency=USD",
			},
		},
		{
			name: "multiple costs",
			costs: []*CostResult{
				{TotalCost: 0.0045, Currency: "USD"},
				{TotalCost: 0.0025, Currency: "USD"},
			},
			expected: map[string]string{
				"X-Cost-Total":   "0.007000;currency=USD",
				"X-Cost-Model-0": "0.004500;currency=USD",
				"X-Cost-Model-1": "0.002500;currency=USD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCostHeaders(tt.costs)

			if len(result) != len(tt.expected) {
				t.Errorf("FormatCostHeaders() returned %d headers, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("FormatCostHeaders() missing header %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("FormatCostHeaders() header %s = %v, want %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		pricing       registry.Pricing
		tokensPerChar float64
		expected      struct {
			inputTokens  int
			outputTokens int
			totalTokens  int
		}
	}{
		{
			name: "basic estimation",
			text: "This is a test message with some content.",
			pricing: registry.Pricing{
				Currency:    "USD",
				InputPer1K:  0.0015,
				OutputPer1K: 0.006,
			},
			tokensPerChar: 0.25, // 4 chars per token
			expected: struct {
				inputTokens  int
				outputTokens int
				totalTokens  int
			}{
				inputTokens:  8,  // 40 chars * 0.25 * 0.8
				outputTokens: 2,  // 40 chars * 0.25 * 0.2
				totalTokens:  10, // 40 chars * 0.25
			},
		},
		{
			name: "empty text",
			text: "",
			pricing: registry.Pricing{
				Currency:    "USD",
				InputPer1K:  0.0015,
				OutputPer1K: 0.006,
			},
			tokensPerChar: 0.25,
			expected: struct {
				inputTokens  int
				outputTokens int
				totalTokens  int
			}{
				inputTokens:  0, // 0 chars * 0.25 * 0.8 = 0
				outputTokens: 0, // 0 chars * 0.25 * 0.2 = 0
				totalTokens:  0, // 0 chars * 0.25 = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateCost(tt.text, tt.pricing, tt.tokensPerChar)

			if result.InputTokens != tt.expected.inputTokens {
				t.Errorf("EstimateCost() InputTokens = %v, want %v", result.InputTokens, tt.expected.inputTokens)
			}
			if result.OutputTokens != tt.expected.outputTokens {
				t.Errorf("EstimateCost() OutputTokens = %v, want %v", result.OutputTokens, tt.expected.outputTokens)
			}
			if result.TotalTokens != tt.expected.totalTokens {
				t.Errorf("EstimateCost() TotalTokens = %v, want %v", result.TotalTokens, tt.expected.totalTokens)
			}
		})
	}
}

func TestCalculator_CalcCostForModel(t *testing.T) {
	// Create a test registry
	registry := &registry.Registry{
		Models: []registry.ModelConfig{
			{
				ID: "gpt-4o-mini",
				Pricing: registry.Pricing{
					Currency:    "USD",
					InputPer1K:  0.00015,
					OutputPer1K: 0.0006,
				},
			},
		},
	}

	calculator := NewCalculator(registry)

	usage := core.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	result, err := calculator.CalcCostForModel("gpt-4o-mini", usage)
	if err != nil {
		t.Fatalf("CalcCostForModel() error = %v", err)
	}

	expectedInputCost := 0.00015
	expectedOutputCost := 0.0003
	expectedTotalCost := 0.00045

	if result.InputCost != expectedInputCost {
		t.Errorf("CalcCostForModel() InputCost = %v, want %v", result.InputCost, expectedInputCost)
	}
	if result.OutputCost != expectedOutputCost {
		t.Errorf("CalcCostForModel() OutputCost = %v, want %v", result.OutputCost, expectedOutputCost)
	}
	if result.TotalCost != expectedTotalCost {
		t.Errorf("CalcCostForModel() TotalCost = %v, want %v", result.TotalCost, expectedTotalCost)
	}
	if result.Currency != "USD" {
		t.Errorf("CalcCostForModel() Currency = %v, want USD", result.Currency)
	}
}

func TestCalculator_CalcCostForModel_NotFound(t *testing.T) {
	registry := &registry.Registry{
		Models: []registry.ModelConfig{},
	}

	calculator := NewCalculator(registry)

	usage := core.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	_, err := calculator.CalcCostForModel("nonexistent-model", usage)
	if err == nil {
		t.Error("CalcCostForModel() expected error for nonexistent model")
	}
}
