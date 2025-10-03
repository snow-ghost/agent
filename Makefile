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
	@echo "  run-heavy   - Run heavy worker (LLM+WASM+KB)"
	@echo "  run-light   - Run light worker (KB only)"
	@echo "  run-router  - Run router (capability-based routing)"
	@echo "  run-llmrouter- Run LLM router (REST API + SSE)"
	@echo "  reindex     - Reindex artifacts for vector search"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Download dependencies"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  check       - Run all checks (fmt, vet, lint, test)"
	@echo "  docker-build- Build Docker image"
	@echo "  docker-up   - Start services with docker-compose"
	@echo "  docker-down - Stop services with docker-compose"
	@echo "  docker-logs - Show logs from all services"
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
	go build -o bin/worker ./cmd/worker

# Build router binary
router:
	@echo "Building router binary..."
	go build -o bin/router ./cmd/router

# Build kb-indexer binary
kb-indexer:
	@echo "Building kb-indexer binary..."
	go build -o bin/kb-indexer ./cmd/kb-indexer

# Build llmrouter binary
llmrouter:
	@echo "Building llmrouter binary..."
	go build -o bin/llmrouter ./cmd/llmrouter

# Build all binaries
binaries: worker router kb-indexer llmrouter
	@echo "All binaries built successfully"

# Build and run the worker
run-worker:
	@echo "Building and running worker..."
	go run ./cmd/worker

# Run heavy worker
run-heavy:
	@echo "Running heavy worker..."
	WORKER_TYPE=heavy WORKER_PORT=8082 go run ./cmd/worker

# Run light worker
run-light:
	@echo "Running light worker..."
	WORKER_TYPE=light WORKER_PORT=8081 go run ./cmd/worker

# Build and run the router
run-router:
	@echo "Building and running router..."
	go run ./cmd/router

# Run LLM router
run-llmrouter:
	@echo "Running LLM router..."
	LLMROUTER_PORT=8085 go run ./cmd/llmrouter

# Reindex artifacts
reindex:
	@echo "Reindexing artifacts..."
	@if [ -z "$(ARTIFACTS_DIR)" ]; then \
		echo "Error: ARTIFACTS_DIR not set. Usage: make reindex ARTIFACTS_DIR=./artifacts"; \
		exit 1; \
	fi
	@echo "Indexing artifacts in $(ARTIFACTS_DIR)..."
	EMBEDDINGS_MODE=mock VECTOR_BACKEND=memory go run ./cmd/kb-indexer -artifacts-dir $(ARTIFACTS_DIR)

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
	rm -f worker router kb-indexer llmrouter
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

# Docker operations
docker-build:
	@echo "Building Docker image..."
	docker build -t agent-worker .

docker-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping services with docker-compose..."
	docker-compose down

docker-logs:
	@echo "Showing logs from all services..."
	docker-compose logs -f

docker-up-nginx:
	@echo "Starting services with nginx..."
	docker-compose --profile with-nginx up -d

# CI pipeline
ci: deps fmt vet lint test build
	@echo "CI pipeline completed successfully"
