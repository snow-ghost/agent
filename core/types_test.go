package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTaskJSONRoundTrip(t *testing.T) {
	task := Task{
		ID:     "t1",
		Domain: "test",
		Spec: Spec{
			SuccessCriteria: []string{"output_valid"},
			Props:           map[string]string{"mode": "simple"},
			MetricsWeights:  map[string]float64{"latency": 0.5},
		},
		Input: json.RawMessage(`{"q":"hello","n":3}`),
		Budget: Budget{
			CPUMillis: 1000,
			MemMB:     128,
			Timeout:   time.Second * 30,
		},
		CreatedAt: time.Now(),
	}

	b, err := json.Marshal(task)
	require.NoError(t, err)

	var got Task
	require.NoError(t, json.Unmarshal(b, &got))

	require.Equal(t, task.ID, got.ID)
	require.Equal(t, task.Domain, got.Domain)
	require.Equal(t, task.Spec, got.Spec)
	require.Equal(t, task.Budget, got.Budget)
	require.Equal(t, string(task.Input), string(got.Input))
	// CreatedAt might have slight precision differences, so check they're close
	require.WithinDuration(t, task.CreatedAt, got.CreatedAt, time.Millisecond)
}

func TestResultJSONRoundTrip(t *testing.T) {
	res := Result{
		Success: true,
		Score:   0.95,
		Output:  json.RawMessage(`{"answer":"world"}`),
		Logs:    "Task completed successfully",
		Metrics: map[string]float64{"latency_ms": 12.5, "accuracy": 0.95},
	}

	b, err := json.Marshal(res)
	require.NoError(t, err)

	var got Result
	require.NoError(t, json.Unmarshal(b, &got))

	require.Equal(t, res.Success, got.Success)
	require.Equal(t, res.Score, got.Score)
	require.Equal(t, string(res.Output), string(got.Output))
	require.Equal(t, res.Logs, got.Logs)
	require.Equal(t, res.Metrics, got.Metrics)
}

func TestSpecJSONRoundTrip(t *testing.T) {
	spec := Spec{
		SuccessCriteria: []string{"output_valid", "latency_ok"},
		Props:           map[string]string{"mode": "test", "version": "v1"},
		MetricsWeights:  map[string]float64{"latency": 0.3, "accuracy": 0.7},
	}

	b, err := json.Marshal(spec)
	require.NoError(t, err)

	var got Spec
	require.NoError(t, json.Unmarshal(b, &got))

	require.Equal(t, spec, got)
}
