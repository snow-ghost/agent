package dsl

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

// Value represents a JSON value in the runtime
type Value interface{}

// Runtime represents the AF-DSL runtime
type Runtime struct {
	variables map[string]Value
	stepCount int
	depth     int
	maxSteps  int
	maxDepth  int
}

// NewRuntime creates a new runtime
func NewRuntime() *Runtime {
	maxSteps := 1000000 // 1e6
	if val := os.Getenv("DSL_MAX_STEPS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxSteps = parsed
		}
	}

	maxDepth := 128
	if val := os.Getenv("DSL_MAX_DEPTH"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxDepth = parsed
		}
	}

	return &Runtime{
		variables: make(map[string]Value),
		stepCount: 0,
		depth:     0,
		maxSteps:  maxSteps,
		maxDepth:  maxDepth,
	}
}

// Execute executes a node in the runtime
func (r *Runtime) Execute(ctx context.Context, node Node) (Value, error) {
	// Check step limit
	if r.stepCount >= r.maxSteps {
		return nil, fmt.Errorf("step limit exceeded: %d", r.maxSteps)
	}

	// Check depth limit
	if r.depth >= r.maxDepth {
		return nil, fmt.Errorf("depth limit exceeded: %d", r.maxDepth)
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.stepCount++
	r.depth++

	var result Value
	var err error

	switch n := node.(type) {
	case *SeqNode:
		result, err = r.executeSeq(ctx, n)
	case *LetNode:
		result, err = r.executeLet(ctx, n)
	case *ReturnNode:
		result, err = r.executeReturn(ctx, n)
	case *IfNode:
		result, err = r.executeIf(ctx, n)
	case *LoopNode:
		result, err = r.executeLoop(ctx, n)
	case *CallNode:
		result, err = r.executeCall(ctx, n)
	case *AssertNode:
		result, err = r.executeAssert(ctx, n)
	case *LitNode:
		result, err = r.executeLit(ctx, n)
	case *VarNode:
		result, err = r.executeVar(ctx, n)
	default:
		err = fmt.Errorf("unknown node type: %T", node)
	}

	r.depth--
	return result, err
}

// executeSeq executes a sequence
func (r *Runtime) executeSeq(ctx context.Context, node *SeqNode) (Value, error) {
	var result Value
	var err error

	for _, stmt := range node.Statements {
		result, err = r.Execute(ctx, stmt)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// executeLet executes a let binding
func (r *Runtime) executeLet(ctx context.Context, node *LetNode) (Value, error) {
	// Evaluate the value
	value, err := r.Execute(ctx, node.Value)
	if err != nil {
		return nil, err
	}

	// Store in variables
	oldValue, exists := r.variables[node.Name]
	r.variables[node.Name] = value

	// Execute body
	result, err := r.Execute(ctx, node.Body)

	// Restore old value
	if exists {
		r.variables[node.Name] = oldValue
	} else {
		delete(r.variables, node.Name)
	}

	return result, err
}

// executeReturn executes a return statement
func (r *Runtime) executeReturn(ctx context.Context, node *ReturnNode) (Value, error) {
	return r.Execute(ctx, node.Value)
}

// executeIf executes a conditional
func (r *Runtime) executeIf(ctx context.Context, node *IfNode) (Value, error) {
	condition, err := r.Execute(ctx, node.Condition)
	if err != nil {
		return nil, err
	}

	if isTruthy(condition) {
		return r.Execute(ctx, node.Then)
	}
	return r.Execute(ctx, node.Else)
}

// executeLoop executes a loop
func (r *Runtime) executeLoop(ctx context.Context, node *LoopNode) (Value, error) {
	var result Value

	for {
		// Check step limit
		if r.stepCount >= r.maxSteps {
			return nil, fmt.Errorf("step limit exceeded in loop: %d", r.maxSteps)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		condition, err := r.Execute(ctx, node.Condition)
		if err != nil {
			return nil, err
		}

		if !isTruthy(condition) {
			break
		}

		result, err = r.Execute(ctx, node.Body)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// executeCall executes a function call
func (r *Runtime) executeCall(ctx context.Context, node *CallNode) (Value, error) {
	// Evaluate arguments
	args := make([]Value, len(node.Args))
	for i, arg := range node.Args {
		val, err := r.Execute(ctx, arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Call built-in function
	return r.callBuiltin(ctx, node.Function, args)
}

// executeAssert executes an assertion
func (r *Runtime) executeAssert(ctx context.Context, node *AssertNode) (Value, error) {
	condition, err := r.Execute(ctx, node.Condition)
	if err != nil {
		return nil, err
	}

	if !isTruthy(condition) {
		return nil, fmt.Errorf("assertion failed: %s", node.Message)
	}

	return true, nil
}

// executeLit executes a literal
func (r *Runtime) executeLit(ctx context.Context, node *LitNode) (Value, error) {
	return node.Value, nil
}

// executeVar executes a variable reference
func (r *Runtime) executeVar(ctx context.Context, node *VarNode) (Value, error) {
	value, exists := r.variables[node.Name]
	if !exists {
		return nil, fmt.Errorf("undefined variable: %s", node.Name)
	}
	return value, nil
}

// callBuiltin calls a built-in function
func (r *Runtime) callBuiltin(ctx context.Context, name string, args []Value) (Value, error) {
	switch name {
	case "split":
		return r.builtinSplit(args)
	case "merge":
		return r.builtinMerge(args)
	case "sorted?":
		return r.builtinSorted(args)
	case "permutes?":
		return r.builtinPermutes(args)
	case "len":
		return r.builtinLen(args)
	case "concat":
		return r.builtinConcat(args)
	case "map":
		return r.builtinMap(args)
	case "filter":
		return r.builtinFilter(args)
	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

// builtinSplit splits a list at the middle
func (r *Runtime) builtinSplit(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("split expects 1 argument, got %d", len(args))
	}

	list, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("split expects a list, got %T", args[0])
	}

	n := len(list)
	mid := n / 2

	left := list[:mid]
	right := list[mid:]

	return map[string]interface{}{
		"left":  left,
		"right": right,
	}, nil
}

// builtinMerge merges two sorted lists
func (r *Runtime) builtinMerge(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("merge expects 2 arguments, got %d", len(args))
	}

	left, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("merge expects list as first argument, got %T", args[0])
	}

	right, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("merge expects list as second argument, got %T", args[1])
	}

	// Check for stable property in third argument
	stable := false
	if len(args) > 2 {
		if props, ok := args[2].(map[string]interface{}); ok {
			if s, exists := props["stable"]; exists {
				if b, ok := s.(bool); ok {
					stable = b
				}
			}
		}
	}

	return r.mergeSorted(left, right, stable), nil
}

// mergeSorted merges two sorted lists with optional stability
func (r *Runtime) mergeSorted(left, right []interface{}, stable bool) []interface{} {
	result := make([]interface{}, 0, len(left)+len(right))
	i, j := 0, 0

	for i < len(left) && j < len(right) {
		leftVal := left[i]
		rightVal := right[j]

		// Compare values
		cmp := r.compareValues(leftVal, rightVal)
		if cmp < 0 || (cmp == 0 && stable) {
			// Take from left (or from left if equal and stable)
			result = append(result, leftVal)
			i++
		} else {
			// Take from right
			result = append(result, rightVal)
			j++
		}
	}

	// Append remaining elements
	for i < len(left) {
		result = append(result, left[i])
		i++
	}
	for j < len(right) {
		result = append(result, right[j])
		j++
	}

	return result
}

// compareValues compares two values for sorting
func (r *Runtime) compareValues(a, b interface{}) int {
	// Convert to comparable types
	aVal := r.toComparable(a)
	bVal := r.toComparable(b)

	switch aTyped := aVal.(type) {
	case float64:
		if bTyped, ok := bVal.(float64); ok {
			if aTyped < bTyped {
				return -1
			} else if aTyped > bTyped {
				return 1
			}
			return 0
		}
	case string:
		if bTyped, ok := bVal.(string); ok {
			if aTyped < bTyped {
				return -1
			} else if aTyped > bTyped {
				return 1
			}
			return 0
		}
	case bool:
		if bTyped, ok := bVal.(bool); ok {
			if !aTyped && bTyped {
				return -1
			} else if aTyped && !bTyped {
				return 1
			}
			return 0
		}
	}

	// Fallback: convert to string and compare
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// toComparable converts a value to a comparable type
func (r *Runtime) toComparable(v interface{}) interface{} {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case string:
		return val
	case bool:
		return val
	default:
		// Try to convert number strings
		if str, ok := v.(string); ok {
			if num, err := strconv.ParseFloat(str, 64); err == nil {
				return num
			}
		}
		return v
	}
}

// builtinSorted checks if a list is sorted
func (r *Runtime) builtinSorted(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sorted? expects 1 argument, got %d", len(args))
	}

	list, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("sorted? expects a list, got %T", args[0])
	}

	for i := 1; i < len(list); i++ {
		if r.compareValues(list[i-1], list[i]) > 0 {
			return false, nil
		}
	}

	return true, nil
}

// builtinPermutes checks if two lists are permutations
func (r *Runtime) builtinPermutes(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("permutes? expects 2 arguments, got %d", len(args))
	}

	a, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("permutes? expects list as first argument, got %T", args[0])
	}

	b, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("permutes? expects list as second argument, got %T", args[1])
	}

	if len(a) != len(b) {
		return false, nil
	}

	// Count occurrences
	counts := make(map[string]int)
	for _, val := range a {
		key := fmt.Sprintf("%v", val)
		counts[key]++
	}

	for _, val := range b {
		key := fmt.Sprintf("%v", val)
		counts[key]--
		if counts[key] < 0 {
			return false, nil
		}
	}

	// Check all counts are zero
	for _, count := range counts {
		if count != 0 {
			return false, nil
		}
	}

	return true, nil
}

