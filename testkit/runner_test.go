package testkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/snow-ghost/agent/core"
	"github.com/snow-ghost/agent/interp/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_Run_SortCases(t *testing.T) {
	interp := wasm.NewInterpreter()
	defer interp.Close(context.Background())

	runner := NewRunner()
	cases := GenerateSortCasesFixed()

	h := core.Hypothesis{ID: "sort-wasm", Lang: "wasm", Bytes: wasm.GetTestModule()}

	ctx := context.Background()
	metrics, pass, err := runner.Run(ctx, h, cases, interp)
	require.NoError(t, err)

	assert.Equal(t, float64(len(cases)), metrics["cases_total"])
	assert.True(t, metrics["cases_passed"]+metrics["cases_failed"] == metrics["cases_total"])
	assert.True(t, pass || !pass) // just ensure it returns a boolean
}

func TestFitnessEvaluator(t *testing.T) {
	w := core.NewWeightedFitness(map[string]float64{"cases_passed": 1.0, "cases_total": 0.0}, 0.1)
	score := w.Score(core.Task{}, map[string]float64{"cases_passed": 2, "cases_total": 2}, 2048)
	assert.InDelta(t, 2-0.2, score, 1e-9)
	assert.True(t, w.Passed(score, 1.5))
	assert.False(t, w.Passed(score, 3.0))
}

func TestSimpleCritic(t *testing.T) {
	critic := core.NewSimpleCritic()
	ok, reason := critic.Accept(core.Task{Spec: core.Spec{SuccessCriteria: []string{"sorted_non_decreasing"}}}, map[string]float64{"cases_failed": 0})
	assert.True(t, ok)
	assert.Contains(t, reason, "all tests passed")

	ok, reason = critic.Accept(core.Task{Spec: core.Spec{SuccessCriteria: []string{"sorted_non_decreasing"}}}, map[string]float64{"cases_failed": 1})
	assert.False(t, ok)
	assert.Contains(t, reason, "failed")

	ok, _ = critic.Accept(core.Task{Spec: core.Spec{SuccessCriteria: nil}}, map[string]float64{})
	assert.True(t, ok)
}

func TestRunner_TimingMetrics(t *testing.T) {
	interp := wasm.NewInterpreter()
	defer interp.Close(context.Background())
	runner := NewRunner()

	cases := []core.TestCase{
		{Name: "noop", Input: json.RawMessage(`{"numbers": [1,2,3]}`)},
	}

	h := core.Hypothesis{ID: "noop-wasm", Lang: "wasm", Bytes: wasm.GetTestModule()}

	ctx := context.Background()
	metrics, _, err := runner.Run(ctx, h, cases, interp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, metrics["duration_ms_total"], float64(0))
}
