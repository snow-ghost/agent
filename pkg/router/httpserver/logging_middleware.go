package httpserver

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

// LoggingConfig holds configuration for the logging middleware
type LoggingConfig struct {
	LogLevel        string   `json:"log_level"`         // DEBUG, INFO, WARN, ERROR
	LogRequestBody  bool     `json:"log_request_body"`  // Whether to log request body
	LogResponseBody bool     `json:"log_response_body"` // Whether to log response body
	SanitizeKeys    []string `json:"sanitize_keys"`     // Keys to sanitize in logs
	MaxBodySize     int      `json:"max_body_size"`     // Maximum body size to log
}

// DefaultLoggingConfig returns a default logging configuration
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		LogLevel:        "INFO",
		LogRequestBody:  false,
		LogResponseBody: false,
		SanitizeKeys:    []string{"password", "token", "key", "secret", "authorization"},
		MaxBodySize:     1024, // 1KB
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture response data
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *loggingResponseWriter) Write(b []byte) (int, error) {
	if rw.body != nil {
		rw.body.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware creates a middleware that logs HTTP requests and responses
func LoggingMiddleware(config *LoggingConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultLoggingConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := uuid.New().String()

			// Add request ID to context
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			r = r.WithContext(ctx)

			// Create response writer wrapper
			var responseBody *bytes.Buffer
			if config.LogResponseBody {
				responseBody = &bytes.Buffer{}
			}
			rw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           responseBody,
			}

			// Log request
			logRequest(r, requestID, config, logger)

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log response
			logResponse(r, rw, requestID, duration, config, logger)
		})
	}
}

// logRequest logs the incoming HTTP request
func logRequest(r *http.Request, requestID string, config *LoggingConfig, logger *slog.Logger) {
	if logger == nil {
		return
	}
	// Extract trace information
	traceID := getTraceID(r.Context())
	spanID := getSpanID(r.Context())

	// Build log attributes
	attrs := []any{
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"content_length", r.ContentLength,
	}

	// Add trace information if available
	if traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}

	// Add headers (excluding sensitive ones)
	headers := sanitizeHeaders(r.Header, config.SanitizeKeys)
	if len(headers) > 0 {
		attrs = append(attrs, "headers", headers)
	}

	// Add request body if configured
	if config.LogRequestBody && r.Body != nil {
		body, err := readRequestBody(r, config.MaxBodySize)
		if err == nil && len(body) > 0 {
			attrs = append(attrs, "request_body", string(body))
		}
	}

	logger.InfoContext(r.Context(), "http_request", attrs...)
}

// logResponse logs the HTTP response
func logResponse(r *http.Request, rw *loggingResponseWriter, requestID string, duration time.Duration, config *LoggingConfig, logger *slog.Logger) {
	if logger == nil {
		return
	}
	// Extract trace information
	traceID := getTraceID(r.Context())
	spanID := getSpanID(r.Context())

	// Build log attributes
	attrs := []any{
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"status_code", rw.statusCode,
		"duration_ms", duration.Milliseconds(),
		"content_length", getBodyLength(rw.body),
	}

	// Add trace information if available
	if traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}

	// Add response body if configured
	if config.LogResponseBody && rw.body != nil && rw.body.Len() > 0 {
		body := rw.body.Bytes()
		if len(body) > config.MaxBodySize {
			body = body[:config.MaxBodySize]
		}
		attrs = append(attrs, "response_body", string(body))
	}

	// Log at appropriate level based on status code
	switch {
	case rw.statusCode >= 500:
		logger.ErrorContext(r.Context(), "http_response", attrs...)
	case rw.statusCode >= 400:
		logger.WarnContext(r.Context(), "http_response", attrs...)
	case duration > 5*time.Second:
		logger.WarnContext(r.Context(), "http_response_slow", attrs...)
	default:
		logger.InfoContext(r.Context(), "http_response", attrs...)
	}
}

// readRequestBody reads the request body and restores it
func readRequestBody(r *http.Request, maxSize int) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)))
	if err != nil {
		return nil, err
	}

	// Restore the body for the next handler
	r.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

// sanitizeHeaders removes sensitive headers from logging
func sanitizeHeaders(headers http.Header, sanitizeKeys []string) map[string]string {
	sanitized := make(map[string]string)

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Check if this key should be sanitized
		shouldSanitize := false
		for _, sanitizeKey := range sanitizeKeys {
			if strings.Contains(lowerKey, strings.ToLower(sanitizeKey)) {
				shouldSanitize = true
				break
			}
		}

		if shouldSanitize {
			sanitized[key] = "***"
		} else {
			sanitized[key] = strings.Join(values, ", ")
		}
	}

	return sanitized
}

// getTraceID extracts trace ID from context
func getTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// getSpanID extracts span ID from context
func getSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// getBodyLength safely gets the length of a buffer, returning 0 if nil
func getBodyLength(body *bytes.Buffer) int {
	if body == nil {
		return 0
	}
	return body.Len()
}

// LogLLMRequest logs LLM-specific request information
func LogLLMRequest(ctx context.Context, logger *slog.Logger, model, provider string, inputTokens, outputTokens int, cost float64, duration time.Duration) {
	if logger == nil {
		return
	}
	attrs := []any{
		"model", model,
		"provider", provider,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"total_tokens", inputTokens + outputTokens,
		"cost", cost,
		"duration_ms", duration.Milliseconds(),
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID := getSpanID(ctx); spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}

	logger.InfoContext(ctx, "llm_request", attrs...)
}

// LogLLMError logs LLM-specific error information
func LogLLMError(ctx context.Context, logger *slog.Logger, model, provider string, err error, duration time.Duration) {
	if logger == nil {
		return
	}

	attrs := []any{
		"model", model,
		"provider", provider,
		"duration_ms", duration.Milliseconds(),
	}

	// Add error message if error is not nil
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	} else {
		attrs = append(attrs, "error", "unknown error")
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID := getSpanID(ctx); spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}

	logger.ErrorContext(ctx, "llm_error", attrs...)
}

// LogCacheHit logs cache hit information
func LogCacheHit(ctx context.Context, logger *slog.Logger, cacheKey string, hitType string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"cache_key", cacheKey,
		"hit_type", hitType,
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}

	logger.DebugContext(ctx, "cache_hit", attrs...)
}

// LogCacheMiss logs cache miss information
func LogCacheMiss(ctx context.Context, logger *slog.Logger, cacheKey string, missType string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"cache_key", cacheKey,
		"miss_type", missType,
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}

	logger.DebugContext(ctx, "cache_miss", attrs...)
}

// LogRateLimit logs rate limiting information
func LogRateLimit(ctx context.Context, logger *slog.Logger, key string, limit int, remaining int) {
	if logger == nil {
		return
	}
	attrs := []any{
		"rate_limit_key", key,
		"limit", limit,
		"remaining", remaining,
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}

	logger.WarnContext(ctx, "rate_limit", attrs...)
}

// LogCircuitBreaker logs circuit breaker information
func LogCircuitBreaker(ctx context.Context, logger *slog.Logger, provider, model, state string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"provider", provider,
		"model", model,
		"state", state,
	}

	// Add trace information
	if traceID := getTraceID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}

	logger.WarnContext(ctx, "circuit_breaker", attrs...)
}
