.PHONY: help test test-verbose lint build clean deps fmt vet worker check ci

# Default target
help:
	@echo "Available targets:"
	@echo "  test        - Run all tests"
	@echo "  test-verbose- Run all tests with verbose output"
	@echo "  lint        - Run linter (golangci-lint)"
	@echo "  build       - Build all packages"
	@echo "  worker      - Build worker binary"
	@echo "  router      - Build router binary"
	@echo "  binaries    - Build all binaries"
	@echo "  run-worker  - Build and run worker"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Download dependencies"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  check       - Run all checks (fmt, vet, lint, test)"
	@echo "  ci          - Run full CI pipeline"
	@echo "  install-tools- Install development tools"
	@echo "  test-core   - Run core package tests"
	@echo "  test-kb     - Run kb package tests"
	@echo "  test-interp - Run interp package tests"
	@echo "  test-testkit- Run testkit package tests"

# Run all tests
test:
	@echo "Running all tests..."
	go test ./...

# Run all tests with verbose output
test-verbose:
	@echo "Running all tests with verbose output..."
	go test -v ./...

# Run specific package tests
test-core:
	@echo "Running core tests..."
	go test ./core

test-kb:
	@echo "Running kb tests..."
	go test ./kb/...

test-interp:
	@echo "Running interp tests..."
	go test ./interp/...

test-testkit:
	@echo "Running testkit tests..."
	go test ./testkit

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Build all packages
build:
	@echo "Building all packages..."
	go build ./...

# Build worker binary
worker:
	@echo "Building worker binary..."
	go build -o worker ./cmd/worker

# Build router binary
router:
	@echo "Building router binary..."
	go build -o router ./cmd/router

# Build all binaries
binaries: worker router
	@echo "All binaries built successfully"

# Build and run the worker
run-worker:
	@echo "Building and running worker..."
	go run ./cmd/worker

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	go clean ./...
	rm -f worker router
	rm -rf ./hypotheses

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/stretchr/testify/assert@latest

# Check if all tools are available
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || (echo "go not found" && exit 1)
	@command -v golangci-lint >/dev/null 2>&1 || echo "golangci-lint not found (run 'make install-tools')"
	@echo "Tools check complete"

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

# CI pipeline
ci: deps fmt vet lint test build
	@echo "CI pipeline completed successfully"
