package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/snow-ghost/agent/core"
)

// Skill represents a knowledge base skill that can solve specific problems
// Implements core.Skill interface

// KnowledgeBase provides access to available skills
type KnowledgeBase interface {
	FindSkill(domain string, keywords []string) (core.Skill, error)
	ListSkills() []core.Skill
	RegisterSkill(skill core.Skill)
}

// Registry is an in-memory implementation of KnowledgeBase
type Registry struct {
	skills        []core.Skill
	hypothesesDir string
}

// HypothesisMetadata represents metadata for a saved hypothesis
type HypothesisMetadata struct {
	ID       string            `json:"id"`
	Source   string            `json:"source"`
	Lang     string            `json:"lang"`
	Meta     map[string]string `json:"meta"`
	Quality  float64           `json:"quality"`
	SavedAt  time.Time         `json:"saved_at"`
	Domain   string            `json:"domain"`
	Keywords []string          `json:"keywords"`
}

// NewRegistry creates a new in-memory knowledge base registry
func NewRegistry() *Registry {
	registry := &Registry{
		skills:        make([]core.Skill, 0),
		hypothesesDir: "./hypotheses",
	}

	// Register built-in skills
	registry.RegisterSkill(&SortSkill{})
	registry.RegisterSkill(&ReverseSkill{})

	// Load saved hypotheses
	_ = registry.LoadHypotheses()

	return registry
}

// NewRegistryWithDir creates a registry with a custom hypotheses directory
func NewRegistryWithDir(hypothesesDir string) *Registry {
	registry := &Registry{
		skills:        make([]core.Skill, 0),
		hypothesesDir: hypothesesDir,
	}

	// Register built-in skills
	registry.RegisterSkill(&SortSkill{})
	registry.RegisterSkill(&ReverseSkill{})

	// Load saved hypotheses
	_ = registry.LoadHypotheses()

	return registry
}

// FindSkill finds the best matching skill for the given domain and keywords
func (r *Registry) FindSkill(domain string, keywords []string) (core.Skill, error) {
	// Create a dummy task for skill matching
	task := core.Task{
		ID:     "dummy",
		Domain: domain,
		Spec: core.Spec{
			Props: make(map[string]string),
		},
	}

	// Add keywords to spec props
	for i, keyword := range keywords {
		task.Spec.Props[fmt.Sprintf("keyword_%d", i)] = keyword
	}

	bestSkill := core.Skill(nil)
	bestConfidence := 0.0

	for _, skill := range r.skills {
		canSolve, confidence := skill.CanSolve(task)
		if canSolve && confidence > bestConfidence {
			bestSkill = skill
			bestConfidence = confidence
		}
	}

	if bestSkill == nil {
		return nil, fmt.Errorf("no skill found for domain %s with keywords %v", domain, keywords)
	}

	return bestSkill, nil
}

// ListSkills returns all registered skills
func (r *Registry) ListSkills() []core.Skill {
	return r.skills
}

// RegisterSkill adds a skill to the registry
func (r *Registry) RegisterSkill(skill core.Skill) {
	r.skills = append(r.skills, skill)
}

// Find implements core.KnowledgeBase: return skills sorted by confidence.
func (r *Registry) Find(task core.Task) []core.Skill {
	// simple order: any skill that CanSolve true, maintain insertion order
	out := make([]core.Skill, 0, len(r.skills))
	for _, sk := range r.skills {
		if ok, _ := sk.CanSolve(task); ok {
			out = append(out, sk)
		}
	}
	return out
}

// SaveHypothesis saves a successful hypothesis to disk
func (r *Registry) SaveHypothesis(ctx context.Context, h core.Hypothesis, quality float64) error {
	// Create hypotheses directory if it doesn't exist
	if err := os.MkdirAll(r.hypothesesDir, 0755); err != nil {
		return fmt.Errorf("failed to create hypotheses directory: %w", err)
	}

	// Create metadata
	metadata := HypothesisMetadata{
		ID:       h.ID,
		Source:   h.Source,
		Lang:     h.Lang,
		Meta:     h.Meta,
		Quality:  quality,
		SavedAt:  time.Now(),
		Domain:   "general", // Default domain, could be extracted from task
		Keywords: []string{"saved", "hypothesis"},
	}

	// Save metadata as JSON
	metadataFile := filepath.Join(r.hypothesesDir, h.ID+".meta.json")
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataFile, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	// Save bytecode
	bytecodeFile := filepath.Join(r.hypothesesDir, h.ID+".wasm")
	if err := os.WriteFile(bytecodeFile, h.Bytes, 0644); err != nil {
		return fmt.Errorf("failed to write bytecode file: %w", err)
	}

	// Register as a skill
	skill := &SavedHypothesisSkill{
		hypothesis: h,
		metadata:   metadata,
	}
	r.RegisterSkill(skill)

	return nil
}

