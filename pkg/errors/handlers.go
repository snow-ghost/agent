package errors

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/snow-ghost/agent/pkg/limiter"
)

// ErrorHandler handles errors and provides appropriate responses
type ErrorHandler struct {
	logger *slog.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *slog.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// HandleError processes an error and returns appropriate HTTP response
func (h *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	// Extract trace ID from context if available
	traceID := getTraceIDFromContext(r.Context())

	// Convert to typed error if not already
	typedErr := h.convertToTypedError(err, traceID)

	// Log the error
	h.logError(typedErr, r)

	// Write HTTP response
	h.writeHTTPResponse(w, typedErr)
}

// convertToTypedError converts a regular error to a typed error
func (h *ErrorHandler) convertToTypedError(err error, traceID string) *TypedError {
	// If already a typed error, return it
	if typedErr, ok := err.(*TypedError); ok {
		if traceID != "" {
			typedErr.TraceID = traceID
		}
		return typedErr
	}

	// Convert based on error type
	typedErr := WrapError(err, InternalError, High, "Internal server error")
	if traceID != "" {
		typedErr.TraceID = traceID
	}

	// Try to classify the error
	if httpErr, ok := err.(*limiter.HTTPError); ok {
		switch httpErr.StatusCode {
		case http.StatusBadRequest:
			typedErr = WrapError(err, ValidationError, Medium, "Invalid request")
		case http.StatusUnauthorized:
			typedErr = WrapError(err, AuthenticationError, High, "Authentication required")
		case http.StatusForbidden:
			typedErr = WrapError(err, AuthorizationError, High, "Access denied")
		case http.StatusNotFound:
			typedErr = WrapError(err, NotFoundError, Medium, "Resource not found")
		case http.StatusTooManyRequests:
			typedErr = WrapError(err, RateLimitError, Medium, "Rate limit exceeded")
		case http.StatusRequestTimeout:
			typedErr = WrapError(err, TimeoutError, High, "Request timeout")
		case http.StatusServiceUnavailable:
			typedErr = WrapError(err, CircuitBreakerError, High, "Service unavailable")
		default:
			if httpErr.StatusCode >= 500 {
				typedErr = WrapError(err, ExternalServiceError, High, "External service error")
			}
		}
	}

	return typedErr
}

// logError logs the error with appropriate level
func (h *ErrorHandler) logError(err *TypedError, r *http.Request) {
	attrs := []any{
		"error_type", err.Type,
		"severity", err.Severity,
		"code", err.Code,
		"trace_id", err.TraceID,
		"service", err.Service,
		"operation", err.Operation,
		"method", r.Method,
		"path", r.URL.Path,
		"user_agent", r.UserAgent(),
		"remote_addr", r.RemoteAddr,
	}

	// Add details if present
	for k, v := range err.Details {
		attrs = append(attrs, k, v)
	}

	// Add retry after if present
	if err.RetryAfter != nil {
		attrs = append(attrs, "retry_after", err.RetryAfter.String())
	}

	// Log with appropriate level based on severity
	switch err.Severity {
	case Low:
		h.logger.Info(err.Message, attrs...)
	case Medium:
		h.logger.Warn(err.Message, attrs...)
	case High:
		h.logger.Error(err.Message, attrs...)
	case Critical:
		h.logger.Error(err.Message, attrs...)
	}
}

// writeHTTPResponse writes the HTTP response for the error
func (h *ErrorHandler) writeHTTPResponse(w http.ResponseWriter, err *TypedError) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Set status code based on error type
	statusCode := h.getStatusCode(err)
	w.WriteHeader(statusCode)

	// Set retry after header if present
	if err.RetryAfter != nil {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", err.RetryAfter.Seconds()))
	}

	// Set trace ID header if present
	if err.TraceID != "" {
		w.Header().Set("X-Trace-ID", err.TraceID)
	}

	// Create error response
	response := ErrorResponse{
		Error:     err.Message,
		Code:      err.Code,
		Type:      string(err.Type),
		Severity:  string(err.Severity),
		Timestamp: err.Timestamp,
		TraceID:   err.TraceID,
		Details:   err.Details,
	}

	// Write response
	fmt.Fprintf(w, `{"error":"%s","code":"%s","type":"%s","severity":"%s","timestamp":"%s","trace_id":"%s"}`,
		response.Error, response.Code, response.Type, response.Severity,
		response.Timestamp.Format(time.RFC3339), response.TraceID)
}

// getStatusCode returns the appropriate HTTP status code for the error
func (h *ErrorHandler) getStatusCode(err *TypedError) int {
	switch err.Type {
	case ValidationError:
		return http.StatusBadRequest
	case AuthenticationError:
		return http.StatusUnauthorized
	case AuthorizationError:
		return http.StatusForbidden
	case NotFoundError:
		return http.StatusNotFound
	case RateLimitError:
		return http.StatusTooManyRequests
	case TimeoutError:
		return http.StatusRequestTimeout
	case CircuitBreakerError:
		return http.StatusServiceUnavailable
	case ExternalServiceError:
		return http.StatusBadGateway
	case ConfigurationError:
		return http.StatusInternalServerError
	case RetryableError, InternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents the JSON error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// getTraceIDFromContext extracts trace ID from context
func getTraceIDFromContext(ctx context.Context) string {
	// This would typically extract from OpenTelemetry trace context
	// For now, return empty string
	return ""
}

// Middleware creates an HTTP middleware for error handling
func (h *ErrorHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Convert panic to typed error
				typedErr := NewInternalError("Internal server error")
				typedErr.Details["panic"] = fmt.Sprintf("%v", err)
				h.HandleError(w, r, typedErr)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RetryableErrorChecker checks if an error is retryable
type RetryableErrorChecker struct{}

// IsRetryable checks if an error should be retried
func (c *RetryableErrorChecker) IsRetryable(err error) bool {
	if typedErr, ok := err.(*TypedError); ok {
		return typedErr.IsRetryable()
	}

	// Check for specific error types that are retryable
	if httpErr, ok := err.(*limiter.HTTPError); ok {
		switch httpErr.StatusCode {
		case http.StatusTooManyRequests, http.StatusRequestTimeout,
			http.StatusServiceUnavailable, http.StatusBadGateway,
			http.StatusGatewayTimeout:
			return true
		}
	}

	return false
}

// GetRetryDelay calculates retry delay for an error
func (c *RetryableErrorChecker) GetRetryDelay(err error, attempt int) time.Duration {
	if typedErr, ok := err.(*TypedError); ok {
		if typedErr.RetryAfter != nil {
			return *typedErr.RetryAfter
		}
	}

	// Default exponential backoff
	baseDelay := time.Second
	maxDelay := 30 * time.Second

	delay := baseDelay * time.Duration(1<<uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}
