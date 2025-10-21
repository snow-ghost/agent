package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/snow-ghost/agent/embeddings"
	"github.com/snow-ghost/agent/kb/fs"
	"github.com/snow-ghost/agent/kb/indexer"
	"github.com/snow-ghost/agent/vectordb"
)

func main() {
	var (
		action       = flag.String("action", "", "Action to perform (list/clear/stats/reindex/query/verify)")
		query        = flag.String("query", "", "Search query for query action")
		artifactsDir = flag.String("artifacts-dir", "./artifacts", "Path to artifacts directory")
		qdrantURL    = flag.String("qdrant-url", "", "Qdrant URL (default from env)")
		collection   = flag.String("collection", "artifacts", "Collection name")
		verbose      = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *action == "" {
		fmt.Fprintf(os.Stderr, "Error: -action flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set Qdrant URL from environment if not provided
	if *qdrantURL == "" {
		*qdrantURL = os.Getenv("QDRANT_URL")
		if *qdrantURL == "" {
			*qdrantURL = "http://localhost:6333"
		}
	}

	// Set environment variables for Qdrant
	os.Setenv("QDRANT_URL", *qdrantURL)
	os.Setenv("QDRANT_COLLECTION", *collection)

	ctx := context.Background()

	switch *action {
	case "list":
		err := listArtifacts(ctx, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing artifacts: %v\n", err)
			os.Exit(1)
		}

	case "clear":
		err := clearCollection(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing collection: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Collection cleared successfully")

	case "stats":
		err := showStats(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting stats: %v\n", err)
			os.Exit(1)
		}

	case "reindex":
		err := reindexArtifacts(ctx, *artifactsDir, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reindexing artifacts: %v\n", err)
			os.Exit(1)
		}

	case "query":
		if *query == "" {
			fmt.Fprintf(os.Stderr, "Error: -query flag is required for query action\n")
			os.Exit(1)
		}
		err := searchArtifacts(ctx, *query, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching artifacts: %v\n", err)
			os.Exit(1)
		}

	case "verify":
		err := verifyConsistency(ctx, *artifactsDir, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying consistency: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action '%s'\n", *action)
		fmt.Fprintf(os.Stderr, "Valid actions: list, clear, stats, reindex, query, verify\n")
		os.Exit(1)
	}
}

// listArtifacts lists all indexed artifacts
func listArtifacts(ctx context.Context, verbose bool) error {
	// Create vector store
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	// Get collection info
	info, err := vectorStore.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection info: %w", err)
	}

	fmt.Printf("Collection: %s\n", vectorStore.GetConfig().Collection)
	fmt.Printf("Total artifacts: %d\n", info)

	if verbose {
		// For now, we can't easily list all artifacts without search
		// This would require implementing a scan operation in the vector store
		fmt.Println("Note: Detailed listing requires search functionality")
	}

	return nil
}

// clearCollection clears all vectors from the collection
func clearCollection(ctx context.Context) error {
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	return vectorStore.Clear(ctx)
}

// showStats shows collection statistics
func showStats(ctx context.Context) error {
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	// Get basic stats
	count, err := vectorStore.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to get count: %w", err)
	}

	config := vectorStore.GetConfig()
	fmt.Printf("Collection: %s\n", config.Collection)
	fmt.Printf("Dimension: %d\n", config.Dimension)
	fmt.Printf("Distance: %s\n", config.Distance)
	fmt.Printf("Total vectors: %d\n", count)

	// Health check
	if err := vectorStore.Health(ctx); err != nil {
		fmt.Printf("Health: UNHEALTHY (%v)\n", err)
	} else {
		fmt.Printf("Health: OK\n")
	}

	return nil
}

// reindexArtifacts re-indexes all artifacts from filesystem to Qdrant
func reindexArtifacts(ctx context.Context, artifactsDir string, verbose bool) error {
	// Create knowledge base
	kb := fs.NewKnowledgeBaseFS(artifactsDir)

	// Create embedder
	embedder, err := embeddings.NewLMStudioEmbedderFromEnv()
	if err != nil {
		// Fallback to mock embedder if LMStudio is not available
		fmt.Println("Warning: LMStudio not available, using mock embedder")
		embedder = &embeddings.MockEmbedder{}
	}

	// Create vector store
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	// Create indexer
	idx := indexer.NewIndexer(embedder, vectorStore)

	// Clear existing index
	if verbose {
		fmt.Println("Clearing existing index...")
	}
	if err := idx.ClearIndex(ctx); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Load all artifacts
	artifacts := kb.ListArtifacts()
	if verbose {
		fmt.Printf("Found %d artifacts to index\n", len(artifacts))
	}

	// Index artifacts in batches
	batchSize := 10
	start := time.Now()
	if err := idx.BatchIndexArtifacts(ctx, artifacts, batchSize); err != nil {
		return fmt.Errorf("failed to index artifacts: %w", err)
	}
	duration := time.Since(start)

	if verbose {
		fmt.Printf("Indexed %d artifacts in %v\n", len(artifacts), duration)
	}

	// Show final stats
	stats, err := idx.GetIndexStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get index stats: %w", err)
	}

	fmt.Printf("Reindexing completed successfully\n")
	fmt.Printf("Total artifacts indexed: %v\n", stats["total_artifacts"])
	fmt.Printf("Embedding model: %v\n", stats["embedding_model"])

	return nil
}

// searchArtifacts searches for artifacts by text query
func searchArtifacts(ctx context.Context, query string, verbose bool) error {
	// Create embedder
	embedder, err := embeddings.NewLMStudioEmbedderFromEnv()
	if err != nil {
		fmt.Println("Warning: LMStudio not available, using mock embedder")
		embedder = &embeddings.MockEmbedder{}
	}

	// Create vector store
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	// Create indexer
	idx := indexer.NewIndexer(embedder, vectorStore)

	// Search
	start := time.Now()
	results, err := idx.SearchArtifacts(ctx, query, 10)
	if err != nil {
		return fmt.Errorf("failed to search artifacts: %w", err)
	}
	duration := time.Since(start)

	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Found %d results in %v\n", len(results), duration)

	for i, manifest := range results {
		fmt.Printf("\n%d. %s@%s\n", i+1, manifest.ID, manifest.Version)
		fmt.Printf("   Domain: %s\n", manifest.Domain)
		fmt.Printf("   Description: %s\n", manifest.Description)
		fmt.Printf("   Language: %s\n", manifest.Lang)
		if len(manifest.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(manifest.Tags, ", "))
		}
	}

	return nil
}

// verifyConsistency checks consistency between filesystem and Qdrant
func verifyConsistency(ctx context.Context, artifactsDir string, verbose bool) error {
	// Create knowledge base
	kb := fs.NewKnowledgeBaseFS(artifactsDir)

	// Create vector store
	vectorStore, err := vectordb.NewQdrantVectorStoreFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer vectorStore.Close()

	// Get counts
	fsArtifacts := kb.ListArtifacts()
	qdrantCount, err := vectorStore.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Qdrant count: %w", err)
	}

	fmt.Printf("Filesystem artifacts: %d\n", len(fsArtifacts))
	fmt.Printf("Qdrant vectors: %d\n", qdrantCount)

	if len(fsArtifacts) != qdrantCount {
		fmt.Printf("WARNING: Count mismatch! Filesystem has %d artifacts but Qdrant has %d vectors\n",
			len(fsArtifacts), qdrantCount)
		return fmt.Errorf("consistency check failed")
	}

	// Check each artifact exists in Qdrant
	missing := 0
	for _, manifest := range fsArtifacts {
		vectorID := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
		_, _, err := vectorStore.Get(ctx, vectorID)
		if err != nil {
			if verbose {
				fmt.Printf("Missing in Qdrant: %s\n", vectorID)
			}
			missing++
		}
	}

	if missing > 0 {
		fmt.Printf("WARNING: %d artifacts missing from Qdrant\n", missing)
		return fmt.Errorf("consistency check failed")
	}

	fmt.Println("Consistency check passed: All filesystem artifacts found in Qdrant")
	return nil
}
