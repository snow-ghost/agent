package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		Port:     getEnv("LLMROUTER_PORT", "8090"),
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

	// Create and start server
	server := httpserver.NewServer(config.Port, logger)

	logger.Info("starting LLM router service",
		"port", config.Port,
		"log_level", config.LogLevel)

	// Start server with graceful shutdown
	if err := server.StartWithGracefulShutdown(); err != nil {
		log.Fatal("failed to start server:", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("LLM router shutting down...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
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
