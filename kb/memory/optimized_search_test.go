package memory

import (
	"context"
	"testing"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
)

// MockSkill implements core.Skill interface for testing
type MockSkill struct {
	id          string
	name        string
	domain      string
	description string
	keywords    []string
}

func (m *MockSkill) Name() string {
	return m.name
}

func (m *MockSkill) Domain() string {
	return m.domain
}

func (m *MockSkill) CanSolve(task core.Task) (bool, float64) {
	// Simple keyword matching for testing
	for _, keyword := range m.keywords {
		if contains(string(task.Input), keyword) {
			return true, 0.8
		}
	}
	return false, 0.0
}

func (m *MockSkill) Execute(ctx context.Context, task core.Task) (core.Result, error) {
	return core.Result{
		Success: true,
		Output:  []byte("mock result"),
		Score:   0.8,
	}, nil
}

func (m *MockSkill) Tests() []core.TestCase {
	return []core.TestCase{}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func TestRegistry_RegisterSkill(t *testing.T) {
	registry := NewRegistry()

	skill := &MockSkill{
		id:          "test-skill",
		name:        "Test Skill",
		domain:      "test.domain",
		description: "A test skill",
		keywords:    []string{"test", "example"},
	}

	registry.RegisterSkill(skill)

	// Check that skill was registered
	skills := registry.ListSkills()
	found := false
	for _, s := range skills {
		if s.Name() == "Test Skill" {
			found = true
			break
		}
	}
	assert.True(t, found, "Skill should be registered")
}

func TestRegistry_Find(t *testing.T) {
	registry := NewRegistry()

	skill := &MockSkill{
		id:          "test-skill",
		name:        "Test Skill",
		domain:      "test.domain",
		description: "A test skill",
		keywords:    []string{"test", "example"},
	}

	registry.RegisterSkill(skill)

	// Create a test task
	task := core.Task{
		ID:     "test-task",
		Domain: "test.domain",
		Input:  []byte("test input"),
		Spec: core.Spec{
			Props: map[string]string{
				"operation": "test",
			},
		},
	}

	// Test finding skills for the task
	skills := registry.Find(task)

	// Should find at least one skill
	assert.GreaterOrEqual(t, len(skills), 1)

	// Check that our skill is in the results
	found := false
	for _, s := range skills {
		if s.Name() == "Test Skill" {
			found = true
			break
		}
	}
	assert.True(t, found, "Test skill should be found")
}

func TestRegistry_SaveHypothesis(t *testing.T) {
	registry := NewRegistry()

	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "test",
		Lang:   "go",
		Bytes:  []byte("test code"),
		Meta:   map[string]string{"test": "value"},
	}

	err := registry.SaveHypothesis(context.Background(), hypothesis, 0.8)
	assert.NoError(t, err)
}

func TestRegistry_LoadHypotheses(t *testing.T) {
	registry := NewRegistry()

	// Load hypotheses (should not error even if file doesn't exist)
	err := registry.LoadHypotheses()
	assert.NoError(t, err)
}
