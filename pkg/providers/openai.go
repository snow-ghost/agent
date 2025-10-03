package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs
type OpenAIProvider struct {
	*BaseProvider
	client *openai.Client
	apiKey string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	client := openai.NewClientWithConfig(config)

	// Create a default registry for cost calculation
	registry := registry.GetDefaultRegistry()

	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}
}

// Chat performs chat completion using OpenAI API
func (p *OpenAIProvider) Chat(ctx context.Context, mc registry.ModelConfig, req core.ChatRequest) (core.ChatResponse, error) {
	// Create a new client with the model config's base URL
	config := openai.DefaultConfig(p.apiKey)
	config.BaseURL = mc.BaseURL
	client := openai.NewClientWithConfig(config)
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
	response, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("openai chat completion failed: %w", err)
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

// Embed generates embeddings using OpenAI API
func (p *OpenAIProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, core.Usage, error) {
	// Create a new client with the model config's base URL
	config := openai.DefaultConfig(p.apiKey)
	config.BaseURL = mc.BaseURL
	client := openai.NewClientWithConfig(config)
	request := openai.EmbeddingRequest{
		Input: input,
		Model: openai.EmbeddingModel(mc.ID),
	}

	response, err := client.CreateEmbeddings(ctx, request)
	if err != nil {
		return nil, core.Usage{}, fmt.Errorf("openai embeddings failed: %w", err)
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

// CreateOpenAIProviderFromConfig creates an OpenAI provider from model config
func CreateOpenAIProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) (*OpenAIProvider, error) {
	apiKey := os.Getenv(mc.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variable %s", mc.APIKeyEnv)
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = mc.BaseURL

	client := openai.NewClientWithConfig(config)

	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}, nil
}
