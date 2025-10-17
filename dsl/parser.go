package dsl

import (
	"fmt"
	"strconv"
)

// Parser represents an S-expression parser
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses the tokens into an AST
func (p *Parser) Parse() (Node, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for EOF
	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", p.current().Pos, p.current().Value)
	}

	return node, nil
}

// parseExpression parses a single expression
func (p *Parser) parseExpression() (Node, error) {
	token := p.current()

	switch token.Type {
	case TokenLParen:
		return p.parseList()
	case TokenLBracket:
		return p.parseArray()
	case TokenSymbol:
		return p.parseSymbol()
	case TokenNumber:
		return p.parseNumber()
	case TokenString:
		return p.parseString()
	case TokenBool:
		return p.parseBool()
	case TokenNull:
		return p.parseNull()
	case TokenEOF:
		return nil, fmt.Errorf("unexpected EOF")
	default:
		return nil, fmt.Errorf("unexpected token at position %d: %s", token.Pos, token.Value)
	}
}

// parseList parses a list expression
func (p *Parser) parseList() (Node, error) {
	// Consume opening parenthesis
	p.advance()

	if p.current().Type == TokenRParen {
		// Empty list
		p.advance()
		return &SeqNode{Statements: []Node{}}, nil
	}

	// Parse the first element (should be a symbol for the operation)
	if p.current().Type != TokenSymbol {
		return nil, fmt.Errorf("expected symbol after '(', got %s at position %d", p.current().Value, p.current().Pos)
	}

	op := p.current().Value
	p.advance()

	switch op {
	case "program":
		return p.parseProgram()
	case "seq":
		return p.parseSeq()
	case "let":
		return p.parseLet()
	case "return":
		return p.parseReturn()
	case "if":
		return p.parseIf()
	case "loop":
		return p.parseLoop()
	case "call":
		return p.parseCall()
	case "assert":
		return p.parseAssert()
	case "get":
		return p.parseGet()
	default:
		return nil, fmt.Errorf("unknown operation: %s at position %d", op, p.current().Pos)
	}
}

// parseProgram parses a program (top-level wrapper)
func (p *Parser) parseProgram() (Node, error) {
	var statements []Node

	for p.current().Type != TokenRParen && p.current().Type != TokenEOF {
		stmt, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after program, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis

	// If there's only one statement, return it directly
	if len(statements) == 1 {
		return statements[0], nil
	}

	return &SeqNode{Statements: statements}, nil
}

// parseSeq parses a sequence
func (p *Parser) parseSeq() (Node, error) {
	var statements []Node

	for p.current().Type != TokenRParen && p.current().Type != TokenEOF {
		stmt, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after seq, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &SeqNode{Statements: statements}, nil
}

// parseArray parses an array literal
func (p *Parser) parseArray() (Node, error) {
	// Consume opening bracket
	p.advance()

	var elements []Node

	for p.current().Type != TokenRBracket && p.current().Type != TokenEOF {
		element, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
	}

	if p.current().Type != TokenRBracket {
		return nil, fmt.Errorf("expected ']' after array, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing bracket
	return &ArrayNode{Elements: elements}, nil
}

// parseLet parses a let binding
func (p *Parser) parseLet() (Node, error) {
	// Check if this is an empty let expression
	if p.current().Type == TokenRParen {
		return nil, fmt.Errorf("let expression cannot be empty")
	}

	// Parse variable name
	if p.current().Type != TokenSymbol {
		return nil, fmt.Errorf("expected variable name after 'let', got %s at position %d", p.current().Value, p.current().Pos)
	}
	name := p.current().Value
	p.advance()

	// Check if we have more arguments
	if p.current().Type == TokenRParen {
		return nil, fmt.Errorf("let expression must have at least a value, got empty after variable name")
	}

	// Parse value
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check if we have a body
	var body Node
	if p.current().Type != TokenRParen {
		// Parse body
		body, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	} else {
		// If no body, create a simple return statement
		body = &ReturnNode{Value: &VarNode{Name: name}}
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after let, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &LetNode{Name: name, Value: value, Body: body}, nil
}

// parseReturn parses a return statement
func (p *Parser) parseReturn() (Node, error) {
	// Parse value
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after return, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &ReturnNode{Value: value}, nil
}

// parseIf parses a conditional
func (p *Parser) parseIf() (Node, error) {
	// Parse condition
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Parse then branch
	then, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Parse else branch
	else_, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after if, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &IfNode{Condition: condition, Then: then, Else: else_}, nil
}

// parseLoop parses a loop
func (p *Parser) parseLoop() (Node, error) {
	// Parse condition
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Parse body
	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after loop, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &LoopNode{Condition: condition, Body: body}, nil
}

// parseCall parses a function call
func (p *Parser) parseCall() (Node, error) {
	// Parse function name
	if p.current().Type != TokenSymbol {
		return nil, fmt.Errorf("expected function name after 'call', got %s at position %d", p.current().Value, p.current().Pos)
	}
	function := p.current().Value
	p.advance()

	var args []Node
	for p.current().Type != TokenRParen && p.current().Type != TokenEOF {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after call, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &CallNode{Function: function, Args: args}, nil
}

// parseAssert parses an assertion
func (p *Parser) parseAssert() (Node, error) {
	// Parse condition
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Parse message (should be a string)
	if p.current().Type != TokenString {
		return nil, fmt.Errorf("expected string message after assert, got %s at position %d", p.current().Value, p.current().Pos)
	}
	message := p.current().Value
	p.advance()

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after assert, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &AssertNode{Condition: condition, Message: message}, nil
}

// parseGet parses a get operation
func (p *Parser) parseGet() (Node, error) {
	// Parse object
	obj, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Parse key
	key, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after get, got %s at position %d", p.current().Value, p.current().Pos)
	}

	p.advance() // consume closing parenthesis
	return &CallNode{Function: "get", Args: []Node{obj, key}}, nil
}

// parseSymbol parses a symbol (variable reference)
func (p *Parser) parseSymbol() (Node, error) {
	token := p.current()
	p.advance()
	return &VarNode{Name: token.Value}, nil
}

// parseNumber parses a number literal
func (p *Parser) parseNumber() (Node, error) {
	token := p.current()
	p.advance()

	// Try to parse as float64 first
	if val, err := strconv.ParseFloat(token.Value, 64); err == nil {
		return &LitNode{Value: val}, nil
	}

	// Try to parse as int64
	if val, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
		return &LitNode{Value: val}, nil
	}

	return nil, fmt.Errorf("invalid number: %s at position %d", token.Value, token.Pos)
}

// parseString parses a string literal
func (p *Parser) parseString() (Node, error) {
	token := p.current()
	p.advance()
	return &LitNode{Value: token.Value}, nil
}

// parseBool parses a boolean literal
func (p *Parser) parseBool() (Node, error) {
	token := p.current()
	p.advance()
	return &LitNode{Value: token.Value == "true"}, nil
}

// parseNull parses a null literal
func (p *Parser) parseNull() (Node, error) {
	p.advance()
	return &LitNode{Value: nil}, nil
}

// current returns the current token
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Value: "", Pos: -1}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// ParseProgram parses an AF-DSL program from source code
func ParseProgram(source string) (Node, error) {
	tokens := Tokenize(source)
	parser := NewParser(tokens)
	return parser.Parse()
}
