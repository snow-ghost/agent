package mutate

import (
	"fmt"
	"time"

	"github.com/snow-ghost/agent/core"
)

// SimpleMutator produces primitive mutations of hypotheses.
// For WASM: toggles metadata parameters (e.g., order asc/desc, resource hints)
// For DSL/IR (future): mocks operator substitutions via metadata.
type SimpleMutator struct{}

func NewSimpleMutator() *SimpleMutator { return &SimpleMutator{} }

// Mutate returns a small set of candidate hypotheses derived from base.
// It never panics and always returns at least one candidate (the base clone with updated ID suffix).
func (m *SimpleMutator) Mutate(base core.Hypothesis) []core.Hypothesis {
	candidates := make([]core.Hypothesis, 0, 6)

	// helper to clone with meta change
	clone := func(idSuffix string, meta map[string]string) core.Hypothesis {
		h := core.Hypothesis{
			ID:     fmt.Sprintf("%s~%s", base.ID, idSuffix),
			Source: base.Source,
			Lang:   base.Lang,
			Bytes:  base.Bytes,
			Meta:   map[string]string{},
		}
		for k, v := range base.Meta {
			h.Meta[k] = v
		}
		for k, v := range meta {
			h.Meta[k] = v
		}
		return h
	}

	// Always include a timestamped slight variant so at least one candidate exists.
	candidates = append(candidates, clone("keep", map[string]string{
		"mut": "keep",
		"ts":  fmt.Sprintf("%d", time.Now().UnixNano()),
	}))

	switch base.Lang {
	case "wasm":
		// Primitive parameter toggles via metadata only (do not alter bytes for stability)
		candidates = append(candidates,
			clone("order:asc", map[string]string{"order": "asc"}),
			clone("order:desc", map[string]string{"order": "desc"}),
			clone("limit:low", map[string]string{"cpu_hint": "low", "mem_hint": "low"}),
			clone("limit:high", map[string]string{"cpu_hint": "high", "mem_hint": "high"}),
		)
	case "dsl", "ir":
		// Mock operator substitutions
		candidates = append(candidates,
			clone("op:swap", map[string]string{"op_subst": "swap"}),
			clone("op:branch", map[string]string{"op_subst": "branch"}),
		)
	default:
		// Unknown language: keep only the base variant
	}

	return candidates
}
