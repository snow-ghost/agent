package dsl

import (
	"fmt"
	"strings"
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenLParen
	TokenRParen
	TokenSymbol
	TokenNumber
	TokenString
	TokenBool
	TokenNull
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenSymbol:
		return "SYMBOL"
	case TokenNumber:
		return "NUMBER"
	case TokenString:
		return "STRING"
	case TokenBool:
		return "BOOL"
	case TokenNull:
		return "NULL"
	default:
		return "UNKNOWN"
	}
}

// Node represents an AST node
type Node interface {
	String() string
	Type() NodeType
}

// NodeType represents the type of an AST node
type NodeType int

const (
	NodeTypeSeq NodeType = iota
	NodeTypeLet
	NodeTypeReturn
	NodeTypeIf
	NodeTypeLoop
	NodeTypeCall
	NodeTypeAssert
	NodeTypeLit
	NodeTypeVar
)

func (nt NodeType) String() string {
	switch nt {
	case NodeTypeSeq:
		return "SEQ"
	case NodeTypeLet:
		return "LET"
	case NodeTypeReturn:
		return "RETURN"
	case NodeTypeIf:
		return "IF"
	case NodeTypeLoop:
		return "LOOP"
	case NodeTypeCall:
		return "CALL"
	case NodeTypeAssert:
		return "ASSERT"
	case NodeTypeLit:
		return "LIT"
	case NodeTypeVar:
		return "VAR"
	default:
		return "UNKNOWN"
	}
}

// SeqNode represents a sequence of statements
type SeqNode struct {
	Statements []Node
}

func (n *SeqNode) String() string {
	var parts []string
	for _, stmt := range n.Statements {
		parts = append(parts, stmt.String())
	}
	return fmt.Sprintf("(seq %s)", strings.Join(parts, " "))
}

func (n *SeqNode) Type() NodeType {
	return NodeTypeSeq
}

// LetNode represents a variable binding
type LetNode struct {
	Name  string
	Value Node
	Body  Node
}

func (n *LetNode) String() string {
	return fmt.Sprintf("(let %s %s %s)", n.Name, n.Value.String(), n.Body.String())
}

func (n *LetNode) Type() NodeType {
	return NodeTypeLet
}

// ReturnNode represents a return statement
type ReturnNode struct {
	Value Node
}

func (n *ReturnNode) String() string {
	return fmt.Sprintf("(return %s)", n.Value.String())
}

func (n *ReturnNode) Type() NodeType {
	return NodeTypeReturn
}

// IfNode represents a conditional statement
type IfNode struct {
	Condition Node
	Then      Node
	Else      Node
}

func (n *IfNode) String() string {
	return fmt.Sprintf("(if %s %s %s)", n.Condition.String(), n.Then.String(), n.Else.String())
}

func (n *IfNode) Type() NodeType {
	return NodeTypeIf
}

// LoopNode represents a loop statement
type LoopNode struct {
	Condition Node
	Body      Node
}

func (n *LoopNode) String() string {
	return fmt.Sprintf("(loop %s %s)", n.Condition.String(), n.Body.String())
}

func (n *LoopNode) Type() NodeType {
	return NodeTypeLoop
}

// CallNode represents a function call
type CallNode struct {
	Function string
	Args     []Node
}

func (n *CallNode) String() string {
	var args []string
	for _, arg := range n.Args {
		args = append(args, arg.String())
	}
	return fmt.Sprintf("(call %s %s)", n.Function, strings.Join(args, " "))
}

func (n *CallNode) Type() NodeType {
	return NodeTypeCall
}

// AssertNode represents an assertion
type AssertNode struct {
	Condition Node
	Message   string
}

func (n *AssertNode) String() string {
	return fmt.Sprintf("(assert %s \"%s\")", n.Condition.String(), n.Message)
}

func (n *AssertNode) Type() NodeType {
	return NodeTypeAssert
}

// LitNode represents a literal value
type LitNode struct {
	Value interface{}
}

func (n *LitNode) String() string {
	switch v := n.Value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (n *LitNode) Type() NodeType {
	return NodeTypeLit
}

// VarNode represents a variable reference
type VarNode struct {
	Name string
}

func (n *VarNode) String() string {
	return n.Name
}

func (n *VarNode) Type() NodeType {
	return NodeTypeVar
}

// Tokenize tokenizes an S-expression string
func Tokenize(input string) []Token {
	var tokens []Token
	pos := 0
	input = strings.TrimSpace(input)

	for pos < len(input) {
		// Skip whitespace
		for pos < len(input) && isWhitespace(input[pos]) {
			pos++
		}
		if pos >= len(input) {
			break
		}

		start := pos
		char := input[pos]

		switch {
		case char == '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "(", Pos: pos})
			pos++
		case char == ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")", Pos: pos})
			pos++
		case char == '"':
			// String literal
			pos++
			value := ""
			for pos < len(input) && input[pos] != '"' {
				if input[pos] == '\\' && pos+1 < len(input) {
					pos++
					switch input[pos] {
					case 'n':
						value += "\n"
					case 't':
						value += "\t"
					case 'r':
						value += "\r"
					case '\\':
						value += "\\"
					case '"':
						value += "\""
					default:
						value += string(input[pos])
					}
				} else {
					value += string(input[pos])
				}
				pos++
			}
			if pos < len(input) {
				pos++ // consume closing quote
			}
			tokens = append(tokens, Token{Type: TokenString, Value: value, Pos: start})
		case isDigit(char) || char == '-':
			// Number literal
			for pos < len(input) && (isDigit(input[pos]) || input[pos] == '.' || input[pos] == 'e' || input[pos] == 'E' || input[pos] == '+' || input[pos] == '-') {
				pos++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Value: input[start:pos], Pos: start})
		case isAlpha(char) || char == '_' || char == '?' || char == '!':
			// Symbol or keyword
			for pos < len(input) && (isAlpha(input[pos]) || isDigit(input[pos]) || input[pos] == '_' || input[pos] == '?' || input[pos] == '!' || input[pos] == '-') {
				pos++
			}
			value := input[start:pos]
			switch value {
			case "true":
				tokens = append(tokens, Token{Type: TokenBool, Value: value, Pos: start})
			case "false":
				tokens = append(tokens, Token{Type: TokenBool, Value: value, Pos: start})
			case "null":
				tokens = append(tokens, Token{Type: TokenNull, Value: value, Pos: start})
			default:
				tokens = append(tokens, Token{Type: TokenSymbol, Value: value, Pos: start})
			}
		default:
			// Unknown character, skip
			pos++
		}
	}

	tokens = append(tokens, Token{Type: TokenEOF, Value: "", Pos: pos})
	return tokens
}

// Helper functions
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
