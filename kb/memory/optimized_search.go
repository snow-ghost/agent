package memory

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/snow-ghost/agent/core"
)

// OptimizedFind implements optimized skill finding with caching and parallel search
func (r *Registry) OptimizedFind(task core.Task) []core.Skill {
	// Create cache key based on task characteristics
	cacheKey := r.createCacheKey(task)

	// Check cache first
	r.mu.RLock()
	if cached, found := r.cache.Get(cacheKey); found {
		r.mu.RUnlock()
		return cached
	}
	r.mu.RUnlock()

	// Perform optimized search
	skills := r.performOptimizedSearch(task)

	// Cache the results
	r.mu.Lock()
	r.cache.Add(cacheKey, skills)
	r.mu.Unlock()

	return skills
}

// createCacheKey creates a cache key based on task characteristics
func (r *Registry) createCacheKey(task core.Task) string {
	// Use domain and description for cache key
	key := fmt.Sprintf("%s:%s", task.Domain, task.Description)
	return key
}

// performOptimizedSearch performs optimized skill search with parallel processing
func (r *Registry) performOptimizedSearch(task core.Task) []core.Skill {
	// First, try domain-based filtering
	domainSkills := r.getSkillsByDomain(task.Domain)

	// If we have domain-specific skills, use them
	if len(domainSkills) > 0 {
		return r.filterSkillsParallel(domainSkills, task)
	}

	// Fallback to keyword-based search
	keywordSkills := r.getSkillsByKeywords(task.Description)
	if len(keywordSkills) > 0 {
		return r.filterSkillsParallel(keywordSkills, task)
	}

	// Final fallback to all skills
	return r.filterSkillsParallel(r.skills, task)
}

// getSkillsByDomain returns skills for a specific domain
func (r *Registry) getSkillsByDomain(domain string) []core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills, exists := r.domainIndex[domain]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	result := make([]core.Skill, len(skills))
	copy(result, skills)
	return result
}

// getSkillsByKeywords returns skills matching keywords
func (r *Registry) getSkillsByKeywords(description string) []core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keywords := r.extractKeywords(description)
	var result []core.Skill
	skillSet := make(map[core.Skill]bool)

	for _, keyword := range keywords {
		if skills, exists := r.keywordIndex[keyword]; exists {
			for _, skill := range skills {
				if !skillSet[skill] {
					result = append(result, skill)
					skillSet[skill] = true
				}
			}
		}
	}

	return result
}

// extractKeywords extracts keywords from description
func (r *Registry) extractKeywords(description string) []string {
	words := strings.Fields(strings.ToLower(description))
	var keywords []string

	for _, word := range words {
		// Filter out common words
		if !r.isCommonWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// isCommonWord checks if a word is a common word that should be filtered out
func (r *Registry) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"this": true, "that": true, "these": true, "those": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
	}

	return commonWords[word]
}

// filterSkillsParallel filters skills using parallel processing
func (r *Registry) filterSkillsParallel(skills []core.Skill, task core.Task) []core.Skill {
	if len(skills) == 0 {
		return nil
	}

	// Use worker pool for parallel processing
	numWorkers := 4
	if len(skills) < numWorkers {
		numWorkers = len(skills)
	}

	chunkSize := len(skills) / numWorkers
	if chunkSize == 0 {
		chunkSize = 1
	}

	type result struct {
		skills []core.Skill
		index  int
	}

	results := make(chan result, numWorkers)
	var wg sync.WaitGroup

	// Process skills in parallel
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == numWorkers-1 {
			end = len(skills)
		}

		if start >= len(skills) {
			break
		}

		wg.Add(1)
		go func(chunk []core.Skill, index int) {
			defer wg.Done()

			var matchingSkills []core.Skill
			for _, skill := range chunk {
				if ok, _ := skill.CanSolve(task); ok {
					matchingSkills = append(matchingSkills, skill)
				}
			}

			results <- result{skills: matchingSkills, index: index}
		}(skills[start:end], i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allSkills []core.Skill
	for result := range results {
		allSkills = append(allSkills, result.skills...)
	}

	// Sort by confidence if skills support it
	r.sortSkillsByConfidence(allSkills, task)

	return allSkills
}

// sortSkillsByConfidence sorts skills by confidence score
func (r *Registry) sortSkillsByConfidence(skills []core.Skill, task core.Task) {
	sort.Slice(skills, func(i, j int) bool {
		// Get confidence scores
		_, confI := skills[i].CanSolve(task)
		_, confJ := skills[j].CanSolve(task)

		// Sort by confidence (higher first)
		return confI > confJ
	})
}

// IndexSkill adds a skill to the domain and keyword indexes
func (r *Registry) IndexSkill(skill core.Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add to domain index if skill has domain information
	if domainSkill, ok := skill.(interface{ GetDomain() string }); ok {
		domain := domainSkill.GetDomain()
		if domain != "" {
			r.domainIndex[domain] = append(r.domainIndex[domain], skill)
		}
	}

	// Add to keyword index if skill has keywords
	if keywordSkill, ok := skill.(interface{ GetKeywords() []string }); ok {
		keywords := keywordSkill.GetKeywords()
		for _, keyword := range keywords {
			r.keywordIndex[keyword] = append(r.keywordIndex[keyword], skill)
		}
	}
}

// ClearCache clears the search cache
func (r *Registry) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache.Purge()
}

// GetCacheStats returns cache statistics
func (r *Registry) GetCacheStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"cache_size":    r.cache.Len(),
		"domain_count":  len(r.domainIndex),
		"keyword_count": len(r.keywordIndex),
		"total_skills":  len(r.skills),
	}
}

// BenchmarkSearch performs a benchmark of the search functionality
func (r *Registry) BenchmarkSearch(task core.Task, iterations int) map[string]time.Duration {
	// Warm up
	r.OptimizedFind(task)

	// Benchmark optimized search
	start := time.Now()
	for i := 0; i < iterations; i++ {
		r.OptimizedFind(task)
	}
	optimizedDuration := time.Since(start)

	// Benchmark original search
	start = time.Now()
	for i := 0; i < iterations; i++ {
		r.Find(task)
	}
	originalDuration := time.Since(start)

	return map[string]time.Duration{
		"optimized": optimizedDuration,
		"original":  originalDuration,
		"speedup":   originalDuration - optimizedDuration,
	}
}
