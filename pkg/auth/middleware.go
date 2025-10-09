package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/snow-ghost/agent/pkg/errors"
)

// APIKey represents an API key with metadata
type APIKey struct {
	Key       string     `json:"key"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RateLimit int        `json:"rate_limit"` // requests per minute
	IsActive  bool       `json:"is_active"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	RequireAuth      bool
	APIKeys          map[string]*APIKey
	DefaultRateLimit int
	JWTSecret        string
	JWTExpiry        time.Duration
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		RequireAuth:      false, // Default to no auth for development
		APIKeys:          make(map[string]*APIKey),
		DefaultRateLimit: 1000, // 1000 requests per minute
		JWTSecret:        "default-secret-change-in-production",
		JWTExpiry:        24 * time.Hour,
	}
}

// RateLimiter is a simple rate limiter implementation
type RateLimiter struct {
	limit    int
	window   time.Duration
	requests []time.Time
	mu       sync.Mutex
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old requests
	var validRequests []time.Time
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Check if we're under the limit
	if len(rl.requests) < rl.limit {
		rl.requests = append(rl.requests, now)
		return true
	}

	return false
}

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	config       *AuthConfig
	rateLimiters map[string]*RateLimiter
	mu           sync.RWMutex
	errorHandler *errors.ErrorHandler
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *AuthConfig, errorHandler *errors.ErrorHandler) *AuthMiddleware {
	if config == nil {
		config = DefaultAuthConfig()
	}

	return &AuthMiddleware{
		config:       config,
		rateLimiters: make(map[string]*RateLimiter),
		errorHandler: errorHandler,
	}
}

// Middleware returns the HTTP middleware function
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and metrics
		if a.shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check if auth is required
		if !a.config.RequireAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key from request
		apiKey, err := a.extractAPIKey(r)
		if err != nil {
			a.errorHandler.HandleError(w, r, errors.NewAuthenticationError("Invalid API key"))
			return
		}

		// Validate API key
		keyInfo, err := a.validateAPIKey(apiKey)
		if err != nil {
			a.errorHandler.HandleError(w, r, errors.NewAuthenticationError("Invalid API key"))
			return
		}

		// Check rate limiting
		if err := a.checkRateLimit(apiKey, keyInfo); err != nil {
			a.errorHandler.HandleError(w, r, errors.NewRateLimitError("Rate limit exceeded", time.Minute))
			return
		}

		// Add API key info to context
		ctx := context.WithValue(r.Context(), "api_key", keyInfo)
		ctx = context.WithValue(ctx, "api_key_name", keyInfo.Name)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// shouldSkipAuth checks if the path should skip authentication
func (a *AuthMiddleware) shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/ready",
		"/favicon.ico",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// extractAPIKey extracts API key from request headers
func (a *AuthMiddleware) extractAPIKey(r *http.Request) (string, error) {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1], nil
		}
		if len(parts) == 1 {
			return parts[0], nil
		}
	}

	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey, nil
	}

	// Check query parameter
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey, nil
	}

	return "", fmt.Errorf("no API key found")
}

// validateAPIKey validates the API key
func (a *AuthMiddleware) validateAPIKey(apiKey string) (*APIKey, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keyInfo, exists := a.config.APIKeys[apiKey]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}

	if !keyInfo.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	return keyInfo, nil
}

// checkRateLimit checks if the API key is within rate limits
func (a *AuthMiddleware) checkRateLimit(apiKey string, keyInfo *APIKey) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create rate limiter for this API key
	limiter, exists := a.rateLimiters[apiKey]
	if !exists {
		rateLimit := keyInfo.RateLimit
		if rateLimit <= 0 {
			rateLimit = a.config.DefaultRateLimit
		}
		// Create a simple rate limiter (this is a placeholder implementation)
		// In a real implementation, you would use a proper rate limiting library
		limiter = &RateLimiter{
			limit:  rateLimit,
			window: time.Minute,
		}
		a.rateLimiters[apiKey] = limiter
	}

	// Check if request is allowed
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// AddAPIKey adds a new API key
func (a *AuthMiddleware) AddAPIKey(name string, rateLimit int, expiresAt *time.Time) (*APIKey, error) {
	apiKey, err := a.generateAPIKey()
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

	a.mu.Lock()
	defer a.mu.Unlock()

	a.config.APIKeys[apiKey] = keyInfo
	return keyInfo, nil
}

// RevokeAPIKey revokes an API key
func (a *AuthMiddleware) RevokeAPIKey(apiKey string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if keyInfo, exists := a.config.APIKeys[apiKey]; exists {
		keyInfo.IsActive = false
		delete(a.rateLimiters, apiKey)
		return nil
	}

	return fmt.Errorf("API key not found")
}

// ListAPIKeys returns all API keys (without the actual key values)
func (a *AuthMiddleware) ListAPIKeys() []*APIKey {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keys := make([]*APIKey, 0, len(a.config.APIKeys))
	for _, keyInfo := range a.config.APIKeys {
		// Don't expose the actual key
		safeKeyInfo := *keyInfo
		safeKeyInfo.Key = "***" + keyInfo.Key[len(keyInfo.Key)-4:]
		keys = append(keys, &safeKeyInfo)
	}

	return keys
}

// generateAPIKey generates a new API key
func (a *AuthMiddleware) generateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Create SHA256 hash
	hash := sha256.Sum256(bytes)

	// Return hex encoded hash
	return hex.EncodeToString(hash[:]), nil
}

// GetAPIKeyFromContext extracts API key info from context
func GetAPIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	keyInfo, ok := ctx.Value("api_key").(*APIKey)
	return keyInfo, ok
}

// GetAPIKeyNameFromContext extracts API key name from context
func GetAPIKeyNameFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value("api_key_name").(string)
	return name, ok
}

// RequireAuth creates a middleware that requires authentication
func RequireAuth(config *AuthConfig, errorHandler *errors.ErrorHandler) func(http.Handler) http.Handler {
	auth := NewAuthMiddleware(config, errorHandler)
	auth.config.RequireAuth = true
	return auth.Middleware
}

// OptionalAuth creates a middleware that allows optional authentication
func OptionalAuth(config *AuthConfig, errorHandler *errors.ErrorHandler) func(http.Handler) http.Handler {
	auth := NewAuthMiddleware(config, errorHandler)
	auth.config.RequireAuth = false
	return auth.Middleware
}
