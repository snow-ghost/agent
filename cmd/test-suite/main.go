package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/snow-ghost/agent/core"
)

// TestResult represents the result of a single test
type TestResult struct {
	TaskID   string        `json:"task_id"`
	TaskFile string        `json:"task_file"`
	Success  bool          `json:"success"`
	Worker   string        `json:"worker"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Output   string        `json:"output,omitempty"`
	Logs     string        `json:"logs,omitempty"`
}

// TestSuiteReport represents the overall test suite results
type TestSuiteReport struct {
	TotalTests    int            `json:"total_tests"`
	PassedTests   int            `json:"passed_tests"`
	FailedTests   int            `json:"failed_tests"`
	TotalDuration time.Duration  `json:"total_duration"`
	WorkerStats   map[string]int `json:"worker_stats"`
	Results       []TestResult   `json:"results"`
	Summary       string         `json:"summary"`
}

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

func main() {
	var (
		taskDir = flag.String("dir", "", "Directory containing test tasks")
		router  = flag.String("router", "http://localhost:9006", "Router URL")
		output  = flag.String("output", "", "Output file for JSON report (optional)")
		verbose = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *taskDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -dir flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Find all task files
	taskFiles, err := findTaskFiles(*taskDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding task files: %v\n", err)
		os.Exit(1)
	}

	if len(taskFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No task files found in %s\n", *taskDir)
		os.Exit(1)
	}

	fmt.Printf("Found %d test tasks in %s\n", len(taskFiles), *taskDir)

	// Run tests
	report, err := runTestSuite(context.Background(), taskFiles, *router, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running test suite: %v\n", err)
		os.Exit(1)
	}

	// Display results
	displayResults(report, *verbose)

	// Save JSON report if requested
	if *output != "" {
		if err := saveReport(report, *output); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Report saved to %s\n", *output)
	}

	// Exit with appropriate code
	if report.FailedTests > 0 {
		os.Exit(1)
	}
}

// findTaskFiles finds all JSON task files in a directory
func findTaskFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// runTestSuite runs all tests and returns a report
func runTestSuite(ctx context.Context, taskFiles []string, routerURL string, verbose bool) (*TestSuiteReport, error) {
	report := &TestSuiteReport{
		TotalTests:  len(taskFiles),
		WorkerStats: make(map[string]int),
		Results:     make([]TestResult, 0, len(taskFiles)),
	}

	start := time.Now()

	for i, taskFile := range taskFiles {
		if verbose {
			fmt.Printf("\n[%d/%d] Running %s...\n", i+1, len(taskFiles), taskFile)
		}

		result, err := runSingleTest(ctx, taskFile, routerURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running test %s: %v\n", taskFile, err)
			result = &TestResult{
				TaskFile: taskFile,
				Success:  false,
				Error:    err.Error(),
			}
		}

		report.Results = append(report.Results, *result)
		report.WorkerStats[result.Worker]++

		if result.Success {
			report.PassedTests++
		} else {
			report.FailedTests++
		}

		if verbose {
			status := "PASS"
			if !result.Success {
				status = "FAIL"
			}
			fmt.Printf("  %s (%v) - Worker: %s\n", status, result.Duration, result.Worker)
		}
	}

	report.TotalDuration = time.Since(start)
	report.Summary = generateSummary(report)

	return report, nil
}

// runSingleTest runs a single test task
func runSingleTest(ctx context.Context, taskFile, routerURL string) (*TestResult, error) {
	// Load task
	task, err := loadTaskFromFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	// Execute task
	start := time.Now()
	response, worker, err := executeTask(ctx, task, routerURL)
	duration := time.Since(start)

	result := &TestResult{
		TaskID:   task.ID,
		TaskFile: taskFile,
		Duration: duration,
		Worker:   worker,
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	result.Success = response.Success
	result.Output = string(response.Output)
	result.Logs = response.Logs
	result.Error = response.Error

	return result, nil
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

// executeTask executes a task via the router
func executeTask(ctx context.Context, task core.Task, routerURL string) (*TaskResponse, string, error) {
	req := TaskRequest{Task: task}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", routerURL+"/solve", strings.NewReader(string(reqData)))
	if err != nil {
		return nil, "unknown", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "unknown", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "unknown", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "unknown", fmt.Errorf("router returned status %d: %s", resp.StatusCode, string(respData))
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(respData, &taskResp); err != nil {
		return nil, "unknown", fmt.Errorf("failed to parse response: %w", err)
	}

	// Determine worker from response headers
	worker := "unknown"
	if workerType := resp.Header.Get("X-Worker-Type"); workerType != "" {
		worker = workerType
	}

	return &taskResp, worker, nil
}

// displayResults displays the test results
func displayResults(report *TestSuiteReport, verbose bool) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("TEST SUITE RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Passed: %d\n", report.PassedTests)
	fmt.Printf("Failed: %d\n", report.FailedTests)
	fmt.Printf("Duration: %v\n", report.TotalDuration)
	fmt.Printf("\nWorker Distribution:\n")
	for worker, count := range report.WorkerStats {
		fmt.Printf("  %s: %d tests\n", worker, count)
	}

	if verbose && len(report.Results) > 0 {
		fmt.Printf("\nDetailed Results:\n")
		for _, result := range report.Results {
			status := "PASS"
			if !result.Success {
				status = "FAIL"
			}
			fmt.Printf("  %s %s (%v) - %s\n", status, result.TaskID, result.Duration, result.Worker)
			if !result.Success && result.Error != "" {
				fmt.Printf("    Error: %s\n", result.Error)
			}
		}
	}

	fmt.Printf("\n%s\n", report.Summary)
}

// generateSummary generates a summary of the test results
func generateSummary(report *TestSuiteReport) string {
	passRate := float64(report.PassedTests) / float64(report.TotalTests) * 100
	avgDuration := report.TotalDuration / time.Duration(report.TotalTests)

	if report.FailedTests == 0 {
		return fmt.Sprintf("All tests passed! (%.1f%% success rate, avg %v per test)",
			passRate, avgDuration)
	}

	return fmt.Sprintf("Test suite completed with %d failures (%.1f%% success rate, avg %v per test)",
		report.FailedTests, passRate, avgDuration)
}

// saveReport saves the test report to a JSON file
func saveReport(report *TestSuiteReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
