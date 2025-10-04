package client

import (
	"context"
	"fmt"

	"github.com/snow-ghost/agent/core"
	routercore "github.com/snow-ghost/agent/pkg/router/core"
)

// Adapter adapts the HTTP client to the core.LLMClient interface
type Adapter struct {
	client *Client
}

// NewAdapter creates a new adapter
func NewAdapter(client *Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

// Generate implements core.LLMClient.Generate
func (a *Adapter) Generate(ctx context.Context, prompt string, options core.LLMOptions) (string, error) {
	// Convert core.LLMOptions to client.ChatRequest
	req := ChatRequest{
		Model:       options.Model,
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
		Caller:      options.Caller,
		Messages: []routercore.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Add system message if provided
	if options.SystemPrompt != "" {
		req.Messages = append([]routercore.Message{
			{
				Role:    "system",
				Content: options.SystemPrompt,
			},
		}, req.Messages...)
	}

	// Add tools if provided
	if len(options.Tools) > 0 {
		req.Tools = make([]routercore.Tool, len(options.Tools))
		for i, tool := range options.Tools {
			req.Tools[i] = routercore.Tool{
				Type: tool.Type,
				Function: &routercore.ToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	// Send request
	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM client error: %w", err)
	}

	return resp.Text, nil
}

// GenerateWithTools implements core.LLMClient.GenerateWithTools
func (a *Adapter) GenerateWithTools(ctx context.Context, prompt string, tools []core.Tool, options core.LLMOptions) (string, []core.ToolCall, error) {
	// Convert tools
	routerTools := make([]routercore.Tool, len(tools))
	for i, tool := range tools {
		routerTools[i] = routercore.Tool{
			Type: tool.Type,
			Function: &routercore.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}

	// Create request
	req := ChatRequest{
		Model:       options.Model,
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
		Caller:      options.Caller,
		Tools:       routerTools,
		Messages: []routercore.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Add system message if provided
	if options.SystemPrompt != "" {
		req.Messages = append([]routercore.Message{
			{
				Role:    "system",
				Content: options.SystemPrompt,
			},
		}, req.Messages...)
	}

	// Send request
	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("LLM client error: %w", err)
	}

	// Convert tool calls back to core format
	coreToolCalls := make([]core.ToolCall, len(resp.ToolCalls))
	for i, toolCall := range resp.ToolCalls {
		coreToolCalls[i] = core.ToolCall{
			ID:   toolCall.ID,
			Name: toolCall.Function.Name,
			Args: toolCall.Function.Arguments,
		}
	}

	return resp.Text, coreToolCalls, nil
}

// Propose implements core.LLMClient.Propose
func (a *Adapter) Propose(ctx context.Context, task core.Task) (string, []core.TestCase, []string, error) {
	// Generate caller from task
	caller := fmt.Sprintf("worker/%s/%s", task.Domain, task.ID)
	return a.ProposeWithCaller(ctx, task, caller)
}

// ProposeWithCaller implements core.LLMClient.ProposeWithCaller
func (a *Adapter) ProposeWithCaller(ctx context.Context, task core.Task, caller string) (string, []core.TestCase, []string, error) {
	// Create a chat request for algorithm generation
	prompt := fmt.Sprintf("Generate a WASM algorithm for the following task:\n\nDescription: %s\nDomain: %s\nSpec: %s",
		task.Description, task.Domain, task.Spec)

	req := ChatRequest{
		Caller: caller,
		Messages: []routercore.Message{
			{
				Role:    "system",
				Content: "You are an expert algorithm developer. Generate WASM bytecode for the given task. Return only the algorithm description, not the actual bytecode.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Send request
	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return "", nil, nil, fmt.Errorf("LLM client error: %w", err)
	}

	// For now, return a simple algorithm description
	// In a real implementation, this would parse the response and generate proper tests/criteria
	algo := resp.Text
	tests := []core.TestCase{
		{
			Name:   "basic_test",
			Input:  []byte(`{"input": "test"}`),
			Oracle: []byte(`{"output": "test"}`),
			Checks: []string{"basic_check"},
			Weight: 1.0,
		},
	}
	criteria := []string{"basic_check", "handles_input"}

	return algo, tests, criteria, nil
}

// Embed implements core.LLMClient.Embed
func (a *Adapter) Embed(ctx context.Context, texts []string, options core.LLMOptions) ([][]float32, error) {
	embeddings, err := a.client.Embed(ctx, texts, options.Caller)
	if err != nil {
		return nil, fmt.Errorf("LLM client embed error: %w", err)
	}

	return embeddings, nil
}

// Health implements core.LLMClient.Health
func (a *Adapter) Health(ctx context.Context) error {
	return a.client.Health(ctx)
}

// Ensure Adapter implements core.LLMClient interface
var _ core.LLMClient = (*Adapter)(nil)
