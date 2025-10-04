package cache

import (
	"context"
	"sync"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
	"golang.org/x/sync/singleflight"
)

// Deduplicator handles in-flight request deduplication
type Deduplicator struct {
	group singleflight.Group
	mu    sync.RWMutex
	stats map[CacheKey]*DedupStats
}

// DedupStats represents deduplication statistics
type DedupStats struct {
	Requests     int64 `json:"requests"`
	Deduplicated int64 `json:"deduplicated"`
	CacheHits    int64 `json:"cache_hits"`
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		stats: make(map[CacheKey]*DedupStats),
	}
}

// Execute executes a function with deduplication
func (d *Deduplicator) Execute(ctx context.Context, key CacheKey, fn func() (core.ChatResponse, error)) (core.ChatResponse, error) {
	// Update stats
	d.updateStats(key, false, false)

	// Use singleflight to deduplicate concurrent requests
	result, err, shared := d.group.Do(string(key), func() (interface{}, error) {
		return fn()
	})

	if err != nil {
		return core.ChatResponse{}, err
	}

	response := result.(core.ChatResponse)

	// Update stats based on whether this was a shared result
	if shared {
		d.updateStats(key, true, false)
	}

	return response, nil
}

// ExecuteWithCache executes a function with both deduplication and caching
func (d *Deduplicator) ExecuteWithCache(
	ctx context.Context,
	key CacheKey,
	cache *LRUCache,
	ttl time.Duration,
	fn func() (core.ChatResponse, error),
) (core.ChatResponse, error) {
	// Check cache first
	if cache != nil {
		if entry, exists := cache.Get(key); exists {
			d.updateStats(key, false, true)
			return entry.Response, nil
		}
	}

	// Update stats
	d.updateStats(key, false, false)

	// Use singleflight to deduplicate concurrent requests
	result, err, shared := d.group.Do(string(key), func() (interface{}, error) {
		response, err := fn()
		if err != nil {
			return nil, err
		}

		// Cache the result
		if cache != nil {
			cache.Set(key, response, ttl)
		}

		return response, nil
	})

	if err != nil {
		return core.ChatResponse{}, err
	}

	response := result.(core.ChatResponse)

	// Update stats based on whether this was a shared result
	if shared {
		d.updateStats(key, true, false)
	}

	return response, nil
}

// updateStats updates deduplication statistics
func (d *Deduplicator) updateStats(key CacheKey, deduplicated, cacheHit bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats, exists := d.stats[key]
	if !exists {
		stats = &DedupStats{}
		d.stats[key] = stats
	}

	stats.Requests++
	if deduplicated {
		stats.Deduplicated++
	}
	if cacheHit {
		stats.CacheHits++
	}
}

// GetStats returns deduplication statistics for a key
func (d *Deduplicator) GetStats(key CacheKey) *DedupStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if stats, exists := d.stats[key]; exists {
		return &DedupStats{
			Requests:     stats.Requests,
			Deduplicated: stats.Deduplicated,
			CacheHits:    stats.CacheHits,
		}
	}

	return &DedupStats{}
}

// GetAllStats returns all deduplication statistics
func (d *Deduplicator) GetAllStats() map[CacheKey]*DedupStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[CacheKey]*DedupStats)
	for key, stats := range d.stats {
		result[key] = &DedupStats{
			Requests:     stats.Requests,
			Deduplicated: stats.Deduplicated,
			CacheHits:    stats.CacheHits,
		}
	}

	return result
}

// Reset resets all statistics
func (d *Deduplicator) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stats = make(map[CacheKey]*DedupStats)
}

// ResetKey resets statistics for a specific key
func (d *Deduplicator) ResetKey(key CacheKey) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.stats, key)
}

// GetDedupRate calculates the deduplication rate for a key
func (d *Deduplicator) GetDedupRate(key CacheKey) float64 {
	stats := d.GetStats(key)
	if stats.Requests == 0 {
		return 0.0
	}
	return float64(stats.Deduplicated) / float64(stats.Requests)
}

// GetCacheHitRate calculates the cache hit rate for a key
func (d *Deduplicator) GetCacheHitRate(key CacheKey) float64 {
	stats := d.GetStats(key)
	if stats.Requests == 0 {
		return 0.0
	}
	return float64(stats.CacheHits) / float64(stats.Requests)
}