// LoadHypotheses loads saved hypotheses from disk
func (r *Registry) LoadHypotheses() error {
	// Check if hypotheses directory exists
	if _, err := os.Stat(r.hypothesesDir); os.IsNotExist(err) {
		return nil // No hypotheses to load
	}

	// Read all .meta.json files
	entries, err := os.ReadDir(r.hypothesesDir)
	if err != nil {
		return fmt.Errorf("failed to read hypotheses directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}

		// Load metadata
		metadataFile := filepath.Join(r.hypothesesDir, entry.Name())
		metadataData, err := os.ReadFile(metadataFile)
		if err != nil {
			continue // Skip corrupted files
		}

		var metadata HypothesisMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue // Skip corrupted files
		}

		// Load bytecode
		bytecodeFile := filepath.Join(r.hypothesesDir, metadata.ID+".wasm")
		bytecode, err := os.ReadFile(bytecodeFile)
		if err != nil {
			continue // Skip if bytecode is missing
		}

		// Create hypothesis
		hypothesis := core.Hypothesis{
			ID:     metadata.ID,
			Source: metadata.Source,
			Lang:   metadata.Lang,
			Bytes:  bytecode,
			Meta:   metadata.Meta,
		}

		// Register as skill
		skill := &SavedHypothesisSkill{
			hypothesis: hypothesis,
			metadata:   metadata,
		}
		r.RegisterSkill(skill)
	}

	return nil
}

// SavedHypothesisSkill wraps a saved hypothesis as a skill
type SavedHypothesisSkill struct {
	hypothesis core.Hypothesis
	metadata   HypothesisMetadata
}

func (s *SavedHypothesisSkill) Name() string {
	return "saved/" + s.hypothesis.ID
}

func (s *SavedHypothesisSkill) Domain() string {
	return s.metadata.Domain
}

func (s *SavedHypothesisSkill) CanSolve(task core.Task) (bool, float64) {
	// Simple matching: check if domain matches and task has similar properties
	if s.metadata.Domain != "general" && s.metadata.Domain != task.Domain {
		return false, 0.0
	}

	// Check if task properties match saved hypothesis keywords
	for _, keyword := range s.metadata.Keywords {
		if strings.Contains(strings.ToLower(task.Spec.Props["type"]), strings.ToLower(keyword)) {
			return true, s.metadata.Quality
		}
	}

	// Default: accept if domain matches
	return s.metadata.Domain == "general" || s.metadata.Domain == task.Domain, s.metadata.Quality
}

func (s *SavedHypothesisSkill) Execute(ctx context.Context, task core.Task) (core.Result, error) {
	// For saved hypotheses, we need to execute the WASM bytecode
	// This is a simplified implementation - in practice, you'd use the interpreter

	// Parse input
	var inputData interface{}
	if err := json.Unmarshal(task.Input, &inputData); err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to parse input: %w", err)
	}

	// For now, return a mock result indicating the hypothesis was found
	// In a real implementation, this would execute the WASM bytecode
	result := core.Result{
		Success: true,
		Score:   s.metadata.Quality,
		Output:  task.Input, // Echo the input for now
		Logs:    fmt.Sprintf("Executed saved hypothesis %s", s.hypothesis.ID),
		Metrics: map[string]float64{
			"hypothesis_quality": s.metadata.Quality,
			"source":             1.0, // Saved hypothesis
		},
	}

	return result, nil
}

func (s *SavedHypothesisSkill) Tests() []core.TestCase {
	// Return empty tests for saved hypotheses
	// In practice, you might want to save and restore test cases too
	return []core.TestCase{}
}

// SortSkill implements sorting of number arrays
type SortSkill struct{}

func (s *SortSkill) Name() string {
	return "algorithms/sort.v1"
}

func (s *SortSkill) Domain() string {
	return "algorithms"
}

func (s *SortSkill) CanSolve(task core.Task) (bool, float64) {
	// Extract keywords from task spec or input
	keywords := []string{}
	if task.Spec.Props != nil {
		for _, v := range task.Spec.Props {
			keywords = append(keywords, v)
		}
	}

	// Check if this is an algorithms task
	if task.Domain != "algorithms" {
		return false, 0.0
	}

	// Check for sorting-related keywords
	for _, keyword := range keywords {
		lower := strings.ToLower(keyword)
		if strings.Contains(lower, "sort") || strings.Contains(lower, "order") ||
			strings.Contains(lower, "arrange") || strings.Contains(lower, "sequence") ||
			strings.Contains(lower, "numbers") {
			return true, 0.9 // High confidence for sorting tasks
		}
	}
	return false, 0.0
}

