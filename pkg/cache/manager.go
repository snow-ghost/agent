package cache

import (
	"context"
	"fmt"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// CacheManager manages caching and deduplication
type CacheManager struct {
	cache        *LRUCache
	deduplicator *Deduplicator
	config       *CacheConfig
}

// NewCacheManager creates a new cache manager
func NewCacheManager(config *CacheConfig) (*CacheManager, error) {
	cache, err := NewLRUCache(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &CacheManager{
		cache:        cache,
		deduplicator: NewDeduplicator(),
		config:       config,
	}, nil
}

// ExecuteWithCache executes a function with caching and deduplication
func (cm *CacheManager) ExecuteWithCache(
	ctx context.Context,
	req CacheRequest,
	fn func() (core.ChatResponse, error),
) (core.ChatResponse, error) {
	// Check if caching is enabled
	if !req.Cache {
		return cm.deduplicator.Execute(ctx, "", fn)
	}

	// Generate cache key
	key, err := GenerateKey(req)
	if err != nil {
		return core.ChatResponse{}, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Determine TTL
	ttl := req.TTL
	if ttl <= 0 {
		ttl = cm.config.DefaultTTL
	}

	// Execute with cache and deduplication
	return cm.deduplicator.ExecuteWithCache(ctx, key, cm.cache, ttl, fn)
}

// Get retrieves a value from the cache
func (cm *CacheManager) Get(req CacheRequest) (*CacheEntry, bool) {
	if !req.Cache {
		return nil, false
	}

	key, err := GenerateKey(req)
	if err != nil {
		return nil, false
	}

	return cm.cache.Get(key)
}

// Set stores a value in the cache
func (cm *CacheManager) Set(req CacheRequest, response core.ChatResponse) error {
	if !req.Cache {
		return nil
	}

	key, err := GenerateKey(req)
	if err != nil {
		return fmt.Errorf("failed to generate cache key: %w", err)
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = cm.config.DefaultTTL
	}

	cm.cache.Set(key, response, ttl)
	return nil
}

// Delete removes a value from the cache
func (cm *CacheManager) Delete(req CacheRequest) error {
	if !req.Cache {
		return nil
	}

	key, err := GenerateKey(req)
	if err != nil {
		return fmt.Errorf("failed to generate cache key: %w", err)
	}

	cm.cache.Delete(key)
	return nil
}

// Clear removes all values from the cache
func (cm *CacheManager) Clear() {
	cm.cache.Clear()
	cm.deduplicator.Reset()
}

// Stats returns comprehensive cache statistics
func (cm *CacheManager) Stats() map[string]interface{} {
	cacheStats := cm.cache.Stats()
	dedupStats := cm.deduplicator.GetAllStats()

	// Calculate total deduplication stats
	var totalRequests, totalDeduplicated, totalCacheHits int64
	for _, stats := range dedupStats {
		totalRequests += stats.Requests
		totalDeduplicated += stats.Deduplicated
		totalCacheHits += stats.CacheHits
	}

	var dedupRate, cacheHitRate float64
	if totalRequests > 0 {
		dedupRate = float64(totalDeduplicated) / float64(totalRequests)
		cacheHitRate = float64(totalCacheHits) / float64(totalRequests)
	}

	return map[string]interface{}{
		"cache": map[string]interface{}{
			"hits":        cacheStats.Hits,
			"misses":      cacheStats.Misses,
			"size":        cacheStats.Size,
			"max_size":    cacheStats.MaxSize,
			"hit_rate":    cacheStats.HitRate,
			"evictions":   cacheStats.Evictions,
			"expirations": cacheStats.Expirations,
		},
		"deduplication": map[string]interface{}{
			"total_requests":     totalRequests,
			"total_deduplicated": totalDeduplicated,
			"total_cache_hits":   totalCacheHits,
			"dedup_rate":         dedupRate,
			"cache_hit_rate":     cacheHitRate,
		},
		"config": map[string]interface{}{
			"max_size":         cm.config.MaxSize,
			"default_ttl":      cm.config.DefaultTTL.String(),
			"cleanup_interval": cm.config.CleanupInterval.String(),
		},
	}
}

// GetKeyStats returns statistics for a specific cache key
func (cm *CacheManager) GetKeyStats(req CacheRequest) map[string]interface{} {
	key, err := GenerateKey(req)
	if err != nil {
		return map[string]interface{}{
			"error": "failed to generate cache key",
		}
	}

	cacheStats := cm.cache.Stats()
	dedupStats := cm.deduplicator.GetStats(key)

	return map[string]interface{}{
		"key": string(key),
		"cache": map[string]interface{}{
			"hits":     cacheStats.Hits,
			"misses":   cacheStats.Misses,
			"hit_rate": cacheStats.HitRate,
		},
		"deduplication": map[string]interface{}{
			"requests":       dedupStats.Requests,
			"deduplicated":   dedupStats.Deduplicated,
			"cache_hits":     dedupStats.CacheHits,
			"dedup_rate":     cm.deduplicator.GetDedupRate(key),
			"cache_hit_rate": cm.deduplicator.GetCacheHitRate(key),
		},
	}
}

// Close closes the cache manager and cleans up resources
func (cm *CacheManager) Close() {
	cm.cache.Close()
}

// IsCached checks if a request is cached
func (cm *CacheManager) IsCached(req CacheRequest) bool {
	if !req.Cache {
		return false
	}

	key, err := GenerateKey(req)
	if err != nil {
		return false
	}

	_, exists := cm.cache.Get(key)
	return exists
}

// GetCacheSize returns the current cache size
func (cm *CacheManager) GetCacheSize() int {
	return cm.cache.Len()
}

// GetCacheKeys returns all cache keys
func (cm *CacheManager) GetCacheKeys() []CacheKey {
	return cm.cache.Keys()
}

// Warmup preloads the cache with common responses
func (cm *CacheManager) Warmup(requests []CacheRequest, responses []core.ChatResponse) error {
	if len(requests) != len(responses) {
		return fmt.Errorf("requests and responses length mismatch")
	}

	for i, req := range requests {
		if err := cm.Set(req, responses[i]); err != nil {
			return fmt.Errorf("failed to warmup cache: %w", err)
		}
	}

	return nil
}
