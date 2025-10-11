package secrets

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecretManager(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.NotNil(t, manager.secrets)
}

func TestNewSecretManager_EmptyKey(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "", // Empty key
		SecretsFile:   "test-secrets.json",
	}

	_, err := NewSecretManager(config)
	assert.Error(t, err)
}

func TestSecretManager_SetGetSecret(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set a secret
	err = manager.SetSecret("test-secret", "test-value", nil, []string{"test"})
	require.NoError(t, err)

	// Get the secret
	secret, err := manager.GetSecret("test-secret")
	require.NoError(t, err)
	assert.Equal(t, "test-secret", secret.Name)
	assert.Equal(t, "test-value", secret.Value)
	assert.Contains(t, secret.Tags, "test")
}

func TestSecretManager_GetNonExistentSecret(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	_, err = manager.GetSecret("non-existent")
	assert.Error(t, err)
}

func TestSecretManager_DeleteSecret(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set a secret
	err = manager.SetSecret("test-secret", "test-value", nil, nil)
	require.NoError(t, err)

	// Delete the secret
	err = manager.DeleteSecret("test-secret")
	require.NoError(t, err)

	// Try to get the deleted secret
	_, err = manager.GetSecret("test-secret")
	assert.Error(t, err)
}

func TestSecretManager_ListSecrets(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets-list.json", // Use different file
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set multiple secrets
	err = manager.SetSecret("secret1", "value1", nil, []string{"tag1"})
	require.NoError(t, err)

	err = manager.SetSecret("secret2", "value2", nil, []string{"tag2"})
	require.NoError(t, err)

	// List secrets
	secretNames := manager.ListSecrets()
	assert.Len(t, secretNames, 2)

	// Check that both secrets are in the list
	secretNameMap := make(map[string]bool)
	for _, name := range secretNames {
		secretNameMap[name] = true
	}
	assert.True(t, secretNameMap["secret1"])
	assert.True(t, secretNameMap["secret2"])
}

func TestSecretManager_SecretWithExpiry(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	expiry := time.Now().Add(24 * time.Hour)
	err = manager.SetSecret("expiring-secret", "value", &expiry, nil)
	require.NoError(t, err)

	secret, err := manager.GetSecret("expiring-secret")
	require.NoError(t, err)
	assert.NotNil(t, secret.ExpiresAt)
	assert.True(t, secret.ExpiresAt.After(time.Now()))
}

func TestSecretManager_UpdateSecret(t *testing.T) {
	config := &SecretConfig{
		EncryptionKey: "test-key-32-characters-long!",
		SecretsFile:   "test-secrets.json",
	}

	manager, err := NewSecretManager(config)
	require.NoError(t, err)

	// Set initial secret
	err = manager.SetSecret("test-secret", "initial-value", nil, nil)
	require.NoError(t, err)

	// Update the secret
	err = manager.SetSecret("test-secret", "updated-value", nil, []string{"updated"})
	require.NoError(t, err)

	// Get the updated secret
	secret, err := manager.GetSecret("test-secret")
	require.NoError(t, err)
	assert.Equal(t, "updated-value", secret.Value)
	assert.Contains(t, secret.Tags, "updated")
}

func TestDefaultSecretConfig(t *testing.T) {
	config := DefaultSecretConfig()

	assert.NotNil(t, config)
	assert.NotEmpty(t, config.SecretsFile)
	assert.NotNil(t, config.Environment)
	assert.Equal(t, 24*time.Hour, config.RotationTime)
}
