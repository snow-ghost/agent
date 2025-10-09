package errors

import (
	"fmt"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// RetryableError indicates the error can be retried
	RetryableError ErrorType = "retryable"
	// CircuitBreakerError indicates circuit breaker is open
	CircuitBreakerError ErrorType = "circuit_breaker"
	// ValidationError indicates input validation failed
	ValidationError ErrorType = "validation"
	// TimeoutError indicates operation timed out
	TimeoutError ErrorType = "timeout"
	// RateLimitError indicates rate limit exceeded
	RateLimitError ErrorType = "rate_limit"
	// AuthenticationError indicates authentication failed
	AuthenticationError ErrorType = "authentication"
	// AuthorizationError indicates authorization failed
	AuthorizationError ErrorType = "authorization"
	// NotFoundError indicates resource not found
	NotFoundError ErrorType = "not_found"
	// InternalError indicates internal system error
	InternalError ErrorType = "internal"
	// ExternalServiceError indicates external service error
	ExternalServiceError ErrorType = "external_service"
	// ConfigurationError indicates configuration error
	ConfigurationError ErrorType = "configuration"
)

// ErrorSeverity represents the severity of the error
type ErrorSeverity string

const (
	// Low severity - informational
	Low ErrorSeverity = "low"
	// Medium severity - warning
	Medium ErrorSeverity = "medium"
	// High severity - error
	High ErrorSeverity = "high"
	// Critical severity - system failure
	Critical ErrorSeverity = "critical"
)

// TypedError represents a structured error with additional metadata
type TypedError struct {
	Type       ErrorType              `json:"type"`
	Severity   ErrorSeverity          `json:"severity"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	TraceID    string                 `json:"trace_id,omitempty"`
	Service    string                 `json:"service,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	RetryAfter *time.Duration         `json:"retry_after,omitempty"`
	Underlying error                  `json:"-"`
}

// Error implements the error interface
func (e *TypedError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Underlying)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *TypedError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns true if the error can be retried
func (e *TypedError) IsRetryable() bool {
	return e.Type == RetryableError || e.Type == TimeoutError || e.Type == RateLimitError
}

// IsCircuitBreaker returns true if the error is due to circuit breaker
func (e *TypedError) IsCircuitBreaker() bool {
	return e.Type == CircuitBreakerError
}

// IsValidation returns true if the error is a validation error
func (e *TypedError) IsValidation() bool {
	return e.Type == ValidationError
}

// IsTimeout returns true if the error is a timeout
func (e *TypedError) IsTimeout() bool {
	return e.Type == TimeoutError
}

// IsRateLimit returns true if the error is due to rate limiting
func (e *TypedError) IsRateLimit() bool {
	return e.Type == RateLimitError
}

// IsAuthError returns true if the error is authentication/authorization related
func (e *TypedError) IsAuthError() bool {
	return e.Type == AuthenticationError || e.Type == AuthorizationError
}

// IsNotFound returns true if the error is due to resource not found
func (e *TypedError) IsNotFound() bool {
	return e.Type == NotFoundError
}

// IsExternalService returns true if the error is from external service
func (e *TypedError) IsExternalService() bool {
	return e.Type == ExternalServiceError
}

// GetRetryAfter returns the retry after duration if set
func (e *TypedError) GetRetryAfter() *time.Duration {
	return e.RetryAfter
}

// WithTraceID adds a trace ID to the error
func (e *TypedError) WithTraceID(traceID string) *TypedError {
	e.TraceID = traceID
	return e
}

// WithService adds a service name to the error
func (e *TypedError) WithService(service string) *TypedError {
	e.Service = service
	return e
}

// WithOperation adds an operation name to the error
func (e *TypedError) WithOperation(operation string) *TypedError {
	e.Operation = operation
	return e
}

// WithRetryAfter adds a retry after duration to the error
func (e *TypedError) WithRetryAfter(duration time.Duration) *TypedError {
	e.RetryAfter = &duration
	return e
}

// WithDetails adds additional details to the error
func (e *TypedError) WithDetails(key string, value interface{}) *TypedError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewTypedError creates a new typed error
func NewTypedError(errorType ErrorType, severity ErrorSeverity, message string) *TypedError {
	return &TypedError{
		Type:      errorType,
		Severity:  severity,
		Message:   message,
		Code:      string(errorType),
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// WrapError wraps an existing error with typed error information
func WrapError(err error, errorType ErrorType, severity ErrorSeverity, message string) *TypedError {
	return &TypedError{
		Type:       errorType,
		Severity:   severity,
		Message:    message,
		Code:       string(errorType),
		Timestamp:  time.Now(),
		Underlying: err,
		Details:    make(map[string]interface{}),
	}
}

// NewRetryableError creates a new retryable error
func NewRetryableError(message string) *TypedError {
	return NewTypedError(RetryableError, Medium, message)
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *TypedError {
	return NewTypedError(ValidationError, Medium, message)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string) *TypedError {
	return NewTypedError(TimeoutError, High, message)
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string, retryAfter time.Duration) *TypedError {
	return NewTypedError(RateLimitError, Medium, message).WithRetryAfter(retryAfter)
}

// NewCircuitBreakerError creates a new circuit breaker error
func NewCircuitBreakerError(message string) *TypedError {
	return NewTypedError(CircuitBreakerError, High, message)
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *TypedError {
	return NewTypedError(AuthenticationError, High, message)
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message string) *TypedError {
	return NewTypedError(AuthorizationError, High, message)
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *TypedError {
	return NewTypedError(NotFoundError, Medium, message)
}

// NewInternalError creates a new internal error
func NewInternalError(message string) *TypedError {
	return NewTypedError(InternalError, High, message)
}

// NewExternalServiceError creates a new external service error
func NewExternalServiceError(message string) *TypedError {
	return NewTypedError(ExternalServiceError, High, message)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string) *TypedError {
	return NewTypedError(ConfigurationError, Critical, message)
}
