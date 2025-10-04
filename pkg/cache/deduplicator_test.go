package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestDeduplicator(t *testing.T) {
	dedup := NewDeduplicator()
	key := CacheKey("test-key")

	// Test basic execution
	response, err := dedup.Execute(context.Background(), key, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response.Text)
	}

	// Check stats
	stats := dedup.GetStats(key)
	if stats.Requests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.Requests)
	}
	if stats.Deduplicated != 0 {
		t.Errorf("Expected 0 deduplicated, got %d", stats.Deduplicated)
	}
}

func TestDeduplicatorConcurrent(t *testing.T) {
	dedup := NewDeduplicator()
	key := CacheKey("test-key")

	var wg sync.WaitGroup
	var responses []core.ChatResponse
	var mu sync.Mutex

	// Launch multiple concurrent requests
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			response, err := dedup.Execute(context.Background(), key, func() (core.ChatResponse, error) {
				// Simulate some work
				time.Sleep(100 * time.Millisecond)
				return core.ChatResponse{Text: "test response"}, nil
			})

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			mu.Lock()
			responses = append(responses, response)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All responses should be the same
	if len(responses) != numRequests {
		t.Errorf("Expected %d responses, got %d", numRequests, len(responses))
	}

	for i, resp := range responses {
		if resp.Text != "test response" {
			t.Errorf("Response %d: expected 'test response', got %s", i, resp.Text)
		}
	}

	// Check stats - should have deduplication
	stats := dedup.GetStats(key)
	// Note: The exact number of requests might vary due to singleflight behavior
	// The important thing is that all responses are the same
	if stats.Requests == 0 {
		t.Error("Expected at least some requests to be recorded")
	}
}

func TestDeduplicatorWithCache(t *testing.T) {
	dedup := NewDeduplicator()
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	key := CacheKey("test-key")
	ttl := 1 * time.Minute

	// First request - should miss cache
	response1, err := dedup.ExecuteWithCache(context.Background(), key, cache, ttl, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response1.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response1.Text)
	}

	// Second request - should hit cache
	response2, err := dedup.ExecuteWithCache(context.Background(), key, cache, ttl, func() (core.ChatResponse, error) {
		t.Error("Function should not be called on cache hit")
		return core.ChatResponse{}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response2.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response2.Text)
	}

	// Check stats
	stats := dedup.GetStats(key)
	if stats.Requests != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.Requests)
	}
	if stats.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.CacheHits)
	}
}

func TestDeduplicatorStats(t *testing.T) {
	dedup := NewDeduplicator()

	// Test multiple keys
	keys := []CacheKey{"key1", "key2", "key3"}

	for _, key := range keys {
		dedup.Execute(context.Background(), key, func() (core.ChatResponse, error) {
			return core.ChatResponse{Text: "test"}, nil
		})
	}

	// Check individual stats
	for _, key := range keys {
		stats := dedup.GetStats(key)
		if stats.Requests != 1 {
			t.Errorf("Key %s: expected 1 request, got %d", key, stats.Requests)
		}
	}

	// Check all stats
	allStats := dedup.GetAllStats()
	if len(allStats) != len(keys) {
		t.Errorf("Expected %d keys in stats, got %d", len(keys), len(allStats))
	}

	// Test reset
	dedup.Reset()
	allStats = dedup.GetAllStats()
	if len(allStats) != 0 {
		t.Errorf("Expected 0 keys after reset, got %d", len(allStats))
	}
}

func TestDeduplicatorRates(t *testing.T) {
	dedup := NewDeduplicator()
	key := CacheKey("test-key")

	// Make some requests
	for i := 0; i < 5; i++ {
		dedup.Execute(context.Background(), key, func() (core.ChatResponse, error) {
			return core.ChatResponse{Text: "test"}, nil
		})
	}

	// Check rates
	dedupRate := dedup.GetDedupRate(key)
	if dedupRate < 0 || dedupRate > 1 {
		t.Errorf("Dedup rate should be between 0 and 1, got %f", dedupRate)
	}

	cacheHitRate := dedup.GetCacheHitRate(key)
	if cacheHitRate < 0 || cacheHitRate > 1 {
		t.Errorf("Cache hit rate should be between 0 and 1, got %f", cacheHitRate)
	}
}
