package observability

import (
	"context"
	"time"

	"github.com/snow-ghost/agent/pkg/logging"
	"github.com/snow-ghost/agent/pkg/metrics"
	"github.com/snow-ghost/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Manager manages all observability components
type Manager struct {
	metrics *metrics.PrometheusMetrics
	tracer  *tracing.Tracer
	logger  *logging.Logger
}

// Config holds observability configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	LogLevel       string
	LogFormat      string
}

// NewManager creates a new observability manager
func NewManager(config Config) (*Manager, error) {
	// Create metrics
	prometheusMetrics := metrics.NewPrometheusMetrics()

	// Create tracer
	tracerConfig := tracing.Config{
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		JaegerEndpoint: config.JaegerEndpoint,
		Environment:    config.Environment,
	}

	tracer, err := tracing.NewTracer(tracerConfig)
	if err != nil {
		return nil, err
	}

	// Create logger
	loggerConfig := logging.Config{
		Level:     config.LogLevel,
		Format:    config.LogFormat,
		Output:    "stdout",
		AddCaller: true,
		AddStack:  false,
	}

	logger, err := logging.NewLogger(loggerConfig)
	if err != nil {
		return nil, err
	}

	return &Manager{
		metrics: prometheusMetrics,
		tracer:  tracer,
		logger:  logger,
	}, nil
}

// GetMetrics returns the metrics instance
func (m *Manager) GetMetrics() *metrics.PrometheusMetrics {
	return m.metrics
}

// GetTracer returns the tracer instance
func (m *Manager) GetTracer() *tracing.Tracer {
	return m.tracer
}

// GetLogger returns the logger instance
func (m *Manager) GetLogger() *logging.Logger {
	return m.logger
}

// StartRequestSpan starts a span for an LLM request with logging
func (m *Manager) StartRequestSpan(ctx context.Context, caller, model, provider, requestID string) (context.Context, trace.Span) {
	// Start span
	ctx, span := m.tracer.StartRequestSpan(ctx, caller, model, provider)

	// Add request ID to span
	span.SetAttributes(
		attribute.String("request_id", requestID),
	)

	// Log request start
	m.logger.WithRequestID(ctx, requestID).WithFields(map[string]interface{}{
		"provider": provider,
		"model":    model,
		"caller":   caller,
	}).Info("LLM request started")

	return ctx, span
}

// RecordRequestMetrics records request metrics
func (m *Manager) RecordRequestMetrics(provider, model, status string, duration time.Duration, inputTokens, outputTokens int, cost float64, currency string) {
	// Record basic metrics
	m.metrics.RecordRequest(provider, model, status)
	m.metrics.RecordLatency(provider, model, duration)
	m.metrics.RecordTokens(provider, model, inputTokens, outputTokens)
	m.metrics.RecordCost(provider, model, currency, cost)
}

// RecordCacheMetrics records cache metrics
func (m *Manager) RecordCacheMetrics(hit bool) {
	if hit {
		m.metrics.RecordCacheHit()
	} else {
		m.metrics.RecordCacheMiss()
	}
}

// RecordRetryMetrics records retry metrics
func (m *Manager) RecordRetryMetrics(provider, model, reason string) {
	m.metrics.RecordRetry(provider, model, reason)
}

// RecordCircuitBreakerMetrics records circuit breaker metrics
func (m *Manager) RecordCircuitBreakerMetrics(provider, model, state string) {
	switch state {
	case "open":
		m.metrics.RecordCircuitOpen(provider, model)
	case "closed":
		m.metrics.RecordCircuitClosed(provider, model)
	case "half-open":
		m.metrics.RecordCircuitHalfOpen(provider, model)
	}
}

// LogRequestCompletion logs request completion
func (m *Manager) LogRequestCompletion(ctx context.Context, provider, model, status string, duration time.Duration, tokens int, cost float64, requestID string) {
	m.logger.LogLLMRequest(ctx, provider, model, status, duration, tokens, cost, requestID)
}

// LogCacheOperation logs cache operation
func (m *Manager) LogCacheOperation(ctx context.Context, operation string, hit bool, requestID string) {
	m.logger.LogCacheOperation(ctx, operation, hit, requestID)
}

// LogRetryOperation logs retry operation
func (m *Manager) LogRetryOperation(ctx context.Context, provider, model, reason string, attempt int, requestID string) {
	m.logger.LogRetry(ctx, provider, model, reason, attempt, requestID)
}

// LogCircuitBreakerOperation logs circuit breaker operation
func (m *Manager) LogCircuitBreakerOperation(ctx context.Context, provider, model, state string, requestID string) {
	m.logger.LogCircuitBreaker(ctx, provider, model, state, requestID)
}

// Shutdown shuts down all observability components
func (m *Manager) Shutdown(ctx context.Context) error {
	// Shutdown tracer
	if err := m.tracer.Shutdown(ctx); err != nil {
		return err
	}

	// Sync logger
	if err := m.logger.Sync(); err != nil {
		return err
	}

	return nil
}

// GetRequestIDFromContext extracts request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, "request_id", requestID)
}

// WithCaller adds caller to context
func WithCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, "caller", caller)
}

// GetCallerFromContext extracts caller from context
func GetCallerFromContext(ctx context.Context) string {
	if caller, ok := ctx.Value("caller").(string); ok {
		return caller
	}
	return "unknown"
}
