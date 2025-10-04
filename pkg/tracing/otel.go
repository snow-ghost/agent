package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps OpenTelemetry tracer
type Tracer struct {
	tracer trace.Tracer
}

// Config holds tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	JaegerEndpoint string
	Environment    string
}

// NewTracer creates a new OpenTelemetry tracer
func NewTracer(config Config) (*Tracer, error) {
	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Tracer{
		tracer: otel.Tracer(config.ServiceName),
	}, nil
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartRequestSpan starts a span for an LLM request
func (t *Tracer) StartRequestSpan(ctx context.Context, caller, model, provider string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("llm.caller", caller),
		attribute.String("llm.model", model),
		attribute.String("llm.provider", provider),
		attribute.String("llm.operation", "chat_completion"),
	}

	return t.tracer.Start(ctx, "llm.request", trace.WithAttributes(attrs...))
}

// StartCacheSpan starts a span for cache operations
func (t *Tracer) StartCacheSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.operation", operation),
	}

	return t.tracer.Start(ctx, "cache.operation", trace.WithAttributes(attrs...))
}

// StartRetrySpan starts a span for retry operations
func (t *Tracer) StartRetrySpan(ctx context.Context, provider, model, reason string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("retry.provider", provider),
		attribute.String("retry.model", model),
		attribute.String("retry.reason", reason),
	}

	return t.tracer.Start(ctx, "llm.retry", trace.WithAttributes(attrs...))
}

// StartCircuitBreakerSpan starts a span for circuit breaker operations
func (t *Tracer) StartCircuitBreakerSpan(ctx context.Context, provider, model, state string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("circuit_breaker.provider", provider),
		attribute.String("circuit_breaker.model", model),
		attribute.String("circuit_breaker.state", state),
	}

	return t.tracer.Start(ctx, "llm.circuit_breaker", trace.WithAttributes(attrs...))
}

// AddSpanAttributes adds attributes to a span
func AddSpanAttributes(span trace.Span, attrs map[string]interface{}) {
	for key, value := range attrs {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		case []string:
			span.SetAttributes(attribute.StringSlice(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// RecordSpanError records an error in a span
func RecordSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(1, err.Error()) // 1 = codes.Error
}

// RecordSpanSuccess records success in a span
func RecordSpanSuccess(span trace.Span) {
	span.SetStatus(0, "success") // 0 = codes.Ok
}

// RecordSpanDuration records duration in a span
func RecordSpanDuration(span trace.Span, duration time.Duration) {
	span.SetAttributes(attribute.Float64("duration_ms", float64(duration.Nanoseconds())/1e6))
}

// RecordSpanTokens records token usage in a span
func RecordSpanTokens(span trace.Span, inputTokens, outputTokens int) {
	span.SetAttributes(
		attribute.Int("tokens.input", inputTokens),
		attribute.Int("tokens.output", outputTokens),
		attribute.Int("tokens.total", inputTokens+outputTokens),
	)
}

// RecordSpanCost records cost in a span
func RecordSpanCost(span trace.Span, cost float64, currency string) {
	span.SetAttributes(
		attribute.Float64("cost.total", cost),
		attribute.String("cost.currency", currency),
	)
}

// Shutdown shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	return otel.GetTracerProvider().(interface{ Shutdown(context.Context) error }).Shutdown(ctx)
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
