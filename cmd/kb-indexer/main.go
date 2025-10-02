package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/embeddings"
	"github.com/snow-ghost/agent/kb/fs"
	"github.com/snow-ghost/agent/kb/indexer"
	"github.com/snow-ghost/agent/vectordb"
)

func main() {
	var (
		artifactsDir = flag.String("artifacts-dir", "./artifacts", "Directory containing artifacts")
		embedderType = flag.String("embedder", "mock", "Embedder type: mock, openai")
		vectorType   = flag.String("vector-store", "memory", "Vector store type: memory, qdrant")
		batchSize    = flag.Int("batch-size", 10, "Batch size for indexing")
		clear        = flag.Bool("clear", false, "Clear existing index before indexing")
		stats        = flag.Bool("stats", false, "Show index statistics")
		verbose      = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	ctx := context.Background()

	// Create embedder
	embedder, err := createEmbedder(*embedderType)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Create vector store
	vectorStore, err := createVectorStore(*vectorType)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Create indexer
	idx := indexer.NewIndexer(embedder, vectorStore)

	// Show stats if requested
	if *stats {
		showStats(ctx, idx)
		return
	}

	// Clear index if requested
	if *clear {
		if *verbose {
			fmt.Println("Clearing existing index...")
		}
		if err := idx.ClearIndex(ctx); err != nil {
			log.Fatalf("Failed to clear index: %v", err)
		}
		fmt.Println("Index cleared successfully")
	}

	// Load artifacts
	if *verbose {
		fmt.Printf("Loading artifacts from %s...\n", *artifactsDir)
	}

	artifacts, err := loadArtifacts(*artifactsDir)
	if err != nil {
		log.Fatalf("Failed to load artifacts: %v", err)
	}

	if len(artifacts) == 0 {
		fmt.Println("No artifacts found to index")
		return
	}

	fmt.Printf("Found %d artifacts to index\n", len(artifacts))

	// Index artifacts
	start := time.Now()
	if err := idx.BatchIndexArtifacts(ctx, artifacts, *batchSize); err != nil {
		log.Fatalf("Failed to index artifacts: %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("Successfully indexed %d artifacts in %v\n", len(artifacts), duration)

	// Show final stats
	showStats(ctx, idx)
}

func createEmbedder(embedderType string) (embeddings.Embedder, error) {
	switch embedderType {
	case "mock":
		factory := embeddings.NewMockEmbedderFactory(nil)
		return factory.CreateEmbedderWithCorpus(), nil
	case "openai":
		return embeddings.NewOpenAIEmbedderFromEnv()
	default:
		return nil, fmt.Errorf("unknown embedder type: %s", embedderType)
	}
}

func createVectorStore(vectorType string) (vectordb.VectorStore, error) {
	switch vectorType {
	case "memory":
		config := vectordb.DefaultConfig()
		return vectordb.NewMemoryVectorStore(config), nil
	case "qdrant":
		return vectordb.NewQdrantVectorStoreFromEnv()
	default:
		return nil, fmt.Errorf("unknown vector store type: %s", vectorType)
	}
}

func loadArtifacts(artifactsDir string) ([]*artifact.Manifest, error) {
	// Create file system knowledge base to load artifacts
	kb := fs.NewKnowledgeBaseFS(artifactsDir)

	// Load all artifacts
	manifests := kb.ListArtifacts()

	// Filter out artifacts that already have embeddings (unless we're reindexing)
	var toIndex []*artifact.Manifest
	for _, manifest := range manifests {
		if len(manifest.Embedding) == 0 {
			toIndex = append(toIndex, manifest)
		}
	}

	return toIndex, nil
}

func showStats(ctx context.Context, idx *indexer.Indexer) {
	stats, err := idx.GetIndexStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get index stats: %v", err)
	}

	fmt.Println("\nIndex Statistics:")
	fmt.Println("================")
	for key, value := range stats {
		fmt.Printf("%-20s: %v\n", key, value)
	}
}

// Test search functionality
func testSearch(ctx context.Context, idx *indexer.Indexer) {
	queries := []string{
		"sort integers stable",
		"parse json data",
		"reverse string text",
		"search algorithm",
		"filter array elements",
	}

	fmt.Println("\nTesting Search:")
	fmt.Println("===============")

	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		results, err := idx.SearchArtifacts(ctx, query, 3)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		for i, result := range results {
			fmt.Printf("  %d. %s@%s (%.3f) - %s\n",
				i+1, result.ID, result.Version, 0.0, result.Description)
		}
	}
}
