package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/snow-ghost/agent/core"
)

// RouterConfig holds configuration for the router
type RouterConfig struct {
	LightWorkerURL string
	HeavyWorkerURL string
	Port           string
}

// LoadRouterConfig loads router configuration from environment variables
func LoadRouterConfig() *RouterConfig {
	return &RouterConfig{
		LightWorkerURL: getEnv("LIGHT_WORKER_URL", "http://localhost:8081"),
		HeavyWorkerURL: getEnv("HEAVY_WORKER_URL", "http://localhost:8082"),
		Port:           getEnv("ROUTER_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Router handles task routing between light and heavy workers
type Router struct {
	config      *RouterConfig
	lightClient *http.Client
	heavyClient *http.Client
}

// NewRouter creates a new router
func NewRouter(config *RouterConfig) *Router {
	return &Router{
		config: config,
		lightClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		heavyClient: &http.Client{
			Timeout: 60 * time.Second, // Heavy workers may take longer
		},
	}
}

// RouteTask determines which worker should handle the task
func (r *Router) RouteTask(task core.Task) string {
	// Check task flags first
	if task.Flags.RequiresSandbox {
		return "heavy"
	}

	// Check complexity
	if task.Flags.MaxComplexity > 5 {
		return "heavy"
	}

	// Check domain - some domains are better suited for heavy workers
	heavyDomains := map[string]bool{
		"ai":                 true,
		"machine_learning":   true,
		"deep_learning":      true,
		"neural_networks":    true,
		"complex_algorithms": true,
	}

	if heavyDomains[task.Domain] {
		return "heavy"
	}

	// Default to light worker for simple tasks
	return "light"
}

// ForwardTask forwards a task to the appropriate worker
func (r *Router) ForwardTask(ctx context.Context, task core.Task, workerType string) (core.Result, error) {
	var client *http.Client
	var url string

	switch workerType {
	case "light":
		client = r.lightClient
		url = r.config.LightWorkerURL + "/solve"
	case "heavy":
		client = r.heavyClient
		url = r.config.HeavyWorkerURL + "/solve"
	default:
		return core.Result{Success: false}, fmt.Errorf("unknown worker type: %s", workerType)
	}

	// Marshal task to JSON
	taskData, err := json.Marshal(task)
	if err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to marshal task: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(taskData))
	if err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Type", workerType)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return core.Result{Success: false}, fmt.Errorf("worker returned status %d", resp.StatusCode)
	}

	// Parse response
	var result core.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return core.Result{Success: false}, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// SolveHandler handles task solving requests
func (r *Router) SolveHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse task from request
	var task core.Task
	if err := json.NewDecoder(req.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Route task
	workerType := r.RouteTask(task)
	slog.Info("routing task", "task_id", task.ID, "worker_type", workerType)

	// Forward to appropriate worker
	result, err := r.ForwardTask(req.Context(), task, workerType)
	if err != nil {
		slog.Error("task forwarding failed", "error", err, "task_id", task.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HealthHandler returns router health status
func (r *Router) HealthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"agent-router","light_worker":"%s","heavy_worker":"%s"}`,
		r.config.LightWorkerURL, r.config.HeavyWorkerURL)
}

func main() {
	// Load configuration
	config := LoadRouterConfig()

	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create router
	router := NewRouter(config)

	// Setup routes
	mux := http.NewServeMux()
	mux.Handle("/solve", http.HandlerFunc(router.SolveHandler))
	mux.Handle("/health", http.HandlerFunc(router.HealthHandler))

	logger.Info("router starting",
		"port", config.Port,
		"light_worker", config.LightWorkerURL,
		"heavy_worker", config.HeavyWorkerURL)

	log.Fatal(http.ListenAndServe(":"+config.Port, mux))
}
