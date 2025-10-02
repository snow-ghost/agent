package capabilities

// Capabilities defines what capabilities a worker has
type Capabilities struct {
	UseKB   bool `json:"use_kb"`   // Can use knowledge base
	UseWASM bool `json:"use_wasm"` // Can execute WASM
	UseLLM  bool `json:"use_llm"`  // Can use LLM
}

// String returns a human-readable representation of capabilities
func (c Capabilities) String() string {
	var caps []string
	if c.UseKB {
		caps = append(caps, "KB")
	}
	if c.UseWASM {
		caps = append(caps, "WASM")
	}
	if c.UseLLM {
		caps = append(caps, "LLM")
	}

	if len(caps) == 0 {
		return "none"
	}

	result := ""
	for i, cap := range caps {
		if i > 0 {
			result += "+"
		}
		result += cap
	}
	return result
}

// CanHandleTask determines if this worker can handle a given task
func (c Capabilities) CanHandleTask(requiresSandbox bool, maxComplexity int) bool {
	// If task requires sandbox, worker must support WASM
	if requiresSandbox && !c.UseWASM {
		return false
	}

	// If task has high complexity, worker should support LLM
	// (This is a heuristic - in practice, you might have more sophisticated logic)
	if maxComplexity > 5 && !c.UseLLM {
		return false
	}

	// All workers should support KB (it's the minimum requirement)
	return c.UseKB
}

// WorkerWithCapabilities is an interface for workers that expose their capabilities
type WorkerWithCapabilities interface {
	Type() string
	Caps() Capabilities
}

// DefaultCapabilities returns default capabilities for different worker types
func DefaultCapabilities(workerType string) Capabilities {
	switch workerType {
	case "light":
		return Capabilities{
			UseKB:   true,
			UseWASM: false,
			UseLLM:  false,
		}
	case "heavy":
		return Capabilities{
			UseKB:   true,
			UseWASM: true,
			UseLLM:  true,
		}
	default:
		return Capabilities{
			UseKB:   true,
			UseWASM: false,
			UseLLM:  false,
		}
	}
}
