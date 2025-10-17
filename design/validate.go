package design

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Default maximum code size in bytes (64KB)
const DefaultMaxCodeBytes = 64 * 1024

// Security limits
const (
	MaxIdentifierLength = 100
	MaxLiteralLength    = 1000
	MaxStringLength     = 10000
)

// Allowed function names for CALL operations
var allowedFunctions = map[string]bool{
	// Built-in functions
	"split":     true,
	"merge":     true,
	"sorted?":   true,
	"permutes?": true,
	"len":       true,
	"concat":    true,
	"map":       true,
	"filter":    true,
	// Array/list functions
	"length":     true,
	"slice":      true,
	"append":     true,
	"prepend":    true,
	"head":       true,
	"tail":       true,
	"merge-sort": true,
	// Arithmetic operators
	"div": true,
	"mod": true,
	"+":   true,
	"-":   true,
	"*":   true,
	"/":   true,
	"%":   true,
	// Comparison operators
	"<=": true,
	">=": true,
	"==": true,
	"!=": true,
	"<":  true,
	">":  true,
	"=":  true,
	// Control flow
	"if":     true,
	"let":    true,
	"return": true,
	"loop":   true,
	"assert": true,
}

// Validate performs comprehensive validation of a HypothesisDesign
func Validate(h HypothesisDesign) error {
	// Check status
	if h.Status != "ok" && h.Status != "cannot_solve" {
		return errors.New("status must be 'ok' or 'cannot_solve'")
	}

	// If status is "cannot_solve", reason should be provided
	if h.Status == "cannot_solve" && h.Reason == "" {
		return errors.New("reason is required when status is 'cannot_solve'")
	}

	// If status is "ok", validate the rest of the structure
	if h.Status == "ok" {
		if err := validateAlgorithm(h.Algorithm); err != nil {
			return fmt.Errorf("algorithm validation failed: %w", err)
		}

		if err := validateCode(h.Code); err != nil {
			return fmt.Errorf("code validation failed: %w", err)
		}

		if err := validateEvaluation(h.Evaluation); err != nil {
			return fmt.Errorf("evaluation validation failed: %w", err)
		}

		if err := validateTests(h.Tests); err != nil {
			return fmt.Errorf("tests validation failed: %w", err)
		}
	}

	return nil
}

func validateAlgorithm(alg struct {
	Name       string `json:"name"`
	Idea       string `json:"idea"`
	Complexity struct {
		Time  string `json:"time"`
		Space string `json:"space"`
	} `json:"complexity"`
}) error {
	if alg.Name == "" {
		return errors.New("algorithm name cannot be empty")
	}
	if alg.Idea == "" {
		return errors.New("algorithm idea cannot be empty")
	}
	if alg.Complexity.Time == "" {
		return errors.New("algorithm complexity time cannot be empty")
	}
	if alg.Complexity.Space == "" {
		return errors.New("algorithm complexity space cannot be empty")
	}
	return nil
}

func validateCode(code struct {
	Lang  string        `json:"lang"`          // "af-dsl" | "wasm"
	Entry string        `json:"entry"`         // "program"
	Src   string        `json:"src"`           // S-expr or base64 wasm
	AST   *AFDSLProgram `json:"ast,omitempty"` // JSON AST for af-dsl
}) error {
	// Check language
	if code.Lang != "af-dsl" && code.Lang != "wasm" {
		return errors.New("code lang must be 'af-dsl' or 'wasm'")
	}

	// Check entry point
	if code.Entry != "program" {
		return errors.New("code entry must be 'program'")
	}

	// Check source code
	if code.Src == "" {
		return errors.New("code src cannot be empty")
	}

	// Check maximum code size
	maxCodeBytes := getMaxCodeBytes()
	if len(code.Src) > maxCodeBytes {
		return fmt.Errorf("code src exceeds maximum size of %d bytes", maxCodeBytes)
	}

	// For wasm, validate base64 encoding
	if code.Lang == "wasm" {
		if err := validateWasmCode(code.Src); err != nil {
			return fmt.Errorf("wasm code validation failed: %w", err)
		}
	}

	// For af-dsl, validate security constraints
	if code.Lang == "af-dsl" {
		// If AST is provided, validate it instead of S-expression
		if code.AST != nil {
			if err := validateAFDSLAST(code.AST); err != nil {
				return fmt.Errorf("af-dsl AST validation failed: %w", err)
			}
		} else if code.Src != "" {
			// Fallback to S-expression validation
			if err := validateAFDSLSecurity(code.Src); err != nil {
				return fmt.Errorf("af-dsl security validation failed: %w", err)
			}
		} else {
			return fmt.Errorf("af-dsl code must have either AST or src")
		}
	}

	return nil
}

