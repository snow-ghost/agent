package fs

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/core"
)

func TestKnowledgeBaseFS(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "kb-fs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create KB
	kb := NewKnowledgeBaseFS(tempDir)

	// Test creating a sample artifact
	if err := CreateSampleArtifact(tempDir); err != nil {
		t.Fatalf("Failed to create sample artifact: %v", err)
	}

	// Reload artifacts
	if err := kb.LoadArtifacts(); err != nil {
		t.Fatalf("Failed to reload artifacts: %v", err)
	}

	// Test finding artifacts
	artifacts := kb.Find("algorithms.sorting")
	if len(artifacts) == 0 {
		t.Fatal("Expected to find sorting artifacts")
	}

	// Test finding by tag
	tagArtifacts := kb.FindByTag("sort")
	if len(tagArtifacts) == 0 {
		t.Fatal("Expected to find artifacts by tag")
	}

	// Test listing artifacts
	allArtifacts := kb.ListArtifacts()
	if len(allArtifacts) == 0 {
		t.Fatal("Expected to find artifacts")
	}

	// Verify artifact structure
	artifact := allArtifacts[0]
	if artifact.ID != "sample.sort.v1" {
		t.Errorf("Expected ID 'sample.sort.v1', got '%s'", artifact.ID)
	}
	if artifact.Domain != "algorithms.sorting" {
		t.Errorf("Expected domain 'algorithms.sorting', got '%s'", artifact.Domain)
	}
	if artifact.Lang != "wasm" {
		t.Errorf("Expected lang 'wasm', got '%s'", artifact.Lang)
	}
}

func TestArtifactSkill(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "artifact-skill-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create KB
	kb := NewKnowledgeBaseFS(tempDir)

	// Create a sample artifact
	if err := CreateSampleArtifact(tempDir); err != nil {
		t.Fatalf("Failed to create sample artifact: %v", err)
	}

	// Reload artifacts
	if err := kb.LoadArtifacts(); err != nil {
		t.Fatalf("Failed to reload artifacts: %v", err)
	}

	// Create artifact knowledge base
	artifactKB := NewArtifactKnowledgeBase(tempDir, nil)

	// Test finding skills
	task := core.Task{
		ID:     "test-1",
		Domain: "algorithms.sorting",
		Spec: core.Spec{
			Props: map[string]string{"type": "sort"},
		},
		Input: json.RawMessage(`[3,1,2]`),
	}

	skills := artifactKB.Find(task)
	if len(skills) == 0 {
		t.Fatal("Expected to find skills for sorting task")
	}

	// Test skill properties
	skill := skills[0]
	if skill.Name() != "sample.sort.v1" {
		t.Errorf("Expected skill name 'sample.sort.v1', got '%s'", skill.Name())
	}
	if skill.Domain() != "algorithms.sorting" {
		t.Errorf("Expected skill domain 'algorithms.sorting', got '%s'", skill.Domain())
	}

	// Test CanSolve
	canSolve, confidence := skill.CanSolve(task)
	if !canSolve {
		t.Error("Expected skill to be able to solve the task")
	}
	if confidence <= 0 {
		t.Error("Expected positive confidence score")
	}

	// Test Tests
	tests := skill.Tests()
	if len(tests) == 0 {
		t.Error("Expected skill to have test cases")
	}
}

func TestManifestValidation(t *testing.T) {
	// Test valid manifest
	manifest := artifact.NewManifest("test.id", "1.0.0", "test.domain", "Test description")
	manifest.SetWASM("code.wasm", []byte{0x00, 0x61, 0x73, 0x6d})

	if err := manifest.Validate(); err != nil {
		t.Errorf("Valid manifest should not error: %v", err)
	}

	// Test invalid manifest (missing ID)
	invalidManifest := &artifact.Manifest{
		Version:     "1.0.0",
		Domain:      "test.domain",
		Description: "Test description",
		Lang:        "wasm",
		Entry:       "solve",
		CodePath:    "code.wasm",
		SHA256:      "test",
	}

	if err := invalidManifest.Validate(); err == nil {
		t.Error("Invalid manifest should error")
	}
}

func TestSaveHypothesis(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "save-hypothesis-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create artifact knowledge base
	artifactKB := NewArtifactKnowledgeBase(tempDir, nil)

	// Create a hypothesis
	hypothesis := core.Hypothesis{
		ID:     "test-hypothesis",
		Source: "llm",
		Lang:   "wasm",
		Bytes:  []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
		Meta: map[string]string{
			"domain":      "algorithms.sorting",
			"description": "Test sorting algorithm",
		},
	}

	// Save hypothesis
	ctx := context.Background()
	if err := artifactKB.SaveHypothesis(ctx, hypothesis, 0.95); err != nil {
		t.Fatalf("Failed to save hypothesis: %v", err)
	}

	// Verify artifact was created
	artifacts := artifactKB.GetArtifactFS().ListArtifacts()
	if len(artifacts) == 0 {
		t.Fatal("Expected to find saved artifact")
	}

	// Check artifact properties
	artifact := artifacts[0]
	if artifact.ID != "hypothesis.test-hypothesis" {
		t.Errorf("Expected ID 'hypothesis.test-hypothesis', got '%s'", artifact.ID)
	}
	if artifact.Domain != "algorithms.sorting" {
		t.Errorf("Expected domain 'algorithms.sorting', got '%s'", artifact.Domain)
	}
	if artifact.Lang != "wasm" {
		t.Errorf("Expected lang 'wasm', got '%s'", artifact.Lang)
	}
}

func TestMigration(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "migration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create sample artifact
	if err := CreateSampleArtifact(tempDir); err != nil {
		t.Fatalf("Failed to create sample artifact: %v", err)
	}

	// Validate artifact
	artifacts, err := ListArtifactsInDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to list artifacts: %v", err)
	}

	if len(artifacts) == 0 {
		t.Fatal("Expected to find artifacts")
	}

	// Validate each artifact
	for _, artifactDir := range artifacts {
		if err := ValidateArtifact(artifactDir); err != nil {
			t.Errorf("Artifact validation failed for %s: %v", artifactDir, err)
		}
	}
}
