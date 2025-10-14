package dsl

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple expression",
			input: "(let x 5)",
			expected: []Token{
				{Type: TokenLParen, Value: "(", Pos: 0},
				{Type: TokenSymbol, Value: "let", Pos: 1},
				{Type: TokenSymbol, Value: "x", Pos: 5},
				{Type: TokenNumber, Value: "5", Pos: 7},
				{Type: TokenRParen, Value: ")", Pos: 8},
				{Type: TokenEOF, Value: "", Pos: 9},
			},
		},
		{
			name:  "string literal",
			input: `(let msg "hello world")`,
			expected: []Token{
				{Type: TokenLParen, Value: "(", Pos: 0},
				{Type: TokenSymbol, Value: "let", Pos: 1},
				{Type: TokenSymbol, Value: "msg", Pos: 5},
				{Type: TokenString, Value: "hello world", Pos: 9},
				{Type: TokenRParen, Value: ")", Pos: 22},
				{Type: TokenEOF, Value: "", Pos: 23},
			},
		},
		{
			name:  "boolean literals",
			input: "(if true false null)",
			expected: []Token{
				{Type: TokenLParen, Value: "(", Pos: 0},
				{Type: TokenSymbol, Value: "if", Pos: 1},
				{Type: TokenBool, Value: "true", Pos: 4},
				{Type: TokenBool, Value: "false", Pos: 9},
				{Type: TokenNull, Value: "null", Pos: 15},
				{Type: TokenRParen, Value: ")", Pos: 19},
				{Type: TokenEOF, Value: "", Pos: 20},
			},
		},
		{
			name:  "nested expressions",
			input: "(let x (call add 1 2))",
			expected: []Token{
				{Type: TokenLParen, Value: "(", Pos: 0},
				{Type: TokenSymbol, Value: "let", Pos: 1},
				{Type: TokenSymbol, Value: "x", Pos: 5},
				{Type: TokenLParen, Value: "(", Pos: 7},
				{Type: TokenSymbol, Value: "call", Pos: 8},
				{Type: TokenSymbol, Value: "add", Pos: 13},
				{Type: TokenNumber, Value: "1", Pos: 17},
				{Type: TokenNumber, Value: "2", Pos: 19},
				{Type: TokenRParen, Value: ")", Pos: 20},
				{Type: TokenRParen, Value: ")", Pos: 21},
				{Type: TokenEOF, Value: "", Pos: 22},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := Tokenize(tt.input)

			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				return
			}

			for i, token := range tokens {
				expected := tt.expected[i]
				if token.Type != expected.Type {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.Type, token.Type)
				}
				if token.Value != expected.Value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.Value, token.Value)
				}
			}
		})
	}
}

func TestParseProgram(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected NodeType
	}{
		{
			name:     "simple let",
			input:    "(let x 5 (return x))",
			expected: NodeTypeLet,
		},
		{
			name:     "return statement",
			input:    "(return 42)",
			expected: NodeTypeReturn,
		},
		{
			name:     "if statement",
			input:    "(if true 1 0)",
			expected: NodeTypeIf,
		},
		{
			name:     "function call",
			input:    "(call add 1 2)",
			expected: NodeTypeCall,
		},
		{
			name:     "sequence",
			input:    "(seq (let x 1 (return x)) (let y 2 (return y)))",
			expected: NodeTypeSeq,
		},
		{
			name:     "assertion",
			input:    `(assert true "test")`,
			expected: NodeTypeAssert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseProgram(tt.input)
			if err != nil {
				t.Errorf("ParseProgram failed: %v", err)
				return
			}

			if node.Type() != tt.expected {
				t.Errorf("Expected node type %s, got %s", tt.expected, node.Type())
			}
		})
	}
}

func TestParseProgram_Errors(t *testing.T) {
	tests := []string{
		"",            // empty input
		"(",           // unmatched parenthesis
		")",           // unmatched parenthesis
		"(let)",       // incomplete let
		"(unknown x)", // unknown operation
		"(let 123 x)", // invalid variable name
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseProgram(input)
			if err == nil {
				t.Errorf("Expected error for input %q, got nil", input)
			}
		})
	}
}
