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

# Create bin directory
bin-dir:
	@mkdir -p bin

# Build all binaries
binaries: bin-dir worker router kb-indexer llmrouter
	@echo "All binaries built successfully"

# Build and run the worker
run-worker:
	@echo "Building and running worker..."
	go run ./cmd/worker

# Run heavy worker
run-heavy:
	@echo "Running heavy worker..."
	WORKER_TYPE=heavy WORKER_PORT=9002 go run ./cmd/worker

# Run light worker
run-light:
	@echo "Running light worker..."
	WORKER_TYPE=light WORKER_PORT=9004 go run ./cmd/worker

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
	rm -f bin/worker bin/router bin/kb-indexer bin/llmrouter
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
	docker build -t llmrouter:latest .

docker-build-no-cache:
	@echo "Building Docker image (no cache)..."
	docker build --no-cache -t llmrouter:latest .

docker-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker-up-build:
	@echo "Building and starting services with docker-compose..."
	docker-compose up --build -d

docker-down:
	@echo "Stopping services with docker-compose..."
	docker-compose down

docker-down-volumes:
	@echo "Stopping services and removing volumes..."
	docker-compose down -v

docker-logs:
	@echo "Showing logs from all services..."
	docker-compose logs -f

docker-logs-llmrouter:
	@echo "Showing logs from llmrouter service..."
	docker-compose logs -f llmrouter

docker-status:
	@echo "Checking service status..."
	docker-compose ps

docker-health:
	@echo "Checking service health..."
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

docker-shell:
	@echo "Opening shell in llmrouter container..."
	docker-compose exec llmrouter /bin/sh

docker-up-tracing:
	@echo "Starting services with tracing..."
	docker-compose --profile tracing up -d

docker-up-monitoring:
	@echo "Starting services with monitoring..."
	docker-compose --profile monitoring up -d

docker-up-all:
	@echo "Starting all services (including monitoring and tracing)..."
	docker-compose --profile tracing --profile monitoring up -d

# Test with real LLM
test-real-llm:
	@echo "Running tests with real LLM..."
	@if [ -z "$$OPENAI_API_KEY" ] && [ -z "$$ANTHROPIC_API_KEY" ]; then \
		echo "Error: No API keys set. Please set OPENAI_API_KEY or ANTHROPIC_API_KEY"; \
		exit 1; \
	fi
	@./scripts/test-real-llm.sh

# Start LLM router for testing
start-llmrouter:
	@echo "Starting LLM Router for testing..."
	@LLMROUTER_PORT=9000 go run ./cmd/llmrouter

# Test with Docker
test-docker:
	@echo "Running tests with Docker LLM Router..."
	@docker-compose -f docker-compose.test.yml up -d llmrouter-test
	@echo "Waiting for LLM Router to start..."
	@sleep 10
	@LLM_ROUTER_URL=http://localhost:9000 go test -v ./worker/heavy -run "TestHeavySolve_WithRealLLM"
	@docker-compose -f docker-compose.test.yml down

# Qdrant Management
qdrant-up:
	@echo "Starting Qdrant..."
	docker-compose up -d qdrant

qdrant-down:
	@echo "Stopping Qdrant..."
	docker-compose stop qdrant

qdrant-logs:
	@echo "Showing Qdrant logs..."
	docker-compose logs -f qdrant

qdrant-stats:
	@echo "Getting Qdrant statistics..."
	go run ./cmd/kb-manager -action stats

qdrant-clear:
	@echo "Clearing Qdrant collection..."
	go run ./cmd/kb-manager -action clear

qdrant-reindex:
	@echo "Reindexing artifacts to Qdrant..."
	go run ./cmd/kb-manager -action reindex -artifacts-dir ./artifacts

qdrant-query:
	@echo "Querying Qdrant collection..."
	@if [ -z "$(QUERY)" ]; then \
		echo "Usage: make qdrant-query QUERY=\"your search query\""; \
		exit 1; \
	fi
	go run ./cmd/kb-manager -action query -query "$(QUERY)"

qdrant-verify:
	@echo "Verifying Qdrant consistency..."
	go run ./cmd/kb-manager -action verify -artifacts-dir ./artifacts

# Test Task Execution
test-simple:
	@echo "Running simple test tasks..."
	go run ./cmd/test-suite -dir ./testdata/tasks/simple

test-complex:
	@echo "Running complex test tasks..."
	go run ./cmd/test-suite -dir ./testdata/tasks/complex

test-decomposable:
	@echo "Running decomposable test tasks..."
	go run ./cmd/test-suite -dir ./testdata/tasks/decomposable

test-all-tasks:
	@echo "Running all test tasks..."
	go run ./cmd/test-suite -dir ./testdata/tasks

test-all-tasks-verbose:
	@echo "Running all test tasks with verbose output..."
	go run ./cmd/test-suite -dir ./testdata/tasks -verbose

test-all-tasks-report:
	@echo "Running all test tasks and generating report..."
	go run ./cmd/test-suite -dir ./testdata/tasks -output test-results.json

# Individual Task Examples
run-task-sort:
	@echo "Running sort numbers task..."
	go run ./cmd/task-runner -task ./testdata/tasks/simple/sort_numbers.json

run-task-fibonacci:
	@echo "Running fibonacci task..."
	go run ./cmd/task-runner -task ./testdata/tasks/complex/fibonacci.json

run-task-pipeline:
	@echo "Running text pipeline task..."
	go run ./cmd/task-runner -task ./testdata/tasks/decomposable/text_pipeline.json -verbose

run-task-custom:
	@echo "Running custom task..."
	@if [ -z "$(TASK)" ]; then \
		echo "Usage: make run-task-custom TASK=path/to/task.json"; \
		exit 1; \
	fi
	go run ./cmd/task-runner -task $(TASK) -verbose

# Docker Integration Tests
docker-test-setup:
	@echo "Setting up Docker test environment..."
	docker-compose up -d qdrant llmrouter light-worker heavy-worker router
	@echo "Waiting for services to start..."
	@sleep 15
	@echo "Reindexing artifacts..."
	@make qdrant-reindex

docker-test-run:
	@echo "Running Docker integration tests..."
	@make docker-test-setup
	@make test-all-tasks
	@make qdrant-stats

docker-test-cleanup:
	@echo "Cleaning up Docker test environment..."
	docker-compose down -v

docker-test-full:
	@echo "Running full Docker test cycle..."
	@make docker-test-run
	@make docker-test-cleanup

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@make qdrant-up
	@sleep 5
	@make qdrant-reindex
	@echo "Development environment ready!"

dev-clean:
	@echo "Cleaning development environment..."
	@make qdrant-clear
	@make docker-down

# CI pipeline
ci: deps fmt vet lint test build
	@echo "CI pipeline completed successfully"