// builtinLen returns the length of a list
func (r *Runtime) builtinLen(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len expects 1 argument, got %d", len(args))
	}

	switch val := args[0].(type) {
	case []interface{}:
		return len(val), nil
	case string:
		return len(val), nil
	default:
		return nil, fmt.Errorf("len expects a list or string, got %T", args[0])
	}
}

// builtinConcat concatenates two lists
func (r *Runtime) builtinConcat(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("concat expects 2 arguments, got %d", len(args))
	}

	a, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("concat expects list as first argument, got %T", args[0])
	}

	b, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("concat expects list as second argument, got %T", args[1])
	}

	result := make([]interface{}, 0, len(a)+len(b))
	result = append(result, a...)
	result = append(result, b...)
	return result, nil
}

// builtinMap applies a function to each element (simplified - just returns the list for now)
func (r *Runtime) builtinMap(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("map expects 2 arguments, got %d", len(args))
	}

	list, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("map expects list as first argument, got %T", args[0])
	}

	// For now, just return the list (simplified implementation)
	return list, nil
}

// builtinFilter filters a list (simplified - just returns the list for now)
func (r *Runtime) builtinFilter(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("filter expects 2 arguments, got %d", len(args))
	}

	list, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("filter expects list as first argument, got %T", args[0])
	}

	// For now, just return the list (simplified implementation)
	return list, nil
}

// isTruthy checks if a value is truthy
func isTruthy(v Value) bool {
	switch val := v.(type) {
	case bool:
		return val
	case nil:
		return false
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	case string:
		return val != ""
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		return true
	}
}

// ExecuteProgram executes an AF-DSL program with input
func ExecuteProgram(ctx context.Context, source string, input Value) (Value, error) {
	// Parse the program
	node, err := ParseProgram(source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Create runtime
	runtime := NewRuntime()

	// Set input variable
	runtime.variables["input"] = input

	// Execute the program
	return runtime.Execute(ctx, node)
}