// validateAFDSLAST validates AF-DSL AST structure and security constraints
func validateAFDSLAST(ast *AFDSLProgram) error {
	// Validate AST structure
	if err := ast.ValidateAFDSLAST(); err != nil {
		return fmt.Errorf("AST structure validation failed: %w", err)
	}

	// Additional validation for common LLM errors
	if err := validateASTForCommonErrors(ast); err != nil {
		return fmt.Errorf("AST common errors validation failed: %w", err)
	}

	// Validate security constraints by converting to S-expression and validating
	afdslCode := ast.ToAFDSL()
	if err := validateAFDSLSecurity(afdslCode); err != nil {
		return fmt.Errorf("AST security validation failed: %w", err)
	}

	return nil
}

// validateASTForCommonErrors validates for common LLM generation errors
func validateASTForCommonErrors(ast *AFDSLProgram) error {
	// Handle case where LLM generates "value" instead of "children" for the root program node
	var children []*AFDSLNode
	if len(ast.Children) > 0 {
		children = ast.Children
	} else if ast.Value != nil {
		// Try to parse the value as a single node
		if valueMap, ok := ast.Value.(map[string]interface{}); ok {
			// Create a temporary node from the value
			tempNode := &AFDSLNode{}
			if valueBytes, err := json.Marshal(valueMap); err == nil {
				if err := json.Unmarshal(valueBytes, tempNode); err == nil {
					children = []*AFDSLNode{tempNode}
				}
			}
		}
	}

	// Check if the program has any children
	if len(children) == 0 {
		return fmt.Errorf("program must have at least one child node")
	}

	// Validate each child node
	for i, child := range children {
		if err := validateNodeForCommonErrors(child, fmt.Sprintf("program.children[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// validateNodeForCommonErrors validates a single node for common errors
func validateNodeForCommonErrors(node *AFDSLNode, path string) error {
	switch node.Type {
	case NodeTypeLet:
		// Let nodes must have at least 2 children (variable and value)
		if len(node.Children) < 2 {
			return fmt.Errorf("%s: let node must have at least 2 children (variable and value), got %d", path, len(node.Children))
		}

		// First child must be a variable
		if node.Children[0].Type != NodeTypeVar {
			return fmt.Errorf("%s: let node first child must be a variable, got %s", path, node.Children[0].Type)
		}

		// Variable must have a value
		if node.Children[0].Value == nil || node.Children[0].Value == "" {
			return fmt.Errorf("%s: let node variable must have a name", path)
		}

	case NodeTypeCall:
		// Call nodes must have at least one argument (function name)
		if len(node.Args) == 0 {
			return fmt.Errorf("%s: call node must have at least one argument (function name)", path)
		}

		// First argument must be a variable (function name)
		if node.Args[0].Type != NodeTypeVar {
			return fmt.Errorf("%s: call node first argument must be a variable (function name), got %s", path, node.Args[0].Type)
		}

	case NodeTypeIf:
		// If nodes must have at least 2 children (condition and then)
		if len(node.Children) < 2 {
			return fmt.Errorf("%s: if node must have at least 2 children (condition and then), got %d", path, len(node.Children))
		}

	case NodeTypeVar:
		// Variable nodes must have a value
		if node.Value == nil || node.Value == "" {
			return fmt.Errorf("%s: var node must have a value (variable name)", path)
		}

	case NodeTypeLiteral:
		// Literal nodes must have a value
		if node.Value == nil {
			return fmt.Errorf("%s: literal node must have a value", path)
		}
	}

	// Recursively validate children
	for i, child := range node.Children {
		if err := validateNodeForCommonErrors(child, fmt.Sprintf("%s.children[%d]", path, i)); err != nil {
			return err
		}
	}

	// Recursively validate args
	for i, arg := range node.Args {
		if err := validateNodeForCommonErrors(arg, fmt.Sprintf("%s.args[%d]", path, i)); err != nil {
			return err
		}
	}

	return nil
}

func validateWasmCode(src string) error {
	// Check if it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return errors.New("wasm code must be valid base64 encoded")
	}

	// Check for forbidden patterns (file paths, network calls, etc.)
	forbiddenPatterns := []string{
		"../", "./", "file://", "http://", "https://",
		"localhost", "127.0.0.1", "0.0.0.0",
	}

	lowerDecoded := strings.ToLower(string(decoded))
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(lowerDecoded, pattern) {
			return fmt.Errorf("wasm code contains forbidden pattern: %s", pattern)
		}
	}

	return nil
}

func validateEvaluation(eval struct {
	Metrics       []string `json:"metrics"`
	Fitness       string   `json:"fitness"`
	PassThreshold float64  `json:"pass_threshold"`
}) error {
	// Check pass threshold is in [0,1]
	if eval.PassThreshold < 0 || eval.PassThreshold > 1 {
		return errors.New("pass_threshold must be between 0 and 1")
	}

	// Check metrics includes "correctness"
	hasCorrectness := false
	for _, metric := range eval.Metrics {
		if metric == "correctness" {
			hasCorrectness = true
			break
		}
	}
	if !hasCorrectness {
		return errors.New("metrics must include 'correctness'")
	}

	// Check fitness formula is not empty
	if eval.Fitness == "" {
		return errors.New("fitness formula cannot be empty")
	}

	return nil
}

func validateTests(tests struct {
	Unit []struct {
		Name   string  `json:"name"`
		Input  string  `json:"input"`
		Oracle string  `json:"oracle"`
		Weight float64 `json:"weight"`
	} `json:"unit"`
	Property []struct {
		Name      string   `json:"name"`
		Generator string   `json:"generator"`
		Checks    []string `json:"checks"`
	} `json:"property"`
}) error {
	// Check unit tests have positive total weight
	totalWeight := 0.0
	for _, unit := range tests.Unit {
		if unit.Name == "" {
			return errors.New("unit test name cannot be empty")
		}
		if unit.Input == "" {
			return errors.New("unit test input cannot be empty")
		}
		if unit.Oracle == "" {
			return errors.New("unit test oracle cannot be empty")
		}
		if unit.Weight < 0 {
			return errors.New("unit test weight cannot be negative")
		}
		totalWeight += unit.Weight
	}

	if totalWeight <= 0 {
		return errors.New("unit tests must have total weight > 0")
	}

	// Check property tests
	for _, prop := range tests.Property {
		if prop.Name == "" {
			return errors.New("property test name cannot be empty")
		}
		if prop.Generator == "" {
			return errors.New("property test generator cannot be empty")
		}
		if len(prop.Checks) == 0 {
			return errors.New("property test must have at least one check")
		}
	}

	return nil
}

// getMaxCodeBytes returns the maximum allowed code size from environment variable or default
func getMaxCodeBytes() int {
	if maxBytesStr := os.Getenv("MAX_CODE_BYTES"); maxBytesStr != "" {
		if maxBytes, err := strconv.Atoi(maxBytesStr); err == nil && maxBytes > 0 {
			return maxBytes
		}
	}
	return DefaultMaxCodeBytes
}

// validateAFDSLSecurity performs security validation on AF-DSL code
func validateAFDSLSecurity(src string) error {
	// Check for valid UTF-8 encoding
	if !utf8.ValidString(src) {
		return errors.New("code contains invalid UTF-8 characters")
	}

	// Parse and validate the S-expression structure
	return validateSExpression(src)
}

// validateSExpression validates S-expression structure and security constraints
func validateSExpression(src string) error {
	// Tokenize the input
	tokens, err := TokenizeSExpression(src)
	if err != nil {
		return fmt.Errorf("failed to tokenize S-expression: %w", err)
	}

	// Validate tokens for security constraints
	for i, token := range tokens {
		if err := validateToken(token, i); err != nil {
			return fmt.Errorf("token validation failed at position %d: %w", i, err)
		}
	}

	// Check for balanced parentheses
	if err := validateBalancedParentheses(tokens); err != nil {
		return fmt.Errorf("parentheses validation failed: %w", err)
	}

	// Check for unknown function calls
	if err := validateFunctionCalls(tokens); err != nil {
		return fmt.Errorf("function call validation failed: %w", err)
	}

	return nil
}

// Token represents a token in the S-expression
type Token struct {
	Type  TokenType
	Value string
}

type TokenType int

const (
	TokenOpenParen TokenType = iota
	TokenCloseParen
	TokenSymbol
	TokenString
	TokenNumber
	TokenBoolean
)

// TokenizeSExpression tokenizes an S-expression string
func TokenizeSExpression(src string) ([]Token, error) {
	var tokens []Token
	runes := []rune(src)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Skip whitespace
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}

		switch r {
		case '(':
			tokens = append(tokens, Token{Type: TokenOpenParen, Value: "("})
		case ')':
			tokens = append(tokens, Token{Type: TokenCloseParen, Value: ")"})
		case '"':
			// String literal
			start := i
			i++ // Skip opening quote
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					i++ // Skip escaped character
				}
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unterminated string literal at position %d", start)
			}
			tokens = append(tokens, Token{Type: TokenString, Value: string(runes[start : i+1])})
		default:
			// Symbol, number, boolean, or operator
			start := i
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' && runes[i] != '\n' && runes[i] != '\r' && runes[i] != '(' && runes[i] != ')' {
				i++
			}
			i-- // Back up one position

			value := string(runes[start : i+1])

			// Determine token type
			if value == "true" || value == "false" {
				tokens = append(tokens, Token{Type: TokenBoolean, Value: value})
			} else if isNumber(value) {
				tokens = append(tokens, Token{Type: TokenNumber, Value: value})
			} else {
				tokens = append(tokens, Token{Type: TokenSymbol, Value: value})
			}
		}
	}

	return tokens, nil
}

