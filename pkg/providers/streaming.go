package providers

import (
	"context"
	"time"

	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
	"github.com/snow-ghost/agent/pkg/streaming"
)

// StreamingProvider defines the interface for providers that support streaming
type StreamingProvider interface {
	Provider

	// ChatStream performs streaming chat completion
	ChatStream(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest, writer *streaming.SSEWriter) error
}

// StreamHandler handles streaming responses
type StreamHandler struct {
	writer     *streaming.SSEWriter
	aggregator *streaming.UsageAggregator
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(writer *streaming.SSEWriter) *StreamHandler {
	return &StreamHandler{
		writer:     writer,
		aggregator: streaming.NewUsageAggregator(),
	}
}

// HandleChunk handles a text chunk
func (h *StreamHandler) HandleChunk(text string) error {
	return h.writer.WriteChunk(text)
}

// HandleUsage handles usage updates
func (h *StreamHandler) HandleUsage(usage core.Usage) {
	h.aggregator.AddUsage(usage)
}

// HandleDone handles completion
func (h *StreamHandler) HandleDone(model, provider string, finishReason string, cost float64, currency string) error {
	finalUsage := h.aggregator.GetUsage()

	return h.writer.WriteDone(finalUsage, cost, currency)
}

// HandleError handles errors
func (h *StreamHandler) HandleError(err error) error {
	return h.writer.WriteError(err)
}

// WriteStart writes the start event
func (h *StreamHandler) WriteStart(model, provider string) error {
	return h.writer.WriteStart(model, provider)
}

// MockStreamingProvider implements streaming for testing
type MockStreamingProvider struct {
	*BaseProvider
}

// NewMockStreamingProvider creates a new mock streaming provider
func NewMockStreamingProvider() *MockStreamingProvider {
	registry := registry.GetDefaultRegistry()
	return &MockStreamingProvider{
		BaseProvider: NewBaseProvider(registry),
	}
}

// Chat performs non-streaming chat completion
func (p *MockStreamingProvider) Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error) {
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	return core.ChatResponse{
		Text: "Mock response from streaming provider",
		Usage: core.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
		Model:        mc.ID,
		Provider:     mc.Provider,
		FinishReason: "stop",
	}, nil
}

// Embed generates embeddings
func (p *MockStreamingProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error) {
	// Mock embeddings
	embeddings := make([][]float32, len(input))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536) // Mock 1536-dimensional embeddings
	}

	usage := core.Usage{
		PromptTokens:     10,
		CompletionTokens: 0,
		TotalTokens:      10,
	}

	return embeddings, usage, nil
}

// ChatStream performs streaming chat completion
func (p *MockStreamingProvider) ChatStream(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest, writer *streaming.SSEWriter) error {
	// Write start event
	if err := writer.WriteStart(mc.ID, mc.Provider); err != nil {
		return err
	}

	// Simulate streaming response
	chunks := []string{
		"This is a mock ",
		"streaming response ",
		"that will be sent ",
		"in chunks to ",
		"demonstrate ",
		"Server-Sent Events ",
		"functionality.",
	}

	handler := NewStreamHandler(writer)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate processing time
		time.Sleep(200 * time.Millisecond)

		// Write chunk
		if err := handler.HandleChunk(chunk); err != nil {
			return err
		}
	}

	// Simulate usage
	usage := core.Usage{
		PromptTokens:     10,
		CompletionTokens: 15,
		TotalTokens:      25,
	}
	handler.HandleUsage(usage)

	// Calculate cost
	cost, currency := p.calculateCost(mc, usage)

	// Write done event
	return handler.HandleDone(mc.ID, mc.Provider, "stop", cost, currency)
}

// calculateCost calculates the cost for the usage
func (p *MockStreamingProvider) calculateCost(mc registry.ModelConfig, usage core.Usage) (float64, string) {
	if p.costCalculator != nil {
		if result, err := p.costCalculator.CalcCostForModel(mc.ID, usage); err == nil {
			return result.TotalCost, result.Currency
		}
	}
	return 0.0, "USD"
}
