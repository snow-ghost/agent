package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecretManager(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.NotNil(t, manager.secrets)
}

func TestNewSecretManager_InvalidKey(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "short", // Too short
		SecretsFile:   "test-secrets.json",
	}

	_, err := NewSecretManager(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryption key must be 32 bytes")
}

func TestSecretManager_EncryptDecrypt(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	plaintext := "sensitive-data"

	// Encrypt
	ciphertext, err := manager.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt
	decrypted, err := manager.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestSecretManager_EncryptDecrypt_EmptyString(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	plaintext := ""

	// Encrypt empty string
	ciphertext, err := manager.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	// Decrypt
	decrypted, err := manager.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestSecretManager_EncryptDecrypt_LongString(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Create a long string
	longString := ""
	for i := 0; i < 1000; i++ {
		longString += "This is a test string. "
	}

	// Encrypt
	ciphertext, err := manager.Encrypt(longString)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	// Decrypt
	decrypted, err := manager.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, longString, decrypted)
}

func TestSecretManager_EncryptDecrypt_SpecialCharacters(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	specialText := "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"

	// Encrypt
	ciphertext, err := manager.Encrypt(specialText)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	// Decrypt
	decrypted, err := manager.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, specialText, decrypted)
}

func TestSecretManager_Decrypt_InvalidCiphertext(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Try to decrypt invalid ciphertext
	_, err = manager.Decrypt("invalid-ciphertext")
	assert.Error(t, err)
}

func TestSecretManager_Decrypt_TooShort(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Try to decrypt too short ciphertext
	_, err = manager.Decrypt("short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestSecretManager_SetGetSecret(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set a secret
	key := "test-secret"
	value := "sensitive-value"

	err = manager.SetSecret(key, value)
	require.NoError(t, err)

	// Get the secret
	retrieved, err := manager.GetSecret(key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestSecretManager_GetSecret_NotFound(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Try to get non-existent secret
	_, err = manager.GetSecret("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestSecretManager_SetSecret_EmptyKey(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Try to set secret with empty key
	err = manager.SetSecret("", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")
}

func TestSecretManager_LoadSecrets_NonExistentFile(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "non-existent.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Load from non-existent file should not error
	err = manager.LoadSecrets()
	require.NoError(t, err)
}

func TestSecretManager_SaveSecrets(t *testing.T) {
	config := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-save-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set some secrets
	err = manager.SetSecret("key1", "value1")
	require.NoError(t, err)
	err = manager.SetSecret("key2", "value2")
	require.NoError(t, err)

	// Save secrets
	err = manager.SaveSecrets()
	require.NoError(t, err)

	// Create new manager and load secrets
	newManager, err := NewSecretManager(config)
	require.NoError(t, err)

	err = newManager.LoadSecrets()
	require.NoError(t, err)

	// Verify secrets were loaded
	value1, err := newManager.GetSecret("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", value1)

	value2, err := newManager.GetSecret("key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", value2)
}

func TestSecretManager_EncryptDecrypt_DifferentKeys(t *testing.T) {
	config1 := &SecretManagerConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets1.json",
	}

	config2 := &SecretManagerConfig{
		EncryptionKey: "different-key-32-characters-long",
		SecretsFile:   "test-secrets2.json",
	}

	manager1, err := NewSecretManager(config1)
	require.NoError(t, err)

	manager2, err := NewSecretManager(config2)
	require.NoError(t, err)

	plaintext := "sensitive-data"

	// Encrypt with manager1
	ciphertext, err := manager1.Encrypt(plaintext)
	require.NoError(t, err)

	// Try to decrypt with manager2 (different key)
	_, err = manager2.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestDefaultSecretManagerConfig(t *testing.T) {
	config := DefaultSecretManagerConfig()

	assert.Equal(t, "./secrets.json", config.SecretsFile)
	assert.NotEmpty(t, config.EncryptionKey)
	assert.Len(t, config.EncryptionKey, 32)
}
