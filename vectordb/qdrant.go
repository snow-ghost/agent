package vectordb

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/qdrant/go-client"
	"github.com/qdrant/go-client/models"
)

// QdrantVectorStore implements a vector store using Qdrant
type QdrantVectorStore struct {
	config     *VectorStoreConfig
	collection string
	client     *client.Client
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
	clientConfig := client.Config{
		Host: url,
	}
	if apiKey != "" {
		clientConfig.APIKey = apiKey
	}

	qdrantClient, err := client.NewClient(&clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	store := &QdrantVectorStore{
		config:     config,
		collection: config.Collection,
		client:     qdrantClient,
	}

	// Ensure collection exists
	if err := store.ensureCollection(ctx.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist
func (q *QdrantVectorStore) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	collections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	collectionExists := false
	for _, collection := range collections {
		if collection.Name == q.collection {
			collectionExists = true
			break
		}
	}

	if !collectionExists {
		// Create collection
		distance := models.DistanceCosine
		switch q.config.Distance {
		case "euclidean":
			distance = models.DistanceEuclid
		case "dot":
			distance = models.DistanceDot
		}

		_, err = q.client.CreateCollection(ctx, &client.CreateCollection{
			CollectionName: q.collection,
			VectorsConfig: &models.VectorParams{
				Size:     uint64(q.config.Dimension),
				Distance: distance,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
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
	vec64 := make([]float64, len(vec))
	for i, v := range vec {
		vec64[i] = float64(v)
	}

	// Upsert point
	_, err := q.client.Upsert(ctx, &client.UpsertPoints{
		CollectionName: q.collection,
		Points: []models.PointStruct{
			{
				ID:      id,
				Vector:  vec64,
				Payload: payload,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

// Search finds the most similar vectors
func (q *QdrantVectorStore) Search(ctx context.Context, vec []float32, topK int) ([]Hit, error) {
	// Convert float32 to float64 for Qdrant
	vec64 := make([]float64, len(vec))
	for i, v := range vec {
		vec64[i] = float64(v)
	}

	// Search points
	searchResult, err := q.client.Search(ctx, &client.SearchPoints{
		CollectionName: q.collection,
		Vector:         vec64,
		Limit:          uint64(topK),
		WithPayload:    true,
		WithVector:     false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert results to Hit format
	hits := make([]Hit, len(searchResult))
	for i, result := range searchResult {
		// Convert payload to string map
		meta := make(map[string]string)
		for k, v := range result.Payload {
			if str, ok := v.(string); ok {
				meta[k] = str
			} else {
				meta[k] = fmt.Sprintf("%v", v)
			}
		}

		hits[i] = Hit{
			ID:    result.ID.String(),
			Score: result.Score,
			Meta:  meta,
		}
	}

	return hits, nil
}

// Delete removes a vector by ID
func (q *QdrantVectorStore) Delete(ctx context.Context, id string) error {
	_, err := q.client.Delete(ctx, &client.DeletePoints{
		CollectionName: q.collection,
		Points:         []models.PointID{models.PointID(id)},
	})
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

// Get retrieves a vector by ID
func (q *QdrantVectorStore) Get(ctx context.Context, id string) ([]float32, map[string]string, error) {
	// Retrieve point
	retrieveResult, err := q.client.Retrieve(ctx, &client.RetrievePoints{
		CollectionName: q.collection,
		Points:         []models.PointID{models.PointID(id)},
		WithPayload:    true,
		WithVector:     true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve vector: %w", err)
	}

	if len(retrieveResult) == 0 {
		return nil, nil, fmt.Errorf("vector with id %s not found", id)
	}

	point := retrieveResult[0]

	// Convert vector back to float32
	vec := make([]float32, len(point.Vector))
	for i, v := range point.Vector {
		vec[i] = float32(v)
	}

	// Convert payload to string map
	meta := make(map[string]string)
	for k, v := range point.Payload {
		if str, ok := v.(string); ok {
			meta[k] = str
		} else {
			meta[k] = fmt.Sprintf("%v", v)
		}
	}

	return vec, meta, nil
}

// Count returns the total number of vectors
func (q *QdrantVectorStore) Count(ctx context.Context) (int, error) {
	// Get collection info
	info, err := q.client.GetCollection(ctx, q.collection)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection info: %w", err)
	}

	return int(info.PointsCount), nil
}

// Clear removes all vectors
func (q *QdrantVectorStore) Clear(ctx context.Context) error {
	// Delete all points in the collection
	_, err := q.client.Delete(ctx, &client.DeletePoints{
		CollectionName: q.collection,
		Filter: &models.Filter{
			Must: []models.Condition{
				{
					IsNull: &models.IsNullCondition{
						IsNull: "id",
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to clear collection: %w", err)
	}

	return nil
}

// GetConfig returns the vector store configuration
func (q *QdrantVectorStore) GetConfig() *VectorStoreConfig {
	return q.config
}

// Close closes the Qdrant client
func (q *QdrantVectorStore) Close() error {
	if q.client != nil {
		return q.client.Close()
	}
	return nil
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
	qdrantPoints := make([]models.PointStruct, len(points))
	for i, point := range points {
		// Convert metadata
		payload := make(map[string]interface{})
		for k, v := range point.Meta {
			payload[k] = v
		}

		// Convert vector
		vec64 := make([]float64, len(point.Vector))
		for j, v := range point.Vector {
			vec64[j] = float64(v)
		}

		qdrantPoints[i] = models.PointStruct{
			ID:      point.ID,
			Vector:  vec64,
			Payload: payload,
		}
	}

	// Upsert in batches
	batchSize := 100
	for i := 0; i < len(qdrantPoints); i += batchSize {
		end := i + batchSize
		if end > len(qdrantPoints) {
			end = len(qdrantPoints)
		}

		_, err := q.client.Upsert(ctx, &client.UpsertPoints{
			CollectionName: q.collection,
			Points:         qdrantPoints[i:end],
		})
		if err != nil {
			return fmt.Errorf("failed to batch upsert vectors: %w", err)
		}
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
