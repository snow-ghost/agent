package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// healthcheck performs a health check on the worker
func healthcheck() {
	portStr := os.Getenv("WORKER_PORT")
	if portStr == "" {
		portStr = "9004"
	}
	apiPort, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Invalid WORKER_PORT: %v\n", err)
		os.Exit(1)
	}
	servicePort := apiPort + 1

	url := fmt.Sprintf("http://localhost:%d/healthz", servicePort)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Health check failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println("Health check passed")
	os.Exit(0)
}
