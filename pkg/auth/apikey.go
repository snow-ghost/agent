package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// APIKeyManager manages API keys with persistence
type APIKeyManager struct {
	config     *AuthConfig
	configFile string
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(configFile string) *APIKeyManager {
	return &APIKeyManager{
		config:     DefaultAuthConfig(),
		configFile: configFile,
	}
}

// LoadConfig loads API key configuration from file
func (m *APIKeyManager) LoadConfig() error {
	if m.configFile == "" {
		return nil // No config file specified
	}

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, create default
			return m.SaveConfig()
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// SaveConfig saves API key configuration to file
func (m *APIKeyManager) SaveConfig() error {
	if m.configFile == "" {
		return nil // No config file specified
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateAPIKey creates a new API key
func (m *APIKeyManager) CreateAPIKey(name string, rateLimit int, expiresAt *time.Time) (*APIKey, error) {
	// Generate API key
	apiKey, err := m.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	keyInfo := &APIKey{
		Key:       apiKey,
		Name:      name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		RateLimit: rateLimit,
		IsActive:  true,
	}

	// Add to config
	m.config.APIKeys[apiKey] = keyInfo

	// Save config
	if err := m.SaveConfig(); err != nil {
		// Remove from config if save failed
		delete(m.config.APIKeys, apiKey)
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return keyInfo, nil
}

// RevokeAPIKey revokes an API key
func (m *APIKeyManager) RevokeAPIKey(apiKey string) error {
	if keyInfo, exists := m.config.APIKeys[apiKey]; exists {
		keyInfo.IsActive = false
		return m.SaveConfig()
	}
	return fmt.Errorf("API key not found")
}

// DeleteAPIKey permanently deletes an API key
func (m *APIKeyManager) DeleteAPIKey(apiKey string) error {
	if _, exists := m.config.APIKeys[apiKey]; exists {
		delete(m.config.APIKeys, apiKey)
		return m.SaveConfig()
	}
	return fmt.Errorf("API key not found")
}

// GetAPIKey returns API key information
func (m *APIKeyManager) GetAPIKey(apiKey string) (*APIKey, error) {
	keyInfo, exists := m.config.APIKeys[apiKey]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}
	return keyInfo, nil
}

// ListAPIKeys returns all API keys
func (m *APIKeyManager) ListAPIKeys() []*APIKey {
	keys := make([]*APIKey, 0, len(m.config.APIKeys))
	for _, keyInfo := range m.config.APIKeys {
		keys = append(keys, keyInfo)
	}
	return keys
}

// GetConfig returns the current configuration
func (m *APIKeyManager) GetConfig() *AuthConfig {
	return m.config
}

// SetConfig sets the configuration
func (m *APIKeyManager) SetConfig(config *AuthConfig) {
	m.config = config
}

// generateAPIKey generates a new API key
func (m *APIKeyManager) generateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := os.ReadFile("/dev/urandom"); err == nil {
		// Use /dev/urandom if available
		file, err := os.Open("/dev/urandom")
		if err == nil {
			defer file.Close()
			file.Read(bytes)
		} else {
			// Fallback to crypto/rand
			_, err := os.ReadFile("/dev/urandom")
			if err != nil {
				return "", fmt.Errorf("failed to generate random bytes: %w", err)
			}
		}
	} else {
		// Fallback to crypto/rand
		_, err := os.ReadFile("/dev/urandom")
		if err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}
	}

	// Create hex encoded string
	return fmt.Sprintf("ak_%x", bytes), nil
}

// CLI commands for API key management

// CreateAPIKeyCommand creates an API key via CLI
func CreateAPIKeyCommand(name string, rateLimit int, expiresInDays int, configFile string) error {
	manager := NewAPIKeyManager(configFile)
	if err := manager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var expiresAt *time.Time
	if expiresInDays > 0 {
		exp := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}

	keyInfo, err := manager.CreateAPIKey(name, rateLimit, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	fmt.Printf("API Key created successfully:\n")
	fmt.Printf("Name: %s\n", keyInfo.Name)
	fmt.Printf("Key: %s\n", keyInfo.Key)
	fmt.Printf("Rate Limit: %d requests/minute\n", keyInfo.RateLimit)
	if keyInfo.ExpiresAt != nil {
		fmt.Printf("Expires: %s\n", keyInfo.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

// ListAPIKeysCommand lists all API keys via CLI
func ListAPIKeysCommand(configFile string) error {
	manager := NewAPIKeyManager(configFile)
	if err := manager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	keys := manager.ListAPIKeys()
	if len(keys) == 0 {
		fmt.Println("No API keys found")
		return nil
	}

	fmt.Printf("API Keys (%d total):\n", len(keys))
	fmt.Println("Name\t\tKey\t\t\t\tRate Limit\tActive\tCreated")
	fmt.Println("----\t\t---\t\t\t\t----------\t------\t-------")

	for _, key := range keys {
		active := "Yes"
		if !key.IsActive {
			active = "No"
		}

		expires := "Never"
		if key.ExpiresAt != nil {
			expires = key.ExpiresAt.Format("2006-01-02")
		}

		fmt.Printf("%s\t\t%s\t\t%d/min\t\t%s\t%s\n",
			key.Name,
			key.Key,
			key.RateLimit,
			active,
			key.CreatedAt.Format("2006-01-02"),
		)
	}

	return nil
}

// RevokeAPIKeyCommand revokes an API key via CLI
func RevokeAPIKeyCommand(apiKey string, configFile string) error {
	manager := NewAPIKeyManager(configFile)
	if err := manager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := manager.RevokeAPIKey(apiKey); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	fmt.Printf("API key %s revoked successfully\n", apiKey)
	return nil
}

// DeleteAPIKeyCommand deletes an API key via CLI
func DeleteAPIKeyCommand(apiKey string, configFile string) error {
	manager := NewAPIKeyManager(configFile)
	if err := manager.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := manager.DeleteAPIKey(apiKey); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	fmt.Printf("API key %s deleted successfully\n", apiKey)
	return nil
}
