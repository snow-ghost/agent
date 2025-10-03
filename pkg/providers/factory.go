package providers

import (
	"fmt"

	"github.com/snow-ghost/agent/pkg/registry"
)

// DefaultProviderFactory implements ProviderFactory
type DefaultProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *DefaultProviderFactory {
	return &DefaultProviderFactory{}
}

// CreateProvider creates a provider instance based on the provider type
func (f *DefaultProviderFactory) CreateProvider(providerType string) (Provider, error) {
	switch providerType {
	case "openai":
		return &OpenAIProvider{}, nil
	case "anthropic":
		return &AnthropicProvider{}, nil
	case "ollama":
		return &OllamaProvider{}, nil
	case "vllm":
		return &VLLMProvider{}, nil
	case "lmstudio":
		return &LMStudioProvider{}, nil
	case "openrouter":
		return &OpenRouterProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateProviderFromConfig creates a provider instance from model configuration
func (f *DefaultProviderFactory) CreateProviderFromConfig(mc registry.ModelConfig, registry *registry.Registry) (Provider, error) {
	switch mc.Provider {
	case "openai":
		return CreateOpenAIProviderFromConfig(mc, registry)
	case "anthropic":
		return CreateAnthropicProviderFromConfig(mc, registry)
	case "ollama":
		return CreateOllamaProviderFromConfig(mc, registry), nil
	case "vllm":
		return CreateVLLMProviderFromConfig(mc, registry)
	case "lmstudio":
		return CreateLMStudioProviderFromConfig(mc, registry)
	case "openrouter":
		return CreateOpenRouterProviderFromConfig(mc, registry)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", mc.Provider)
	}
}

// GetSupportedProviders returns a list of supported provider types
func (f *DefaultProviderFactory) GetSupportedProviders() []string {
	return []string{
		"openai",
		"anthropic",
		"ollama",
		"vllm",
		"lmstudio",
		"openrouter",
	}
}
