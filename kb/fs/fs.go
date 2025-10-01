package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/snow-ghost/agent/artifact"
)

// KnowledgeBaseFS implements a file-system based knowledge base
type KnowledgeBaseFS struct {
	artifactsDir string
	cache        map[string]*artifact.Manifest
	index        map[string][]*artifact.Manifest // domain -> manifests
	tagIndex     map[string][]*artifact.Manifest // tag -> manifests
	mu           sync.RWMutex
}

// NewKnowledgeBaseFS creates a new file-system based knowledge base
func NewKnowledgeBaseFS(artifactsDir string) *KnowledgeBaseFS {
	kb := &KnowledgeBaseFS{
		artifactsDir: artifactsDir,
		cache:        make(map[string]*artifact.Manifest),
		index:        make(map[string][]*artifact.Manifest),
		tagIndex:     make(map[string][]*artifact.Manifest),
	}

	// Load artifacts on startup
	kb.LoadArtifacts()

	return kb
}

// LoadArtifacts loads all artifacts from the artifacts directory
func (kb *KnowledgeBaseFS) LoadArtifacts() error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	// Clear existing cache
	kb.cache = make(map[string]*artifact.Manifest)
	kb.index = make(map[string][]*artifact.Manifest)
	kb.tagIndex = make(map[string][]*artifact.Manifest)

	// Create artifacts directory if it doesn't exist
	if err := os.MkdirAll(kb.artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Walk through artifact directories
	return filepath.Walk(kb.artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for manifest.json files
		if info.Name() == "manifest.json" {
			if err := kb.loadManifest(path); err != nil {
				fmt.Printf("Warning: failed to load manifest %s: %v\n", path, err)
			}
		}

		return nil
	})
}

// loadManifest loads a single manifest file
func (kb *KnowledgeBaseFS) loadManifest(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest, err := artifact.FromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Verify SHA256 if it's a WASM artifact
	if manifest.Lang == "wasm" && manifest.SHA256 != "" {
		codePath := manifest.GetCodePath(kb.artifactsDir)
		if err := kb.verifySHA256(codePath, manifest.SHA256); err != nil {
			return fmt.Errorf("SHA256 verification failed: %w", err)
		}
	}

	// Add to cache
	key := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
	kb.cache[key] = manifest

	// Add to domain index
	kb.index[manifest.Domain] = append(kb.index[manifest.Domain], manifest)

	// Add to tag index
	for _, tag := range manifest.Tags {
		kb.tagIndex[tag] = append(kb.tagIndex[tag], manifest)
	}

	return nil
}

// verifySHA256 verifies the SHA256 hash of a file
func (kb *KnowledgeBaseFS) verifySHA256(filePath, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// Find finds artifacts by domain
func (kb *KnowledgeBaseFS) Find(domain string) []*artifact.Manifest {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	manifests, exists := kb.index[domain]
	if !exists {
		return []*artifact.Manifest{}
	}

	// Return a copy to avoid race conditions
	result := make([]*artifact.Manifest, len(manifests))
	copy(result, manifests)
	return result
}

// FindByTag finds artifacts by tag
func (kb *KnowledgeBaseFS) FindByTag(tag string) []*artifact.Manifest {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	manifests, exists := kb.tagIndex[tag]
	if !exists {
		return []*artifact.Manifest{}
	}

	// Return a copy to avoid race conditions
	result := make([]*artifact.Manifest, len(manifests))
	copy(result, manifests)
	return result
}

// FindByID finds an artifact by ID and version
func (kb *KnowledgeBaseFS) FindByID(id, version string) *artifact.Manifest {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	key := fmt.Sprintf("%s@%s", id, version)
	return kb.cache[key]
}

// SaveArtifact saves an artifact to the file system
func (kb *KnowledgeBaseFS) SaveArtifact(manifest *artifact.Manifest, code []byte) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Create artifact directory
	artifactDir := manifest.GetArtifactPath(kb.artifactsDir)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Save manifest
	manifestPath := manifest.GetManifestPath(kb.artifactsDir)
	manifestData, err := manifest.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Save code if it's a WASM artifact
	if manifest.Lang == "wasm" && len(code) > 0 {
		codePath := manifest.GetCodePath(kb.artifactsDir)
		if err := os.WriteFile(codePath, code, 0644); err != nil {
			return fmt.Errorf("failed to write code: %w", err)
		}
	}

	// Add to cache
	key := fmt.Sprintf("%s@%s", manifest.ID, manifest.Version)
	kb.cache[key] = manifest

	// Add to domain index
	kb.index[manifest.Domain] = append(kb.index[manifest.Domain], manifest)

	// Add to tag index
	for _, tag := range manifest.Tags {
		kb.tagIndex[tag] = append(kb.tagIndex[tag], manifest)
	}

	return nil
}

// ListArtifacts returns all artifacts
func (kb *KnowledgeBaseFS) ListArtifacts() []*artifact.Manifest {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	result := make([]*artifact.Manifest, 0, len(kb.cache))
	for _, manifest := range kb.cache {
		result = append(result, manifest)
	}

	return result
}

// DeleteArtifact deletes an artifact
func (kb *KnowledgeBaseFS) DeleteArtifact(id, version string) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	key := fmt.Sprintf("%s@%s", id, version)
	manifest, exists := kb.cache[key]
	if !exists {
		return fmt.Errorf("artifact not found: %s@%s", id, version)
	}

	// Remove from file system
	artifactDir := manifest.GetArtifactPath(kb.artifactsDir)
	if err := os.RemoveAll(artifactDir); err != nil {
		return fmt.Errorf("failed to remove artifact directory: %w", err)
	}

	// Remove from cache
	delete(kb.cache, key)

	// Remove from domain index
	manifests := kb.index[manifest.Domain]
	for i, m := range manifests {
		if m.ID == id && m.Version == version {
			kb.index[manifest.Domain] = append(manifests[:i], manifests[i+1:]...)
			break
		}
	}

	// Remove from tag index
	for _, tag := range manifest.Tags {
		manifests := kb.tagIndex[tag]
		for i, m := range manifests {
			if m.ID == id && m.Version == version {
				kb.tagIndex[tag] = append(manifests[:i], manifests[i+1:]...)
				break
			}
		}
	}

	return nil
}

// Search searches for artifacts by query
func (kb *KnowledgeBaseFS) Search(query string) []*artifact.Manifest {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*artifact.Manifest

	for _, manifest := range kb.cache {
		// Search in ID, description, and tags
		if strings.Contains(strings.ToLower(manifest.ID), query) ||
			strings.Contains(strings.ToLower(manifest.Description), query) ||
			strings.Contains(strings.ToLower(manifest.Domain), query) {
			results = append(results, manifest)
			continue
		}

		// Search in tags
		for _, tag := range manifest.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, manifest)
				break
			}
		}
	}

	return results
}
