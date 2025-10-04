package cache

import (
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/snow-ghost/agent/pkg/router/core"
)

// LRUCache implements an LRU cache with TTL support
type LRUCache struct {
	cache    *lru.Cache[CacheKey, *CacheEntry]
	config   *CacheConfig
	stats    *CacheStats
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(config *CacheConfig) (*LRUCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache, err := lru.New[CacheKey, *CacheEntry](config.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	c := &LRUCache{
		cache:    cache,
		config:   config,
		stats:    &CacheStats{MaxSize: config.MaxSize},
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c, nil
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key CacheKey) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache.Get(key)
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if entry.IsExpired() {
		c.cache.Remove(key)
		c.stats.Expirations++
		c.stats.Misses++
		return nil, false
	}

	// Update access info
	entry.Touch()
	c.stats.Hits++
	return entry, true
}

// Set stores a value in the cache
func (c *LRUCache) Set(key CacheKey, response core.ChatResponse, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use default TTL if not specified
	if ttl <= 0 {
		ttl = c.config.DefaultTTL
	}

	now := time.Now()
	entry := &CacheEntry{
		Response:     response,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
		AccessCount:  0,
		LastAccessed: now,
	}

	// Check if we need to evict
	if c.cache.Len() >= c.config.MaxSize {
		// Remove oldest entry
		oldestKey, _, _ := c.cache.GetOldest()
		if oldestKey != "" {
			c.cache.Remove(oldestKey)
			c.stats.Evictions++
		}
	}

	c.cache.Add(key, entry)
	c.stats.Size = c.cache.Len()
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key CacheKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Remove(key)
	c.stats.Size = c.cache.Len()
}

// Clear removes all values from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Purge()
	c.stats.Size = 0
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := *c.stats
	stats.Size = c.cache.Len()
	stats.CalculateHitRate()
	return stats
}

// Reset resets cache statistics
func (c *LRUCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats = &CacheStats{MaxSize: c.config.MaxSize}
}

// Close stops the cache and cleans up resources
func (c *LRUCache) Close() {
	close(c.stopChan)
}

// cleanup periodically removes expired entries
func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopChan:
			return
		}
	}
}

// cleanupExpired removes expired entries
func (c *LRUCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := c.cache.Keys()
	expiredCount := 0

	for _, key := range keys {
		if entry, exists := c.cache.Peek(key); exists && entry.IsExpired() {
			c.cache.Remove(key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		c.stats.Expirations += int64(expiredCount)
		c.stats.Size = c.cache.Len()
	}
}

// GetWithTTL retrieves a value and its remaining TTL
func (c *LRUCache) GetWithTTL(key CacheKey) (*CacheEntry, time.Duration, bool) {
	entry, exists := c.Get(key)
	if !exists {
		return nil, 0, false
	}

	remaining := time.Until(entry.ExpiresAt)
	return entry, remaining, true
}

// SetWithTTL stores a value with a specific TTL
func (c *LRUCache) SetWithTTL(key CacheKey, response core.ChatResponse, ttl time.Duration) {
	c.Set(key, response, ttl)
}

// Keys returns all cache keys
func (c *LRUCache) Keys() []CacheKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache.Keys()
}

// Len returns the number of items in the cache
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache.Len()
}
