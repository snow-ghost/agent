# Agent - AI Problem Solver

A Go-based AI agent system that uses evolutionary algorithms and WebAssembly to solve computational problems. The agent learns from successful solutions and builds a knowledge base of reusable skills.

## Features

- **Knowledge Base**: In-memory registry of skills with persistence
- **WASM Interpreter**: Sandboxed execution using wazero runtime
- **Evolutionary Algorithm**: Mutates and improves solutions over time
- **LLM Integration**: Mock LLM client for algorithm proposals
- **Hypothesis Persistence**: Saves successful solutions for reuse
- **Structured Logging**: JSON logs with contextual information
- **Metrics & Health**: HTTP endpoints for monitoring
- **Policy Guard**: Security controls and resource limits

## Quick Start

### Prerequisites

- Go 1.21 or later
- Make (optional, for build automation)

### Installation

```bash
# Clone the repository
git clone https://github.com/snow-ghost/agent.git
cd agent

# Install dependencies
go mod tidy

# Build the worker
go build -o worker-bin ./cmd/worker
```

### Running the Agent

```bash
# Start the worker with default settings
./worker-bin

# Or with custom configuration
WORKER_PORT=8080 LLM_MODE=mock ./worker-bin
```

The worker will start on port 8080 (or the port specified by `WORKER_PORT`) and provide:
- `/solve` - POST endpoint for submitting tasks
- `/health` - Health check endpoint
- `/metrics` - Prometheus-compatible metrics

## Configuration

The agent can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_PORT` | `8081` | HTTP server port |
| `LLM_MODE` | `mock` | LLM mode (`mock` or `disabled`) |
| `POLICY_ALLOW_TOOLS` | `example.com,api.example.com` | Comma-separated list of allowed domains for HTTP tools |
| `SANDBOX_MEM_MB` | `4` | WASM sandbox memory limit in MB |
| `TASK_TIMEOUT` | `30s` | Default task timeout duration |
| `HYPOTHESES_DIR` | `./hypotheses` | Directory for saving successful hypotheses |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

### Example Configuration

```bash
export WORKER_PORT=8080
export LLM_MODE=mock
export POLICY_ALLOW_TOOLS="api.github.com,api.openai.com"
export SANDBOX_MEM_MB=8
export TASK_TIMEOUT=60s
export HYPOTHESES_DIR="/var/lib/agent/hypotheses"
export LOG_LEVEL=debug

./worker
```

## Usage

### Submitting a Task

Send a POST request to `/solve` with a JSON task:

**Via Router (recommended):**
```bash
curl -X POST http://localhost:8083/solve \
  -H "Content-Type: application/json" \
  -d '{
    "id": "sort-task-1",
    "domain": "algorithms.sorting",
    "spec": {
      "success_criteria": ["sorted_non_decreasing"],
      "props": {"type": "sort"},
      "metrics_weights": {"cases_passed": 1.0, "cases_total": 0.0}
    },
    "input": "{\"numbers\": [3,1,2]}",
    "budget": {
      "cpu_millis": 1000,
      "timeout": "5s"
    },
    "flags": {
      "requires_sandbox": true,
      "max_complexity": 5
    },
    "created_at": "2024-01-01T00:00:00Z"
  }'
```

**Direct to Worker:**
```bash
curl -X POST http://localhost:8081/solve \
  -H "Content-Type: application/json" \
  -d '{
    "id": "sort-task-1",
    "domain": "algorithms.sorting",
    "spec": {
      "success_criteria": ["sorted_non_decreasing"],
      "props": {"type": "sort"},
      "metrics_weights": {"cases_passed": 1.0, "cases_total": 0.0}
    },
    "input": "{\"numbers\": [3,1,2]}",
    "budget": {
      "cpu_millis": 1000,
      "timeout": "5s"
    },
    "flags": {
      "requires_sandbox": true,
      "max_complexity": 5
    },
    "created_at": "2024-01-01T00:00:00Z"
  }'
