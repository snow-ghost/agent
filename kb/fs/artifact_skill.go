package fs

import (
	"context"
	"fmt"
	"os"

	"github.com/snow-ghost/agent/artifact"
	"github.com/snow-ghost/agent/core"
)

// ArtifactSkill wraps a Manifest to implement the core.Skill interface
type ArtifactSkill struct {
	manifest *artifact.Manifest
	kb       *KnowledgeBaseFS
	wasmExec core.Interpreter
	goSkills map[string]core.Skill // Registry for Go skills during migration
}

// NewArtifactSkill creates a new ArtifactSkill
func NewArtifactSkill(manifest *artifact.Manifest, kb *KnowledgeBaseFS, wasmExec core.Interpreter) *ArtifactSkill {
	return &ArtifactSkill{
		manifest: manifest,
		kb:       kb,
		wasmExec: wasmExec,
		goSkills: make(map[string]core.Skill),
	}
}

// RegisterGoSkill registers a Go skill for migration period
func (as *ArtifactSkill) RegisterGoSkill(pkgFunc string, skill core.Skill) {
	as.goSkills[pkgFunc] = skill
}

// Name returns the skill name
func (as *ArtifactSkill) Name() string {
	return as.manifest.ID
}

// Domain returns the skill domain
func (as *ArtifactSkill) Domain() string {
	return as.manifest.Domain
}

// CanSolve checks if this skill can solve the given task
func (as *ArtifactSkill) CanSolve(task core.Task) (bool, float64) {
	// Check domain match
	if as.manifest.Domain != task.Domain {
		return false, 0.0
	}

	// Check tags against task properties
	for _, tag := range as.manifest.Tags {
		if task.Spec.Props != nil {
			for _, prop := range task.Spec.Props {
				if prop == tag {
					return true, 0.9 // High confidence for exact tag match
				}
			}
		}
	}

	// Check if any tag matches task domain keywords
	domainKeywords := []string{"sort", "reverse", "search", "filter", "transform"}
	for _, keyword := range domainKeywords {
		if task.Domain == keyword {
			for _, tag := range as.manifest.Tags {
				if tag == keyword {
					return true, 0.8 // Good confidence for domain keyword match
				}
			}
		}
	}

	// Domain match only
	return true, 0.5 // Medium confidence for domain match only
}

// Execute executes the skill
func (as *ArtifactSkill) Execute(ctx context.Context, task core.Task) (core.Result, error) {
	switch as.manifest.Lang {
	case "wasm":
		return as.executeWASM(ctx, task)
	case "go-skill":
		return as.executeGoSkill(ctx, task)
	default:
		return core.Result{Success: false}, fmt.Errorf("unsupported artifact language: %s", as.manifest.Lang)
	}
}

// executeWASM executes a WASM artifact
func (as *ArtifactSkill) executeWASM(ctx context.Context, task core.Task) (core.Result, error) {
	if as.wasmExec == nil {
		return core.Result{Success: false}, fmt.Errorf("WASM interpreter not available")
	}

	// Load WASM code
	codePath := as.manifest.GetCodePath(as.kb.artifactsDir)
	code, err := os.ReadFile(codePath)
	if err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to read WASM code: %w", err)
	}

	// Create hypothesis from artifact
	hypothesis := core.Hypothesis{
		ID:     as.manifest.ID,
		Source: "artifact",
		Lang:   "wasm",
		Bytes:  code,
		Meta: map[string]string{
			"version":     as.manifest.Version,
			"description": as.manifest.Description,
			"domain":      as.manifest.Domain,
		},
	}

	// Execute WASM
	return as.wasmExec.Execute(ctx, hypothesis, task)
}

// executeGoSkill executes a Go skill
func (as *ArtifactSkill) executeGoSkill(ctx context.Context, task core.Task) (core.Result, error) {
	skill, exists := as.goSkills[as.manifest.Entry]
	if !exists {
		return core.Result{Success: false}, fmt.Errorf("Go skill not found: %s", as.manifest.Entry)
	}

	return skill.Execute(ctx, task)
}

// Tests returns the test cases for this skill
func (as *ArtifactSkill) Tests() []core.TestCase {
	return as.manifest.Tests
}

// GetManifest returns the underlying manifest
func (as *ArtifactSkill) GetManifest() *artifact.Manifest {
	return as.manifest
}

// ArtifactKnowledgeBase implements core.KnowledgeBase using artifacts
type ArtifactKnowledgeBase struct {
	fs        *KnowledgeBaseFS
	wasmExec  core.Interpreter
	goSkills  map[string]core.Skill
	artifacts map[string]*ArtifactSkill
}

