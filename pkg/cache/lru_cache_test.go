package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

func TestLRUCache(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         10,
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	}

	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test basic set/get
	key := CacheKey("test-key")
	response := core.ChatResponse{
		Text:  "test response",
		Model: "test-model",
	}

	cache.Set(key, response, 0)

	entry, exists := cache.Get(key)
	if !exists {
		t.Error("Expected entry to exist")
	}
	if entry.Response.Text != "test response" {
		t.Errorf("Expected 'test response', got %s", entry.Response.Text)
	}

	// Test stats
	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Expected 0 misses, got %d", stats.Misses)
	}
}

func TestLRUCacheExpiration(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         10,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: 25 * time.Millisecond,
	}

	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	key := CacheKey("test-key")
	response := core.ChatResponse{Text: "test response"}

	cache.Set(key, response, 50*time.Millisecond)

	// Should exist initially
	_, exists := cache.Get(key)
	if !exists {
		t.Error("Expected entry to exist initially")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, exists = cache.Get(key)
	if exists {
		t.Error("Expected entry to be expired")
	}

	// Check stats
	stats := cache.Stats()
	if stats.Expirations == 0 {
		t.Error("Expected expirations to be > 0")
	}
}

func TestLRUCacheEviction(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         3,
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}

	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Fill cache beyond capacity
	for i := 0; i < 5; i++ {
		key := CacheKey(fmt.Sprintf("key-%d", i))
		response := core.ChatResponse{Text: fmt.Sprintf("response-%d", i)}
		cache.Set(key, response, 0)
	}

	// Check that cache size is at max
	if cache.Len() != 3 {
		t.Errorf("Expected cache size to be 3, got %d", cache.Len())
	}

	// Check stats
	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Error("Expected evictions to be > 0")
	}
}

func TestLRUCacheTouch(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         10,
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}

	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	key := CacheKey("test-key")
	response := core.ChatResponse{Text: "test response"}

	cache.Set(key, response, 0)

	// Access multiple times
	for i := 0; i < 3; i++ {
		entry, exists := cache.Get(key)
		if !exists {
			t.Error("Expected entry to exist")
		}
		if entry.AccessCount != i+1 {
			t.Errorf("Expected access count to be %d, got %d", i+1, entry.AccessCount)
		}
	}
}

func TestLRUCacheClear(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         10,
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}

	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Add some entries
	for i := 0; i < 3; i++ {
		key := CacheKey(fmt.Sprintf("key-%d", i))
		response := core.ChatResponse{Text: fmt.Sprintf("response-%d", i)}
		cache.Set(key, response, 0)
	}

	if cache.Len() != 3 {
		t.Errorf("Expected cache size to be 3, got %d", cache.Len())
	}

	// Clear cache
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Expected cache size to be 0, got %d", cache.Len())
	}
}
