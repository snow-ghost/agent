package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/snow-ghost/agent/design"
	"github.com/snow-ghost/agent/pkg/router/core"
	"github.com/snow-ghost/agent/prompts"
)

// Designer interface for algorithm design operations
type Designer interface {
	Design(ctx context.Context, taskJSON string) (design.HypothesisDesign, []byte, error)
}

// ExtractFirstJSONObject extracts the first valid JSON object from potentially dirty input
// Handles markdown code blocks, text before/after JSON, and malformed responses
func ExtractFirstJSONObject(data []byte) ([]byte, error) {
	content := string(data)

	// Remove markdown code blocks
	markdownRegex := regexp.MustCompile("```(?:json)?\\s*\\n?(.*?)\\n?```")
	matches := markdownRegex.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		// Use the first code block content
		content = matches[0][1]
	}

	// Find the first '{' character
	start := strings.Index(content, "{")
	if start == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// Extract from the first '{' to the end
	jsonCandidate := content[start:]

	// Use bracket balancing to find the end of the JSON object
	jsonBytes, err := extractBalancedJSON([]byte(jsonCandidate))
	if err != nil {
		return nil, fmt.Errorf("failed to extract balanced JSON: %w", err)
	}

	// Validate that it's valid JSON
	var testObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &testObj); err != nil {
		return nil, fmt.Errorf("extracted content is not valid JSON: %w", err)
	}

	return jsonBytes, nil
}

// extractBalancedJSON finds the first complete JSON object using bracket balancing
func extractBalancedJSON(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// Skip whitespace
	start := 0
	for start < len(data) && unicode.IsSpace(rune(data[start])) {
		start++
	}

	if start >= len(data) || data[start] != '{' {
		return nil, fmt.Errorf("input does not start with '{'")
	}

	// Track bracket balance
	balance := 0
	inString := false
	escapeNext := false

	for i := start; i < len(data); i++ {
		char := data[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				balance++
			} else if char == '}' {
				balance--
				if balance == 0 {
					// Found the end of the JSON object
					return data[start : i+1], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unbalanced brackets in JSON")
}

// DesignClient represents a client for design operations
type DesignClient struct {
	BaseURL string
	Model   string
	Caller  string
	HTTP    *http.Client
}

// NewDesignClient creates a new design client
func NewDesignClient(baseURL, model, caller string, httpClient *http.Client) *DesignClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &DesignClient{
		BaseURL: baseURL,
		Model:   model,
		Caller:  caller,
		HTTP:    httpClient,
	}
}

// NewDesignClientFromEnv creates a design client from environment variables
func NewDesignClientFromEnv() *DesignClient {
	baseURL := os.Getenv("LLM_ROUTER_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "lmstudio:qwen3-4b-2507"
	}

	caller := os.Getenv("LLM_CALLER")
	if caller == "" {
		caller = "design-client"
	}

	return NewDesignClient(baseURL, model, caller, nil)
}

// DesignRequest represents a design request to the LLM router
type DesignRequest struct {
	Model          string            `json:"model"`
	Messages       []core.Message    `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Caller         string            `json:"caller,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DesignResponse represents a design response from the LLM router
type DesignResponse struct {
	Text         string     `json:"text"`
	Usage        core.Usage `json:"usage"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	FinishReason string     `json:"finish_reason"`
	RawResponse  []byte     `json:"-"` // Store raw response for debugging
}

// Design sends a design request to the LLM router and returns a validated HypothesisDesign
func (c *DesignClient) Design(ctx context.Context, taskJSON string) (design.HypothesisDesign, []byte, error) {
	// Create the design request
	req := DesignRequest{
		Model: c.Model,
		Messages: []core.Message{
			{
				Role:    "system",
				Content: prompts.SystemPrompt,
			},
			{
				Role:    "user",
				Content: taskJSON,
			},
		},
		ResponseFormat: map[string]string{
			"type": "json",
		},
		Caller: c.Caller,
		Metadata: map[string]string{
			"task_domain": "algorithm_design",
		},
	}

	// Marshal request
	reqData, err := json.Marshal(req)
	if err != nil {
		return design.HypothesisDesign{}, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/chat", strings.NewReader(string(reqData)))
	if err != nil {
		return design.HypothesisDesign{}, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Caller", c.Caller)

	// Send request
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return design.HypothesisDesign{}, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read raw response
	rawResponse, err := readAllBytes(resp.Body)
	if err != nil {
		return design.HypothesisDesign{}, nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return design.HypothesisDesign{}, rawResponse, fmt.Errorf("LLM router returned status %d: %s", resp.StatusCode, string(rawResponse))
	}

	// Parse response
	var designResp DesignResponse
	if err := json.Unmarshal(rawResponse, &designResp); err != nil {
		return design.HypothesisDesign{}, rawResponse, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract and sanitize JSON from the response text
	jsonBytes, err := ExtractFirstJSONObject([]byte(designResp.Text))
	if err != nil {
		return design.HypothesisDesign{}, rawResponse, fmt.Errorf("failed to extract JSON from response: %w", err)
	}
	jsonStr := string(jsonBytes)

	// Parse the sanitized JSON into HypothesisDesign
	var hypothesis design.HypothesisDesign
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&hypothesis); err != nil {
		return design.HypothesisDesign{}, rawResponse, fmt.Errorf("failed to parse design JSON: %w", err)
	}

	// Validate the design
	if err := design.Validate(hypothesis); err != nil {
		return design.HypothesisDesign{}, rawResponse, fmt.Errorf("design validation failed: %w", err)
	}

	return hypothesis, rawResponse, nil
}

// readAllBytes reads all bytes from a reader (helper function)
func readAllBytes(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	return result, nil
}

// Ensure DesignClient implements Designer interface
var _ Designer = (*DesignClient)(nil)
