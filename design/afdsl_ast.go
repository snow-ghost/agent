package design

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AFDSLNode represents a node in the AF-DSL AST
type AFDSLNode struct {
	Type     string       `json:"type"`
	Value    interface{}  `json:"value,omitempty"`
	Children []*AFDSLNode `json:"children,omitempty"`
	Args     []*AFDSLNode `json:"args,omitempty"`
}

// AFDSLProgram represents the root of an AF-DSL program
type AFDSLProgram struct {
	Type     string       `json:"type"`
	Children []*AFDSLNode `json:"children"`
	Value    interface{}  `json:"value,omitempty"` // For LLM compatibility
}

// Node types
const (
	NodeTypeProgram = "program"
	NodeTypeLet     = "let"
	NodeTypeIf      = "if"
	NodeTypeCall    = "call"
	NodeTypeReturn  = "return"
	NodeTypeLoop    = "loop"
	NodeTypeAssert  = "assert"
	NodeTypeVar     = "var"
	NodeTypeLiteral = "literal"
	NodeTypeArray   = "array"
)

// ToAFDSL converts JSON AST to AF-DSL S-expression string
func (p *AFDSLProgram) ToAFDSL() string {
	var parts []string

	// Handle case where LLM generates "value" instead of "children"
	if len(p.Children) == 0 && p.Value != nil {
		// Try to parse the value as a single node
		if valueMap, ok := p.Value.(map[string]interface{}); ok {
			// Create a temporary node from the value
			tempNode := &AFDSLNode{}
			if valueBytes, err := json.Marshal(valueMap); err == nil {
				if err := json.Unmarshal(valueBytes, tempNode); err == nil {
					parts = append(parts, tempNode.toAFDSL())
				}
			}
		}
	} else {
		// Normal case with children
		for _, child := range p.Children {
			parts = append(parts, child.toAFDSL())
		}
	}

	return fmt.Sprintf("(program %s)", strings.Join(parts, " "))
}

// toAFDSL converts a node to AF-DSL S-expression string
func (n *AFDSLNode) toAFDSL() string {
	switch n.Type {
	case NodeTypeLet:
		if len(n.Children) >= 2 {
			var parts []string
			parts = append(parts, "(let")
			parts = append(parts, n.Children[0].toAFDSL()) // variable name
			parts = append(parts, n.Children[1].toAFDSL()) // value
			if len(n.Children) > 2 {
				parts = append(parts, n.Children[2].toAFDSL()) // body
			}
			parts = append(parts, ")")
			return strings.Join(parts, " ")
		}
		return "(let)"

	case NodeTypeIf:
		if len(n.Children) >= 2 {
			var parts []string
			parts = append(parts, "(if")
			parts = append(parts, n.Children[0].toAFDSL()) // condition
			parts = append(parts, n.Children[1].toAFDSL()) // then
			if len(n.Children) > 2 {
				parts = append(parts, n.Children[2].toAFDSL()) // else
			}
			parts = append(parts, ")")
			return strings.Join(parts, " ")
		}
		return "(if)"

	case NodeTypeCall:
		if len(n.Args) > 0 {
			var parts []string
			parts = append(parts, "(call")
			parts = append(parts, n.Args[0].toAFDSL()) // function name
			for i := 1; i < len(n.Args); i++ {
				parts = append(parts, n.Args[i].toAFDSL()) // arguments
			}
			parts = append(parts, ")")
			return strings.Join(parts, " ")
		}
		return "(call)"

	case NodeTypeReturn:
		if len(n.Children) > 0 {
			return fmt.Sprintf("(return %s)", n.Children[0].toAFDSL())
		}
		return "(return)"

	case NodeTypeLoop:
		if len(n.Children) >= 2 {
			return fmt.Sprintf("(loop %s %s)", n.Children[0].toAFDSL(), n.Children[1].toAFDSL())
		}
		return "(loop)"

	case NodeTypeAssert:
		if len(n.Children) >= 2 {
			return fmt.Sprintf("(assert %s %s)", n.Children[0].toAFDSL(), n.Children[1].toAFDSL())
		}
		return "(assert)"

	case NodeTypeVar:
		if n.Value != nil {
			return fmt.Sprintf("%v", n.Value)
		}
		return "var"

	case NodeTypeLiteral:
		if n.Value != nil {
			switch v := n.Value.(type) {
			case string:
				return fmt.Sprintf("\"%s\"", v)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
		return "literal"

	case NodeTypeArray:
		if len(n.Children) > 0 {
			var parts []string
			for _, child := range n.Children {
				parts = append(parts, child.toAFDSL())
			}
			return fmt.Sprintf("[%s]", strings.Join(parts, " "))
		}
		return "[]"

	default:
		return fmt.Sprintf("(%s)", n.Type)
	}
}

// ParseAFDSLFromJSON parses AF-DSL JSON representation into AST
func ParseAFDSLFromJSON(jsonData []byte) (*AFDSLProgram, error) {
	var program AFDSLProgram
	if err := json.Unmarshal(jsonData, &program); err != nil {
		return nil, fmt.Errorf("failed to parse AF-DSL JSON: %w", err)
	}
	return &program, nil
}

// ValidateAFDSLAST validates the AF-DSL AST structure
func (p *AFDSLProgram) ValidateAFDSLAST() error {
	if p.Type != NodeTypeProgram {
		return fmt.Errorf("root node must be of type 'program'")
	}

	for _, child := range p.Children {
		if err := child.validateNode(); err != nil {
			return fmt.Errorf("invalid node: %w", err)
		}
	}

	return nil
}

// validateNode validates a single node
func (n *AFDSLNode) validateNode() error {
	switch n.Type {
	case NodeTypeLet:
		if len(n.Children) < 2 {
			return fmt.Errorf("let node must have at least 2 children (variable and value)")
		}
		if n.Children[0].Type != NodeTypeVar {
			return fmt.Errorf("let node first child must be a variable")
		}

	case NodeTypeIf:
		if len(n.Children) < 2 {
			return fmt.Errorf("if node must have at least 2 children (condition and then)")
		}

	case NodeTypeCall:
		if len(n.Args) == 0 {
			return fmt.Errorf("call node must have at least one argument (function name)")
		}
		if n.Args[0].Type != NodeTypeVar {
			return fmt.Errorf("call node first argument must be a variable (function name)")
		}

	case NodeTypeReturn, NodeTypeLoop, NodeTypeAssert:
		// These can have 0 or more children

	case NodeTypeVar, NodeTypeLiteral:
		if n.Value == nil {
			return fmt.Errorf("%s node must have a value", n.Type)
		}

	case NodeTypeArray:
		// Arrays can be empty

	default:
		return fmt.Errorf("unknown node type: %s", n.Type)
	}

	// Recursively validate children
	for _, child := range n.Children {
		if err := child.validateNode(); err != nil {
			return err
		}
	}

	// Recursively validate args
	for _, arg := range n.Args {
		if err := arg.validateNode(); err != nil {
			return err
		}
	}

	return nil
}
