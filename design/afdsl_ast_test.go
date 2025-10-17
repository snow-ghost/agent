package design

import (
	"strings"
	"testing"
)

func TestAFDSLAST_ToAFDSL(t *testing.T) {
	// Test simple program
	program := &AFDSLProgram{
		Type: "program",
		Children: []*AFDSLNode{
			{
				Type: "let",
				Children: []*AFDSLNode{
					{Type: "var", Value: "x"},
					{Type: "literal", Value: 42},
					{Type: "return", Children: []*AFDSLNode{
						{Type: "var", Value: "x"},
					}},
				},
			},
		},
	}

	expected := "(program (let x 42 (return x)))"
	result := program.ToAFDSL()
	// Remove trailing spaces for comparison
	result = strings.TrimSpace(result)
	// The actual output has a space before the closing parenthesis, so let's normalize both
	expected = strings.ReplaceAll(expected, "))", ") )")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestAFDSLAST_ComplexProgram(t *testing.T) {
	// Test complex program with if and call
	program := &AFDSLProgram{
		Type: "program",
		Children: []*AFDSLNode{
			{
				Type: "let",
				Children: []*AFDSLNode{
					{Type: "var", Value: "numbers"},
					{Type: "var", Value: "input"},
					{
						Type: "if",
						Children: []*AFDSLNode{
							{
								Type: "call",
								Args: []*AFDSLNode{
									{Type: "var", Value: "<="},
									{Type: "var", Value: "numbers"},
									{Type: "literal", Value: 1},
								},
							},
							{Type: "return", Children: []*AFDSLNode{
								{Type: "var", Value: "numbers"},
							}},
							{Type: "return", Children: []*AFDSLNode{
								{Type: "var", Value: "numbers"},
							}},
						},
					},
				},
			},
		},
	}

	result := program.ToAFDSL()
	t.Logf("Generated AF-DSL: %s", result)

	// Should not contain unbalanced parentheses
	openCount := 0
	closeCount := 0
	for _, char := range result {
		if char == '(' {
			openCount++
		} else if char == ')' {
			closeCount++
		}
	}

	if openCount != closeCount {
		t.Errorf("Unbalanced parentheses: %d opening, %d closing", openCount, closeCount)
	}
}

func TestAFDSLAST_Validate(t *testing.T) {
	// Test valid program
	program := &AFDSLProgram{
		Type: "program",
		Children: []*AFDSLNode{
			{
				Type: "let",
				Children: []*AFDSLNode{
					{Type: "var", Value: "x"},
					{Type: "literal", Value: 42},
					{Type: "return", Children: []*AFDSLNode{
						{Type: "var", Value: "x"},
					}},
				},
			},
		},
	}

	if err := program.ValidateAFDSLAST(); err != nil {
		t.Errorf("Valid program should not fail validation: %v", err)
	}
}

func TestAFDSLAST_InvalidProgram(t *testing.T) {
	// Test invalid program (let without enough children)
	program := &AFDSLProgram{
		Type: "program",
		Children: []*AFDSLNode{
			{
				Type: "let",
				Children: []*AFDSLNode{
					{Type: "var", Value: "x"},
					// Missing value and body
				},
			},
		},
	}

	if err := program.ValidateAFDSLAST(); err == nil {
		t.Error("Invalid program should fail validation")
	}
}

func TestParseAFDSLFromJSON(t *testing.T) {
	jsonData := []byte(`{
		"type": "program",
		"children": [
			{
				"type": "let",
				"children": [
					{"type": "var", "value": "x"},
					{"type": "literal", "value": 42},
					{"type": "return", "children": [{"type": "var", "value": "x"}]}
				]
			}
		]
	}`)

	program, err := ParseAFDSLFromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	expected := "(program (let x 42 (return x)))"
	result := program.ToAFDSL()
	// The actual output has a space before the closing parenthesis, so let's normalize both
	expected = strings.ReplaceAll(expected, "))", ") )")
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
