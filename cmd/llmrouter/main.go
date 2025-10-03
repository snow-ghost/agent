package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/snow-ghost/agent/pkg/router/httpserver"
)

func main() {
	// Get port from environment or use default
	port := os.Getenv("LLMROUTER_PORT")
	if port == "" {
		port = "8080"
	}

	// Setup logging
	logLevel := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch logLevel {
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

	// Create and start server
	server := httpserver.NewServer(port, logger)

	logger.Info("starting LLM router service",
		"port", port,
		"log_level", logLevel)

	if err := server.Start(); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
