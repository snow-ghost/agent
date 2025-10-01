package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/snow-ghost/agent/worker"
	"github.com/snow-ghost/agent/worker/telemetry"
)

func main() {
	// Load configuration
	config := worker.LoadConfig()

	// Setup structured logging
	logLevel := parseLogLevel(config.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Create worker using factory
	workerInstance, err := worker.NewWorker(config)
	if err != nil {
		logger.Error("failed to create worker", "error", err)
		os.Exit(1)
	}

	// Create ingestor
	ing := worker.NewIngestor(workerInstance.Solve)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/solve", ing)

	// Get telemetry from worker if available
	// Try to get telemetry from the worker
	var telemetryHandler http.HandlerFunc
	var metricsHandler http.HandlerFunc

	// Check if worker has GetTelemetry method (common pattern)
	if telemetryGetter, ok := workerInstance.(interface{ GetTelemetry() *telemetry.Telemetry }); ok {
		telemetry := telemetryGetter.GetTelemetry()
		telemetryHandler = telemetry.HealthHandler
		metricsHandler = telemetry.MetricsHandler
	} else {
		// Fallback health endpoint
		telemetryHandler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ok","service":"agent-worker","type":"%s"}`, workerInstance.Type())
		}
		metricsHandler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"metrics":"not_available"}`)
		}
	}

	mux.Handle("/health", telemetryHandler)
	mux.Handle("/metrics", metricsHandler)

	logger.Info("worker starting",
		"port", config.WorkerPort,
		"llm_mode", config.LLMMode,
		"hypotheses_dir", config.HypothesesDir,
		"log_level", config.LogLevel)

	log.Fatal(http.ListenAndServe(":"+config.WorkerPort, mux))
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
