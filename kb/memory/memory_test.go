package memory

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_FindSkill(t *testing.T) {
	registry := NewRegistry()

	t.Run("find sort skill", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "algorithms",
			Spec: core.Spec{
				Props: map[string]string{"operation": "sort"},
			},
		}

		skill, err := registry.FindSkill("algorithms", []string{"sort", "numbers"})
		require.NoError(t, err)
		assert.Equal(t, "algorithms/sort.v1", skill.Name())
		assert.Equal(t, "algorithms", skill.Domain())

		canSolve, confidence := skill.CanSolve(task)
		assert.True(t, canSolve)
		assert.Greater(t, confidence, 0.0)
	})

	t.Run("find reverse skill", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "text",
			Spec: core.Spec{
				Props: map[string]string{"operation": "reverse"},
			},
		}

		skill, err := registry.FindSkill("text", []string{"reverse", "string"})
		require.NoError(t, err)
		assert.Equal(t, "text/reverse.v1", skill.Name())
		assert.Equal(t, "text", skill.Domain())

		canSolve, confidence := skill.CanSolve(task)
		assert.True(t, canSolve)
		assert.Greater(t, confidence, 0.0)
	})

	t.Run("no skill found", func(t *testing.T) {
		_, err := registry.FindSkill("unknown", []string{"test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no skill found")
	})
}

func TestSortSkill_CanSolve(t *testing.T) {
	skill := &SortSkill{}

	t.Run("can solve with sort keyword", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "algorithms",
			Spec: core.Spec{
				Props: map[string]string{"operation": "sort"},
			},
		}

		canSolve, confidence := skill.CanSolve(task)
		assert.True(t, canSolve)
		assert.Greater(t, confidence, 0.0)
	})

	t.Run("cannot solve wrong domain", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "text",
			Spec: core.Spec{
				Props: map[string]string{"operation": "sort"},
			},
		}

		canSolve, confidence := skill.CanSolve(task)
		assert.False(t, canSolve)
		assert.Equal(t, 0.0, confidence)
	})
}

func TestSortSkill_Execute(t *testing.T) {
	skill := &SortSkill{}
	ctx := context.Background()

	t.Run("sort float64 numbers", func(t *testing.T) {
		input := map[string]any{
			"numbers": []float64{3, 1, 4, 1, 5},
		}
		inputJSON, _ := json.Marshal(input)

		task := core.Task{
			ID:     "t1",
			Domain: "algorithms",
			Input:  inputJSON,
			Budget: core.Budget{
				CPUMillis: 1000,
				MemMB:     128,
				Timeout:   time.Second * 30,
			},
			CreatedAt: time.Now(),
		}

		result, err := skill.Execute(ctx, task)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1.0, result.Score)

		// Parse output
		var output map[string]any
		err = json.Unmarshal(result.Output, &output)
		require.NoError(t, err)

		expected := []interface{}{1.0, 1.0, 3.0, 4.0, 5.0}
		assert.Equal(t, expected, output["sorted"])
		assert.Equal(t, 5, int(output["count"].(float64)))
	})

	t.Run("missing numbers input", func(t *testing.T) {
		input := map[string]any{}
		inputJSON, _ := json.Marshal(input)

		task := core.Task{
			ID:     "t1",
			Domain: "algorithms",
			Input:  inputJSON,
			Budget: core.Budget{
				CPUMillis: 1000,
				MemMB:     128,
				Timeout:   time.Second * 30,
			},
			CreatedAt: time.Now(),
		}

		_, err := skill.Execute(ctx, task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'numbers' input")
	})
}

func TestReverseSkill_CanSolve(t *testing.T) {
	skill := &ReverseSkill{}

	t.Run("can solve with reverse keyword", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "text",
			Spec: core.Spec{
				Props: map[string]string{"operation": "reverse"},
			},
		}

		canSolve, confidence := skill.CanSolve(task)
		assert.True(t, canSolve)
		assert.Greater(t, confidence, 0.0)
	})

	t.Run("cannot solve wrong domain", func(t *testing.T) {
		task := core.Task{
			ID:     "t1",
			Domain: "algorithms",
			Spec: core.Spec{
				Props: map[string]string{"operation": "reverse"},
			},
		}

		canSolve, confidence := skill.CanSolve(task)
		assert.False(t, canSolve)
		assert.Equal(t, 0.0, confidence)
	})
}

func TestReverseSkill_Execute(t *testing.T) {
	skill := &ReverseSkill{}
	ctx := context.Background()

	t.Run("reverse simple string", func(t *testing.T) {
		input := map[string]any{
			"text": "hello",
		}
		inputJSON, _ := json.Marshal(input)

		task := core.Task{
			ID:     "t1",
			Domain: "text",
			Input:  inputJSON,
			Budget: core.Budget{
				CPUMillis: 1000,
				MemMB:     128,
				Timeout:   time.Second * 30,
			},
			CreatedAt: time.Now(),
		}

		result, err := skill.Execute(ctx, task)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1.0, result.Score)

		// Parse output
		var output map[string]any
		err = json.Unmarshal(result.Output, &output)
		require.NoError(t, err)

		assert.Equal(t, "olleh", output["reversed"])
		assert.Equal(t, "hello", output["original"])
		assert.Equal(t, 5, int(output["length"].(float64)))
	})

	t.Run("missing text input", func(t *testing.T) {
		input := map[string]any{}
		inputJSON, _ := json.Marshal(input)

		task := core.Task{
			ID:     "t1",
			Domain: "text",
			Input:  inputJSON,
			Budget: core.Budget{
				CPUMillis: 1000,
				MemMB:     128,
				Timeout:   time.Second * 30,
			},
			CreatedAt: time.Now(),
		}

		_, err := skill.Execute(ctx, task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'text' input")
	})
}

func TestRegistry_ListSkills(t *testing.T) {
	registry := NewRegistry()
	skills := registry.ListSkills()

	assert.Len(t, skills, 2)

	skillNames := make([]string, len(skills))
	for i, skill := range skills {
		skillNames[i] = skill.Name()
	}

	assert.Contains(t, skillNames, "algorithms/sort.v1")
	assert.Contains(t, skillNames, "text/reverse.v1")
}

func TestSkill_Tests(t *testing.T) {
	sortSkill := &SortSkill{}
	reverseSkill := &ReverseSkill{}

	t.Run("sort skill tests", func(t *testing.T) {
		tests := sortSkill.Tests()
		assert.Len(t, tests, 1)
		assert.Equal(t, "sort_simple", tests[0].Name)
		assert.Contains(t, string(tests[0].Input), "numbers")
		assert.Contains(t, string(tests[0].Oracle), "sorted")
	})

	t.Run("reverse skill tests", func(t *testing.T) {
		tests := reverseSkill.Tests()
		assert.Len(t, tests, 1)
		assert.Equal(t, "reverse_simple", tests[0].Name)
		assert.Contains(t, string(tests[0].Input), "text")
		assert.Contains(t, string(tests[0].Oracle), "reversed")
	})
}
