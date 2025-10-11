package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// LMStudioProvider implements the Provider interface for LM Studio (OpenAI-compatible)
type LMStudioProvider struct {
	*BaseProvider
	client *openai.Client
	apiKey string
}

// NewLMStudioProvider creates a new LM Studio provider
func NewLMStudioProvider(baseURL, apiKey string) *LMStudioProvider {
	config := openai.DefaultConfig(apiKey)
	// Ensure baseURL ends with /v1 for LM Studio compatibility
	if baseURL[len(baseURL)-1] == '/' {
		config.BaseURL = baseURL + "v1"
	} else {
		config.BaseURL = baseURL + "/v1"
	}

	client := openai.NewClientWithConfig(config)

	// Create a default registry for cost calculation
	registry := registry.GetDefaultRegistry()

	return &LMStudioProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}
}

// Chat performs chat completion using LM Studio (OpenAI-compatible API)
func (p *LMStudioProvider) Chat(ctx context.Context, mc registry.ModelConfig, req interface{}) (interface{}, error) {
	// Type assert req to core.ChatRequest
	chatReq, ok := req.(core.ChatRequest)
	if !ok {
		return core.ChatResponse{}, fmt.Errorf("invalid request type, expected core.ChatRequest")
	}
	// Convert messages
	messages := make([]openai.ChatCompletionMessage, len(chatReq.Messages))
	for i, msg := range chatReq.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
	}

	// Convert tools if provided
	var tools []openai.Tool
	for _, tool := range chatReq.Tools {
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
		Temperature: chatReq.Temperature,
		TopP:        chatReq.TopP,
		MaxTokens:   chatReq.MaxTokens,
		Stream:      chatReq.Stream,
	}

	if len(tools) > 0 {
		request.Tools = tools
	}

	// Make API call
	response, err := p.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("lmstudio chat completion failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return core.ChatResponse{}, fmt.Errorf("lmstudio chat completion returned no choices")
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

// Embed generates embeddings using LM Studio (OpenAI-compatible API)
func (p *LMStudioProvider) Embed(ctx context.Context, mc registry.ModelConfig, input []string) ([][]float32, interface{}, error) {
	request := openai.EmbeddingRequest{
		Input: input,
		Model: openai.EmbeddingModel(mc.ID),
	}

	response, err := p.client.CreateEmbeddings(ctx, request)
	if err != nil {
		return nil, core.Usage{}, fmt.Errorf("lmstudio embeddings failed: %w", err)
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

// CreateLMStudioProviderFromConfig creates an LM Studio provider from model config
func CreateLMStudioProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) (*LMStudioProvider, error) {
	// LM Studio typically doesn't require API keys
	apiKey := os.Getenv(mc.APIKeyEnv)
	if apiKey == "" {
		apiKey = "dummy-key" // LM Studio often works without authentication
	}

	config := openai.DefaultConfig(apiKey)
	// Ensure baseURL ends with /v1 for LM Studio compatibility
	baseURL := mc.BaseURL
	if baseURL[len(baseURL)-1] == '/' {
		config.BaseURL = baseURL + "v1"
	} else {
		config.BaseURL = baseURL + "/v1"
	}

	client := openai.NewClientWithConfig(config)

	return &LMStudioProvider{
		BaseProvider: NewBaseProvider(registry),
		client:       client,
		apiKey:       apiKey,
	}, nil
}
