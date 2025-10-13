package client

import (
	"testing"
)

func TestExtractFirstJSONObject_CleanJSON(t *testing.T) {
	input := `{"status": "ok", "algorithm": {"name": "test"}}`
	expected := `{"status": "ok", "algorithm": {"name": "test"}}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

func TestExtractFirstJSONObject_WithMarkdown(t *testing.T) {
	input := `Here's the JSON response:

` + "```json" + `
{"status": "ok", "algorithm": {"name": "test"}}
` + "```" + `

This is the end.`
	expected := `{"status": "ok", "algorithm": {"name": "test"}}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

func TestExtractFirstJSONObject_WithTextBeforeAfter(t *testing.T) {
	input := `Some text before the JSON {"status": "ok", "algorithm": {"name": "test"}} and some text after`
	expected := `{"status": "ok", "algorithm": {"name": "test"}}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

func TestExtractFirstJSONObject_NestedJSON(t *testing.T) {
	input := `Text before {"outer": {"inner": {"deep": "value"}}} text after`
	expected := `{"outer": {"inner": {"deep": "value"}}}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

func TestExtractFirstJSONObject_WithEscapedStrings(t *testing.T) {
	input := `{"message": "Hello \"world\" with \\n newline"}`
	expected := `{"message": "Hello \"world\" with \\n newline"}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}

func TestExtractFirstJSONObject_NoJSON(t *testing.T) {
	input := `This is just plain text with no JSON at all`

	_, err := ExtractFirstJSONObject([]byte(input))
	if err == nil {
		t.Error("Expected error for input with no JSON, got nil")
	}
}

func TestExtractFirstJSONObject_UnbalancedBrackets(t *testing.T) {
	input := `{"status": "ok", "algorithm": {"name": "test"}`

	_, err := ExtractFirstJSONObject([]byte(input))
	if err == nil {
		t.Error("Expected error for unbalanced brackets, got nil")
	}
}

func TestExtractFirstJSONObject_InvalidJSON(t *testing.T) {
	input := `{"status": "ok", "algorithm": {"name": "test",}}`

	_, err := ExtractFirstJSONObject([]byte(input))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestExtractFirstJSONObject_MultipleJSONObjects(t *testing.T) {
	input := `First: {"first": "object"} Second: {"second": "object"}`
	expected := `{"first": "object"}`

	result, err := ExtractFirstJSONObject([]byte(input))
	if err != nil {
		t.Fatalf("ExtractFirstJSONObject failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result))
	}
}
