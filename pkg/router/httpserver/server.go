package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/snow-ghost/agent/pkg/accounting"
	"github.com/snow-ghost/agent/pkg/cache"
	"github.com/snow-ghost/agent/pkg/cost"
	"github.com/snow-ghost/agent/pkg/limiter"
	"github.com/snow-ghost/agent/pkg/observability"
	"github.com/snow-ghost/agent/pkg/providers"
	"github.com/snow-ghost/agent/pkg/registry"
	"github.com/snow-ghost/agent/pkg/router/core"
	"github.com/snow-ghost/agent/pkg/routing"
	"github.com/snow-ghost/agent/pkg/streaming"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Server represents the HTTP server
type Server struct {
	port              string
	logger            *slog.Logger
	router            *http.ServeMux
	registry          *registry.Registry
	costCalculator    *cost.Calculator
	modelRouter       *routing.ModelRouter
	protectionManager *limiter.ProtectionManager
	cacheManager      *cache.CacheManager
	observability     *observability.Manager
	accounting        *accounting.Manager
}

// NewServer creates a new HTTP server
func NewServer(port string, logger *slog.Logger) *Server {
	// Load registry
	loader := registry.NewLoader("")
	reg, err := loader.LoadRegistry()
	if err != nil {
		logger.Warn("failed to load registry, using default", "error", err)
		reg = registry.GetDefaultRegistry()
	}

	// Create cache manager
	cacheConfig := cache.DefaultCacheConfig()
	cacheManager, err := cache.NewCacheManager(cacheConfig)
	if err != nil {
		logger.Warn("failed to create cache manager, caching disabled", "error", err)
		cacheManager = nil
	}

	// Create observability manager
	obsConfig := observability.Config{
		ServiceName:    "llmrouter",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		JaegerEndpoint: "http://localhost:14268/api/traces",
		LogLevel:       "info",
		LogFormat:      "json",
	}

	obsManager, err := observability.NewManager(obsConfig)
	if err != nil {
		logger.Warn("failed to create observability manager, using basic logging", "error", err)
		obsManager = nil
	}

	// Create accounting manager
	accountingConfig := accounting.Config{
		UseSQLite: false, // Use in-memory for now
		DBPath:    "costs.db",
	}

	accountingManager, err := accounting.NewManager(accountingConfig)
	if err != nil {
		logger.Warn("failed to create accounting manager, cost tracking disabled", "error", err)
		accountingManager = nil
	}

	s := &Server{
		port:              port,
		logger:            logger,
		router:            http.NewServeMux(),
		registry:          reg,
		costCalculator:    cost.NewCalculator(reg),
		modelRouter:       routing.NewModelRouter(reg),
		protectionManager: limiter.NewProtectionManager(reg),
		cacheManager:      cacheManager,
		observability:     obsManager,
		accounting:        accountingManager,
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
	v1.HandleFunc("/strategies", s.handleStrategies)
	v1.HandleFunc("/protection", s.handleProtection)
	v1.HandleFunc("/cache", s.handleCache)

	s.router.Handle("/v1/", http.StripPrefix("/v1", v1))
}

// observabilityMiddleware adds observability to HTTP requests
func (s *Server) observabilityMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to context
		ctx := observability.WithRequestID(r.Context(), requestID)
		ctx = observability.WithCaller(ctx, r.Header.Get("X-Caller"))

		// Start span if observability is available
		if s.observability != nil {
			var span trace.Span
			ctx, span = s.observability.GetTracer().StartSpan(ctx, "http.request")
			defer span.End()

			// Add span attributes
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("request_id", requestID),
			)
		}

		// Create response writer wrapper
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Record metrics and logs
		duration := time.Since(start)
		if s.observability != nil {
			s.observability.GetLogger().LogRequest(
				ctx, r.Method, r.URL.Path, wrapped.statusCode, duration, requestID,
			)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
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
	fmt.Fprintf(w, "\n")

	// LLM-specific metrics
	fmt.Fprintf(w, "# HELP llm_requests_total Total number of LLM requests\n")
	fmt.Fprintf(w, "# TYPE llm_requests_total counter\n")
	fmt.Fprintf(w, "llm_requests_total{provider=\"mock\",model=\"mock\",status=\"success\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_latency_seconds LLM request latency in seconds\n")
	fmt.Fprintf(w, "# TYPE llm_latency_seconds histogram\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.005\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.01\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.025\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.05\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.1\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.25\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"0.5\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"1\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"2.5\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"5\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"10\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_bucket{provider=\"mock\",model=\"mock\",le=\"+Inf\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_sum{provider=\"mock\",model=\"mock\"} 0\n")
	fmt.Fprintf(w, "llm_latency_seconds_count{provider=\"mock\",model=\"mock\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_tokens_input_total Total number of input tokens processed\n")
	fmt.Fprintf(w, "# TYPE llm_tokens_input_total counter\n")
	fmt.Fprintf(w, "llm_tokens_input_total{provider=\"mock\",model=\"mock\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_tokens_output_total Total number of output tokens generated\n")
	fmt.Fprintf(w, "# TYPE llm_tokens_output_total counter\n")
	fmt.Fprintf(w, "llm_tokens_output_total{provider=\"mock\",model=\"mock\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_cost_total Total cost of LLM requests\n")
	fmt.Fprintf(w, "# TYPE llm_cost_total counter\n")
	fmt.Fprintf(w, "llm_cost_total{provider=\"mock\",model=\"mock\",currency=\"USD\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_cache_hits_total Total number of cache hits\n")
	fmt.Fprintf(w, "# TYPE llm_cache_hits_total counter\n")
	fmt.Fprintf(w, "llm_cache_hits_total 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_cache_misses_total Total number of cache misses\n")
	fmt.Fprintf(w, "# TYPE llm_cache_misses_total counter\n")
	fmt.Fprintf(w, "llm_cache_misses_total 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_retries_total Total number of retries\n")
	fmt.Fprintf(w, "# TYPE llm_retries_total counter\n")
	fmt.Fprintf(w, "llm_retries_total{provider=\"mock\",model=\"mock\",reason=\"429\"} 0\n")
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP llm_circuit_open_total Total number of circuit breaker opens\n")
	fmt.Fprintf(w, "# TYPE llm_circuit_open_total counter\n")
	fmt.Fprintf(w, "llm_circuit_open_total{provider=\"mock\",model=\"mock\"} 0\n")
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

	ctx := r.Context()
	requestID := observability.GetRequestIDFromContext(ctx)
	caller := observability.GetCallerFromContext(ctx)

	// Check budget if specified
	if s.accounting != nil {
		budgetHeader := r.Header.Get("X-Budget-Amount")
		if budgetHeader != "" {
			budgetInfo, err := s.accounting.CheckBudget(caller, budgetHeader)
			if err != nil {
				s.logger.Warn("failed to check budget", "error", err, "request_id", requestID)
			} else if budgetInfo.Exceeded {
				s.writeError(w, "Budget exceeded", "BUDGET_EXCEEDED", http.StatusPaymentRequired)
				return
			}
		}
	}

	// Check if caching is enabled and not streaming
	cacheEnabled := req.Metadata["cache"] == "true" && !req.Stream
	var cacheReq cache.CacheRequest
	if cacheEnabled && s.cacheManager != nil {
		cacheReq = cache.CacheRequest{
			Model:       req.Model,
			Messages:    req.Messages,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   req.MaxTokens,
			Tools:       req.Tools,
			Metadata:    req.Metadata,
			Cache:       true,
		}

		// Check cache first
		if entry, exists := s.cacheManager.Get(cacheReq); exists {
			if s.observability != nil {
				s.observability.RecordCacheMetrics(true)
				s.observability.LogCacheOperation(ctx, "get", true, requestID)
			}
			s.logger.Info("cache hit", "model", req.Model, "request_id", requestID)
			s.addCostHeaders(w, entry.Response.Model, entry.Response.Usage)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			json.NewEncoder(w).Encode(entry.Response)
			return
		}
	}

	// Select model using routing strategy
	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		strategy = "tag-based" // Default strategy
	}

	selectedModel, err := s.modelRouter.SelectModel(ctx, strategy, req.Metadata)
	if err != nil {
		s.logger.Error("model selection failed", "error", err, "strategy", strategy, "request_id", requestID)
		s.writeError(w, "Model selection failed", "MODEL_SELECTION_FAILED", http.StatusInternalServerError)
		return
	}

	// Start LLM request span
	var span trace.Span
	if s.observability != nil {
		ctx, span = s.observability.StartRequestSpan(ctx, caller, selectedModel.ID, selectedModel.Provider, requestID)
		defer span.End()
	}

	s.logger.Info("model selected", "model", selectedModel.ID, "strategy", strategy, "domain", req.Metadata["task_domain"], "request_id", requestID)

	// For now, return a mock response with selected model info
	response := core.ChatResponse{
		Text: fmt.Sprintf("Hello! This is a mock response from the LLM router using model %s (strategy: %s).", selectedModel.ID, strategy),
		Usage: core.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
		Model:        selectedModel.ID,
		Provider:     selectedModel.Provider,
		FinishReason: "stop",
	}

	// Calculate cost
	costResult, err := s.costCalculator.CalcCostForModel(selectedModel.ID, response.Usage)
	if err != nil {
		s.logger.Warn("failed to calculate cost", "error", err, "request_id", requestID)
		costResult = &cost.CostResult{TotalCost: 0, Currency: "USD"}
	}

	// Record metrics and logs
	if s.observability != nil {
		s.observability.RecordRequestMetrics(
			selectedModel.Provider, selectedModel.ID, "success",
			time.Since(time.Now()), // Mock duration
			response.Usage.PromptTokens, response.Usage.CompletionTokens,
			costResult.TotalCost, costResult.Currency,
		)
		s.observability.LogRequestCompletion(
			ctx, selectedModel.Provider, selectedModel.ID, "success",
			time.Since(time.Now()), response.Usage.TotalTokens, costResult.TotalCost, requestID,
		)
	}

	// Record cost in accounting
	if s.accounting != nil {
		err := s.accounting.RecordLLMCost(
			caller, selectedModel.Provider, selectedModel.ID, requestID,
			response.Usage.PromptTokens, response.Usage.CompletionTokens,
			costResult.InputCost, costResult.OutputCost, costResult.TotalCost,
			costResult.Currency,
		)
		if err != nil {
			s.logger.Warn("failed to record cost", "error", err, "request_id", requestID)
		}
	}

	// Cache the response if enabled
	if cacheEnabled && s.cacheManager != nil {
		if err := s.cacheManager.Set(cacheReq, response); err != nil {
			s.logger.Warn("failed to cache response", "error", err, "request_id", requestID)
		} else {
			s.logger.Info("response cached", "model", selectedModel.ID, "request_id", requestID)
		}
		w.Header().Set("X-Cache", "MISS")
	} else {
		w.Header().Set("X-Cache", "DISABLED")
	}

	// Add cost headers
	s.addCostHeaders(w, selectedModel.ID, response.Usage)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChatStream handles streaming chat completion requests
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req core.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Create SSE writer
	sseWriter, err := streaming.NewSSEWriter(w)
	if err != nil {
		s.logger.Error("failed to create SSE writer", "error", err)
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Select model using routing strategy
	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		strategy = "tag-based" // Default strategy
	}

	selectedModel, err := s.modelRouter.SelectModel(r.Context(), strategy, req.Metadata)
	if err != nil {
		s.logger.Error("model selection failed", "error", err, "strategy", strategy)
		sseWriter.WriteError(fmt.Errorf("model selection failed: %w", err))
		return
	}

	s.logger.Info("streaming model selected", "model", selectedModel.ID, "strategy", strategy, "domain", req.Metadata["task_domain"])

	// Create mock streaming provider for now
	provider := providers.NewMockStreamingProvider()

	// Perform streaming chat
	if err := provider.ChatStream(r.Context(), *selectedModel, req, sseWriter); err != nil {
		s.logger.Error("streaming chat failed", "error", err)
		sseWriter.WriteError(err)
		return
	}

	// Close the stream
	sseWriter.Close()
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

	// Convert registry models to API models
	var models []core.Model
	for _, modelConfig := range s.registry.Models {
		model := core.Model{
			ID:       modelConfig.ID,
			Provider: modelConfig.Provider,
			Type:     modelConfig.Kind,
			Metadata: map[string]string{
				"base_url":      modelConfig.BaseURL,
				"api_key_env":   modelConfig.APIKeyEnv,
				"currency":      modelConfig.Pricing.Currency,
				"input_per_1k":  fmt.Sprintf("%.6f", modelConfig.Pricing.InputPer1K),
				"output_per_1k": fmt.Sprintf("%.6f", modelConfig.Pricing.OutputPer1K),
				"max_rpm":       fmt.Sprintf("%d", modelConfig.MaxRPM),
				"max_tpm":       fmt.Sprintf("%d", modelConfig.MaxTPM),
			},
		}

		// Add tags to metadata
		if len(modelConfig.Tags) > 0 {
			model.Metadata["tags"] = fmt.Sprintf("%v", modelConfig.Tags)
		}

		models = append(models, model)
	}

	response := core.ModelsResponse{
		Models: models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStrategies handles strategy listing requests
func (s *Server) handleStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	strategies := s.modelRouter.GetAvailableStrategies()

	response := map[string]interface{}{
		"strategies": strategies,
		"default":    "tag-based",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProtection handles protection mechanism statistics requests
func (s *Server) handleProtection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get model ID from query parameter
	modelID := r.URL.Query().Get("model")

	var response map[string]interface{}
	if modelID != "" {
		// Get stats for specific model
		response = s.protectionManager.GetStats(modelID)
	} else {
		// Get stats for all models
		response = s.protectionManager.GetAllStats()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCache handles cache statistics and management requests
func (s *Server) handleCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cacheManager == nil {
		s.writeError(w, "Cache not available", "CACHE_DISABLED", http.StatusServiceUnavailable)
		return
	}

	// Get cache statistics
	stats := s.cacheManager.Stats()

	// Add cache status
	stats["status"] = "enabled"
	stats["size"] = s.cacheManager.GetCacheSize()
	stats["keys"] = len(s.cacheManager.GetCacheKeys())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleCosts handles cost reporting requests
func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.accounting == nil {
		s.writeError(w, "Cost tracking not available", "ACCOUNTING_DISABLED", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	caller := r.URL.Query().Get("caller")
	provider := r.URL.Query().Get("provider")
	model := r.URL.Query().Get("model")
	currency := r.URL.Query().Get("currency")
	groupBy := r.URL.Query().Get("groupBy")
	format := r.URL.Query().Get("format")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse time range
	var from, to *time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &t
		}
	}

	// Parse pagination
	limit := 0
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// Build filter
	filter := accounting.CostFilter{
		From:     from,
		To:       to,
		Caller:   caller,
		Provider: provider,
		Model:    model,
		Currency: currency,
		GroupBy:  groupBy,
		Limit:    limit,
		Offset:   offset,
	}

	// Determine format
	exportFormat := accounting.ExportFormatJSON
	if format == "csv" {
		exportFormat = accounting.ExportFormatCSV
	}

	// Generate report
	if groupBy != "" || format == "csv" {
		// Export format
		data, err := s.accounting.ExportCosts(filter, exportFormat)
		if err != nil {
			s.writeError(w, "Failed to export costs", "EXPORT_FAILED", http.StatusInternalServerError)
			return
		}

		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", "attachment; filename=costs.csv")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(data)
	} else {
		// JSON report
		report, err := s.accounting.GetCostReport(filter)
		if err != nil {
			s.writeError(w, "Failed to get cost report", "REPORT_FAILED", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
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

// addCostHeaders adds cost headers to the response
func (s *Server) addCostHeaders(w http.ResponseWriter, modelID string, usage core.Usage) {
	costResult, err := s.costCalculator.CalcCostForModel(modelID, usage)
	if err != nil {
		s.logger.Warn("failed to calculate cost", "error", err, "model", modelID)
		// Add a fallback header to indicate cost calculation failed
		w.Header().Set("X-Cost-Error", "calculation-failed")
		return
	}

	headers := cost.FormatCostHeaders([]*cost.CostResult{costResult})
	for key, value := range headers {
		w.Header().Set(key, value)
	}
}
