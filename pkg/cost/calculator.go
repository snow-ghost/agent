package cost

import (
	"fmt"
	"math"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// CostResult represents the calculated cost breakdown
type CostResult struct {
	InputCost    float64 `json:"input_cost"`
	OutputCost   float64 `json:"output_cost"`
	TotalCost    float64 `json:"total_cost"`
	Currency     string  `json:"currency"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
}

// Calculator handles cost calculations
type Calculator struct {
	registry *registry.Registry
}

// NewCalculator creates a new cost calculator
func NewCalculator(registry *registry.Registry) *Calculator {
	return &Calculator{
		registry: registry,
	}
}

// CalcCost calculates the cost for usage and pricing
func CalcCost(u core.Usage, p registry.Pricing) (inputCost, outputCost, total float64) {
	// Calculate input cost (per 1k tokens)
	inputCost = float64(u.PromptTokens) * p.InputPer1K / 1000.0

	// Calculate output cost (per 1k tokens)
	outputCost = float64(u.CompletionTokens) * p.OutputPer1K / 1000.0

	// Round to 6 decimal places for precision
	inputCost = math.Round(inputCost*1000000) / 1000000
	outputCost = math.Round(outputCost*1000000) / 1000000

	total = inputCost + outputCost
	total = math.Round(total*1000000) / 1000000

	return inputCost, outputCost, total
}

// CalcCostForModel calculates cost for a specific model
func (c *Calculator) CalcCostForModel(modelID string, usage core.Usage) (*CostResult, error) {
	// Find the model in the registry
	var modelConfig *registry.ModelConfig
	for _, model := range c.registry.Models {
		if model.ID == modelID {
			modelConfig = &model
			break
		}
	}

	if modelConfig == nil {
		return nil, fmt.Errorf("model %s not found in registry", modelID)
	}

	inputCost, outputCost, totalCost := CalcCost(usage, modelConfig.Pricing)

	return &CostResult{
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Currency:     modelConfig.Pricing.Currency,
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}, nil
}

// CalcCostForProvider calculates cost for a provider (aggregated)
func (c *Calculator) CalcCostForProvider(provider string, usage core.Usage) (*CostResult, error) {
	// Find models for this provider
	var totalInputCost, totalOutputCost, totalCost float64
	var currency string
	var totalInputTokens, totalOutputTokens, totalTokens int

	for _, model := range c.registry.Models {
		if model.Provider == provider {
			// Use the first model's pricing as reference
			if currency == "" {
				currency = model.Pricing.Currency
			}

			inputCost, outputCost, _ := CalcCost(usage, model.Pricing)
			totalInputCost += inputCost
			totalOutputCost += outputCost
			totalCost += inputCost + outputCost

			totalInputTokens += usage.PromptTokens
			totalOutputTokens += usage.CompletionTokens
			totalTokens += usage.TotalTokens
		}
	}

	if currency == "" {
		return nil, fmt.Errorf("provider %s not found in registry", provider)
	}

	return &CostResult{
		InputCost:    totalInputCost,
		OutputCost:   totalOutputCost,
		TotalCost:    totalCost,
		Currency:     currency,
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		TotalTokens:  totalTokens,
	}, nil
}

// FormatCostHeader formats cost for HTTP header
func FormatCostHeader(cost *CostResult) string {
	return fmt.Sprintf("X-Cost-Total=%.6f;currency=%s", cost.TotalCost, cost.Currency)
}

// FormatCostHeaders formats multiple cost headers
func FormatCostHeaders(costs []*CostResult) map[string]string {
	headers := make(map[string]string)

	if len(costs) == 0 {
		return headers
	}

	// Single cost
	if len(costs) == 1 {
		headers["X-Cost-Total"] = fmt.Sprintf("%.6f;currency=%s", costs[0].TotalCost, costs[0].Currency)
		return headers
	}

	// Multiple costs - aggregate
	var totalCost float64
	var currency string

	for _, cost := range costs {
		if currency == "" {
			currency = cost.Currency
		}
		totalCost += cost.TotalCost
	}

	headers["X-Cost-Total"] = fmt.Sprintf("%.6f;currency=%s", totalCost, currency)

	// Add breakdown headers
	for i, cost := range costs {
		headers[fmt.Sprintf("X-Cost-Model-%d", i)] = fmt.Sprintf("%.6f;currency=%s", cost.TotalCost, cost.Currency)
	}

	return headers
}

// EstimateCost estimates cost based on text length when usage is not available
func EstimateCost(text string, pricing registry.Pricing, tokensPerChar float64) *CostResult {
	// Estimate tokens based on character count
	estimatedTokens := int(float64(len(text)) * tokensPerChar)
	// Don't enforce minimum for empty text

	// Assume 80% input, 20% output for estimation
	inputTokens := int(float64(estimatedTokens) * 0.8)
	outputTokens := int(float64(estimatedTokens) * 0.2)

	usage := core.Usage{
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		TotalTokens:      estimatedTokens,
	}

	inputCost, outputCost, totalCost := CalcCost(usage, pricing)

	return &CostResult{
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    totalCost,
		Currency:     pricing.Currency,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  estimatedTokens,
	}
}
