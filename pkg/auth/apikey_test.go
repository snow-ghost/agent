package auth

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.Len(t, key, 64) // 32 bytes = 64 hex chars
}

func TestHashAPIKey(t *testing.T) {
	key := "test-key"
	hash := HashAPIKey(key)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 = 64 hex chars
	assert.NotEqual(t, key, hash)
}

func TestVerifyAPIKey(t *testing.T) {
	key := "test-key"
	hash := HashAPIKey(key)

	assert.True(t, VerifyAPIKey(key, hash))
	assert.False(t, VerifyAPIKey("wrong-key", hash))
	assert.False(t, VerifyAPIKey(key, "wrong-hash"))
}

func TestNewAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		keyName     string
		rateLimit   int
		expiry      *time.Time
		permissions []string
		expected    *APIKey
	}{
		{
			name:        "basic key",
			keyName:     "test-key",
			rateLimit:   100,
			expiry:      nil,
			permissions: []string{"read", "write"},
			expected: &APIKey{
				Name:        "test-key",
				IsActive:    true,
				RateLimit:   100,
				ExpiresAt:   nil,
				Permissions: []string{"read", "write"},
			},
		},
		{
			name:        "key with expiry",
			keyName:     "expiring-key",
			rateLimit:   50,
			expiry:      func() *time.Time { t := time.Now().Add(24 * time.Hour); return &t }(),
			permissions: []string{"read"},
			expected: &APIKey{
				Name:        "expiring-key",
				IsActive:    true,
				RateLimit:   50,
				Permissions: []string{"read"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := NewAPIKey(tt.keyName, tt.rateLimit, tt.expiry, tt.permissions)
			require.NoError(t, err)

			assert.NotEmpty(t, key.Key)
			assert.Equal(t, tt.expected.Name, key.Name)
			assert.Equal(t, tt.expected.IsActive, key.IsActive)
			assert.Equal(t, tt.expected.RateLimit, key.RateLimit)
			assert.Equal(t, tt.expected.ExpiresAt, key.ExpiresAt)
			assert.Equal(t, tt.expected.Permissions, key.Permissions)
			assert.True(t, key.CreatedAt.Before(time.Now().Add(time.Second)))
			assert.True(t, key.CreatedAt.After(time.Now().Add(-time.Second)))
		})
	}
}

