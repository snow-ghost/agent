package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/snow-ghost/agent/worker"
	"github.com/snow-ghost/agent/worker/capabilities"
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
	mux.Handle("/caps", http.HandlerFunc(createCapsHandler(workerInstance)))
	mux.Handle("/ready", http.HandlerFunc(createReadyHandler(workerInstance)))

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

// createCapsHandler creates a capabilities handler for the worker
func createCapsHandler(workerInstance worker.Worker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Check if worker supports capabilities
		if capsWorker, ok := workerInstance.(capabilities.WorkerWithCapabilities); ok {
			caps := capsWorker.Caps()
			response := map[string]interface{}{
				"worker_type": workerInstance.Type(),
				"capabilities": map[string]bool{
					"use_kb":   caps.UseKB,
					"use_wasm": caps.UseWASM,
					"use_llm":  caps.UseLLM,
				},
				"capabilities_string": caps.String(),
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// Fallback for workers without capabilities
			response := map[string]interface{}{
				"worker_type": workerInstance.Type(),
				"capabilities": map[string]bool{
					"use_kb":   true, // All workers support KB
					"use_wasm": workerInstance.Type() == "heavy",
					"use_llm":  workerInstance.Type() == "heavy",
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}
}

// createReadyHandler creates a readiness handler for the worker
func createReadyHandler(workerInstance worker.Worker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// For now, all workers are always ready
		// In a real implementation, you might check dependencies, health, etc.
		response := map[string]interface{}{
			"status":      "ready",
			"worker_type": workerInstance.Type(),
			"ready":       true,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
