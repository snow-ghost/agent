package vectordb

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/qdrant/go-client/qdrant"
	workermetrics "github.com/snow-ghost/agent/worker/metrics"
)

// QdrantVectorStore implements a vector store using Qdrant
type QdrantVectorStore struct {
	config     *VectorStoreConfig
	collection string
	client     *qdrant.Client
}

// NewQdrantVectorStore creates a new Qdrant vector store
func NewQdrantVectorStore(config *VectorStoreConfig) (*QdrantVectorStore, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Get Qdrant configuration from environment
	url := getEnv("QDRANT_URL", "localhost:6333")
	apiKey := getEnv("QDRANT_API_KEY", "")

	// Create Qdrant client
	clientConfig := &qdrant.Config{
		Host: url,
	}
	if apiKey != "" {
		clientConfig.APIKey = apiKey
	}

	qdrantClient, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	store := &QdrantVectorStore{
		config:     config,
		collection: config.Collection,
		client:     qdrantClient,
	}

	// Ensure collection exists
	if err := store.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist
func (q *QdrantVectorStore) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	collections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}

	// Check if our collection exists
	for _, collection := range collections {
		if collection.Name == q.collection {
			return nil // Collection already exists
		}
	}

	// Create collection
	var distance qdrant.Distance
	switch q.config.Distance {
	case "cosine":
		distance = qdrant.Distance_Cosine
	case "euclidean":
		distance = qdrant.Distance_Euclidean
	case "dot":
		distance = qdrant.Distance_Dot
	default:
		distance = qdrant.Distance_Cosine // Default to cosine
	}

	_, err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(q.config.Dimension),
			Distance: distance,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// Upsert stores or updates a vector with metadata
func (q *QdrantVectorStore) Upsert(ctx context.Context, id string, vec []float32, meta map[string]string) error {
	// Convert metadata to Qdrant format
	payload := make(map[string]interface{})
	for k, v := range meta {
		payload[k] = v
	}

	// Convert float32 to float64 for Qdrant
	vector := make([]float64, len(vec))
	for i, v := range vec {
		vector[i] = float64(v)
	}

	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewID(id),
			Vectors: qdrant.NewVectors(vector...),
			Payload: qdrant.NewValueMap(payload),
		},
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collection,
		Points:         points,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

// Search finds the most similar vectors
func (q *QdrantVectorStore) Search(ctx context.Context, vec []float32, topK int) ([]Hit, error) {
	start := time.Now()

	// Convert float32 to float64 for Qdrant
	vector := make([]float64, len(vec))
	for i, v := range vec {
		vector[i] = float64(v)
	}

	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          uint64(topK),
		WithPayload:    true,
		WithVector:     false,
	})

	if err != nil {
		duration := time.Since(start).Seconds()
		workermetrics.ObserveRAGSearch(ctx, "qdrant", duration, 0)
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert results to Hit format
	hits := make([]Hit, len(searchResult))
	for i, result := range searchResult {
		// Convert payload back to string map
		meta := make(map[string]string)
		if result.Payload != nil {
			for k, v := range result.Payload.GetMapValue().GetFields() {
				if str, ok := v.GetStringValue(); ok {
					meta[k] = str
				}
			}
		}

		hits[i] = Hit{
			ID:     result.Id.GetUuid(),
			Score:  result.Score,
			Meta:   meta,
			Vector: nil, // Don't include vector in results by default
		}
	}

	duration := time.Since(start).Seconds()
	workermetrics.ObserveRAGSearch(ctx, "qdrant", duration, len(hits))

	return hits, nil
}

// Delete removes a vector by ID
func (q *QdrantVectorStore) Delete(ctx context.Context, id string) error {
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: q.collection,
		Points:         qdrant.NewPointsSelector(qdrant.NewID(id)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}
	return nil
}

// Get retrieves a vector by ID
func (q *QdrantVectorStore) Get(ctx context.Context, id string) ([]float32, map[string]string, error) {
	points, err := q.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: q.collection,
		Ids:            []*qdrant.PointId{qdrant.NewID(id)},
		WithPayload:    true,
		WithVector:     true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve vector: %w", err)
	}

	if len(points) == 0 {
		return nil, nil, fmt.Errorf("vector not found: %s", id)
	}

	point := points[0]

	// Convert vector back to float32
	var vector []float32
	if point.Vectors != nil {
		vector = make([]float32, len(point.Vectors.Vector))
		for i, v := range point.Vectors.Vector {
			vector[i] = float32(v)
		}
	}

	// Convert payload to string map
	meta := make(map[string]string)
	if point.Payload != nil {
		for k, v := range point.Payload.GetMapValue().GetFields() {
			if str, ok := v.GetStringValue(); ok {
				meta[k] = str
			}
		}
	}

	return vector, meta, nil
}

// Count returns the total number of vectors
func (q *QdrantVectorStore) Count(ctx context.Context) (int, error) {
	count, err := q.client.Count(ctx, &qdrant.CountPoints{
		CollectionName: q.collection,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}
	return int(count), nil
}

// Clear removes all vectors
func (q *QdrantVectorStore) Clear(ctx context.Context) error {
	// For now, we'll delete the collection and recreate it
	// This is simpler than trying to delete all points
	_, err := q.client.DeleteCollection(ctx, q.collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Recreate the collection
	return q.ensureCollection(ctx)
}

// GetConfig returns the vector store configuration
func (q *QdrantVectorStore) GetConfig() *VectorStoreConfig {
	return q.config
}

// Close closes the Qdrant client
func (q *QdrantVectorStore) Close() error {
	return q.client.Close()
}

// Health checks if the Qdrant service is available
func (q *QdrantVectorStore) Health(ctx context.Context) error {
	_, err := q.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("Qdrant health check failed: %w", err)
	}
	return nil
}

// BatchUpsert performs batch upsert for better performance
func (q *QdrantVectorStore) BatchUpsert(ctx context.Context, points []BatchPoint) error {
	if len(points) == 0 {
		return nil
	}

	// Convert to Qdrant format
	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, point := range points {
		// Convert metadata
		payload := make(map[string]interface{})
		for k, v := range point.Meta {
			payload[k] = v
		}

		// Convert vector
		vector := make([]float64, len(point.Vector))
		for j, v := range point.Vector {
			vector[j] = float64(v)
		}

		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(point.ID),
			Vectors: qdrant.NewVectors(vector...),
			Payload: qdrant.NewValueMap(payload),
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collection,
		Points:         qdrantPoints,
	})
	if err != nil {
		return fmt.Errorf("failed to batch upsert vectors: %w", err)
	}

	return nil
}

// BatchPoint represents a point for batch operations
type BatchPoint struct {
	ID     string
	Vector []float32
	Meta   map[string]string
}

// NewQdrantVectorStoreFromEnv creates a Qdrant vector store using environment variables
func NewQdrantVectorStoreFromEnv() (*QdrantVectorStore, error) {
	config := &VectorStoreConfig{
		Collection: getEnv("QDRANT_COLLECTION", "artifacts"),
		Dimension:  getEnvInt("QDRANT_DIMENSION", 1536),
		Distance:   getEnv("QDRANT_DISTANCE", "cosine"),
		Options:    make(map[string]string),
	}

	return NewQdrantVectorStore(config)
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