// isNumber checks if a string represents a number
func isNumber(s string) bool {
	if s == "" {
		return false
	}

	// Check for integer
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	// Check for float
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	return false
}

// validateToken validates a single token for security constraints
func validateToken(token Token, position int) error {
	switch token.Type {
	case TokenString:
		// Check string length
		if len(token.Value) > MaxStringLength {
			return fmt.Errorf("string literal too long: %d characters (max %d)", len(token.Value), MaxStringLength)
		}

		// Validate string content (no null bytes, control characters)
		content := token.Value[1 : len(token.Value)-1] // Remove quotes
		for _, r := range content {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return fmt.Errorf("string contains invalid control character: %d", r)
			}
		}

	case TokenSymbol:
		// Check identifier length
		if len(token.Value) > MaxIdentifierLength {
			return fmt.Errorf("identifier too long: %d characters (max %d)", len(token.Value), MaxIdentifierLength)
		}

		// Check for valid identifier characters
		if !isValidIdentifier(token.Value) {
			return fmt.Errorf("invalid identifier: %s", token.Value)
		}

	case TokenNumber:
		// Check number length
		if len(token.Value) > MaxLiteralLength {
			return fmt.Errorf("number literal too long: %d characters (max %d)", len(token.Value), MaxLiteralLength)
		}
	}

	return nil
}

