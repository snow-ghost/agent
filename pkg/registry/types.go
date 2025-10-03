package registry

// Pricing represents pricing information for a model
type Pricing struct {
	Currency    string  `json:"currency" yaml:"currency"`
	InputPer1K  float64 `json:"input_per_1k" yaml:"input_per_1k"`
	OutputPer1K float64 `json:"output_per_1k" yaml:"output_per_1k"`
}

// ModelConfig represents configuration for a model
type ModelConfig struct {
	ID            string                 `json:"id" yaml:"id"`             // "openai:gpt-4o-mini"
	Provider      string                 `json:"provider" yaml:"provider"` // openai|anthropic|ollama|vllm|lmstudio|openrouter
	BaseURL       string                 `json:"base_url" yaml:"base_url"`
	APIKeyEnv     string                 `json:"api_key_env" yaml:"api_key_env"`
	Kind          string                 `json:"kind" yaml:"kind"` // chat|complete|embed
	Pricing       Pricing                `json:"pricing" yaml:"pricing"`
	DefaultParams map[string]interface{} `json:"default_params" yaml:"default_params"`
	MaxRPM        int                    `json:"max_rpm,omitempty" yaml:"max_rpm,omitempty"` // requests per minute
	MaxTPM        int                    `json:"max_tpm,omitempty" yaml:"max_tpm,omitempty"` // tokens per minute
	Tags          []string               `json:"tags,omitempty" yaml:"tags,omitempty"`       // routing hints: general, code, embed
}

// Registry represents the model registry
type Registry struct {
	Models []ModelConfig `json:"models" yaml:"models"`
}

// GetModelByID returns a model configuration by ID
func (r *Registry) GetModelByID(id string) *ModelConfig {
	for _, model := range r.Models {
		if model.ID == id {
			return &model
		}
	}
	return nil
}

// GetModelsByProvider returns all models for a specific provider
func (r *Registry) GetModelsByProvider(provider string) []ModelConfig {
	var models []ModelConfig
	for _, model := range r.Models {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// GetModelsByKind returns all models of a specific kind
func (r *Registry) GetModelsByKind(kind string) []ModelConfig {
	var models []ModelConfig
	for _, model := range r.Models {
		if model.Kind == kind {
			models = append(models, model)
		}
	}
	return models
}

// GetModelsByTag returns all models with a specific tag
func (r *Registry) GetModelsByTag(tag string) []ModelConfig {
	var models []ModelConfig
	for _, model := range r.Models {
		for _, modelTag := range model.Tags {
			if modelTag == tag {
				models = append(models, model)
				break
			}
		}
	}
	return models
}

// GetAllProviders returns a list of all unique providers
func (r *Registry) GetAllProviders() []string {
	providers := make(map[string]bool)
	for _, model := range r.Models {
		providers[model.Provider] = true
	}

	var result []string
	for provider := range providers {
		result = append(result, provider)
	}
	return result
}

// GetTotalModels returns the total number of models
func (r *Registry) GetTotalModels() int {
	return len(r.Models)
}
