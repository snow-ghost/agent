package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHypothesisPersistence(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create registry with temp directory
	registry := NewRegistryWithDir(tempDir)

	// Create a test hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis-1",
		Source: "llm",
		Lang:   "wasm",
		Bytes:  []byte("test wasm bytecode"),
		Meta:   map[string]string{"test": "true"},
	}

	// Save hypothesis
	err := registry.SaveHypothesis(context.Background(), hypothesis, 0.95)
	require.NoError(t, err)

	// Check that files were created
	metadataFile := filepath.Join(tempDir, "test-hypothesis-1.meta.json")
	bytecodeFile := filepath.Join(tempDir, "test-hypothesis-1.wasm")

	assert.FileExists(t, metadataFile)
	assert.FileExists(t, bytecodeFile)

	// Check metadata content
	metadataData, err := os.ReadFile(metadataFile)
	require.NoError(t, err)

	var metadata HypothesisMetadata
	err = metadata.UnmarshalJSON(metadataData)
	require.NoError(t, err)

	assert.Equal(t, "test-hypothesis-1", metadata.ID)
	assert.Equal(t, "llm", metadata.Source)
	assert.Equal(t, "wasm", metadata.Lang)
	assert.Equal(t, 0.95, metadata.Quality)
	assert.Equal(t, "general", metadata.Domain)
}

func TestLoadHypotheses(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files
	metadata := HypothesisMetadata{
		ID:       "loaded-hypothesis-1",
		Source:   "llm",
		Lang:     "wasm",
		Meta:     map[string]string{"test": "true"},
		Quality:  0.85,
		SavedAt:  time.Now(),
		Domain:   "algorithms",
		Keywords: []string{"sort", "numbers"},
	}

	// Write metadata file
	metadataFile := filepath.Join(tempDir, "loaded-hypothesis-1.meta.json")
	metadataData, err := metadata.MarshalJSON()
	require.NoError(t, err)
	err = os.WriteFile(metadataFile, metadataData, 0644)
	require.NoError(t, err)

	// Write bytecode file
	bytecodeFile := filepath.Join(tempDir, "loaded-hypothesis-1.wasm")
	err = os.WriteFile(bytecodeFile, []byte("test wasm bytecode"), 0644)
	require.NoError(t, err)

	// Create new registry and load hypotheses
	registry := NewRegistryWithDir(tempDir)

	// Check that the hypothesis was loaded as a skill
	skills := registry.ListSkills()
	found := false
	for _, skill := range skills {
		if skill.Name() == "saved/loaded-hypothesis-1" {
			found = true
			break
		}
	}
	assert.True(t, found, "Saved hypothesis should be loaded as a skill")
}

func TestSavedHypothesisSkill(t *testing.T) {
	// Create a test hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-skill-1",
		Source: "llm",
		Lang:   "wasm",
		Bytes:  []byte("test wasm bytecode"),
		Meta:   map[string]string{"test": "true"},
	}

	metadata := HypothesisMetadata{
		ID:       "test-skill-1",
		Source:   "llm",
		Lang:     "wasm",
		Meta:     map[string]string{"test": "true"},
		Quality:  0.9,
		SavedAt:  time.Now(),
		Domain:   "algorithms",
		Keywords: []string{"sort", "numbers"},
	}

	// Create skill
	skill := &SavedHypothesisSkill{
		hypothesis: hypothesis,
		metadata:   metadata,
	}

	// Test skill properties
	assert.Equal(t, "saved/test-skill-1", skill.Name())
	assert.Equal(t, "algorithms", skill.Domain())

	// Test CanSolve
	task := core.Task{
		Domain: "algorithms",
		Spec: core.Spec{
			Props: map[string]string{"type": "numbers"},
		},
	}

	canSolve, confidence := skill.CanSolve(task)
	assert.True(t, canSolve)
	assert.Equal(t, 0.9, confidence)

	// Test Execute with valid input
	task.Input = []byte(`[3,1,2]`)
	result, err := skill.Execute(context.Background(), task)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 0.9, result.Score)
	assert.Contains(t, result.Logs, "Executed saved hypothesis test-skill-1")
}

// Add MarshalJSON and UnmarshalJSON methods for HypothesisMetadata
func (h *HypothesisMetadata) MarshalJSON() ([]byte, error) {
	type Alias HypothesisMetadata
	return json.Marshal((*Alias)(h))
}

func (h *HypothesisMetadata) UnmarshalJSON(data []byte) error {
	type Alias HypothesisMetadata
	return json.Unmarshal(data, (*Alias)(h))
}