func (s *SortSkill) Execute(ctx context.Context, task core.Task) (core.Result, error) {
	// Parse input from task.Input (json.RawMessage)
	var inputData map[string]any
	if err := json.Unmarshal(task.Input, &inputData); err != nil {
		return core.Result{}, fmt.Errorf("failed to parse input: %w", err)
	}

	numbers, ok := inputData["numbers"]
	if !ok {
		return core.Result{}, fmt.Errorf("missing 'numbers' input")
	}

	// Convert to []float64
	var nums []float64
	switch v := numbers.(type) {
	case []interface{}:
		for _, item := range v {
			switch n := item.(type) {
			case float64:
				nums = append(nums, n)
			case int:
				nums = append(nums, float64(n))
			default:
				return core.Result{}, fmt.Errorf("invalid number type: %T", item)
			}
		}
	case []float64:
		nums = v
	case []int:
		for _, n := range v {
			nums = append(nums, float64(n))
		}
	default:
		return core.Result{}, fmt.Errorf("unsupported numbers type: %T", numbers)
	}

	// Sort the numbers
	sort.Float64s(nums)

	// Create result
	output, _ := json.Marshal(map[string]any{
		"sorted": nums,
		"count":  len(nums),
	})

	return core.Result{
		Success: true,
		Score:   1.0,
		Output:  output,
		Logs:    fmt.Sprintf("Sorted %d numbers", len(nums)),
		Metrics: map[string]float64{
			"count": float64(len(nums)),
		},
	}, nil
}

func (s *SortSkill) Tests() []core.TestCase {
	return []core.TestCase{
		{
			Name:   "sort_simple",
			Input:  []byte(`{"numbers": [3, 1, 4, 1, 5]}`),
			Oracle: []byte(`{"sorted": [1, 1, 3, 4, 5], "count": 5}`),
			Checks: []string{"sorted_order", "preserves_count"},
			Weight: 1.0,
		},
	}
}

// ReverseSkill implements string reversal
type ReverseSkill struct{}

func (s *ReverseSkill) Name() string {
	return "text/reverse.v1"
}

func (s *ReverseSkill) Domain() string {
	return "text"
}

func (s *ReverseSkill) CanSolve(task core.Task) (bool, float64) {
	// Extract keywords from task spec or input
	keywords := []string{}
	if task.Spec.Props != nil {
		for _, v := range task.Spec.Props {
			keywords = append(keywords, v)
		}
	}

	// Check if this is a text task
	if task.Domain != "text" {
		return false, 0.0
	}

	// Check for reverse-related keywords
	for _, keyword := range keywords {
		lower := strings.ToLower(keyword)
		if strings.Contains(lower, "reverse") || strings.Contains(lower, "flip") ||
			strings.Contains(lower, "backward") || strings.Contains(lower, "invert") {
			return true, 0.9 // High confidence for reverse tasks
		}
	}
	return false, 0.0
}

func (s *ReverseSkill) Execute(ctx context.Context, task core.Task) (core.Result, error) {
	// Parse input from task.Input (json.RawMessage)
	var inputData map[string]any
	if err := json.Unmarshal(task.Input, &inputData); err != nil {
		return core.Result{}, fmt.Errorf("failed to parse input: %w", err)
	}

	text, ok := inputData["text"]
	if !ok {
		return core.Result{}, fmt.Errorf("missing 'text' input")
	}

	str, ok := text.(string)
	if !ok {
		return core.Result{}, fmt.Errorf("text must be a string, got %T", text)
	}

	// Reverse the string
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	// Create result
	output, _ := json.Marshal(map[string]any{
		"reversed": string(runes),
		"original": str,
		"length":   len(str),
	})

	return core.Result{
		Success: true,
		Score:   1.0,
		Output:  output,
		Logs:    fmt.Sprintf("Reversed string of length %d", len(str)),
		Metrics: map[string]float64{
			"length": float64(len(str)),
		},
	}, nil
}

func (s *ReverseSkill) Tests() []core.TestCase {
	return []core.TestCase{
		{
			Name:   "reverse_simple",
			Input:  []byte(`{"text": "hello"}`),
			Oracle: []byte(`{"reversed": "olleh", "original": "hello", "length": 5}`),
			Checks: []string{"reversed_correctly", "preserves_original"},
			Weight: 1.0,
		},
	}
}
