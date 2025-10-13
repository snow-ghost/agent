package metrics

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	coremetrics "github.com/snow-ghost/agent/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	// HTTP metrics
	httpRequestsTotalProm   *prometheus.CounterVec
	httpRequestDurationProm *prometheus.HistogramVec

	httpRequestsTotalOTel   metric.Int64Counter
	httpRequestDurationOTel metric.Float64Histogram
)

// InitHTTPMetrics initializes HTTP-specific metrics
func InitHTTPMetrics() {
	if mode == "" {
		Init() // Ensure base metrics are initialized
	}

	if mode == "otel" {
		meter := coremetrics.Meter()
		if meter == nil {
			meter = otel.Meter("agent-llmrouter-http")
		}

		httpRequestsTotalOTel, _ = meter.Int64Counter(namespace + ".http_requests_total")
		httpRequestDurationOTel, _ = meter.Float64Histogram(namespace + ".http_request_duration_seconds")
		return
	}

	// Prometheus
	reg := coremetrics.Registry()
	if reg == nil {
		return
	}

	httpRequestsTotalProm = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests",
		},
		[]string{"path", "method", "code"},
	)

	httpRequestDurationProm = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
}

// HTTPMetricsMiddleware creates middleware for HTTP metrics
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	InitHTTPMetrics()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Process request
		next.ServeHTTP(wrapped, r.WithContext(r.Context()))

		// Calculate duration
		duration := time.Since(start)

		// Normalize path to reduce cardinality
		path := normalizePath(r.URL.Path)
		method := r.Method
		code := statusCodeToString(wrapped.statusCode)

		// Record metrics
		recordHTTPMetrics(r.Context(), path, method, code, duration)
	})
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

// normalizePath reduces cardinality by templating common patterns
func normalizePath(path string) string {
	// Remove query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Template common patterns
	path = regexp.MustCompile(`/v1/chat/[^/]+`).ReplaceAllString(path, "/v1/chat/{id}")
	path = regexp.MustCompile(`/v1/complete/[^/]+`).ReplaceAllString(path, "/v1/complete/{id}")
	path = regexp.MustCompile(`/v1/embed/[^/]+`).ReplaceAllString(path, "/v1/embed/{id}")
	path = regexp.MustCompile(`/v1/models/[^/]+`).ReplaceAllString(path, "/v1/models/{id}")
	path = regexp.MustCompile(`/v1/costs/[^/]+`).ReplaceAllString(path, "/v1/costs/{id}")
	path = regexp.MustCompile(`/v1/strategies/[^/]+`).ReplaceAllString(path, "/v1/strategies/{id}")
	path = regexp.MustCompile(`/v1/protection/[^/]+`).ReplaceAllString(path, "/v1/protection/{id}")
	path = regexp.MustCompile(`/v1/cache/[^/]+`).ReplaceAllString(path, "/v1/cache/{id}")

	// Limit path length
	if len(path) > 100 {
		path = path[:100] + "..."
	}

	return path
}

// statusCodeToString converts status code to string
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// recordHTTPMetrics records HTTP metrics
func recordHTTPMetrics(ctx context.Context, path, method, code string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("path", path),
		attribute.String("method", method),
		attribute.String("code", code),
	}

	if mode == "otel" {
		httpRequestsTotalOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
		httpRequestDurationOTel.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs[:2]...))
		return
	}

	// Prometheus
	if httpRequestsTotalProm != nil {
		httpRequestsTotalProm.WithLabelValues(path, method, code).Inc()
	}
	if httpRequestDurationProm != nil {
		httpRequestDurationProm.WithLabelValues(path, method).Observe(duration.Seconds())
	}
}