func TestLoadAPIKeysFromFile(t *testing.T) {
	// Test with non-existent file
	keys, err := LoadAPIKeysFromFile("non-existent.json")
	require.NoError(t, err)
	assert.Empty(t, keys)

	// Test with valid file
	tempFile := "test-api-keys.json"
	defer os.Remove(tempFile)

	testKeys := map[string]*APIKey{
		"key1": {
			Key:       "key1",
			Name:      "test1",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
		"key2": {
			Key:       "key2",
			Name:      "test2",
			IsActive:  false,
			CreatedAt: time.Now(),
		},
	}

	err = SaveAPIKeysToFile(tempFile, testKeys)
	require.NoError(t, err)

	loadedKeys, err := LoadAPIKeysFromFile(tempFile)
	require.NoError(t, err)
	assert.Equal(t, len(testKeys), len(loadedKeys))
	assert.Equal(t, testKeys["key1"].Name, loadedKeys["key1"].Name)
	assert.Equal(t, testKeys["key2"].Name, loadedKeys["key2"].Name)
}

func TestSaveAPIKeysToFile(t *testing.T) {
	tempFile := "test-save-keys.json"
	defer os.Remove(tempFile)

	keys := map[string]*APIKey{
		"test-key": {
			Key:       "test-key",
			Name:      "test",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}

	err := SaveAPIKeysToFile(tempFile, keys)
	require.NoError(t, err)

	// Verify file exists and is readable
	_, err = os.Stat(tempFile)
	require.NoError(t, err)

	// Load and verify content
	loadedKeys, err := LoadAPIKeysFromFile(tempFile)
	require.NoError(t, err)
	assert.Equal(t, keys["test-key"].Name, loadedKeys["test-key"].Name)
}

func TestAddAPIKeyToFile(t *testing.T) {
	tempFile := "test-add-key.json"
	defer os.Remove(tempFile)

	// Start with empty file
	keys := make(map[string]*APIKey)
	err := SaveAPIKeysToFile(tempFile, keys)
	require.NoError(t, err)

	// Add a key
	newKey := &APIKey{
		Key:       "new-key",
		Name:      "new",
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	err = AddAPIKeyToFile(tempFile, newKey)
	require.NoError(t, err)

	// Verify key was added
	loadedKeys, err := LoadAPIKeysFromFile(tempFile)
	require.NoError(t, err)
	assert.Contains(t, loadedKeys, "new-key")
	assert.Equal(t, newKey.Name, loadedKeys["new-key"].Name)

	// Try to add duplicate key
	err = AddAPIKeyToFile(tempFile, newKey)
	assert.Error(t, err)
}

func TestRemoveAPIKeyFromFile(t *testing.T) {
	tempFile := "test-remove-key.json"
	defer os.Remove(tempFile)

	// Start with a key
	keys := map[string]*APIKey{
		"key-to-remove": {
			Key:       "key-to-remove",
			Name:      "remove-me",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}
	err := SaveAPIKeysToFile(tempFile, keys)
	require.NoError(t, err)

	// Remove the key
	err = RemoveAPIKeyFromFile(tempFile, "key-to-remove")
	require.NoError(t, err)

	// Verify key was removed
	loadedKeys, err := LoadAPIKeysFromFile(tempFile)
	require.NoError(t, err)
	assert.NotContains(t, loadedKeys, "key-to-remove")

	// Try to remove non-existent key
	err = RemoveAPIKeyFromFile(tempFile, "non-existent")
	assert.Error(t, err)
}

func TestUpdateAPIKeyInFile(t *testing.T) {
	tempFile := "test-update-key.json"
	defer os.Remove(tempFile)

	// Start with a key
	originalKey := &APIKey{
		Key:       "update-key",
		Name:      "original",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	keys := map[string]*APIKey{
		"update-key": originalKey,
	}
	err := SaveAPIKeysToFile(tempFile, keys)
	require.NoError(t, err)

	// Update the key
	updatedKey := &APIKey{
		Key:       "update-key",
		Name:      "updated",
		IsActive:  false,
		CreatedAt: originalKey.CreatedAt,
	}

	err = UpdateAPIKeyInFile(tempFile, updatedKey)
	require.NoError(t, err)

	// Verify key was updated
	loadedKeys, err := LoadAPIKeysFromFile(tempFile)
	require.NoError(t, err)
	assert.Equal(t, "updated", loadedKeys["update-key"].Name)
	assert.False(t, loadedKeys["update-key"].IsActive)

	// Try to update non-existent key
	nonExistentKey := &APIKey{
		Key:  "non-existent",
		Name: "test",
	}
	err = UpdateAPIKeyInFile(tempFile, nonExistentKey)
	assert.Error(t, err)
}

func TestListAPIKeysFromFile(t *testing.T) {
	tempFile := "test-list-keys.json"
	defer os.Remove(tempFile)

	// Create test keys
	keys := map[string]*APIKey{
		"key1": {
			Key:       "key1",
			Name:      "test1",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
		"key2": {
			Key:       "key2",
			Name:      "test2",
			IsActive:  false,
			CreatedAt: time.Now(),
		},
	}
	err := SaveAPIKeysToFile(tempFile, keys)
	require.NoError(t, err)

	// List keys
	listedKeys, err := ListAPIKeysFromFile(tempFile)
	require.NoError(t, err)

	assert.Len(t, listedKeys, 2)

	// Verify keys are masked
	for _, key := range listedKeys {
		assert.Equal(t, "********", key.Key)
	}

	// Verify other fields are preserved
	key1 := listedKeys[0]
	if key1.Name == "test1" {
		assert.True(t, key1.IsActive)
	} else {
		assert.False(t, key1.IsActive)
	}
}
