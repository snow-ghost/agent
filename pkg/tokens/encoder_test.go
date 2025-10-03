package tokens

import (
	"testing"
)

func TestMockEncoder_Count(t *testing.T) {
	encoder := NewMockEncoder()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 1, // minimum 1 token
		},
		{
			name:     "short text",
			text:     "Hello",
			expected: 1, // 5 chars / 4 = 1
		},
		{
			name:     "medium text",
			text:     "This is a test message",
			expected: 5, // 22 chars / 4 = 5
		},
		{
			name:     "long text",
			text:     "This is a very long text that should produce multiple tokens when counted",
			expected: 18, // 70 chars / 4 = 17.5, rounded to 18
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := encoder.Count(tt.text)
			if err != nil {
				t.Fatalf("Count() error = %v", err)
			}
			if count != tt.expected {
				t.Errorf("Count() = %v, want %v", count, tt.expected)
			}
		})
	}
}

func TestMockEncoder_Encode(t *testing.T) {
	encoder := NewMockEncoder()

	text := "Hello world"
	tokens, err := encoder.Encode(text)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Should have some tokens
	if len(tokens) == 0 {
		t.Error("Encode() returned empty tokens")
	}

	// Should be consistent with Count
	count, err := encoder.Count(text)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if len(tokens) != count {
		t.Errorf("Encode() returned %d tokens, Count() returned %d", len(tokens), count)
	}
}

func TestMockEncoder_Decode(t *testing.T) {
	encoder := NewMockEncoder()

	_, err := encoder.Decode([]int{1, 2, 3})
	if err == nil {
		t.Error("Decode() expected error for mock encoder")
	}
}

func TestTiktokenEncoder_Count(t *testing.T) {
	encoder, err := NewTiktokenEncoder("cl100k_base")
	if err != nil {
		t.Fatalf("NewTiktokenEncoder() error = %v", err)
	}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "simple text",
			text:     "Hello world",
			expected: 2, // "Hello" and " world"
		},
		{
			name:     "longer text",
			text:     "This is a test message with multiple words",
			expected: 8, // Tokenized by tiktoken
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := encoder.Count(tt.text)
			if err != nil {
				t.Fatalf("Count() error = %v", err)
			}
			if count != tt.expected {
				t.Errorf("Count() = %v, want %v", count, tt.expected)
			}
		})
	}
}

func TestTiktokenEncoder_EncodeDecode(t *testing.T) {
	encoder, err := NewTiktokenEncoder("cl100k_base")
	if err != nil {
		t.Fatalf("NewTiktokenEncoder() error = %v", err)
	}

	text := "Hello world, this is a test!"
	tokens, err := encoder.Encode(text)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := encoder.Decode(tokens)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if decoded != text {
		t.Errorf("Decode() = %v, want %v", decoded, text)
	}
}

func TestEncoderRegistry_GetEncoder(t *testing.T) {
	registry := NewEncoderRegistry()

	// Test fallback for unknown model
	encoder := registry.GetEncoder("unknown-model")
	if encoder == nil {
		t.Error("GetEncoder() returned nil for unknown model")
	}

	// Test registered model
	mockEncoder := NewMockEncoder()
	registry.RegisterEncoder("test-model", mockEncoder)

	retrievedEncoder := registry.GetEncoder("test-model")
	if retrievedEncoder != mockEncoder {
		t.Error("GetEncoder() returned wrong encoder for registered model")
	}
}

func TestEncoderRegistry_CountTokens(t *testing.T) {
	registry := NewEncoderRegistry()

	// Test with fallback encoder
	count, err := registry.CountTokens("unknown-model", "Hello world")
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	if count == 0 {
		t.Error("CountTokens() returned 0 for fallback encoder")
	}

	// Test with registered encoder
	mockEncoder := NewMockEncoder()
	registry.RegisterEncoder("test-model", mockEncoder)

	count, err = registry.CountTokens("test-model", "Hello world")
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	expectedCount, _ := mockEncoder.Count("Hello world")
	if count != expectedCount {
		t.Errorf("CountTokens() = %v, want %v", count, expectedCount)
	}
}

func TestEncoderRegistry_CountTokensInMessages(t *testing.T) {
	registry := NewEncoderRegistry()

	messages := []string{"Hello", "world", "test"}
	count, err := registry.CountTokensInMessages("unknown-model", messages)
	if err != nil {
		t.Fatalf("CountTokensInMessages() error = %v", err)
	}

	// Should be sum of individual counts
	expectedCount := 0
	for _, msg := range messages {
		msgCount, _ := registry.CountTokens("unknown-model", msg)
		expectedCount += msgCount
	}

	if count != expectedCount {
		t.Errorf("CountTokensInMessages() = %v, want %v", count, expectedCount)
	}
}

func TestGetDefaultRegistry(t *testing.T) {
	registry := GetDefaultRegistry()

	if registry == nil {
		t.Fatal("GetDefaultRegistry() returned nil")
	}

	// Test that common models are registered
	testModels := []string{
		"gpt-4o-mini",
		"claude-3-5-sonnet-20241022",
		"llama3.2",
	}

	for _, model := range testModels {
		encoder := registry.GetEncoder(model)
		if encoder == nil {
			t.Errorf("GetDefaultRegistry() missing encoder for model %s", model)
		}

		// Test that encoder works
		count, err := registry.CountTokens(model, "test")
		if err != nil {
			t.Errorf("GetDefaultRegistry() encoder for %s failed: %v", model, err)
		}
		if count == 0 {
			t.Errorf("GetDefaultRegistry() encoder for %s returned 0 tokens", model)
		}
	}
}
