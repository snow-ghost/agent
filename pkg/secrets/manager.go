package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Secret represents a secret with metadata
type Secret struct {
	Name      string     `json:"name"`
	Value     string     `json:"value"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
}

// SecretConfig holds secret management configuration
type SecretConfig struct {
	EncryptionKey string            `json:"encryption_key"`
	SecretsFile   string            `json:"secrets_file"`
	Environment   map[string]string `json:"environment"`
	RotationTime  time.Duration     `json:"rotation_time"`
}

// DefaultSecretConfig returns default secret configuration
func DefaultSecretConfig() *SecretConfig {
	return &SecretConfig{
		EncryptionKey: getEnv("SECRET_ENCRYPTION_KEY", ""),
		SecretsFile:   getEnv("SECRETS_FILE", "./secrets.json"),
		Environment:   make(map[string]string),
		RotationTime:  24 * time.Hour,
	}
}

// SecretManager manages secrets with encryption and rotation
type SecretManager struct {
	config    *SecretConfig
	secrets   map[string]*Secret
	mu        sync.RWMutex
	encryptor *Encryptor
}

// Encryptor handles encryption/decryption of secrets
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	// Ensure key is 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		// Pad with zeros
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		// Truncate
		keyBytes = keyBytes[:32]
	}

	return &Encryptor{key: keyBytes}, nil
}

// Encrypt encrypts a string
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a string
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// NewSecretManager creates a new secret manager
func NewSecretManager(config *SecretConfig) (*SecretManager, error) {
	if config == nil {
		config = DefaultSecretConfig()
	}

	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	manager := &SecretManager{
		config:    config,
		secrets:   make(map[string]*Secret),
		encryptor: encryptor,
	}

	// Load secrets from file
	if err := manager.loadSecrets(); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	// Load secrets from environment
	manager.loadFromEnvironment()

	return manager, nil
}

// GetSecret retrieves a secret by name
func (m *SecretManager) GetSecret(name string) (*Secret, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	secret, exists := m.secrets[name]
	if !exists {
		return nil, fmt.Errorf("secret %s not found", name)
	}

	// Check if secret has expired
	if secret.ExpiresAt != nil && time.Now().After(*secret.ExpiresAt) {
		return nil, fmt.Errorf("secret %s has expired", name)
	}

	// Decrypt the value
	decryptedValue, err := m.encryptor.Decrypt(secret.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret %s: %w", name, err)
	}

	// Return a copy with decrypted value
	return &Secret{
		Name:      secret.Name,
		Value:     decryptedValue,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
		ExpiresAt: secret.ExpiresAt,
		Tags:      secret.Tags,
	}, nil
}

// SetSecret stores a secret
func (m *SecretManager) SetSecret(name, value string, expiresAt *time.Time, tags []string) error {
	// Encrypt the value
	encryptedValue, err := m.encryptor.Encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret %s: %w", name, err)
	}

	secret := &Secret{
		Name:      name,
		Value:     encryptedValue,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Tags:      tags,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.secrets[name] = secret

	// Save to file
	return m.saveSecrets()
}

// DeleteSecret removes a secret
func (m *SecretManager) DeleteSecret(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.secrets[name]; !exists {
		return fmt.Errorf("secret %s not found", name)
	}

	delete(m.secrets, name)
	return m.saveSecrets()
}

// ListSecrets returns all secret names
func (m *SecretManager) ListSecrets() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.secrets))
	for name := range m.secrets {
		names = append(names, name)
	}
	return names
}

// RotateSecret rotates a secret value
func (m *SecretManager) RotateSecret(name string, newValue string) error {
	secret, err := m.GetSecret(name)
	if err != nil {
		return err
	}

	return m.SetSecret(name, newValue, secret.ExpiresAt, secret.Tags)
}

// loadSecrets loads secrets from file
func (m *SecretManager) loadSecrets() error {
	if m.config.SecretsFile == "" {
		return nil
	}

	data, err := os.ReadFile(m.config.SecretsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, start with empty secrets
		}
		return err
	}

	var secrets map[string]*Secret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return err
	}

	m.secrets = secrets
	return nil
}

// saveSecrets saves secrets to file
func (m *SecretManager) saveSecrets() error {
	if m.config.SecretsFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(m.config.SecretsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.secrets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.config.SecretsFile, data, 0600)
}

// loadFromEnvironment loads secrets from environment variables
func (m *SecretManager) loadFromEnvironment() {
	for key, value := range m.config.Environment {
		if value != "" {
			// Check if secret already exists
			if _, exists := m.secrets[key]; !exists {
				// Encrypt and store
				if encryptedValue, err := m.encryptor.Encrypt(value); err == nil {
					m.secrets[key] = &Secret{
						Name:      key,
						Value:     encryptedValue,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Tags:      []string{"environment"},
					}
				}
			}
		}
	}
}

// GetSecretValue is a convenience method that returns just the value
func (m *SecretManager) GetSecretValue(name string) (string, error) {
	secret, err := m.GetSecret(name)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// GetSecretWithDefault returns a secret value or a default if not found
func (m *SecretManager) GetSecretWithDefault(name, defaultValue string) string {
	value, err := m.GetSecretValue(name)
	if err != nil {
		return defaultValue
	}
	return value
}

// CleanupExpiredSecrets removes expired secrets
func (m *SecretManager) CleanupExpiredSecrets() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for name, secret := range m.secrets {
		if secret.ExpiresAt != nil && now.After(*secret.ExpiresAt) {
			expired = append(expired, name)
		}
	}

	for _, name := range expired {
		delete(m.secrets, name)
	}

	if len(expired) > 0 {
		return m.saveSecrets()
	}

	return nil
}

// StartRotation starts automatic secret rotation
func (m *SecretManager) StartRotation(ctx context.Context) {
	ticker := time.NewTicker(m.config.RotationTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up expired secrets
			if err := m.CleanupExpiredSecrets(); err != nil {
				// Log error but continue
				fmt.Printf("Failed to cleanup expired secrets: %v\n", err)
			}
		}
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Global secret manager instance
var globalSecretManager *SecretManager

// InitGlobalSecretManager initializes the global secret manager
func InitGlobalSecretManager(config *SecretConfig) error {
	manager, err := NewSecretManager(config)
	if err != nil {
		return err
	}
	globalSecretManager = manager
	return nil
}

// GetGlobalSecretManager returns the global secret manager
func GetGlobalSecretManager() *SecretManager {
	return globalSecretManager
}

// GetSecret is a convenience function that uses the global secret manager
func GetSecret(name string) (*Secret, error) {
	if globalSecretManager == nil {
		return nil, fmt.Errorf("secret manager not initialized")
	}
	return globalSecretManager.GetSecret(name)
}

// GetSecretValue is a convenience function that uses the global secret manager
func GetSecretValue(name string) (string, error) {
	if globalSecretManager == nil {
		return "", fmt.Errorf("secret manager not initialized")
	}
	return globalSecretManager.GetSecretValue(name)
}

// GetSecretWithDefault is a convenience function that uses the global secret manager
func GetSecretWithDefault(name, defaultValue string) string {
	if globalSecretManager == nil {
		return defaultValue
	}
	return globalSecretManager.GetSecretWithDefault(name, defaultValue)
}
