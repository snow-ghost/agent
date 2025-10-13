package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_Init(t *testing.T) {
	// Test that Init doesn't panic
	assert.NotPanics(t, func() {
		Init()
	})
}

func TestMetrics_ObserveLLMRequest(t *testing.T) {
	// Test that ObserveLLMRequest doesn't panic
	ctx := context.Background()

	assert.NotPanics(t, func() {
		ObserveLLMRequest(ctx, "test-provider", "test-model", "ok", "miss",
			time.Second, 100, 50, 0.01, "USD")
	})
}

func TestMetrics_IncLLMRetry(t *testing.T) {
	// Test that IncLLMRetry doesn't panic
	ctx := context.Background()

	assert.NotPanics(t, func() {
		IncLLMRetry(ctx, "test-provider", "test-model")
	})
}

func TestMetrics_IncLLMCircuitOpen(t *testing.T) {
	// Test that IncLLMCircuitOpen doesn't panic
	ctx := context.Background()

	assert.NotPanics(t, func() {
		IncLLMCircuitOpen(ctx, "test-provider", "test-model")
	})
}

func TestHTTPMetrics_InitHTTPMetrics(t *testing.T) {
	// Test that InitHTTPMetrics doesn't panic
	assert.NotPanics(t, func() {
		InitHTTPMetrics()
	})
}

func TestHTTPMetrics_Middleware(t *testing.T) {
	// Test that HTTPMetricsMiddleware creates a working middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := HTTPMetricsMiddleware(handler)
	require.NotNil(t, middleware)

	// Test that middleware works
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		middleware.ServeHTTP(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestHTTPMetrics_NormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/v1/chat", "/v1/chat"},
		{"/v1/chat/123", "/v1/chat/{id}"},
		{"/v1/complete/abc", "/v1/complete/{id}"},
		{"/v1/embed/xyz", "/v1/embed/{id}"},
		{"/v1/models/test", "/v1/models/{id}"},
		{"/v1/costs/123", "/v1/costs/{id}"},
		{"/v1/strategies/abc", "/v1/strategies/{id}"},
		{"/v1/protection/xyz", "/v1/protection/{id}"},
		{"/v1/cache/test", "/v1/cache/{id}"},
		{"/v1/chat/123?param=value", "/v1/chat/{id}"},
		{"/very/long/path/that/exceeds/one/hundred/characters/and/should/be/truncated/to/prevent/high/cardinality/in/metrics", "/very/long/path/that/exceeds/one/hundred/characters/and/should/be/truncated/to/prevent/high/cardinal..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTTPMetrics_StatusCodeToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{501, "5xx"},
		{599, "5xx"},
		{100, "unknown"},
		{600, "5xx"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.input)), func(t *testing.T) {
			result := statusCodeToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
