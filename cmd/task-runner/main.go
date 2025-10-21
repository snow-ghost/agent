package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/snow-ghost/agent/core"
)

// TaskRequest represents a task submission request
type TaskRequest struct {
	Task core.Task `json:"task"`
}

// TaskResponse represents a task execution response
type TaskResponse struct {
	Success  bool          `json:"success"`
	Output   []byte        `json:"output"`
	Logs     string        `json:"logs"`
	Worker   string        `json:"worker,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// WorkerInfo represents information about which worker handled the task
type WorkerInfo struct {
	Type         string   `json:"type"`
	URL          string   `json:"url"`
	Port         int      `json:"port"`
	Capabilities []string `json:"capabilities"`
}

func main() {
	var (
		taskFile = flag.String("task", "", "Path to task JSON file")
		worker   = flag.String("worker", "auto", "Target worker (auto/light/heavy)")
		router   = flag.String("router", "http://localhost:9006", "Router URL")
		verbose  = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *taskFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -task flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load task from file
	task, err := loadTaskFromFile(*taskFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading task: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Loaded task: %s\n", task.ID)
		fmt.Printf("Domain: %s\n", task.Domain)
		fmt.Printf("Description: %s\n", task.Description)
		fmt.Printf("Budget: %dms CPU, %dMB RAM, %v timeout\n",
			task.Budget.CPUMillis, task.Budget.MemMB, task.Budget.Timeout)
	}

	// Execute task
	start := time.Now()
	result, workerInfo, err := executeTask(context.Background(), task, *worker, *router)
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing task: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Task: %s\n", task.ID)
	fmt.Printf("Worker: %s (%s)\n", workerInfo.Type, workerInfo.URL)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Success: %t\n", result.Success)

	if result.Logs != "" {
		fmt.Printf("Logs: %s\n", result.Logs)
	}

	if result.Success {
		fmt.Printf("Output: %s\n", string(result.Output))
	} else {
		fmt.Printf("Error: %s\n", result.Error)
	}

	// Exit with appropriate code
	if !result.Success {
		os.Exit(1)
	}
}

// loadTaskFromFile loads a task from a JSON file
func loadTaskFromFile(filename string) (core.Task, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return core.Task{}, fmt.Errorf("failed to read task file: %w", err)
	}

	var task core.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return core.Task{}, fmt.Errorf("failed to parse task JSON: %w", err)
	}

	return task, nil
}

// executeTask executes a task via the router or direct worker
func executeTask(ctx context.Context, task core.Task, worker, routerURL string) (*TaskResponse, *WorkerInfo, error) {
	if worker == "auto" {
		return executeViaRouter(ctx, task, routerURL)
	}

	// Execute directly via worker
	workerURL := getWorkerURL(worker)
	return executeViaWorker(ctx, task, workerURL)
}

// executeViaRouter executes a task via the router
func executeViaRouter(ctx context.Context, task core.Task, routerURL string) (*TaskResponse, *WorkerInfo, error) {
	req := TaskRequest{Task: task}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", routerURL+"/solve", bytes.NewReader(reqData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("router returned status %d: %s", resp.StatusCode, string(respData))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(respData, &taskResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Try to get worker info from response headers or response body
	workerInfo := &WorkerInfo{
		Type: "unknown",
		URL:  "unknown",
	}

	// Check if worker info is in the response
	if workerType := resp.Header.Get("X-Worker-Type"); workerType != "" {
		workerInfo.Type = workerType
	}
	if workerURL := resp.Header.Get("X-Worker-URL"); workerURL != "" {
		workerInfo.URL = workerURL
	}

	return &taskResp, workerInfo, nil
}

// executeViaWorker executes a task directly via a worker
func executeViaWorker(ctx context.Context, task core.Task, workerURL string) (*TaskResponse, *WorkerInfo, error) {
	req := TaskRequest{Task: task}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", workerURL+"/solve", bytes.NewReader(reqData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("worker returned status %d: %s", resp.StatusCode, string(respData))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(respData, &taskResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Determine worker type from URL
	workerType := "unknown"
	if workerURL == "http://localhost:9004" || workerURL == "http://light-worker:9004" {
		workerType = "light"
	} else if workerURL == "http://localhost:9002" || workerURL == "http://heavy-worker:9002" {
		workerType = "heavy"
	}

	workerInfo := &WorkerInfo{
		Type: workerType,
		URL:  workerURL,
	}

	return &taskResp, workerInfo, nil
}

// getWorkerURL returns the URL for a specific worker type
func getWorkerURL(workerType string) string {
	switch workerType {
	case "light":
		return "http://localhost:9004"
	case "heavy":
		return "http://localhost:9002"
	default:
		return "http://localhost:9004" // Default to light worker
	}
}
