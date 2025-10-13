package design

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Default maximum code size in bytes (64KB)
const DefaultMaxCodeBytes = 64 * 1024

// Validate performs comprehensive validation of a HypothesisDesign
func Validate(h HypothesisDesign) error {
	// Check status
	if h.Status != "ok" && h.Status != "cannot_solve" {
		return errors.New("status must be 'ok' or 'cannot_solve'")
	}

	// If status is "cannot_solve", reason should be provided
	if h.Status == "cannot_solve" && h.Reason == "" {
		return errors.New("reason is required when status is 'cannot_solve'")
	}

	// If status is "ok", validate the rest of the structure
	if h.Status == "ok" {
		if err := validateAlgorithm(h.Algorithm); err != nil {
			return fmt.Errorf("algorithm validation failed: %w", err)
		}

		if err := validateCode(h.Code); err != nil {
			return fmt.Errorf("code validation failed: %w", err)
		}

		if err := validateEvaluation(h.Evaluation); err != nil {
			return fmt.Errorf("evaluation validation failed: %w", err)
		}

		if err := validateTests(h.Tests); err != nil {
			return fmt.Errorf("tests validation failed: %w", err)
		}
	}

	return nil
}

func validateAlgorithm(alg struct {
	Name       string `json:"name"`
	Idea       string `json:"idea"`
	Complexity struct {
		Time  string `json:"time"`
		Space string `json:"space"`
	} `json:"complexity"`
}) error {
	if alg.Name == "" {
		return errors.New("algorithm name cannot be empty")
	}
	if alg.Idea == "" {
		return errors.New("algorithm idea cannot be empty")
	}
	if alg.Complexity.Time == "" {
		return errors.New("algorithm complexity time cannot be empty")
	}
	if alg.Complexity.Space == "" {
		return errors.New("algorithm complexity space cannot be empty")
	}
	return nil
}

func validateCode(code struct {
	Lang  string `json:"lang"`  // "af-dsl" | "wasm"
	Entry string `json:"entry"` // "program"
	Src   string `json:"src"`   // S-expr or base64 wasm
}) error {
	// Check language
	if code.Lang != "af-dsl" && code.Lang != "wasm" {
		return errors.New("code lang must be 'af-dsl' or 'wasm'")
	}

	// Check entry point
	if code.Entry != "program" {
		return errors.New("code entry must be 'program'")
	}

	// Check source code
	if code.Src == "" {
		return errors.New("code src cannot be empty")
	}

	// Check maximum code size
	maxCodeBytes := getMaxCodeBytes()
	if len(code.Src) > maxCodeBytes {
		return fmt.Errorf("code src exceeds maximum size of %d bytes", maxCodeBytes)
	}

	// For wasm, validate base64 encoding
	if code.Lang == "wasm" {
		if err := validateWasmCode(code.Src); err != nil {
			return fmt.Errorf("wasm code validation failed: %w", err)
		}
	}

	return nil
}

func validateWasmCode(src string) error {
	// Check if it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return errors.New("wasm code must be valid base64 encoded")
	}

	// Check for forbidden patterns (file paths, network calls, etc.)
	forbiddenPatterns := []string{
		"../", "./", "file://", "http://", "https://",
		"localhost", "127.0.0.1", "0.0.0.0",
	}

	lowerDecoded := strings.ToLower(string(decoded))
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(lowerDecoded, pattern) {
			return fmt.Errorf("wasm code contains forbidden pattern: %s", pattern)
		}
	}

	return nil
}

func validateEvaluation(eval struct {
	Metrics       []string `json:"metrics"`
	Fitness       string   `json:"fitness"`
	PassThreshold float64  `json:"pass_threshold"`
}) error {
	// Check pass threshold is in [0,1]
	if eval.PassThreshold < 0 || eval.PassThreshold > 1 {
		return errors.New("pass_threshold must be between 0 and 1")
	}

	// Check metrics includes "correctness"
	hasCorrectness := false
	for _, metric := range eval.Metrics {
		if metric == "correctness" {
			hasCorrectness = true
			break
		}
	}
	if !hasCorrectness {
		return errors.New("metrics must include 'correctness'")
	}

	// Check fitness formula is not empty
	if eval.Fitness == "" {
		return errors.New("fitness formula cannot be empty")
	}

	return nil
}

func validateTests(tests struct {
	Unit []struct {
		Name   string  `json:"name"`
		Input  string  `json:"input"`
		Oracle string  `json:"oracle"`
		Weight float64 `json:"weight"`
	} `json:"unit"`
	Property []struct {
		Name      string   `json:"name"`
		Generator string   `json:"generator"`
		Checks    []string `json:"checks"`
	} `json:"property"`
}) error {
	// Check unit tests have positive total weight
	totalWeight := 0.0
	for _, unit := range tests.Unit {
		if unit.Name == "" {
			return errors.New("unit test name cannot be empty")
		}
		if unit.Input == "" {
			return errors.New("unit test input cannot be empty")
		}
		if unit.Oracle == "" {
			return errors.New("unit test oracle cannot be empty")
		}
		if unit.Weight < 0 {
			return errors.New("unit test weight cannot be negative")
		}
		totalWeight += unit.Weight
	}

	if totalWeight <= 0 {
		return errors.New("unit tests must have total weight > 0")
	}

	// Check property tests
	for _, prop := range tests.Property {
		if prop.Name == "" {
			return errors.New("property test name cannot be empty")
		}
		if prop.Generator == "" {
			return errors.New("property test generator cannot be empty")
		}
		if len(prop.Checks) == 0 {
			return errors.New("property test must have at least one check")
		}
	}

	return nil
}

// getMaxCodeBytes returns the maximum allowed code size from environment variable or default
func getMaxCodeBytes() int {
	if maxBytesStr := os.Getenv("MAX_CODE_BYTES"); maxBytesStr != "" {
		if maxBytes, err := strconv.Atoi(maxBytesStr); err == nil && maxBytes > 0 {
			return maxBytes
		}
	}
	return DefaultMaxCodeBytes
}
