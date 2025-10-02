package embeddings

import (
	"context"
	"math"
	"strings"
	"unicode"
)

// MockEmbedder implements a simple TF-IDF based embedder for testing
type MockEmbedder struct {
	config     *EmbeddingConfig
	vocabulary map[string]int
	docCounts  map[string]int
	totalDocs  int
}

// NewMockEmbedder creates a new mock embedder
func NewMockEmbedder(config *EmbeddingConfig) *MockEmbedder {
	if config == nil {
		config = DefaultConfig()
	}

	return &MockEmbedder{
		config:     config,
		vocabulary: make(map[string]int),
		docCounts:  make(map[string]int),
		totalDocs:  0,
	}
}

// EmbedText converts text to a TF-IDF vector
func (m *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Tokenize text
	tokens := m.tokenize(text)

	// Count term frequencies
	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	// Create vocabulary if not exists
	for token := range tf {
		if _, exists := m.vocabulary[token]; !exists {
			m.vocabulary[token] = len(m.vocabulary)
		}
	}

	// Calculate TF-IDF vector with fixed dimension
	dimension := m.config.Dimension
	vector := make([]float32, dimension)

	for token, freq := range tf {
		if idx, exists := m.vocabulary[token]; exists && idx < dimension {
			// Calculate TF
			tfScore := 1.0 + math.Log(float64(freq))

			// Calculate IDF (simplified)
			idfScore := 1.0
			if m.totalDocs > 0 {
				docFreq := m.docCounts[token]
				if docFreq > 0 {
					idfScore = math.Log(float64(m.totalDocs) / float64(docFreq))
				}
			}

			// TF-IDF score
			tfidf := tfScore * idfScore
			vector[idx] = float32(tfidf)
		}
	}

	// Normalize vector
	m.normalize(vector)

	return vector, nil
}

// AddDocument adds a document to the corpus for IDF calculation
func (m *MockEmbedder) AddDocument(text string) {
	tokens := m.tokenize(text)
	uniqueTokens := make(map[string]bool)

	for _, token := range tokens {
		uniqueTokens[token] = true
	}

	for token := range uniqueTokens {
		m.docCounts[token]++
	}

	m.totalDocs++
}

// tokenize splits text into tokens
func (m *MockEmbedder) tokenize(text string) []string {
	// Convert to lowercase and split by whitespace/punctuation
	text = strings.ToLower(text)

	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	// Filter out very short tokens
	var filtered []string
	for _, token := range tokens {
		if len(token) >= 2 {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// normalize normalizes a vector to unit length
func (m *MockEmbedder) normalize(vector []float32) {
	var norm float64
	for _, v := range vector {
		norm += float64(v * v)
	}

	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vector {
			vector[i] = float32(float64(vector[i]) / norm)
		}
	}
}

// GetVocabulary returns the current vocabulary
func (m *MockEmbedder) GetVocabulary() map[string]int {
	return m.vocabulary
}

// GetDocumentCounts returns document frequency counts
func (m *MockEmbedder) GetDocumentCounts() map[string]int {
	return m.docCounts
}

// GetTotalDocuments returns the total number of documents processed
func (m *MockEmbedder) GetTotalDocuments() int {
	return m.totalDocs
}

// Reset clears the vocabulary and document counts
func (m *MockEmbedder) Reset() {
	m.vocabulary = make(map[string]int)
	m.docCounts = make(map[string]int)
	m.totalDocs = 0
}

// MockEmbedderFactory creates a mock embedder with pre-trained vocabulary
type MockEmbedderFactory struct {
	config *EmbeddingConfig
}

// NewMockEmbedderFactory creates a new factory
func NewMockEmbedderFactory(config *EmbeddingConfig) *MockEmbedderFactory {
	if config == nil {
		config = DefaultConfig()
	}
	return &MockEmbedderFactory{config: config}
}

// CreateEmbedder creates a new mock embedder
func (f *MockEmbedderFactory) CreateEmbedder() *MockEmbedder {
	return NewMockEmbedder(f.config)
}

// PreTrain trains the embedder on a corpus of texts
func (f *MockEmbedderFactory) PreTrain(texts []string) *MockEmbedder {
	embedder := NewMockEmbedder(f.config)

	// Add all texts to the corpus
	for _, text := range texts {
		embedder.AddDocument(text)
	}

	return embedder
}

// CreateEmbedderWithCorpus creates an embedder trained on common programming terms
func (f *MockEmbedderFactory) CreateEmbedderWithCorpus() *MockEmbedder {
	// Common programming and algorithm terms
	corpus := []string{
		"sort integers stable algorithm",
		"parse json data structure",
		"reverse string text manipulation",
		"search binary tree data structure",
		"filter array elements",
		"transform data mapping",
		"validate input parameters",
		"optimize performance algorithm",
		"compress data storage",
		"encrypt security cryptography",
		"hash function checksum",
		"graph traversal algorithm",
		"dynamic programming optimization",
		"recursive function call",
		"iterative loop processing",
		"database query sql",
		"api rest http request",
		"authentication security login",
		"authorization permission access",
		"logging debug information",
		"error handling exception",
		"testing unit integration",
		"deployment production release",
		"monitoring metrics performance",
		"scaling horizontal vertical",
	}

	return f.PreTrain(corpus)
}
