package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/snow-ghost/agent/pkg/metrics"
	"github.com/snow-ghost/agent/pkg/ports"
	"github.com/snow-ghost/agent/worker"
	"github.com/snow-ghost/agent/worker/capabilities"
	"github.com/snow-ghost/agent/worker/telemetry"
)

func main() {
	// Check if this is a healthcheck command
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthcheck()
		return
	}

	// Load configuration
	config := worker.LoadConfig()

	// Setup structured logging
	logLevel := parseLogLevel(config.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Log loaded configuration
	config.LogConfig(logger)

	// Create worker using factory
	workerInstance, err := worker.NewWorker(config)
	if err != nil {
		logger.Error("failed to create worker", "error", err)
		os.Exit(1)
	}

	// Create ingestor
	ing := worker.NewIngestor(workerInstance.Solve)

	// Validate API port and compute service port
	apiPort, err := strconv.Atoi(config.WorkerPort)
	if err != nil {
		logger.Error("invalid WORKER_PORT", "error", err)
		os.Exit(1)
	}
	if err := ports.ValidateAPIPort(apiPort); err != nil {
		logger.Error("worker API port invalid", "error", err)
		os.Exit(1)
	}
	servicePort, err := ports.DeriveServicePort(apiPort)
	if err != nil {
		logger.Error("worker service port invalid", "error", err)
		os.Exit(1)
	}

	// Setup HTTP routes (API)
	mux := http.NewServeMux()
	mux.Handle("/solve", ing)

	// Get telemetry from worker if available
	// Try to get telemetry from the worker
	var telemetryHandler http.HandlerFunc

	// Check if worker has GetTelemetry method (common pattern)
	if telemetryGetter, ok := workerInstance.(interface{ GetTelemetry() *telemetry.Telemetry }); ok {
		telemetry := telemetryGetter.GetTelemetry()
		telemetryHandler = telemetry.HealthHandler
	} else {
		// Fallback health endpoint
		telemetryHandler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ok","service":"agent-worker","type":"%s"}`, workerInstance.Type())
		}
		// Metrics are served by the shared metrics package on the metrics server
	}

	// Health and metrics will be exposed on a separate service port
	mux.Handle("/caps", http.HandlerFunc(createCapsHandler(workerInstance)))
	mux.Handle("/ready", http.HandlerFunc(createReadyHandler(workerInstance)))

	logger.Info("worker starting",
		"port", config.WorkerPort,
		"llm_mode", config.LLMMode,
		"hypotheses_dir", config.HypothesesDir,
		"log_level", config.LogLevel)

	// Create HTTP server with graceful shutdown
	server := &http.Server{
		Addr:    ":" + config.WorkerPort,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("worker server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("worker server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Service endpoints on METRICS_ADDR (defaults to 0.0.0.0:servicePort)
	if err := metrics.Init(); err != nil {
		logger.Error("metrics init failed", "error", err)
	}
	metricsPath := os.Getenv("METRICS_PATH")
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = "0.0.0.0:" + strconv.Itoa(servicePort)
	}

	serviceMux := http.NewServeMux()
	serviceMux.Handle("/healthz", telemetryHandler)
	serviceMux.Handle(metricsPath, metrics.Handler())
	serviceSrv := &http.Server{Addr: metricsAddr, Handler: serviceMux}
	go func() {
		logger.Info("worker service endpoints starting", "addr", serviceSrv.Addr)
		if err := serviceSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("worker service endpoints failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("worker shutting down...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown servers
	if err := serviceSrv.Shutdown(ctx); err != nil {
		logger.Error("worker service endpoints shutdown failed", "error", err)
	}
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("worker server shutdown failed", "error", err)
	} else {
		logger.Info("worker server shutdown complete")
	}

	// Cleanup worker resources
	if cleanupWorker, ok := workerInstance.(interface{ Close() error }); ok {
		if err := cleanupWorker.Close(); err != nil {
			logger.Error("worker cleanup failed", "error", err)
		} else {
			logger.Info("worker cleanup complete")
		}
	}

	logger.Info("worker shutdown complete")
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
