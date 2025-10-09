package memory

import (
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_OptimizedFind(t *testing.T) {
	// Create a test registry
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		cache:        nil, // Will be created in the function
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
		cacheTTL:     5 * time.Minute,
	}

	// Add some test skills
	skills := []core.Skill{
		{
			ID:          "skill1",
			Domain:      "algorithms.sorting",
			Description: "Bubble sort algorithm",
			Keywords:    []string{"sort", "bubble", "algorithm"},
		},
		{
			ID:          "skill2",
			Domain:      "algorithms.searching",
			Description: "Binary search algorithm",
			Keywords:    []string{"search", "binary", "algorithm"},
		},
		{
			ID:          "skill3",
			Domain:      "algorithms.sorting",
			Description: "Quick sort algorithm",
			Keywords:    []string{"sort", "quick", "algorithm"},
		},
	}

	// Add skills to registry
	for _, skill := range skills {
		registry.skills = append(registry.skills, skill)
	}

	// Build indexes
	registry.buildIndexes()

	tests := []struct {
		name     string
		task     core.Task
		expected []string // Expected skill IDs
	}{
		{
			name: "find by domain",
			task: core.Task{
				Domain: "algorithms.sorting",
			},
			expected: []string{"skill1", "skill3"},
		},
		{
			name: "find by keyword",
			task: core.Task{
				Description: "I need to search for something",
			},
			expected: []string{"skill2"},
		},
		{
			name: "find by multiple keywords",
			task: core.Task{
				Description: "I need to sort data using bubble sort",
			},
			expected: []string{"skill1"},
		},
		{
			name: "no matches",
			task: core.Task{
				Domain: "nonexistent.domain",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := registry.OptimizedFind(tt.task)

			// Extract skill IDs from results
			skillIDs := make([]string, len(results))
			for i, skill := range results {
				skillIDs[i] = skill.ID
			}

			assert.ElementsMatch(t, tt.expected, skillIDs)
		})
	}
}

func TestRegistry_buildIndexes(t *testing.T) {
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	// Add test skills
	skills := []core.Skill{
		{
			ID:          "skill1",
			Domain:      "algorithms.sorting",
			Description: "Bubble sort",
			Keywords:    []string{"sort", "bubble"},
		},
		{
			ID:          "skill2",
			Domain:      "algorithms.searching",
			Description: "Binary search",
			Keywords:    []string{"search", "binary"},
		},
		{
			ID:          "skill3",
			Domain:      "algorithms.sorting",
			Description: "Quick sort",
			Keywords:    []string{"sort", "quick"},
		},
	}

	registry.skills = skills
	registry.buildIndexes()

	// Check domain index
	assert.Len(t, registry.domainIndex["algorithms.sorting"], 2)
	assert.Len(t, registry.domainIndex["algorithms.searching"], 1)

	// Check keyword index
	assert.Len(t, registry.keywordIndex["sort"], 2)
	assert.Len(t, registry.keywordIndex["search"], 1)
	assert.Len(t, registry.keywordIndex["bubble"], 1)
	assert.Len(t, registry.keywordIndex["quick"], 1)
	assert.Len(t, registry.keywordIndex["binary"], 1)
}

func TestRegistry_createCacheKey(t *testing.T) {
	registry := &Registry{}

	task := core.Task{
		ID:          "task1",
		Domain:      "algorithms.sorting",
		Description: "Sort an array of numbers",
		Spec: core.Spec{
			SuccessCriteria: []string{"sorted"},
			Props:           map[string]string{"type": "sort"},
		},
	}

	key := registry.createCacheKey(task)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "task1")
	assert.Contains(t, key, "algorithms.sorting")
}

func TestRegistry_GetCacheStats(t *testing.T) {
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	// Add some skills
	skills := []core.Skill{
		{ID: "skill1", Domain: "domain1"},
		{ID: "skill2", Domain: "domain2"},
	}
	registry.skills = skills
	registry.buildIndexes()

	stats := registry.GetCacheStats()

	assert.Equal(t, 0, stats["cache_size"]) // No cache yet
	assert.Equal(t, 2, stats["domain_count"])
	assert.Equal(t, 0, stats["keyword_count"]) // No keywords in test skills
	assert.Equal(t, 2, stats["total_skills"])
}

func TestRegistry_OptimizedFind_WithCache(t *testing.T) {
	// This test would require setting up a proper cache
	// For now, we'll test the basic functionality without cache
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	// Add a test skill
	skill := core.Skill{
		ID:          "test-skill",
		Domain:      "test.domain",
		Description: "Test skill",
		Keywords:    []string{"test"},
	}
	registry.skills = append(registry.skills, skill)
	registry.buildIndexes()

	task := core.Task{
		Domain: "test.domain",
	}

	results := registry.OptimizedFind(task)
	assert.Len(t, results, 1)
	assert.Equal(t, "test-skill", results[0].ID)
}

func TestRegistry_OptimizedFind_EmptyRegistry(t *testing.T) {
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	task := core.Task{
		Domain: "any.domain",
	}

	results := registry.OptimizedFind(task)
	assert.Empty(t, results)
}

func TestRegistry_OptimizedFind_CaseInsensitive(t *testing.T) {
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	// Add skill with lowercase keywords
	skill := core.Skill{
		ID:          "test-skill",
		Domain:      "test.domain",
		Description: "Test skill",
		Keywords:    []string{"test", "algorithm"},
	}
	registry.skills = append(registry.skills, skill)
	registry.buildIndexes()

	// Search with uppercase description
	task := core.Task{
		Description: "I need to TEST something",
	}

	results := registry.OptimizedFind(task)
	assert.Len(t, results, 1)
	assert.Equal(t, "test-skill", results[0].ID)
}

func TestRegistry_OptimizedFind_MultipleMatches(t *testing.T) {
	registry := &Registry{
		skills:       make([]core.Skill, 0),
		domainIndex:  make(map[string][]core.Skill),
		keywordIndex: make(map[string][]core.Skill),
	}

	// Add multiple skills with same keyword
	skills := []core.Skill{
		{
			ID:          "skill1",
			Domain:      "domain1",
			Description: "First skill",
			Keywords:    []string{"common"},
		},
		{
			ID:          "skill2",
			Domain:      "domain2",
			Description: "Second skill",
			Keywords:    []string{"common"},
		},
	}

	for _, skill := range skills {
		registry.skills = append(registry.skills, skill)
	}
	registry.buildIndexes()

	task := core.Task{
		Description: "I need something common",
	}

	results := registry.OptimizedFind(task)
	assert.Len(t, results, 2)

	// Check that both skills are returned
	skillIDs := make([]string, len(results))
	for i, skill := range results {
		skillIDs[i] = skill.ID
	}
	assert.Contains(t, skillIDs, "skill1")
	assert.Contains(t, skillIDs, "skill2")
}
