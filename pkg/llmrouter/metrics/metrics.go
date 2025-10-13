package metrics

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	coremetrics "github.com/snow-ghost/agent/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	mode      string
	namespace string

	// Prometheus instruments
	llmRequestsTotalProm    *prometheus.CounterVec
	llmRequestDurationProm  *prometheus.HistogramVec
	llmTokensInputProm      *prometheus.CounterVec
	llmTokensOutputProm     *prometheus.CounterVec
	llmCostTotalProm        *prometheus.CounterVec
	llmRetriesTotalProm     *prometheus.CounterVec
	llmCircuitOpenTotalProm *prometheus.CounterVec

	// OTel instruments
	meter                   metric.Meter
	llmRequestsTotalOTel    metric.Int64Counter
	llmRequestDurationOTel  metric.Float64Histogram
	llmTokensInputOTel      metric.Int64Counter
	llmTokensOutputOTel     metric.Int64Counter
	llmCostTotalOTel        metric.Float64Counter
	llmRetriesTotalOTel     metric.Int64Counter
	llmCircuitOpenTotalOTel metric.Int64Counter
)

// Init sets up LLM router metrics according to METRICS_MODE.
func Init() {
	if mode != "" {
		return
	}

	// Ensure global metrics core is initialized
	_ = coremetrics.Init()

	mode = strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_MODE")))
	if mode == "" {
		mode = "prom"
	}
	namespace = os.Getenv("METRICS_NAMESPACE")
	if namespace == "" {
		namespace = "agent"
	}

	if mode == "otel" {
		meter = coremetrics.Meter()
		// If nil, the global provider returns a no-op meter
		if meter == nil {
			meter = otel.Meter("agent-llmrouter")
		}

		llmRequestsTotalOTel, _ = meter.Int64Counter(namespace + ".llm_requests_total")
		llmRequestDurationOTel, _ = meter.Float64Histogram(namespace + ".llm_request_duration_seconds")
		llmTokensInputOTel, _ = meter.Int64Counter(namespace + ".llm_tokens_input_total")
		llmTokensOutputOTel, _ = meter.Int64Counter(namespace + ".llm_tokens_output_total")
		llmCostTotalOTel, _ = meter.Float64Counter(namespace + ".llm_cost_total")
		llmRetriesTotalOTel, _ = meter.Int64Counter(namespace + ".llm_retries_total")
		llmCircuitOpenTotalOTel, _ = meter.Int64Counter(namespace + ".llm_circuit_open_total")
		return
	}

	// prom
	reg := coremetrics.Registry()
	if reg == nil {
		return
	}

	llmRequestsTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_requests_total", Help: "Total LLM requests"},
		[]string{"provider", "model", "status", "cache"},
	)
	llmRequestDurationProm = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: namespace, Name: "llm_request_duration_seconds", Help: "LLM request duration", Buckets: prometheus.DefBuckets},
		[]string{"provider", "model"},
	)
	llmTokensInputProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_tokens_input_total", Help: "Total input tokens"},
		[]string{"provider", "model"},
	)
	llmTokensOutputProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_tokens_output_total", Help: "Total output tokens"},
		[]string{"provider", "model"},
	)
	llmCostTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_cost_total", Help: "Total cost"},
		[]string{"provider", "model", "currency"},
	)
	llmRetriesTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_retries_total", Help: "Total retries"},
		[]string{"provider", "model"},
	)
	llmCircuitOpenTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "llm_circuit_open_total", Help: "Total circuit breaker opens"},
		[]string{"provider", "model"},
	)

	reg.MustRegister(llmRequestsTotalProm, llmRequestDurationProm, llmTokensInputProm, llmTokensOutputProm, llmCostTotalProm, llmRetriesTotalProm, llmCircuitOpenTotalProm)
}

// ObserveLLMRequest records a complete LLM request with all metrics.
func ObserveLLMRequest(ctx context.Context, provider, model, status, cache string, duration time.Duration, inputTokens, outputTokens int64, cost float64, currency string) {
	Init()

	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("status", status),
		attribute.String("cache", cache),
	}

	if mode == "otel" {
		llmRequestsTotalOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
		llmRequestDurationOTel.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs[:2]...))

		if inputTokens > 0 {
			llmTokensInputOTel.Add(ctx, inputTokens, metric.WithAttributes(attrs[:2]...))
		}
		if outputTokens > 0 {
			llmTokensOutputOTel.Add(ctx, outputTokens, metric.WithAttributes(attrs[:2]...))
		}
		if cost > 0 {
			costAttrs := append(attrs[:2], attribute.String("currency", currency))
			llmCostTotalOTel.Add(ctx, cost, metric.WithAttributes(costAttrs...))
		}
		return
	}

	// Prometheus
	if llmRequestsTotalProm != nil {
		llmRequestsTotalProm.WithLabelValues(provider, model, status, cache).Inc()
	}
	if llmRequestDurationProm != nil {
		llmRequestDurationProm.WithLabelValues(provider, model).Observe(duration.Seconds())
	}
	if llmTokensInputProm != nil && inputTokens > 0 {
		llmTokensInputProm.WithLabelValues(provider, model).Add(float64(inputTokens))
	}
	if llmTokensOutputProm != nil && outputTokens > 0 {
		llmTokensOutputProm.WithLabelValues(provider, model).Add(float64(outputTokens))
	}
	if llmCostTotalProm != nil && cost > 0 {
		llmCostTotalProm.WithLabelValues(provider, model, currency).Add(cost)
	}
}

// IncLLMRetry increments the retry counter.
func IncLLMRetry(ctx context.Context, provider, model string) {
	Init()

	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}

	if mode == "otel" {
		llmRetriesTotalOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
		return
	}

	if llmRetriesTotalProm != nil {
		llmRetriesTotalProm.WithLabelValues(provider, model).Inc()
	}
}

// IncLLMCircuitOpen increments the circuit breaker open counter.
func IncLLMCircuitOpen(ctx context.Context, provider, model string) {
	Init()

	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}

	if mode == "otel" {
		llmCircuitOpenTotalOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
		return
	}

	if llmCircuitOpenTotalProm != nil {
		llmCircuitOpenTotalProm.WithLabelValues(provider, model).Inc()
	}
}
