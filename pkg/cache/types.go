package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// CacheKey represents a cache key
type CacheKey string

// CacheEntry represents a cached response
type CacheEntry struct {
	Response     core.ChatResponse `json:"response"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	AccessCount  int               `json:"access_count"`
	LastAccessed time.Time         `json:"last_accessed"`
}

// IsExpired checks if the cache entry is expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Touch updates the access time and count
func (e *CacheEntry) Touch() {
	e.LastAccessed = time.Now()
	e.AccessCount++
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxSize         int           `json:"max_size"`         // Maximum number of entries
	DefaultTTL      time.Duration `json:"default_ttl"`      // Default TTL for entries
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to clean expired entries
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:         1000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
}

// CacheRequest represents a request with caching options
type CacheRequest struct {
	Model       string            `json:"model"`
	Messages    []core.Message    `json:"messages"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Tools       []core.Tool       `json:"tools,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Cache options
	Cache bool          `json:"cache,omitempty"`
	TTL   time.Duration `json:"ttl,omitempty"`
}

// GenerateKey generates a cache key for a request
func GenerateKey(req CacheRequest) (CacheKey, error) {
	// Create a normalized version of the request for hashing
	normalized := struct {
		Model       string         `json:"model"`
		Messages    []core.Message `json:"messages"`
		Temperature float32        `json:"temperature"`
		TopP        float32        `json:"top_p"`
		MaxTokens   int            `json:"max_tokens"`
		Tools       []core.Tool    `json:"tools"`
		// Exclude metadata and cache options from key
	}{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Tools:       req.Tools,
	}

	// Serialize to JSON
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Generate SHA256 hash
	hash := sha256.Sum256(data)
	return CacheKey(fmt.Sprintf("%x", hash)), nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	HitRate     float64 `json:"hit_rate"`
	Evictions   int64   `json:"evictions"`
	Expirations int64   `json:"expirations"`
}

// CalculateHitRate calculates the hit rate
func (s *CacheStats) CalculateHitRate() {
	total := s.Hits + s.Misses
	if total > 0 {
		s.HitRate = float64(s.Hits) / float64(total)
	} else {
		s.HitRate = 0.0
	}
}
