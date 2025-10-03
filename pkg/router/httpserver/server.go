package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/snow-ghost/agent/pkg/router/core"
)

// Server represents the HTTP server
type Server struct {
	port   string
	logger *slog.Logger
	router *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(port string, logger *slog.Logger) *Server {
	s := &Server{
		port:   port,
		logger: logger,
		router: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all the HTTP routes
func (s *Server) setupRoutes() {
	// Health and metrics
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/metrics", s.handleMetrics)

	// API v1 routes
	v1 := http.NewServeMux()
	v1.HandleFunc("/chat", s.handleChat)
	v1.HandleFunc("/chat/stream", s.handleChatStream)
	v1.HandleFunc("/complete", s.handleComplete)
	v1.HandleFunc("/embed", s.handleEmbed)
	v1.HandleFunc("/models", s.handleModels)
	v1.HandleFunc("/costs", s.handleCosts)

	s.router.Handle("/v1/", http.StripPrefix("/v1", v1))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "port", s.port)
	return http.ListenAndServe(":"+s.port, s.router)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"llmrouter","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleMetrics handles metrics requests
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	// Basic Prometheus metrics
	fmt.Fprintf(w, "# HELP llmrouter_requests_total Total number of requests\n")
	fmt.Fprintf(w, "# TYPE llmrouter_requests_total counter\n")
	fmt.Fprintf(w, "llmrouter_requests_total 0\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "# HELP llmrouter_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE llmrouter_uptime_seconds gauge\n")
	fmt.Fprintf(w, "llmrouter_uptime_seconds 0\n")
}

// handleChat handles chat completion requests
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req core.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	response := core.ChatResponse{
		Text: "Hello! This is a mock response from the LLM router.",
		Usage: core.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
		Model:        req.Model,
		Provider:     "mock",
		FinishReason: "stop",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChatStream handles streaming chat completion requests
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if streaming is requested
	stream := r.URL.Query().Get("stream")
	if stream != "1" {
		// Redirect to non-streaming endpoint
		s.handleChat(w, r)
		return
	}

	var req core.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send streaming response
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Mock streaming response
	chunks := []string{"Hello", "! This", " is a", " mock", " streaming", " response", "."}

	for i, chunk := range chunks {
		streamChunk := core.StreamChunk{
			ID:      fmt.Sprintf("chunk-%d", i),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Data: map[string]interface{}{
				"delta": map[string]string{
					"content": chunk,
				},
			},
		}

		data, _ := json.Marshal(streamChunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond) // Simulate streaming delay
	}

	// Send final chunk with usage
	finalChunk := core.StreamChunk{
		ID:      "chunk-final",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Data: map[string]interface{}{
			"delta": map[string]string{
				"finish_reason": "stop",
			},
		},
		Usage: &core.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}

	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleComplete handles text completion requests
func (s *Server) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req core.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	response := core.CompleteResponse{
		Text: "This is a mock completion response.",
		Usage: core.Usage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
		},
		Model:        req.Model,
		Provider:     "mock",
		FinishReason: "stop",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEmbed handles embedding requests
func (s *Server) handleEmbed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req core.EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	embeddings := make([]core.Embedding, len(req.Input))
	for i := range req.Input {
		// Generate mock embedding (1536 dimensions)
		embedding := make([]float32, 1536)
		for j := range embedding {
			embedding[j] = float32(i+j) / 1000.0 // Mock values
		}
		embeddings[i] = core.Embedding{
			Index:     i,
			Embedding: embedding,
		}
	}

	response := core.EmbedResponse{
		Data:     embeddings,
		Usage:    core.Usage{TotalTokens: len(req.Input) * 10},
		Model:    req.Model,
		Provider: "mock",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModels handles model listing requests
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, return an empty list as requested
	response := core.ModelsResponse{
		Models: []core.Model{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCosts handles cost information requests
func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	groupBy := r.URL.Query().Get("groupBy")

	// For now, return empty costs
	response := core.CostsResponse{
		Costs: []core.CostEntry{},
		Summary: map[string]interface{}{
			"from":       from,
			"to":         to,
			"group_by":   groupBy,
			"total_cost": 0.0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, message, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := core.ErrorResponse{
		Error: message,
		Code:  code,
	}

	json.NewEncoder(w).Encode(errorResp)
}
