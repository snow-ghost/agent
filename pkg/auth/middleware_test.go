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
			name: "custom config",
			config: &AuthConfig{
				RequireAuth:      true,
				APIKeys:          make(map[string]*APIKey),
				DefaultRateLimit: 100,
				JWTSecret:        "test-secret",
				JWTExpiry:        time.Hour,
			},
			expected: &AuthMiddleware{
				config: &AuthConfig{
					RequireAuth:      true,
					APIKeys:          make(map[string]*APIKey),
					DefaultRateLimit: 100,
					JWTSecret:        "test-secret",
					JWTExpiry:        time.Hour,
				},
				rateLimiters: make(map[string]*RateLimiter),
				errorHandler: errorHandler,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewAuthMiddleware(tt.config, errorHandler)
			assert.Equal(t, tt.expected.config, middleware.config)
			assert.NotNil(t, middleware.rateLimiters)
			assert.Equal(t, tt.expected.errorHandler, middleware.errorHandler)
		})
	}
}

func TestAuthMiddleware_Middleware_NoAuthRequired(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      false,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with no auth header when auth is not required
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Middleware_WithValidKey(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      true,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	// Add a valid API key
	apiKey := &APIKey{
		Key:       "test-key",
		Name:      "test",
		CreatedAt: time.Now(),
		RateLimit: 100,
		IsActive:  true,
	}
	config.APIKeys["test-key"] = apiKey

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with valid auth header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Middleware_WithInvalidKey(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      true,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with invalid auth header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Middleware_WithInactiveKey(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      true,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	// Add an inactive API key
	apiKey := &APIKey{
		Key:       "test-key",
		Name:      "test",
		CreatedAt: time.Now(),
		RateLimit: 100,
		IsActive:  false,
	}
	config.APIKeys["test-key"] = apiKey

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with inactive auth header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Middleware_WithExpiredKey(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      true,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	// Add an expired API key
	expiredTime := time.Now().Add(-24 * time.Hour)
	apiKey := &APIKey{
		Key:       "test-key",
		Name:      "test",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: &expiredTime,
		RateLimit: 100,
		IsActive:  true,
	}
	config.APIKeys["test-key"] = apiKey

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with expired auth header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Middleware_NoAuthHeader(t *testing.T) {
	config := &AuthConfig{
		RequireAuth:      true,
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 100,
		JWTSecret:        "test-secret",
		JWTExpiry:        time.Hour,
	}

	errorHandler := &errors.ErrorHandler{}
	middleware := NewAuthMiddleware(config, errorHandler)

	// Test with no auth header
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := &RateLimiter{
		limit:    2,
		window:   time.Minute,
		requests: []time.Time{},
	}

	// First request should be allowed
	assert.True(t, limiter.Allow())

	// Second request should be allowed
	assert.True(t, limiter.Allow())

	// Third request should be denied
	assert.False(t, limiter.Allow())
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := &RateLimiter{
		limit:    1,
		window:   10 * time.Millisecond, // Very short window for testing
		requests: []time.Time{},
	}

	// First request should be allowed
	assert.True(t, limiter.Allow())

	// Second request should be denied
	assert.False(t, limiter.Allow())

	// Wait for window to expire
	time.Sleep(20 * time.Millisecond)

	// Request should be allowed again after window expires
	assert.True(t, limiter.Allow())
}
