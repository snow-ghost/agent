package indexer

import (
	"context"
	"fmt"
	"strings"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/embeddings"
	"github.com/snow-ghost/agent/vectordb"
)

// Indexer handles indexing of artifacts for vector search
type Indexer struct {
	embedder    embeddings.Embedder
	vectorStore vectordb.VectorStore
}

// NewIndexer creates a new artifact indexer
func NewIndexer(embedder embeddings.Embedder, vectorStore vectordb.VectorStore) *Indexer {
	return &Indexer{
		embedder:    embedder,
		vectorStore: vectorStore,
	}
}

// IndexArtifact indexes a single artifact
func (i *Indexer) IndexArtifact(ctx context.Context, manifest *artifact.Manifest) error {
	// Generate text for embedding
	text := i.generateTextForEmbedding(manifest)

	// Create embedding
	embedding, err := i.embedder.EmbedText(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	// Prepare metadata
	meta := map[string]string{
		"id":          manifest.ID,
		"version":     manifest.Version,
		"domain":      manifest.Domain,
		"description": manifest.Description,
		"lang":        manifest.Lang,
		"entry":       manifest.Entry,
		"created_at":  manifest.CreatedAt,
	}

	// Add tags to metadata
	for i, tag := range manifest.Tags {
		meta[fmt.Sprintf("tag_%d", i)] = tag
	}

	// Add test count
	meta["test_count"] = fmt.Sprintf("%d", len(manifest.Tests))

	// Create unique ID for vector store
	vectorID := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)

	// Store in vector database
	if err := i.vectorStore.Upsert(ctx, vectorID, embedding, meta); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	// Update manifest with embedding info
	manifest.Embedding = embedding
	manifest.EmbeddingModel = i.getEmbeddingModelName()

	return nil
}

// IndexArtifacts indexes multiple artifacts
func (i *Indexer) IndexArtifacts(ctx context.Context, manifests []*artifact.Manifest) error {
	for _, manifest := range manifests {
		if err := i.IndexArtifact(ctx, manifest); err != nil {
			return fmt.Errorf("failed to index artifact %s@%s: %w", manifest.ID, manifest.Version, err)
		}
	}
	return nil
}

// SearchArtifacts searches for artifacts by text query
func (i *Indexer) SearchArtifacts(ctx context.Context, query string, topK int) ([]*artifact.Manifest, error) {
	// Create embedding for query
	queryEmbedding, err := i.embedder.EmbedText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Search vector store
	hits, err := i.vectorStore.Search(ctx, queryEmbedding, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert hits to manifests
	var manifests []*artifact.Manifest
	for _, hit := range hits {
		manifest := i.hitToManifest(hit)
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// generateTextForEmbedding creates a text representation of the manifest for embedding
func (i *Indexer) generateTextForEmbedding(manifest *artifact.Manifest) string {
	var parts []string

	// Add ID and version
	parts = append(parts, fmt.Sprintf("artifact %s version %s", manifest.ID, manifest.Version))

	// Add domain
	parts = append(parts, fmt.Sprintf("domain %s", manifest.Domain))

	// Add description
	if manifest.Description != "" {
		parts = append(parts, manifest.Description)
	}

	// Add tags
	if len(manifest.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("tags %s", strings.Join(manifest.Tags, " ")))
	}

	// Add language and entry point
	parts = append(parts, fmt.Sprintf("%s function %s", manifest.Lang, manifest.Entry))

	// Add test information
	if len(manifest.Tests) > 0 {
		parts = append(parts, fmt.Sprintf("with %d test cases", len(manifest.Tests)))
	}

	return strings.Join(parts, " ")
}

// hitToManifest converts a vector search hit to a manifest
func (i *Indexer) hitToManifest(hit vectordb.Hit) *artifact.Manifest {
	manifest := &artifact.Manifest{
		ID:             hit.Meta["id"],
		Version:        hit.Meta["version"],
		Domain:         hit.Meta["domain"],
		Description:    hit.Meta["description"],
		Lang:           hit.Meta["lang"],
		Entry:          hit.Meta["entry"],
		CreatedAt:      hit.Meta["created_at"],
		Embedding:      hit.Vector,
		EmbeddingModel: i.getEmbeddingModelName(),
	}

	// Reconstruct tags
	for i := 0; ; i++ {
		tagKey := fmt.Sprintf("tag_%d", i)
		if tag, exists := hit.Meta[tagKey]; exists {
			manifest.Tags = append(manifest.Tags, tag)
		} else {
			break
		}
	}

	return manifest
}

// getEmbeddingModelName returns the name of the embedding model
func (i *Indexer) getEmbeddingModelName() string {
	// Try to get model name from embedder if it has a config
	if configGetter, ok := i.embedder.(interface {
		GetConfig() *embeddings.EmbeddingConfig
	}); ok {
		config := configGetter.GetConfig()
		if config != nil && config.Model != "" {
			return config.Model
		}
	}

	// Default fallback
	return "unknown"
}

// ClearIndex clears all indexed artifacts
func (i *Indexer) ClearIndex(ctx context.Context) error {
	return i.vectorStore.Clear(ctx)
}

// GetIndexStats returns statistics about the index
func (i *Indexer) GetIndexStats(ctx context.Context) (map[string]interface{}, error) {
	count, err := i.vectorStore.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vector count: %w", err)
	}

	stats := map[string]interface{}{
		"total_artifacts": count,
		"embedding_model": i.getEmbeddingModelName(),
	}

	// Add vector store specific stats if available
	if statsGetter, ok := i.vectorStore.(interface{ GetStats() map[string]interface{} }); ok {
		vectorStats := statsGetter.GetStats()
		for k, v := range vectorStats {
			stats[k] = v
		}
	}

	return stats, nil
}

// BatchIndexArtifacts indexes artifacts in batches for better performance
func (idx *Indexer) BatchIndexArtifacts(ctx context.Context, manifests []*artifact.Manifest, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	for i := 0; i < len(manifests); i += batchSize {
		end := i + batchSize
		if end > len(manifests) {
			end = len(manifests)
		}

		batch := manifests[i:end]
		if err := idx.IndexArtifacts(ctx, batch); err != nil {
			return fmt.Errorf("failed to index batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}
