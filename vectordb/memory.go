package vectordb

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
)

// MemoryVectorStore implements an in-memory vector store using cosine similarity
type MemoryVectorStore struct {
	config   *VectorStoreConfig
	vectors  map[string][]float32
	metadata map[string]map[string]string
	mu       sync.RWMutex
}

// NewMemoryVectorStore creates a new in-memory vector store
func NewMemoryVectorStore(config *VectorStoreConfig) *MemoryVectorStore {
	if config == nil {
		config = DefaultConfig()
	}

	return &MemoryVectorStore{
		config:   config,
		vectors:  make(map[string][]float32),
		metadata: make(map[string]map[string]string),
	}
}

// Upsert stores or updates a vector with metadata
func (m *MemoryVectorStore) Upsert(ctx context.Context, id string, vec []float32, meta map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate vector dimension
	if len(vec) != m.config.Dimension {
		return fmt.Errorf("vector dimension %d does not match expected %d", len(vec), m.config.Dimension)
	}

	// Normalize vector for cosine similarity
	normalized := make([]float32, len(vec))
	copy(normalized, vec)
	m.normalize(normalized)

	// Store vector and metadata
	m.vectors[id] = normalized
	m.metadata[id] = make(map[string]string)
	for k, v := range meta {
		m.metadata[id][k] = v
	}

	return nil
}

// Search finds the most similar vectors using cosine similarity
func (m *MemoryVectorStore) Search(ctx context.Context, vec []float32, topK int) ([]Hit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Validate vector dimension
	if len(vec) != m.config.Dimension {
		return nil, fmt.Errorf("vector dimension %d does not match expected %d", len(vec), m.config.Dimension)
	}

	// Normalize query vector
	queryVec := make([]float32, len(vec))
	copy(queryVec, vec)
	m.normalize(queryVec)

	// Calculate similarities
	var hits []Hit
	for id, storedVec := range m.vectors {
		score := m.cosineSimilarity(queryVec, storedVec)

		// Create metadata copy
		meta := make(map[string]string)
		if storedMeta, exists := m.metadata[id]; exists {
			for k, v := range storedMeta {
				meta[k] = v
			}
		}

		hits = append(hits, Hit{
			ID:    id,
			Score: score,
			Meta:  meta,
		})
	}

	// Sort by score (descending)
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	// Return top K results
	if topK > 0 && topK < len(hits) {
		hits = hits[:topK]
	}

	return hits, nil
}

// Delete removes a vector by ID
func (m *MemoryVectorStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.vectors, id)
	delete(m.metadata, id)

	return nil
}

// Get retrieves a vector by ID
func (m *MemoryVectorStore) Get(ctx context.Context, id string) ([]float32, map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vec, exists := m.vectors[id]
	if !exists {
		return nil, nil, fmt.Errorf("vector with id %s not found", id)
	}

	// Create copy of vector
	vecCopy := make([]float32, len(vec))
	copy(vecCopy, vec)

	// Create copy of metadata
	meta := make(map[string]string)
	if storedMeta, exists := m.metadata[id]; exists {
		for k, v := range storedMeta {
			meta[k] = v
		}
	}

	return vecCopy, meta, nil
}

// Count returns the total number of vectors
func (m *MemoryVectorStore) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.vectors), nil
}

// Clear removes all vectors
func (m *MemoryVectorStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.vectors = make(map[string][]float32)
	m.metadata = make(map[string]map[string]string)

	return nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func (m *MemoryVectorStore) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// normalize normalizes a vector to unit length
func (m *MemoryVectorStore) normalize(vec []float32) {
	var norm float64
	for _, v := range vec {
		norm += float64(v * v)
	}

	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}
}

// GetConfig returns the vector store configuration
func (m *MemoryVectorStore) GetConfig() *VectorStoreConfig {
	return m.config
}

// SetDimension updates the expected vector dimension
func (m *MemoryVectorStore) SetDimension(dimension int) {
	m.config.Dimension = dimension
}

// GetStats returns statistics about the vector store
func (m *MemoryVectorStore) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_vectors": len(m.vectors),
		"dimension":     m.config.Dimension,
		"distance":      m.config.Distance,
		"collection":    m.config.Collection,
	}
}
