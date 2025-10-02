package vectordb

import (
	"context"
)

// Hit represents a search result from vector store
type Hit struct {
	ID     string            `json:"id"`
	Score  float64           `json:"score"`
	Meta   map[string]string `json:"meta"`
	Vector []float32         `json:"vector,omitempty"`
}

// VectorStore defines the interface for vector storage and retrieval
type VectorStore interface {
	// Upsert stores or updates a vector with metadata
	Upsert(ctx context.Context, id string, vec []float32, meta map[string]string) error

	// Search finds the most similar vectors
	Search(ctx context.Context, vec []float32, topK int) ([]Hit, error)

	// Delete removes a vector by ID
	Delete(ctx context.Context, id string) error

	// Get retrieves a vector by ID
	Get(ctx context.Context, id string) ([]float32, map[string]string, error)

	// Count returns the total number of vectors
	Count(ctx context.Context) (int, error)

	// Clear removes all vectors
	Clear(ctx context.Context) error
}

// SearchOptions provides additional options for search
type SearchOptions struct {
	TopK          int               `json:"top_k"`
	Filter        map[string]string `json:"filter,omitempty"`
	MinScore      float64           `json:"min_score,omitempty"`
	IncludeVector bool              `json:"include_vector,omitempty"`
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		TopK:          10,
		MinScore:      0.0,
		IncludeVector: false,
	}
}

// VectorStoreConfig holds configuration for vector stores
type VectorStoreConfig struct {
	Collection string            `json:"collection"`
	Dimension  int               `json:"dimension"`
	Distance   string            `json:"distance"` // "cosine", "euclidean", "dot"
	Options    map[string]string `json:"options,omitempty"`
}

// DefaultConfig returns default vector store configuration
func DefaultConfig() *VectorStoreConfig {
	return &VectorStoreConfig{
		Collection: "artifacts",
		Dimension:  1536,
		Distance:   "cosine",
		Options:    make(map[string]string),
	}
}
