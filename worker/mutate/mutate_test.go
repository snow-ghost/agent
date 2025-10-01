package mutate

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleMutator_CandidatesCount(t *testing.T) {
	m := NewSimpleMutator()
	base := core.Hypothesis{ID: "algo", Source: "test", Lang: "wasm", Bytes: []byte{1, 2, 3}, Meta: map[string]string{"version": "v1"}}
	cands := m.Mutate(base)
	assert.GreaterOrEqual(t, len(cands), 3) // at least keep + a couple toggles
}

func TestSimpleMutator_NoPanicOnExecute(t *testing.T) {
	m := NewSimpleMutator()
	base := core.Hypothesis{ID: "algo", Source: "test", Lang: "wasm", Bytes: wasm.GetTestModule(), Meta: map[string]string{"version": "v1"}}
	cands := m.Mutate(base)

	interp := wasm.NewInterpreter()
	defer interp.Close(context.Background())

	task := core.Task{
		ID:     "mut-test",
		Domain: "algorithms",
		Input:  []byte(`{"numbers": [3,1,2]}`),
		Budget: core.Budget{CPUMillis: 1000, MemMB: 64, Timeout: time.Second * 2},
	}

	for _, h := range cands {
		_, err := interp.Execute(context.Background(), h, task)
		require.NoError(t, err)
	}
}