// NewArtifactKnowledgeBase creates a new artifact-based knowledge base
func NewArtifactKnowledgeBase(artifactsDir string, wasmExec core.Interpreter) *ArtifactKnowledgeBase {
	fs := NewKnowledgeBaseFS(artifactsDir)

	kb := &ArtifactKnowledgeBase{
		fs:        fs,
		wasmExec:  wasmExec,
		goSkills:  make(map[string]core.Skill),
		artifacts: make(map[string]*ArtifactSkill),
	}

	// Convert all manifests to ArtifactSkills
	kb.loadArtifacts()

	return kb
}

// loadArtifacts loads all artifacts and converts them to skills
func (kb *ArtifactKnowledgeBase) loadArtifacts() {
	manifests := kb.fs.ListArtifacts()

	// Clear existing artifacts
	kb.artifacts = make(map[string]*ArtifactSkill)

	for _, manifest := range manifests {
		skill := NewArtifactSkill(manifest, kb.fs, kb.wasmExec)

		// Register Go skills if needed
		if manifest.Lang == "go-skill" {
			skill.RegisterGoSkill(manifest.Entry, kb.goSkills[manifest.Entry])
		}

		key := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
		kb.artifacts[key] = skill
	}
}

// RegisterGoSkill registers a Go skill for migration
func (kb *ArtifactKnowledgeBase) RegisterGoSkill(pkgFunc string, skill core.Skill) {
	kb.goSkills[pkgFunc] = skill
}

// Find finds skills that can solve the given task
func (kb *ArtifactKnowledgeBase) Find(task core.Task) []core.Skill {
	var skills []core.Skill

	// Find by domain
	manifests := kb.fs.Find(task.Domain)
	for _, manifest := range manifests {
		key := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
		if skill, exists := kb.artifacts[key]; exists {
			if canSolve, _ := skill.CanSolve(task); canSolve {
				skills = append(skills, skill)
			}
		}
	}

	// Find by tags
	if task.Spec.Props != nil {
		for _, prop := range task.Spec.Props {
			manifests := kb.fs.FindByTag(prop)
			for _, manifest := range manifests {
				key := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
				if skill, exists := kb.artifacts[key]; exists {
					if canSolve, _ := skill.CanSolve(task); canSolve {
						// Avoid duplicates
						found := false
						for _, s := range skills {
							if s.Name() == skill.Name() {
								found = true
								break
							}
						}
						if !found {
							skills = append(skills, skill)
						}
					}
				}
			}
		}
	}

	return skills
}

// ListSkills returns all available skills
func (kb *ArtifactKnowledgeBase) ListSkills() []core.Skill {
	var skills []core.Skill
	for _, skill := range kb.artifacts {
		skills = append(skills, skill)
	}
	return skills
}

// RegisterSkill registers a new skill (for compatibility)
func (kb *ArtifactKnowledgeBase) RegisterSkill(skill core.Skill) {
	// This is for backward compatibility during migration
	// In the new system, skills are loaded from artifacts
}

// SaveHypothesis saves a hypothesis as an artifact
func (kb *ArtifactKnowledgeBase) SaveHypothesis(ctx context.Context, h core.Hypothesis, quality float64) error {
	// Generate artifact ID and version
	artifactID := fmt.Sprintf("hypothesis.%s", h.ID)
	version := "1.0.0"

	// Create manifest
	manifest := artifact.NewManifest(artifactID, version, "generated", "Generated hypothesis")
	manifest.AddTag("hypothesis")
	manifest.AddTag("generated")

	// Set up based on language
	switch h.Lang {
	case "wasm":
		codePath := "code.wasm"
		if err := manifest.SetWASM(codePath, h.Bytes); err != nil {
			return fmt.Errorf("failed to set WASM manifest: %w", err)
		}
	case "go-skill":
		// Extract package function from metadata
		pkgFunc := h.Meta["pkg_func"]
		if pkgFunc == "" {
			pkgFunc = "unknown.Func"
		}
		manifest.SetGoSkill(pkgFunc)
	default:
		return fmt.Errorf("unsupported hypothesis language: %s", h.Lang)
	}

	// Add metadata
	if h.Meta != nil {
		if domain, exists := h.Meta["domain"]; exists {
			manifest.Domain = domain
		}
		if desc, exists := h.Meta["description"]; exists {
			manifest.Description = desc
		}
	}

	// Add quality as a tag
	qualityTag := fmt.Sprintf("quality-%.2f", quality)
	manifest.AddTag(qualityTag)

	// Save artifact
	if err := kb.fs.SaveArtifact(manifest, h.Bytes); err != nil {
		return err
	}

	// Reload artifacts to include the new one
	kb.loadArtifacts()

	return nil
}

// GetArtifactFS returns the underlying file system
func (kb *ArtifactKnowledgeBase) GetArtifactFS() *KnowledgeBaseFS {
	return kb.fs
}