// isValidIdentifier checks if a string is a valid identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// Check for comparison operators (special case)
	comparisonOps := []string{"<=", ">=", "==", "!=", "<", ">", "=", "+", "-", "*", "/", "%"}
	for _, op := range comparisonOps {
		if s == op {
			return true
		}
	}

	// Must start with letter or allowed symbol
	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_' || first == '?' || first == '!') {
		return false
	}

	// Rest must be letters, digits, or allowed symbols
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '?' || c == '!' || c == '-') {
			return false
		}
	}

	return true
}

// validateBalancedParentheses checks if parentheses are balanced
func validateBalancedParentheses(tokens []Token) error {
	balance := 0
	for _, token := range tokens {
		switch token.Type {
		case TokenOpenParen:
			balance++
		case TokenCloseParen:
			balance--
			if balance < 0 {
				return errors.New("unbalanced parentheses: closing without opening")
			}
		}
	}

	if balance != 0 {
		return errors.New("unbalanced parentheses: unclosed expressions")
	}

	return nil
}

// validateFunctionCalls checks that all function calls use allowed functions
func validateFunctionCalls(tokens []Token) error {
	for i, token := range tokens {
		if token.Type == TokenSymbol && token.Value == "call" {
			// Find the function name (next symbol after "call")
			if i+2 < len(tokens) && tokens[i+1].Type == TokenOpenParen {
				// Look for the function name in the call expression
				for j := i + 2; j < len(tokens); j++ {
					if tokens[j].Type == TokenSymbol {
						funcName := tokens[j].Value
						if !allowedFunctions[funcName] {
							return fmt.Errorf("unknown function call: %s (not in allowed list)", funcName)
						}
						break
					}
					if tokens[j].Type == TokenCloseParen {
						break
					}
				}
			}
		}
	}

	return nil
}
