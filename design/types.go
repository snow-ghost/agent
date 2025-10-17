package design

// HypothesisDesign represents the JSON response from the LLM for algorithm design tasks
type HypothesisDesign struct {
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	Algorithm struct {
		Name       string `json:"name"`
		Idea       string `json:"idea"`
		Complexity struct {
			Time  string `json:"time"`
			Space string `json:"space"`
		} `json:"complexity"`
	} `json:"algorithm"`
	Code struct {
		Lang  string        `json:"lang"`          // "af-dsl" | "wasm"
		Entry string        `json:"entry"`         // "program"
		Src   string        `json:"src"`           // S-expr or base64 wasm (deprecated for af-dsl)
		AST   *AFDSLProgram `json:"ast,omitempty"` // JSON AST for af-dsl
	} `json:"code"`
	Evaluation struct {
		Metrics       []string `json:"metrics"`
		Fitness       string   `json:"fitness"`
		PassThreshold float64  `json:"pass_threshold"`
	} `json:"evaluation"`
	Tests struct {
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
	} `json:"tests"`
}
