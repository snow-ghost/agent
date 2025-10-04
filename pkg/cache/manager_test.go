package cache

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestCacheManager(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         10,
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	}

	manager, err := NewCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Test basic execution with cache enabled
	req := CacheRequest{
		Model:    "test-model",
		Messages: []core.Message{{Role: "user", Content: "test"}},
		Cache:    true,
	}

	response, err := manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response.Text)
	}

	// Test cache hit
	response2, err := manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		t.Error("Function should not be called on cache hit")
		return core.ChatResponse{}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response2.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response.Text)
	}
}

func TestCacheManagerWithoutCache(t *testing.T) {
	manager, err := NewCacheManager(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Test execution with cache disabled
	req := CacheRequest{
		Model:    "test-model",
		Messages: []core.Message{{Role: "user", Content: "test"}},
		Cache:    false,
	}

	response, err := manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if response.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", response.Text)
	}

	// Should not be cached
	if manager.IsCached(req) {
		t.Error("Request should not be cached when cache is disabled")
	}
}

func TestCacheManagerStats(t *testing.T) {
	manager, err := NewCacheManager(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Make some requests
	req := CacheRequest{
		Model:    "test-model",
		Messages: []core.Message{{Role: "user", Content: "test"}},
		Cache:    true,
	}

	// First request - cache miss
	manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	// Second request - cache hit
	manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		t.Error("Function should not be called on cache hit")
		return core.ChatResponse{}, nil
	})

	// Check stats
	stats := manager.Stats()

	cacheStats, ok := stats["cache"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected cache stats")
	}

	if cacheStats["hits"] != int64(1) {
		t.Errorf("Expected 1 cache hit, got %v", cacheStats["hits"])
	}
	if cacheStats["misses"] != int64(1) {
		t.Errorf("Expected 1 cache miss, got %v", cacheStats["misses"])
	}
}

func TestCacheManagerKeyStats(t *testing.T) {
	manager, err := NewCacheManager(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	req := CacheRequest{
		Model:    "test-model",
		Messages: []core.Message{{Role: "user", Content: "test"}},
		Cache:    true,
	}

	// Make some requests
	manager.ExecuteWithCache(context.Background(), req, func() (core.ChatResponse, error) {
		return core.ChatResponse{Text: "test response"}, nil
	})

	// Check key stats
	keyStats := manager.GetKeyStats(req)

	if keyStats["key"] == "" {
		t.Error("Expected key to be present in stats")
	}

	dedupStats, ok := keyStats["deduplication"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected deduplication stats")
	}

	if dedupStats["requests"] != int64(1) {
		t.Errorf("Expected 1 request, got %v", dedupStats["requests"])
	}
}

func TestCacheManagerWarmup(t *testing.T) {
	manager, err := NewCacheManager(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Test warmup
	requests := []CacheRequest{
		{
			Model:    "test-model",
			Messages: []core.Message{{Role: "user", Content: "test1"}},
			Cache:    true,
		},
		{
			Model:    "test-model",
			Messages: []core.Message{{Role: "user", Content: "test2"}},
			Cache:    true,
		},
	}

	responses := []core.ChatResponse{
		{Text: "response1"},
		{Text: "response2"},
	}

	err = manager.Warmup(requests, responses)
	if err != nil {
		t.Errorf("Expected no error during warmup, got %v", err)
	}

	// Check that items are cached
	for i, req := range requests {
		if !manager.IsCached(req) {
			t.Errorf("Request %d should be cached after warmup", i)
		}
	}
}

func TestCacheManagerClear(t *testing.T) {
	manager, err := NewCacheManager(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Add some items
	req := CacheRequest{
		Model:    "test-model",
		Messages: []core.Message{{Role: "user", Content: "test"}},
		Cache:    true,
	}

	manager.Set(req, core.ChatResponse{Text: "test response"})

	if !manager.IsCached(req) {
		t.Error("Request should be cached")
	}

	// Clear cache
	manager.Clear()

	if manager.IsCached(req) {
		t.Error("Request should not be cached after clear")
	}

	if manager.GetCacheSize() != 0 {
		t.Errorf("Expected cache size to be 0, got %d", manager.GetCacheSize())
	}
}
