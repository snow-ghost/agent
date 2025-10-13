package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/snow-ghost/agent/pkg/metrics"
	"github.com/snow-ghost/agent/pkg/ports"
	"github.com/snow-ghost/agent/pkg/router/httpserver"
)

// LLMRouterConfig holds configuration for the LLM router
type LLMRouterConfig struct {
	Port     string
	LogLevel string
}

// LoadLLMRouterConfig loads LLM router configuration from environment variables
func LoadLLMRouterConfig() *LLMRouterConfig {
	return &LLMRouterConfig{
		Port:     getEnv("LLMROUTER_PORT", "9000"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}
}

// LogConfig logs the LLM router configuration in a structured format
func (c *LLMRouterConfig) LogConfig(logger interface{}) {
	// Use type assertion to check if logger has Info method
	if slogLogger, ok := logger.(interface {
		Info(msg string, args ...any)
	}); ok {
		slogLogger.Info("llmrouter configuration loaded",
			"port", c.Port,
			"log_level", c.LogLevel,
		)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Check if this is a healthcheck command
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthcheck()
		return
	}

	// Load configuration
	config := LoadLLMRouterConfig()

	// Setup logging
	var level slog.Level
	switch config.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	// Log loaded configuration
	config.LogConfig(logger)

	// Validate and derive ports
	apiPort, err := strconv.Atoi(config.Port)
	if err != nil {
		log.Fatal("invalid LLMROUTER_PORT, must be integer:", err)
	}
	if err := ports.ValidateAPIPort(apiPort); err != nil {
		log.Fatal("invalid API port:", err)
	}
	servicePort, err := ports.DeriveServicePort(apiPort)
	if err != nil {
		log.Fatal("invalid derived service port:", err)
	}

	// Create and start API server
	server := httpserver.NewServer(config.Port, logger)

	logger.Info("starting LLM router service",
		"port", config.Port,
		"log_level", config.LogLevel)

	// Start API server with graceful shutdown
	if err := server.StartWithGracefulShutdown(); err != nil {
		log.Fatal("failed to start server:", err)
	}

	// Initialize metrics and start metrics/service server on METRICS_ADDR
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
	serviceMux.Handle(metricsPath, metrics.Handler())
	serviceMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	serviceSrv := &http.Server{Addr: metricsAddr, Handler: serviceMux}
	go func() {
		logger.Info("starting service endpoints", "addr", metricsAddr)
		if err := serviceSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("service endpoints server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("LLM router shutting down...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown servers
	if err := serviceSrv.Shutdown(ctx); err != nil {
		logger.Error("LLM router service server shutdown failed", "error", err)
	}
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("LLM router server shutdown failed", "error", err)
	} else {
		logger.Info("LLM router server shutdown complete")
	}

	// Close server resources
	if err := server.Close(); err != nil {
		logger.Error("LLM router cleanup failed", "error", err)
	} else {
		logger.Info("LLM router cleanup complete")
	}

	logger.Info("LLM router shutdown complete")
}
