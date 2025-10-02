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
	"strconv"
	"time"

	"github.com/snow-ghost/agent/core"
)

// RouterConfig holds configuration for the router
type RouterConfig struct {
	LightWorkerURL      string
	HeavyWorkerURL      string
	Port                string
	ComplexityThreshold int
}

// LoadRouterConfig loads router configuration from environment variables
func LoadRouterConfig() *RouterConfig {
	return &RouterConfig{
		LightWorkerURL:      getEnv("LIGHT_WORKER_URL", "http://localhost:8081"),
		HeavyWorkerURL:      getEnv("HEAVY_WORKER_URL", "http://localhost:8082"),
		Port:                getEnv("ROUTER_PORT", "8080"),
		ComplexityThreshold: getEnvInt("COMPLEXITY_THRESHOLD", 5),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
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

// RouteTask determines which worker should handle the task based on capabilities
func (r *Router) RouteTask(task core.Task) (string, string) {
	// Check if task requires sandbox (WASM)
	requiresSandbox := task.Flags.RequiresSandbox

	// Check complexity threshold
	maxComplexity := task.Flags.MaxComplexity
	highComplexity := maxComplexity > r.config.ComplexityThreshold

	// Route to heavy worker if:
	// 1. Task requires sandbox (WASM)
	// 2. Task has high complexity (needs LLM)
	if requiresSandbox || highComplexity {
		return r.config.HeavyWorkerURL, "heavy"
	}

	// Otherwise route to light worker
	return r.config.LightWorkerURL, "light"
}

// RouteTaskLegacy determines which worker should handle the task (legacy method)
func (r *Router) RouteTaskLegacy(task core.Task) string {
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
	workerURL, workerType := r.RouteTask(task)
	slog.Info("routing task", "task_id", task.ID, "worker_type", workerType, "worker_url", workerURL)

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

// CapsHandler returns worker capabilities
func (r *Router) CapsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	caps := map[string]interface{}{
		"light_worker": map[string]interface{}{
			"url": r.config.LightWorkerURL,
			"capabilities": map[string]bool{
				"use_kb":   true,
				"use_wasm": false,
				"use_llm":  false,
			},
		},
		"heavy_worker": map[string]interface{}{
			"url": r.config.HeavyWorkerURL,
			"capabilities": map[string]bool{
				"use_kb":   true,
				"use_wasm": true,
				"use_llm":  true,
			},
		},
		"routing_rules": map[string]interface{}{
			"requires_sandbox":         "heavy",
			"max_complexity_threshold": r.config.ComplexityThreshold,
			"high_complexity":          "heavy",
			"default":                  "light",
		},
	}

	json.NewEncoder(w).Encode(caps)
}

// ReadyHandler returns readiness status
func (r *Router) ReadyHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if workers are reachable
	lightReady := r.checkWorkerReady(r.config.LightWorkerURL)
	heavyReady := r.checkWorkerReady(r.config.HeavyWorkerURL)

	allReady := lightReady && heavyReady

	status := "ready"
	if !allReady {
		status = "not_ready"
	}

	response := map[string]interface{}{
		"status": status,
		"workers": map[string]bool{
			"light": lightReady,
			"heavy": heavyReady,
		},
	}

	if allReady {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// checkWorkerReady checks if a worker is ready
func (r *Router) checkWorkerReady(workerURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(workerURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
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
	mux.Handle("/caps", http.HandlerFunc(router.CapsHandler))
	mux.Handle("/ready", http.HandlerFunc(router.ReadyHandler))

	logger.Info("router starting",
		"port", config.Port,
		"light_worker", config.LightWorkerURL,
		"heavy_worker", config.HeavyWorkerURL)

	log.Fatal(http.ListenAndServe(":"+config.Port, mux))
}
