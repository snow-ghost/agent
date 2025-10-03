package core

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant", "tool"
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Tool represents a tool that can be called
type Tool struct {
	Type     string                 `json:"type"`
	Function *ToolFunction          `json:"function,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolFunction defines a function tool
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function *ToolCallFunction      `json:"function,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCallFunction contains the function call details
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Tools       []Tool            `json:"tools,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Caller      string            `json:"caller,omitempty"` // tenant/project
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Text         string     `json:"text"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        Usage      `json:"usage"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	FinishReason string     `json:"finish_reason"`
}

// CompleteRequest represents a text completion request
type CompleteRequest struct {
	Model       string            `json:"model"`
	Prompt      string            `json:"prompt"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Caller      string            `json:"caller,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CompleteResponse represents a text completion response
type CompleteResponse struct {
	Text         string            `json:"text"`
	Usage        Usage             `json:"usage"`
	Model        string            `json:"model"`
	Provider     string            `json:"provider"`
	FinishReason string            `json:"finish_reason"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// EmbedRequest represents an embedding request
type EmbedRequest struct {
	Model    string            `json:"model"`
	Input    []string          `json:"input"`
	Caller   string            `json:"caller,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmbedResponse represents an embedding response
type EmbedResponse struct {
	Data     []Embedding       `json:"data"`
	Usage    Usage             `json:"usage"`
	Model    string            `json:"model"`
	Provider string            `json:"provider"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Embedding represents a single embedding
type Embedding struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// Model represents an available model
type Model struct {
	ID       string            `json:"id"`
	Provider string            `json:"provider"`
	Type     string            `json:"type"` // "chat", "complete", "embed"
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ModelsResponse represents the response for listing models
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// CostEntry represents a cost entry for billing
type CostEntry struct {
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	Caller           string  `json:"caller"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`
	Timestamp        string  `json:"timestamp"`
}

// CostsRequest represents a request for cost information
type CostsRequest struct {
	From    string `json:"from,omitempty"`     // ISO date
	To      string `json:"to,omitempty"`       // ISO date
	GroupBy string `json:"group_by,omitempty"` // "provider", "model", "caller"
}

// CostsResponse represents the response for cost information
type CostsResponse struct {
	Costs   []CostEntry            `json:"costs"`
	Summary map[string]interface{} `json:"summary,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Data    interface{} `json:"data"`
	Usage   *Usage      `json:"usage,omitempty"`
}
