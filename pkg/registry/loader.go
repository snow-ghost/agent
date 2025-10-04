package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading model configurations
type Loader struct {
	configPath string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// LoadRegistry loads the model registry from configuration file
func (l *Loader) LoadRegistry() (*Registry, error) {
	// Check if config path is provided via environment
	if configPath := os.Getenv("CONFIG"); configPath != "" {
		l.configPath = configPath
	}

	// Use default config if none provided
	if l.configPath == "" {
		l.configPath = "router.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		// Return empty registry if no config file
		return &Registry{Models: []ModelConfig{}}, nil
	}

	// Read the configuration file
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", l.configPath, err)
	}

	// Parse YAML
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &registry, nil
}

// LoadRegistryFromBytes loads registry from byte data
func LoadRegistryFromBytes(data []byte) (*Registry, error) {
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}
	return &registry, nil
}

// SaveRegistry saves the registry to a YAML file
func (l *Loader) SaveRegistry(registry *Registry) error {
	// Use config path or default
	configPath := l.configPath
	if configPath == "" {
		configPath = "router.yaml"
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// FindModel finds a model by ID in the registry
func (r *Registry) FindModel(modelID string) *ModelConfig {
	for _, model := range r.Models {
		if model.ID == modelID {
			return &model
		}
	}
	return nil
}

// GetDefaultRegistry returns a registry with some default models
func GetDefaultRegistry() *Registry {
	return &Registry{
		Models: []ModelConfig{
			{
				ID:        "openai:gpt-4o-mini",
				Provider:  "openai",
				BaseURL:   "https://api.openai.com/v1",
				APIKeyEnv: "OPENAI_API_KEY",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.00015,
					OutputPer1K: 0.0006,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				MaxRPM: 10000,
				MaxTPM: 200000,
				Tags:   []string{"general", "fast"},
			},
			{
				ID:        "openai:gpt-4o",
				Provider:  "openai",
				BaseURL:   "https://api.openai.com/v1",
				APIKeyEnv: "OPENAI_API_KEY",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.005,
					OutputPer1K: 0.015,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				MaxRPM: 5000,
				MaxTPM: 100000,
				Tags:   []string{"general", "advanced"},
			},
			{
				ID:        "openai:text-embedding-3-small",
				Provider:  "openai",
				BaseURL:   "https://api.openai.com/v1",
				APIKeyEnv: "OPENAI_API_KEY",
				Kind:      "embed",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.00002,
					OutputPer1K: 0.0,
				},
				DefaultParams: map[string]interface{}{
					"dimensions": 1536,
				},
				MaxRPM: 10000,
				MaxTPM: 1000000,
				Tags:   []string{"embed", "fast"},
			},
			{
				ID:        "anthropic:claude-3-5-sonnet-20241022",
				Provider:  "anthropic",
				BaseURL:   "https://api.anthropic.com",
				APIKeyEnv: "ANTHROPIC_API_KEY",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.003,
					OutputPer1K: 0.015,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				MaxRPM: 5000,
				MaxTPM: 100000,
				Tags:   []string{"general", "advanced"},
			},
			{
				ID:        "ollama:llama3.2",
				Provider:  "ollama",
				BaseURL:   "http://localhost:11434",
				APIKeyEnv: "",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.0,
					OutputPer1K: 0.0,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				MaxRPM: 1000,
				MaxTPM: 10000,
				Tags:   []string{"local", "general"},
			},
			{
				ID:        "openai:gpt-4o-mini",
				Provider:  "openai",
				BaseURL:   "https://api.openai.com/v1",
				APIKeyEnv: "OPENAI_API_KEY",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.00015,
					OutputPer1K: 0.0006,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.3,
					"max_tokens":  4096,
				},
				MaxRPM: 10000,
				MaxTPM: 200000,
				Tags:   []string{"code", "programming", "fast"},
			},
			{
				ID:        "anthropic:claude-3-5-sonnet-20241022",
				Provider:  "anthropic",
				BaseURL:   "https://api.anthropic.com",
				APIKeyEnv: "ANTHROPIC_API_KEY",
				Kind:      "chat",
				Pricing: Pricing{
					Currency:    "USD",
					InputPer1K:  0.003,
					OutputPer1K: 0.015,
				},
				DefaultParams: map[string]interface{}{
					"temperature": 0.3,
					"max_tokens":  4096,
				},
				MaxRPM: 5000,
				MaxTPM: 100000,
				Tags:   []string{"code", "programming", "advanced"},
			},
		},
	}
}
