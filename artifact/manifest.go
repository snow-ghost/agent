package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/snow-ghost/agent/core"
)

// Manifest represents the metadata for an artifact
type Manifest struct {
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	Domain         string          `json:"domain"`
	Description    string          `json:"description"`
	Tags           []string        `json:"tags"`
	Lang           string          `json:"lang"`      // "wasm" | "go-skill"
	Entry          string          `json:"entry"`     // export: "solve" (wasm) or "pkg.Func" (go-skill)
	CodePath       string          `json:"code_path"` // path to .wasm or empty for go-skill
	SHA256         string          `json:"sha256"`
	EmbeddingModel string          `json:"embedding_model,omitempty"`
	Embedding      []float32       `json:"embedding,omitempty"`
	Tests          []core.TestCase `json:"tests"`
	CreatedAt      string          `json:"created_at"`
}

// NewManifest creates a new manifest with default values
func NewManifest(id, version, domain, description string) *Manifest {
	return &Manifest{
		ID:          id,
		Version:     version,
		Domain:      domain,
		Description: description,
		Tags:        []string{},
		Tests:       []core.TestCase{},
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

// SetWASM sets the manifest for a WASM artifact
func (m *Manifest) SetWASM(codePath string, code []byte) error {
	m.Lang = "wasm"
	m.Entry = "solve"
	m.CodePath = codePath

	// Calculate SHA256
	hash := sha256.Sum256(code)
	m.SHA256 = hex.EncodeToString(hash[:])

	return nil
}

// SetGoSkill sets the manifest for a Go skill
func (m *Manifest) SetGoSkill(pkgFunc string) {
	m.Lang = "go-skill"
	m.Entry = pkgFunc
	m.CodePath = ""
	m.SHA256 = "" // No code file for Go skills
}

// AddTag adds a tag to the manifest
func (m *Manifest) AddTag(tag string) {
	for _, t := range m.Tags {
		if t == tag {
			return // Tag already exists
		}
	}
	m.Tags = append(m.Tags, tag)
}

// AddTest adds a test case to the manifest
func (m *Manifest) AddTest(test core.TestCase) {
	m.Tests = append(m.Tests, test)
}

// Validate checks if the manifest is valid
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("manifest ID is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	if m.Domain == "" {
		return fmt.Errorf("manifest domain is required")
	}
	if m.Lang == "" {
		return fmt.Errorf("manifest language is required")
	}
	if m.Entry == "" {
		return fmt.Errorf("manifest entry point is required")
	}

	if m.Lang == "wasm" {
		if m.CodePath == "" {
			return fmt.Errorf("WASM artifacts require code_path")
		}
		if m.SHA256 == "" {
			return fmt.Errorf("WASM artifacts require SHA256")
		}
	}

	return nil
}

// ToJSON converts the manifest to JSON
func (m *Manifest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// FromJSON creates a manifest from JSON
func FromJSON(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetArtifactPath returns the path to the artifact directory
func (m *Manifest) GetArtifactPath(baseDir string) string {
	return fmt.Sprintf("%s/%s@%s", baseDir, m.ID, m.Version)
}

// GetManifestPath returns the path to the manifest file
func (m *Manifest) GetManifestPath(baseDir string) string {
	return fmt.Sprintf("%s/manifest.json", m.GetArtifactPath(baseDir))
}

// GetCodePath returns the path to the code file
func (m *Manifest) GetCodePath(baseDir string) string {
	if m.CodePath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", m.GetArtifactPath(baseDir), m.CodePath)
}
