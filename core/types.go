package core

import (
	"encoding/json"
	"time"
)

type Budget struct {
	CPUMillis int           // CPU limits
	MemMB     int           // memory limits
	Timeout   time.Duration // task timeout
}

type Task struct {
	ID        string
	Domain    string
	Spec      Spec // criteria/contracts
	Input     json.RawMessage
	Budget    Budget
	CreatedAt time.Time
}

type Spec struct {
	SuccessCriteria []string          // declarative/readable
	Props           map[string]string // key properties
	MetricsWeights  map[string]float64
}

type Result struct {
	Success bool
	Score   float64
	Output  json.RawMessage
	Logs    string
	Metrics map[string]float64
}

type Hypothesis struct {
	ID     string
	Source string            // "kb:sort.v1" | "llm:...", etc.
	Lang   string            // "go-skill" | "wasm-ir"
	Bytes  []byte            // code/bytecode/IR
	Meta   map[string]string // domain, version, etc.
}
