package streaming

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// SSEWriter handles Server-Sent Events writing
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("response writer does not support flushing")
	}

	return &SSEWriter{
		w:       w,
		flusher: flusher,
	}, nil
}

// WriteEvent writes an SSE event
func (s *SSEWriter) WriteEvent(event string, data interface{}) error {
	// Write event type
	if event != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
			return err
		}
	}

	// Write data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Split data into lines and write each line
	lines := strings.Split(string(jsonData), "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(s.w, "data: %s\n", line); err != nil {
			return err
		}
	}

	// Write empty line to end event
	if _, err := fmt.Fprintf(s.w, "\n"); err != nil {
		return err
	}

	// Flush the data
	s.flusher.Flush()
	return nil
}

// WriteError writes an error event
func (s *SSEWriter) WriteError(err error) error {
	errorData := map[string]interface{}{
		"error": err.Error(),
		"type":  "error",
	}
	return s.WriteEvent("error", errorData)
}

// WriteDone writes a done event with final usage
func (s *SSEWriter) WriteDone(usage core.Usage, cost float64, currency string) error {
	doneData := map[string]interface{}{
		"usage": usage,
		"cost": map[string]interface{}{
			"total":    cost,
			"currency": currency,
		},
		"type": "done",
	}
	return s.WriteEvent("done", doneData)
}

// WriteChunk writes a text chunk
func (s *SSEWriter) WriteChunk(text string) error {
	chunkData := map[string]interface{}{
		"text": text,
		"type": "chunk",
	}
	return s.WriteEvent("chunk", chunkData)
}

// WriteStart writes a start event
func (s *SSEWriter) WriteStart(model string, provider string) error {
	startData := map[string]interface{}{
		"model":    model,
		"provider": provider,
		"type":     "start",
	}
	return s.WriteEvent("start", startData)
}

// Close closes the SSE stream
func (s *SSEWriter) Close() error {
	// Write final newline
	if _, err := fmt.Fprintf(s.w, "\n"); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	Text         string     `json:"text"`
	Usage        core.Usage `json:"usage"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	FinishReason string     `json:"finish_reason"`
	Cost         float64    `json:"cost,omitempty"`
	Currency     string     `json:"currency,omitempty"`
}

// StreamChunk represents a single chunk in the stream
type StreamChunk struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
}

// StreamHandler handles streaming requests
type StreamHandler struct {
	onChunk func(chunk StreamChunk) error
	onDone  func(response StreamResponse) error
	onError func(err error) error
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

// SetChunkHandler sets the chunk handler
func (h *StreamHandler) SetChunkHandler(fn func(chunk StreamChunk) error) {
	h.onChunk = fn
}

// SetDoneHandler sets the done handler
func (h *StreamHandler) SetDoneHandler(fn func(response StreamResponse) error) {
	h.onDone = fn
}

// SetErrorHandler sets the error handler
func (h *StreamHandler) SetErrorHandler(fn func(err error) error) {
	h.onError = fn
}

// HandleChunk handles a chunk
func (h *StreamHandler) HandleChunk(chunk StreamChunk) error {
	if h.onChunk != nil {
		return h.onChunk(chunk)
	}
	return nil
}

// HandleDone handles completion
func (h *StreamHandler) HandleDone(response StreamResponse) error {
	if h.onDone != nil {
		return h.onDone(response)
	}
	return nil
}

// HandleError handles errors
func (h *StreamHandler) HandleError(err error) error {
	if h.onError != nil {
		return h.onError(err)
	}
	return nil
}

// ParseSSEStream parses an SSE stream from a reader
func ParseSSEStream(ctx context.Context, reader *bufio.Reader, handler *StreamHandler) error {
	var currentEvent string
	var currentData strings.Builder

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line indicates end of event
		if line == "" {
			if currentData.Len() > 0 {
				data := currentData.String()
				if err := processEvent(currentEvent, data, handler); err != nil {
					return err
				}
			}
			currentEvent = ""
			currentData.Reset()
			continue
		}

		// Parse event type
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if currentData.Len() > 0 {
				currentData.WriteString("\n")
			}
			currentData.WriteString(data)
			continue
		}

		// Ignore other lines
	}

	return nil
}

// processEvent processes a single SSE event
func processEvent(eventType, data string, handler *StreamHandler) error {
	switch eventType {
	case "chunk":
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("failed to unmarshal chunk: %w", err)
		}
		return handler.HandleChunk(chunk)

	case "done":
		var response StreamResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			return fmt.Errorf("failed to unmarshal done: %w", err)
		}
		return handler.HandleDone(response)

	case "error":
		var errorData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &errorData); err != nil {
			return fmt.Errorf("failed to unmarshal error: %w", err)
		}
		if errorMsg, ok := errorData["error"].(string); ok {
			return handler.HandleError(fmt.Errorf(errorMsg))
		}
		return handler.HandleError(fmt.Errorf("unknown error"))

	default:
		// Ignore unknown event types
		return nil
	}
}

// UsageAggregator aggregates usage statistics during streaming
type UsageAggregator struct {
	usage core.Usage
	mu    sync.RWMutex
}

// NewUsageAggregator creates a new usage aggregator
func NewUsageAggregator() *UsageAggregator {
	return &UsageAggregator{}
}

// AddUsage adds usage to the aggregator
func (u *UsageAggregator) AddUsage(usage core.Usage) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.usage.PromptTokens += usage.PromptTokens
	u.usage.CompletionTokens += usage.CompletionTokens
	u.usage.TotalTokens += usage.TotalTokens
}

// GetUsage returns the aggregated usage
func (u *UsageAggregator) GetUsage() core.Usage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.usage
}

// Reset resets the usage aggregator
func (u *UsageAggregator) Reset() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.usage = core.Usage{}
}