```

### Task Format

```json
{
  "id": "unique-task-id",
  "domain": "problem-domain",
  "spec": {
    "success_criteria": ["criterion1", "criterion2"],
    "props": {"key": "value"},
    "metrics_weights": {"metric": 1.0}
  },
  "input": "{\"data\": [1,2,3]}",
  "budget": {
    "cpu_millis": 1000,
    "timeout": "30s"
  },
  "flags": {
    "requires_sandbox": true,
    "max_complexity": 5
  },
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Response Format

```json
{
  "Success": true,
  "Score": 0.95,
  "Output": "{\"result\": [1,2,3]}",
  "Logs": "Task solved by KB skill algorithms/sort.v1",
  "Metrics": {
    "cases_passed": 5,
    "cases_total": 5,
    "execution_time_ms": 150
  }
}
```

## Monitoring

### Health Check

**Router:**
```bash
curl http://localhost:8083/health
```

**Worker:**
```bash
curl http://localhost:8081/health
```

Response:
```json
{"status":"ok","service":"agent-worker"}
```

### Metrics

**Router:**
```bash
curl http://localhost:8083/metrics
```

**Worker:**
```bash
curl http://localhost:8081/metrics
```

Returns Prometheus-compatible metrics including:
- `tasks_total` - Total tasks processed
- `tasks_solved` - Successfully solved tasks
- `tasks_failed` - Failed tasks
- `avg_solve_time_ms` - Average solve time
- `test_pass_rate` - Test pass rate
- Go runtime metrics (`memstats`, `cmdline`)

## Development

### Building

```bash
# Build all components
make build

# Build specific component
go build -o worker-bin ./cmd/worker
```

### Testing

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run specific package tests
go test ./kb/memory
```

### Linting

```bash
# Run all linters
make lint

# Format code
make fmt

# Run go vet
make vet
```

### Development Workflow

```bash
# Install development tools
make install-tools

# Run full CI pipeline
make ci

# Clean build artifacts
make clean
```

## Artifact Knowledge Base

The system now supports a unified artifact-based knowledge base that replaces Go skills with standardized artifacts containing WASM code and metadata.

### Artifact Structure

Each artifact is stored in a directory with the following structure:
```
artifacts/
├── artifact-id@version/
│   ├── manifest.json    # Artifact metadata
│   └── code.wasm       # WASM bytecode (for WASM artifacts)
```

### Manifest Format

```json
{
  "id": "alg.sort.v1",
  "version": "1.0.0",
  "domain": "algorithms.sorting",
  "description": "Stable integer sort",
  "tags": ["sort", "stable"],
  "lang": "wasm",
  "entry": "solve",
  "code_path": "code.wasm",
  "sha256": "...",
  "tests": [...],
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Features

- **Unified Storage**: Both WASM and Go skills stored as artifacts
- **SHA256 Verification**: Automatic integrity checking for WASM artifacts
- **Tag-based Search**: Find artifacts by domain, tags, or keywords
- **Vector Search (RAG)**: Semantic search using embeddings for better artifact discovery
- **Automatic Migration**: Existing Go skills can be converted to artifacts
- **Hypothesis Persistence**: LLM-generated solutions saved as artifacts

### Usage

The system automatically uses the artifact-based knowledge base when `ARTIFACTS_DIR` is configured. Workers will:
1. Load all artifacts on startup
2. Convert them to skills for task solving
3. Save successful hypotheses as new artifacts
4. Support both WASM and Go skill artifacts during migration

### Vector Search (RAG)

The system includes advanced vector search capabilities for semantic artifact discovery:

#### Embedders
- **Mock TF-IDF**: Local TF-IDF based embedder for testing
- **OpenAI**: Production-ready embeddings using OpenAI's API

#### Vector Stores
- **Memory**: In-memory cosine similarity search
- **Qdrant**: Production vector database (placeholder implementation)

#### Indexing Artifacts
```bash
# Index artifacts using mock embedder
./kb-indexer -artifacts-dir ./artifacts -embedder mock -vector-store memory

# Index with OpenAI embeddings
export OPENAI_API_KEY=your_key
./kb-indexer -artifacts-dir ./artifacts -embedder openai -vector-store memory

# Show index statistics
./kb-indexer -stats
```

#### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDINGS_MODEL` | `text-embedding-3-small` | OpenAI embedding model |
| `EMBEDDINGS_DIMENSION` | `1536` | Vector dimension |
| `QDRANT_URL` | `localhost:6333` | Qdrant server URL |
| `QDRANT_API_KEY` | - | Qdrant API key |
| `QDRANT_COLLECTION` | `artifacts` | Qdrant collection name |

### Testing the Artifact System

1. **Start with artifacts directory:**
   ```bash
   export ARTIFACTS_DIR=./artifacts
   ./worker-bin
   ```

2. **Submit a task that will be solved by artifacts:**
   ```bash
   curl -X POST http://localhost:8083/solve \
     -H "Content-Type: application/json" \
     -d '{
       "id": "test-sort",
       "domain": "algorithms.sorting",
       "spec": {
         "success_criteria": ["sorted_non_decreasing"],
         "props": {"type": "sort"},
         "metrics_weights": {"cases_passed": 1.0}
       },
       "input": "{\"numbers\": [3,1,2]}",
       "budget": {"cpu_millis": 1000, "timeout": "5s"},
       "flags": {"requires_sandbox": true, "max_complexity": 5},
       "created_at": "2024-01-01T00:00:00Z"
     }'
   ```

3. **Check that artifacts are created:**
   ```bash
   ls -la ./artifacts/
   # Should show artifact directories with manifest.json and code.wasm
   ```

4. **Verify hypothesis persistence:**
   - First run: Task solved by LLM, hypothesis saved as artifact
   - Second run: Task solved by artifact from knowledge base

## Docker Deployment

### Quick Start

1. **Build and start all services:**
   ```bash
   make docker-up
   ```

2. **Access the services:**
   - Router: http://localhost:8083
   - Light Worker: http://localhost:8081
   - Heavy Worker: http://localhost:8082

3. **With Nginx load balancer:**
   ```bash
   make docker-up-nginx
   ```
   - Access via: http://localhost (port 80)

### Service Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   Nginx         │    │   Router        │
│  (Port 80)      │───▶│  (Port 8083)    │
│  Load Balancer  │    │  Capability-    │
│                 │    │  Based Router   │
└─────────────────┘    └─────────────────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            ┌─────────────────┐    ┌─────────────────┐
            │  Light Worker   │    │  Heavy Worker   │
            │  (Port 8081)    │    │  (Port 8082)    │
            │  KB Only        │    │  LLM+WASM+KB    │
            │  Capabilities:  │    │  Capabilities:  │
            │  KB             │    │  KB+WASM+LLM    │
            └─────────────────┘    └─────────────────┘
```

### Worker Capabilities

The system supports two types of workers with different capabilities:

#### Light Worker
- **Capabilities**: KB only
- **Use Cases**: Simple tasks that can be solved with existing knowledge
- **Performance**: Fast, low resource usage
- **Endpoints**: `/solve`, `/health`, `/metrics`, `/caps`, `/ready`

#### Heavy Worker  
- **Capabilities**: KB + WASM + LLM
- **Use Cases**: Complex tasks requiring code generation and execution
- **Performance**: Slower, higher resource usage
- **Endpoints**: `/solve`, `/health`, `/metrics`, `/caps`, `/ready`

#### Routing Logic
Tasks are automatically routed based on their requirements:
- **Requires Sandbox** → Heavy Worker (needs WASM)
- **High Complexity** (> threshold) → Heavy Worker (needs LLM)
- **Default** → Light Worker (KB only)

### Docker Commands

```bash
# Build Docker image
make docker-build

# Start all services
make docker-up

# Stop all services
make docker-down

# View logs
make docker-logs

# Start with nginx
make docker-up-nginx
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_TYPE` | `heavy` | Worker type: `light` or `heavy` |
| `WORKER_PORT` | `8081` | Port for worker service |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `HYPOTHESES_DIR` | `./hypotheses` | Directory for saved hypotheses (legacy) |
| `ARTIFACTS_DIR` | `./artifacts` | Directory for artifact-based knowledge base |
| `LLM_MODE` | `mock` | LLM mode: `mock` or `real` |
| `SANDBOX_MEM_MB` | `4` | Memory limit for WASM sandbox |
| `TASK_TIMEOUT` | `30s` | Default task timeout |
| `COMPLEXITY_THRESHOLD` | `5` | Complexity threshold for heavy worker routing |

### Health Checks

All services include comprehensive health check endpoints:

#### Router Endpoints
- `GET /health` - Basic health status
- `GET /caps` - Worker capabilities and routing rules
- `GET /ready` - Readiness status (checks worker availability)

#### Worker Endpoints
- `GET /health` - Basic health status
- `GET /metrics` - Prometheus-compatible metrics
- `GET /caps` - Worker capabilities
- `GET /ready` - Readiness status

#### Example Usage
```bash
# Check router capabilities
curl http://localhost:8083/caps

# Check if all workers are ready
curl http://localhost:8083/ready

# Check specific worker capabilities
curl http://localhost:8081/caps  # Light worker
curl http://localhost:8082/caps  # Heavy worker
```

## Architecture

### Components

- **Core**: Domain types, interfaces, and business logic
- **KB/Memory**: In-memory knowledge base with persistence
- **Interp/WASM**: WebAssembly interpreter using wazero
- **LLM/Mock**: Mock LLM client for algorithm proposals
- **TestKit**: Test runner and evaluation framework
- **Worker**: Main solver with evolutionary algorithm
- **Policy**: Security controls and resource limits

### Data Flow

1. **Task Submission**: HTTP request → Ingestor → Solver
2. **Knowledge Base Check**: Search for existing skills
3. **LLM Proposal**: Generate algorithm if no KB match
4. **Evolution**: Mutate and test hypotheses
5. **Execution**: Run WASM in sandboxed environment
6. **Persistence**: Save successful solutions to KB
7. **Response**: Return result with metrics

### Security

- **Sandboxed Execution**: WASM runs in isolated environment
- **Resource Limits**: Memory and CPU constraints
- **Policy Guard**: Tool allowlisting and timeout controls
- **Input Validation**: JSON schema validation

## Troubleshooting

### Common Issues

**Worker won't start:**
- Check if port is available
- Verify Go version (1.21+)
- Run `go mod tidy` to update dependencies

**Tasks failing:**
- Check input format matches expected schema
- Verify domain matches available skills
- Check logs for detailed error messages

**Memory issues:**
- Increase `SANDBOX_MEM_MB` for complex tasks
- Monitor metrics for memory usage patterns

### Debugging

Enable debug logging:
```bash
export LOG_LEVEL=debug
./worker-bin
```

Check worker logs for detailed execution information.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters
5. Submit a pull request

## Support

For issues and questions:
- Create an issue on GitHub
- Check the troubleshooting section
- Review the logs for error details