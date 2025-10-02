package indexer

import (
	"context"
	"strings"
	"testing"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/embeddings"
	"github.com/snow-ghost/agent/vectordb"
)

func TestIndexer(t *testing.T) {
	// Create mock embedder with correct dimension
	embedderConfig := embeddings.DefaultConfig()
	embedderConfig.Dimension = 50
	factory := embeddings.NewMockEmbedderFactory(embedderConfig)
	embedder := factory.CreateEmbedderWithCorpus()

	// Create memory vector store with matching dimension
	vectorConfig := vectordb.DefaultConfig()
	vectorConfig.Dimension = 50
	vectorStore := vectordb.NewMemoryVectorStore(vectorConfig)

	// Create indexer
	indexer := NewIndexer(embedder, vectorStore)

	// Create test artifacts
	artifacts := createTestArtifacts()

	ctx := context.Background()

	// Test indexing
	t.Run("IndexArtifacts", func(t *testing.T) {
		if err := indexer.IndexArtifacts(ctx, artifacts); err != nil {
			t.Fatalf("Failed to index artifacts: %v", err)
		}

		// Verify artifacts were indexed
		count, err := vectorStore.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to get count: %v", err)
		}
		if count != len(artifacts) {
			t.Errorf("Expected %d artifacts, got %d", len(artifacts), count)
		}
	})

	// Test search
	t.Run("SearchArtifacts", func(t *testing.T) {
		testCases := []struct {
			query    string
			expected string // Expected top result ID
		}{
			{"sort integers stable", "sort.integers.v1"},
			{"parse json data", "parse.json.v1"},
			{"reverse string text", "reverse.string.v1"},
		}

		for _, tc := range testCases {
			t.Run(tc.query, func(t *testing.T) {
				results, err := indexer.SearchArtifacts(ctx, tc.query, 3)
				if err != nil {
					t.Fatalf("Search failed: %v", err)
				}

				if len(results) == 0 {
					t.Fatal("No results returned")
				}

				// Check if expected result is in top results
				found := false
				for _, result := range results {
					if result.ID == tc.expected {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected result %s not found in top results", tc.expected)
					for i, result := range results {
						t.Logf("  %d. %s@%s - %s", i+1, result.ID, result.Version, result.Description)
					}
				}
			})
		}
	})

	// Test stats
	t.Run("GetIndexStats", func(t *testing.T) {
		stats, err := indexer.GetIndexStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats["total_artifacts"] != len(artifacts) {
			t.Errorf("Expected %d total artifacts, got %v", len(artifacts), stats["total_artifacts"])
		}

		if stats["embedding_model"] == "" {
			t.Error("Expected embedding model to be set")
		}
	})

	// Test clear
	t.Run("ClearIndex", func(t *testing.T) {
		if err := indexer.ClearIndex(ctx); err != nil {
			t.Fatalf("Failed to clear index: %v", err)
		}

		count, err := vectorStore.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to get count after clear: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 artifacts after clear, got %d", count)
		}
	})
}

func TestBatchIndexing(t *testing.T) {
	// Create mock embedder with correct dimension
	embedderConfig := embeddings.DefaultConfig()
	embedderConfig.Dimension = 50
	factory := embeddings.NewMockEmbedderFactory(embedderConfig)
	embedder := factory.CreateEmbedderWithCorpus()

	// Create memory vector store with matching dimension
	vectorConfig := vectordb.DefaultConfig()
	vectorConfig.Dimension = 50
	vectorStore := vectordb.NewMemoryVectorStore(vectorConfig)

	// Create indexer
	indexer := NewIndexer(embedder, vectorStore)

	// Create test artifacts
	artifacts := createTestArtifacts()

	ctx := context.Background()

	// Test batch indexing
	if err := indexer.BatchIndexArtifacts(ctx, artifacts, 2); err != nil {
		t.Fatalf("Failed to batch index artifacts: %v", err)
	}

	// Verify all artifacts were indexed
	count, err := vectorStore.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != len(artifacts) {
		t.Errorf("Expected %d artifacts, got %d", len(artifacts), count)
	}
}

func createTestArtifacts() []*artifact.Manifest {
	artifacts := []*artifact.Manifest{
		{
			ID:          "sort.integers.v1",
			Version:     "1.0.0",
			Domain:      "algorithms.sorting",
			Description: "Stable integer sorting algorithm",
			Tags:        []string{"sort", "stable", "integers", "algorithm"},
			Lang:        "wasm",
			Entry:       "solve",
			CreatedAt:   "2024-01-01T00:00:00Z",
		},
		{
			ID:          "parse.json.v1",
			Version:     "1.0.0",
			Domain:      "data.parsing",
			Description: "JSON parsing and validation",
			Tags:        []string{"parse", "json", "validation", "data"},
			Lang:        "wasm",
			Entry:       "solve",
			CreatedAt:   "2024-01-01T00:00:00Z",
		},
		{
			ID:          "reverse.string.v1",
			Version:     "1.0.0",
			Domain:      "text.manipulation",
			Description: "String reversal utility",
			Tags:        []string{"reverse", "string", "text", "utility"},
			Lang:        "wasm",
			Entry:       "solve",
			CreatedAt:   "2024-01-01T00:00:00Z",
		},
	}

	return artifacts
}

func TestTextGeneration(t *testing.T) {
	indexer := &Indexer{}

	manifest := &artifact.Manifest{
		ID:          "test.artifact.v1",
		Version:     "1.0.0",
		Domain:      "test.domain",
		Description: "Test artifact for unit testing",
		Tags:        []string{"test", "unit", "artifact"},
		Lang:        "wasm",
		Entry:       "solve",
		Tests:       []core.TestCase{{Name: "test1"}, {Name: "test2"}},
	}

	text := indexer.generateTextForEmbedding(manifest)

	// Check that all important parts are included
	expectedParts := []string{
		"artifact test.artifact.v1",
		"version 1.0.0",
		"domain test.domain",
		"Test artifact for unit testing",
		"tags test unit artifact",
		"wasm function solve",
		"with 2 test cases",
	}

	for _, part := range expectedParts {
		if !strings.Contains(text, part) {
			t.Errorf("Expected text to contain '%s', got: %s", part, text)
		}
	}
}
