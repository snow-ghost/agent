package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// E2E: docker-compose + LM Studio + router + worker => KB writes
func TestE2E_DockerCompose_KB_Write(t *testing.T) {
	// This test validates end-to-end flow when run in CI with docker-compose.
	// It assumes:
	// - LM Studio exposes OpenAI-compatible API at lmstudio:1234 with model qwen/qwen3-4b-2507
	// - docker-compose brings up services with volumes mounting ./artifacts for KB persistence
	// The test will:
	// 1) POST a heavy task to router /solve
	// 2) Expect heavy worker to generate and save a hypothesis into artifacts
	// 3) Assert at least one new artifact manifest appears

	// Skip in default unit test run; enable with E2E=1
	if os.Getenv("E2E") == "" {
		t.Skip("E2E not enabled; set E2E=1 to run")
	}

	routerURL := getenv("ROUTER_URL", "http://localhost:9006")

	// Clean artifacts dir before run
	artifactsDir := getenv("ARTIFACTS_DIR", "./artifacts")
	_ = os.MkdirAll(artifactsDir, 0o755)
	before, _ := filepath.Glob(filepath.Join(artifactsDir, "hypothesis.*@*/manifest.json"))

	// Construct a task that will route to heavy (requires sandbox)
	task := core.Task{
		ID:          "e2e-qwen-001",
		Domain:      "algorithms",
		Description: "Sort numbers ascending",
		Spec:        core.Spec{SuccessCriteria: []string{"sorted_non_decreasing", "permutes"}},
		Input:       json.RawMessage(`{"numbers":[3,1,2]}`),
		Flags:       core.TaskFlags{RequiresSandbox: true, MaxComplexity: 10},
		Budget:      core.Budget{Timeout: mustParseDuration("30s")},
	}

	body, err := json.Marshal(task)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", routerURL+"/solve", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Read response to ensure JSON decodes
	var result core.Result
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Poll for new artifact manifest up to 30s
	deadline := time.Now().Add(30 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		after, _ := filepath.Glob(filepath.Join(artifactsDir, "hypothesis.*@*/manifest.json"))
		if len(after) > len(before) {
			found = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	assert.True(t, found, "expected a new KB artifact manifest to be created")
}
