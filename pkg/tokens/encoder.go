package tokens

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// Encoder represents a token encoder for a specific model
type Encoder interface {
	Encode(text string) ([]int, error)
	Decode(tokens []int) (string, error)
	Count(text string) (int, error)
}

// TiktokenEncoder implements Encoder using tiktoken-go
type TiktokenEncoder struct {
	encoding *tiktoken.Tiktoken
}

// NewTiktokenEncoder creates a new tiktoken encoder
func NewTiktokenEncoder(encodingName string) (*TiktokenEncoder, error) {
	encoding, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding %s: %w", encodingName, err)
	}

	return &TiktokenEncoder{
		encoding: encoding,
	}, nil
}

// Encode converts text to tokens
func (e *TiktokenEncoder) Encode(text string) ([]int, error) {
	return e.encoding.Encode(text, nil, nil), nil
}

// Decode converts tokens to text
func (e *TiktokenEncoder) Decode(tokens []int) (string, error) {
	return e.encoding.Decode(tokens), nil
}

// Count returns the number of tokens in text
func (e *TiktokenEncoder) Count(text string) (int, error) {
	tokens := e.encoding.Encode(text, nil, nil)
	return len(tokens), nil
}

// MockEncoder implements Encoder with simple character-based counting
type MockEncoder struct{}

// NewMockEncoder creates a new mock encoder
func NewMockEncoder() *MockEncoder {
	return &MockEncoder{}
}

// Encode converts text to mock tokens (character-based)
func (e *MockEncoder) Encode(text string) ([]int, error) {
	// Simple character-based tokenization - same logic as Count
	count := len(text) / 4
	if count < 1 && len(text) > 0 {
		count = 1
	}

	tokens := make([]int, count)
	for i := 0; i < count; i++ {
		tokens[i] = i
	}
	return tokens, nil
}

// Decode converts mock tokens to text (not implemented)
func (e *MockEncoder) Decode(tokens []int) (string, error) {
	return "", fmt.Errorf("mock decoder not implemented")
}

// Count returns the number of tokens in text (character-based)
func (e *MockEncoder) Count(text string) (int, error) {
	// Simple estimation: ~4 characters per token
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count, nil
}

// EncoderRegistry manages model-to-encoder mappings
type EncoderRegistry struct {
	encoders map[string]Encoder
	fallback Encoder
}

// NewEncoderRegistry creates a new encoder registry
func NewEncoderRegistry() *EncoderRegistry {
	return &EncoderRegistry{
		encoders: make(map[string]Encoder),
		fallback: NewMockEncoder(),
	}
}

// RegisterEncoder registers an encoder for a model
func (r *EncoderRegistry) RegisterEncoder(modelID string, encoder Encoder) {
	r.encoders[modelID] = encoder
}

// GetEncoder returns the encoder for a model, or fallback if not found
func (r *EncoderRegistry) GetEncoder(modelID string) Encoder {
	if encoder, exists := r.encoders[modelID]; exists {
		return encoder
	}
	return r.fallback
}

// CountTokens counts tokens in text using the appropriate encoder
func (r *EncoderRegistry) CountTokens(modelID, text string) (int, error) {
	encoder := r.GetEncoder(modelID)
	return encoder.Count(text)
}

// CountTokensInMessages counts tokens in a list of messages
func (r *EncoderRegistry) CountTokensInMessages(modelID string, messages []string) (int, error) {
	total := 0
	for _, message := range messages {
		count, err := r.CountTokens(modelID, message)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

// GetDefaultRegistry returns a registry with common model encoders
func GetDefaultRegistry() *EncoderRegistry {
	registry := NewEncoderRegistry()

	// Register common OpenAI models
	openaiModels := []string{
		"gpt-4", "gpt-4-turbo", "gpt-4o", "gpt-4o-mini",
		"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		"text-embedding-3-small", "text-embedding-3-large",
		"text-embedding-ada-002",
	}

	for _, model := range openaiModels {
		if encoder, err := NewTiktokenEncoder("cl100k_base"); err == nil {
			registry.RegisterEncoder(model, encoder)
		}
	}

	// Register Anthropic models (use cl100k_base as approximation)
	anthropicModels := []string{
		"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022",
		"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307",
	}

	for _, model := range anthropicModels {
		if encoder, err := NewTiktokenEncoder("cl100k_base"); err == nil {
			registry.RegisterEncoder(model, encoder)
		}
	}

	// Register local models (use mock encoder)
	localModels := []string{
		"llama3.2", "codellama", "mistral", "mixtral",
	}

	for _, model := range localModels {
		registry.RegisterEncoder(model, NewMockEncoder())
	}

	return registry
}
