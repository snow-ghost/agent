package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/snow-ghost/agent/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthMiddleware(t *testing.T) {
	errorHandler := &errors.ErrorHandler{}

	tests := []struct {
		name     string
		config   *AuthConfig
		expected *AuthMiddleware
	}{
		{
			name:   "nil config uses default",
			config: nil,
			expected: &AuthMiddleware{
				config:       DefaultAuthConfig(),
				rateLimiters: make(map[string]*RateLimiter),
				errorHandler: errorHandler,
			},
		},
		{
			name:   "provided config is used",
			config: &AuthConfig{RequireAuth: true, DefaultRateLimit: 100},
			expected: &AuthMiddleware{
				config:       &AuthConfig{RequireAuth: true, DefaultRateLimit: 100},
				rateLimiters: make(map[string]*RateLimiter),
				errorHandler: errorHandler,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewAuthMiddleware(tt.config, errorHandler)
			assert.Equal(t, tt.expected.config.RequireAuth, middleware.config.RequireAuth)
			assert.Equal(t, tt.expected.config.DefaultRateLimit, middleware.config.DefaultRateLimit)
			assert.NotNil(t, middleware.rateLimiters)
			assert.Equal(t, errorHandler, middleware.errorHandler)
		})
	}
}

func TestAuthMiddleware_Middleware(t *testing.T) {
	errorHandler := &errors.ErrorHandler{}

	tests := []struct {
		name           string
		requireAuth    bool
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "no auth required",
			requireAuth:    false,
			apiKey:         "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "auth required, no API key",
			requireAuth:    true,
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "auth required, invalid API key",
			requireAuth:    true,
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AuthConfig{
				RequireAuth:      tt.requireAuth,
				DefaultRateLimit: 100,
				APIKeys:          make(map[string]*APIKey),
			}

			// Add a valid API key for testing
			if tt.requireAuth {
				validKey := &APIKey{
					Key:      "valid-key",
					Name:     "test-key",
					IsActive: true,
				}
				config.APIKeys["valid-key"] = validKey
			}

			middleware := NewAuthMiddleware(config, errorHandler)

			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Apply middleware
			middleware.Middleware(handler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		window   time.Duration
		requests int
		expected bool
	}{
		{
			name:     "within limit",
			limit:    10,
			window:   time.Minute,
			requests: 5,
			expected: true,
		},
		{
			name:     "at limit",
			limit:    5,
			window:   time.Minute,
			requests: 5,
			expected: false,
		},
		{
			name:     "over limit",
			limit:    3,
			window:   time.Minute,
			requests: 5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &RateLimiter{
				limit:    tt.limit,
				window:   tt.window,
				requests: make([]time.Time, 0),
			}

			// Make requests
			for i := 0; i < tt.requests; i++ {
				limiter.Allow()
			}

			// Check if the last request is allowed
			result := limiter.Allow()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthMiddleware_AddAPIKey(t *testing.T) {
	errorHandler := &errors.ErrorHandler{}
	config := &AuthConfig{
		RequireAuth:      true,
		DefaultRateLimit: 100,
		APIKeys:          make(map[string]*APIKey),
	}

	middleware := NewAuthMiddleware(config, errorHandler)

	key := &APIKey{
		Key:      "test-key",
		Name:     "test",
		IsActive: true,
	}

	err := middleware.AddAPIKey(key)
	assert.NoError(t, err)
	assert.Contains(t, middleware.config.APIKeys, "test-key")
}

func TestAuthMiddleware_RemoveAPIKey(t *testing.T) {
	errorHandler := &errors.ErrorHandler{}
	config := &AuthConfig{
		RequireAuth:      true,
		DefaultRateLimit: 100,
		APIKeys:          make(map[string]*APIKey),
	}

	middleware := NewAuthMiddleware(config, errorHandler)

	// Add a key first
	key := &APIKey{
		Key:      "test-key",
		Name:     "test",
		IsActive: true,
	}
	middleware.config.APIKeys["test-key"] = key

	// Remove the key
	err := middleware.RemoveAPIKey("test-key")
	assert.NoError(t, err)
	assert.NotContains(t, middleware.config.APIKeys, "test-key")

	// Try to remove non-existent key
	err = middleware.RemoveAPIKey("non-existent")
	assert.Error(t, err)
}

func TestAuthMiddleware_GetAPIKey(t *testing.T) {
	errorHandler := &errors.ErrorHandler{}
	config := &AuthConfig{
		RequireAuth:      true,
		DefaultRateLimit: 100,
		APIKeys:          make(map[string]*APIKey),
	}

	middleware := NewAuthMiddleware(config, errorHandler)

	// Add a key
	key := &APIKey{
		Key:      "test-key",
		Name:     "test",
		IsActive: true,
	}
	middleware.config.APIKeys["test-key"] = key

	// Get existing key
	retrievedKey, err := middleware.GetAPIKey("test-key")
	assert.NoError(t, err)
	assert.Equal(t, key, retrievedKey)

	// Get non-existent key
	_, err = middleware.GetAPIKey("non-existent")
	assert.Error(t, err)
}

func TestDefaultAuthConfig(t *testing.T) {
	config := DefaultAuthConfig()

	assert.False(t, config.RequireAuth)
	assert.Equal(t, "./api-keys.json", config.APIKeyFile)
	assert.Equal(t, "default-secret-change-in-production", config.JWTSecret)
	assert.Equal(t, 24*time.Hour, config.JWTExpiry)
	assert.Equal(t, 1000, config.DefaultRateLimit)
}
