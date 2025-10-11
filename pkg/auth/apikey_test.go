package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyManager_CreateAPIKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	key, err := manager.CreateAPIKey("test-key", 100, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, key.Key)
	assert.Equal(t, "test-key", key.Name)
	assert.Equal(t, 100, key.RateLimit)
	assert.True(t, key.IsActive)
}

func TestAPIKeyManager_GetAPIKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	key, err := manager.CreateAPIKey("test-key", 100, nil)
	require.NoError(t, err)

	retrieved, err := manager.GetAPIKey(key.Key)
	require.NoError(t, err)
	assert.Equal(t, key.Key, retrieved.Key)
	assert.Equal(t, key.Name, retrieved.Name)
}

func TestAPIKeyManager_RevokeAPIKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	key, err := manager.CreateAPIKey("test-key", 100, nil)
	require.NoError(t, err)

	err = manager.RevokeAPIKey(key.Key)
	require.NoError(t, err)

	retrieved, err := manager.GetAPIKey(key.Key)
	require.NoError(t, err)
	assert.False(t, retrieved.IsActive)
}

func TestAPIKeyManager_DeleteAPIKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	key, err := manager.CreateAPIKey("test-key", 100, nil)
	require.NoError(t, err)

	err = manager.DeleteAPIKey(key.Key)
	require.NoError(t, err)

	_, err = manager.GetAPIKey(key.Key)
	assert.Error(t, err)
}

func TestAPIKeyManager_ListAPIKeys(t *testing.T) {
	manager := NewAPIKeyManager("")

	// Create multiple keys
	_, err := manager.CreateAPIKey("key1", 100, nil)
	require.NoError(t, err)

	_, err = manager.CreateAPIKey("key2", 200, nil)
	require.NoError(t, err)

	keys := manager.ListAPIKeys()
	assert.Len(t, keys, 2)

	// Check that both keys are in the list
	keyNames := make(map[string]bool)
	for _, key := range keys {
		keyNames[key.Name] = true
	}
	assert.True(t, keyNames["key1"])
	assert.True(t, keyNames["key2"])
}

func TestAPIKeyManager_KeyWithExpiry(t *testing.T) {
	manager := NewAPIKeyManager("")

	expiry := time.Now().Add(24 * time.Hour)
	key, err := manager.CreateAPIKey("expiring-key", 100, &expiry)
	require.NoError(t, err)

	assert.NotNil(t, key.ExpiresAt)
	assert.True(t, key.ExpiresAt.After(time.Now()))
}

func TestAPIKeyManager_GetNonExistentKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	_, err := manager.GetAPIKey("non-existent-key")
	assert.Error(t, err)
}

func TestAPIKeyManager_RevokeNonExistentKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	err := manager.RevokeAPIKey("non-existent-key")
	assert.Error(t, err)
}

func TestAPIKeyManager_DeleteNonExistentKey(t *testing.T) {
	manager := NewAPIKeyManager("")

	err := manager.DeleteAPIKey("non-existent-key")
	assert.Error(t, err)
}
