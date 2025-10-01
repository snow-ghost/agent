package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/core"
)

// GoSkillRegistry holds Go skills during migration
type GoSkillRegistry struct {
	skills map[string]core.Skill
}

// NewGoSkillRegistry creates a new Go skill registry
func NewGoSkillRegistry() *GoSkillRegistry {
	return &GoSkillRegistry{
		skills: make(map[string]core.Skill),
	}
}

// Register registers a Go skill
func (r *GoSkillRegistry) Register(pkgFunc string, skill core.Skill) {
	r.skills[pkgFunc] = skill
}

// Get retrieves a Go skill
func (r *GoSkillRegistry) Get(pkgFunc string) (core.Skill, bool) {
	skill, exists := r.skills[pkgFunc]
	return skill, exists
}

// MigrateGoSkillToArtifact migrates a Go skill to an artifact
func MigrateGoSkillToArtifact(skill core.Skill, pkgFunc, artifactsDir string) error {
	// Create manifest
	manifest := artifact.NewManifest(
		fmt.Sprintf("go.%s", skill.Name()),
		"1.0.0",
		skill.Domain(),
		fmt.Sprintf("Migrated Go skill: %s", skill.Name()),
	)

	// Set as Go skill
	manifest.SetGoSkill(pkgFunc)

	// Add tags based on domain and name
	manifest.AddTag("go-skill")
	manifest.AddTag("migrated")
	manifest.AddTag(skill.Domain())
	manifest.AddTag(skill.Name())

	// Add test cases
	tests := skill.Tests()
	for _, test := range tests {
		manifest.AddTest(test)
	}

	// Create artifact directory
	artifactDir := manifest.GetArtifactPath(artifactsDir)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Save manifest
	manifestPath := manifest.GetManifestPath(artifactsDir)
	manifestData, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// MigrateExistingSkills migrates all existing Go skills to artifacts
func MigrateExistingSkills(artifactsDir string) error {
	// Create artifacts directory
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Create registry
	registry := NewGoSkillRegistry()

	// Register built-in Go skills
	// This would be done by the main application
	// registry.Register("sort.SortSkill", sortSkill)
	// registry.Register("reverse.ReverseSkill", reverseSkill)

	// Migrate each skill
	for pkgFunc, skill := range registry.skills {
		if err := MigrateGoSkillToArtifact(skill, pkgFunc, artifactsDir); err != nil {
			return fmt.Errorf("failed to migrate skill %s: %w", pkgFunc, err)
		}
	}

	return nil
}

// ExportManifest exports a manifest to JSON file
func ExportManifest(manifest *artifact.Manifest, outputPath string) error {
	data, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}

// ImportManifest imports a manifest from JSON file
func ImportManifest(manifestPath string) (*artifact.Manifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	manifest, err := artifact.FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return manifest, nil
}

// ValidateArtifact validates an artifact directory
func ValidateArtifact(artifactDir string) error {
	manifestPath := filepath.Join(artifactDir, "manifest.json")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest.json not found in %s", artifactDir)
	}

	// Load and validate manifest
	manifest, err := ImportManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Check code file if it's a WASM artifact
	if manifest.Lang == "wasm" && manifest.CodePath != "" {
		codePath := filepath.Join(artifactDir, manifest.CodePath)
		if _, err := os.Stat(codePath); os.IsNotExist(err) {
			return fmt.Errorf("code file not found: %s", codePath)
		}
	}

	return nil
}

// ListArtifactsInDir lists all artifacts in a directory
func ListArtifactsInDir(artifactsDir string) ([]string, error) {
	var artifacts []string

	err := filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() != filepath.Base(artifactsDir) {
			// Check if this directory contains a manifest.json
			manifestPath := filepath.Join(path, "manifest.json")
			if _, err := os.Stat(manifestPath); err == nil {
				artifacts = append(artifacts, path)
			}
		}

		return nil
	})

	return artifacts, err
}

// CleanupArtifacts removes invalid artifacts
func CleanupArtifacts(artifactsDir string) error {
	artifacts, err := ListArtifactsInDir(artifactsDir)
	if err != nil {
		return fmt.Errorf("failed to list artifacts: %w", err)
	}

	var invalidArtifacts []string

	for _, artifactDir := range artifacts {
		if err := ValidateArtifact(artifactDir); err != nil {
			fmt.Printf("Invalid artifact %s: %v\n", artifactDir, err)
			invalidArtifacts = append(invalidArtifacts, artifactDir)
		}
	}

	// Remove invalid artifacts
	for _, artifactDir := range invalidArtifacts {
		fmt.Printf("Removing invalid artifact: %s\n", artifactDir)
		if err := os.RemoveAll(artifactDir); err != nil {
			fmt.Printf("Failed to remove %s: %v\n", artifactDir, err)
		}
	}

	return nil
}

// CreateSampleArtifact creates a sample artifact for testing
func CreateSampleArtifact(artifactsDir string) error {
	// Create a sample WASM artifact
	manifest := artifact.NewManifest(
		"sample.sort.v1",
		"1.0.0",
		"algorithms.sorting",
		"Sample sorting algorithm",
	)

	// Add tags
	manifest.AddTag("sort")
	manifest.AddTag("sample")
	manifest.AddTag("algorithms")

	// Set as WASM
	sampleWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00} // Minimal WASM header
	if err := manifest.SetWASM("code.wasm", sampleWASM); err != nil {
		return fmt.Errorf("failed to set WASM: %w", err)
	}

	// Add sample test
	test := core.TestCase{
		Name:   "sort_test_1",
		Input:  []byte(`[3,1,2]`),
		Oracle: []byte(`[1,2,3]`),
		Checks: []string{"sorted_non_decreasing"},
		Weight: 1.0,
	}
	manifest.AddTest(test)

	// Save artifact
	artifactDir := manifest.GetArtifactPath(artifactsDir)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Save manifest
	manifestPath := manifest.GetManifestPath(artifactsDir)
	manifestData, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Save WASM code
	codePath := manifest.GetCodePath(artifactsDir)
	if err := os.WriteFile(codePath, sampleWASM, 0644); err != nil {
		return fmt.Errorf("failed to write WASM code: %w", err)
	}

	return nil
}
