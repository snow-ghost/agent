package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// OpenRouterProvider implements the Provider interface for OpenRouter (OpenAI-compatible)
type OpenRouterProvider struct {
	*BaseProvider
	client *openai.Client
	apiKey string
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(baseURL, apiKey string) *OpenRouterProvider {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	client := openai.NewClientWithConfig(config)

	// Create a default registry for cost calculation
	registry := registry.GetDefaultRegistry()

	return &OpenRouterProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}
}

// Chat performs chat completion using OpenRouter (OpenAI-compatible API)
func (p *OpenRouterProvider) Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error) {
	// Convert messages
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
	}

	// Convert tools if provided
	var tools []openai.Tool
	for _, tool := range req.Tools {
		openaiTool := openai.Tool{
			Type: openai.ToolType(tool.Type),
		}
		if tool.Function != nil {
			openaiTool.Function = &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			}
		}
		tools = append(tools, openaiTool)
	}

	// Build request
	request := openai.ChatCompletionRequest{
		Model:       mc.ID,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}

	if len(tools) > 0 {
		request.Tools = tools
	}

	// Make API call
	response, err := p.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("openrouter chat completion failed: %w", err)
	}

	// Convert response
	chatResp := core.ChatResponse{
		Text: response.Choices[0].Message.Content,
		Usage: core.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
		Model:        mc.ID,
		Provider:     mc.Provider,
		FinishReason: string(response.Choices[0].FinishReason),
	}

	// Convert tool calls if present
	if len(response.Choices[0].Message.ToolCalls) > 0 {
		for _, tc := range response.Choices[0].Message.ToolCalls {
			toolCall := core.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
			}
			if tc.Function.Name != "" {
				toolCall.Function = &core.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
			chatResp.ToolCalls = append(chatResp.ToolCalls, toolCall)
		}
	}

	return chatResp, nil
}

// Embed generates embeddings using OpenRouter (OpenAI-compatible API)
func (p *OpenRouterProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error) {
	request := openai.EmbeddingRequest{
		Input: input,
		Model: openai.EmbeddingModel(mc.ID),
	}

	response, err := p.client.CreateEmbeddings(ctx, request)
	if err != nil {
		return nil, core.Usage{}, fmt.Errorf("openrouter embeddings failed: %w", err)
	}

	// Convert embeddings
	embeddings := make([][]float32, len(response.Data))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	usage := core.Usage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: 0, // Embeddings don't have completion tokens
		TotalTokens:      response.Usage.TotalTokens,
	}

	return embeddings, usage, nil
}

// CreateOpenRouterProviderFromConfig creates an OpenRouter provider from model config
func CreateOpenRouterProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) (*OpenRouterProvider, error) {
	apiKey := os.Getenv(mc.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variable %s", mc.APIKeyEnv)
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = mc.BaseURL

	client := openai.NewClientWithConfig(config)

	return &OpenRouterProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}, nil
}
