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

// MarshalJSON implements custom JSON marshaling for Budget
func (b Budget) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CPUMillis int    `json:"cpu_millis"`
		MemMB     int    `json:"mem_mb"`
		Timeout   string `json:"timeout"`
	}{
		CPUMillis: b.CPUMillis,
		MemMB:     b.MemMB,
		Timeout:   b.Timeout.String(),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Budget
func (b *Budget) UnmarshalJSON(data []byte) error {
	var aux struct {
		CPUMillis int    `json:"cpu_millis"`
		MemMB     int    `json:"mem_mb"`
		Timeout   string `json:"timeout"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	b.CPUMillis = aux.CPUMillis
	b.MemMB = aux.MemMB

	if aux.Timeout != "" {
		duration, err := time.ParseDuration(aux.Timeout)
		if err != nil {
			return err
		}
		b.Timeout = duration
	}

	return nil
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
