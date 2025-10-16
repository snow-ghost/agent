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
WORKER_PORT=9002 LLM_MODE=mock ./worker-bin

# Start with artifact-based knowledge base
ARTIFACTS_DIR=./artifacts ./worker-bin

# Start with vector search enabled
ARTIFACTS_DIR=./artifacts EMBEDDINGS_MODE=mock VECTOR_BACKEND=memory ./worker-bin
```

The worker will start on port 9002 (or the port specified by `WORKER_PORT`) and provide:
- `/solve` - POST endpoint for submitting tasks
- `/health` - Health check endpoint
- `/metrics` - Prometheus-compatible metrics

## Configuration

The agent can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_PORT` | `9002` | HTTP server port |
| `LLM_MODE` | `mock` | LLM mode (`mock` or `disabled`) |
| `LLM_ROUTER_URL` | `http://localhost:9000` | LLM router base URL |
| `LLM_MODEL` | `lmstudio:qwen/qwen3-4b-2507` | Default LLM model for design |
| `MAX_CODE_BYTES` | `65536` | Max AF-DSL source size (bytes) |
| `DSL_MAX_STEPS` | `100000` | AF-DSL runtime max execution steps |
| `DSL_MAX_DEPTH` | `128` | AF-DSL runtime max call depth |
| `PROP_K` | `64` | Property tests to generate |
| `POLICY_ALLOW_TOOLS` | `example.com,api.example.com` | Comma-separated list of allowed domains for HTTP tools |
| `SANDBOX_MEM_MB` | `4` | WASM sandbox memory limit in MB |
| `TASK_TIMEOUT` | `30s` | Default task timeout duration |
| `HYPOTHESES_DIR` | `./hypotheses` | Directory for saving successful hypotheses |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

#### Metrics Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_MODE` | `prom` | Metrics mode: `prom` (Prometheus) or `otel` (OpenTelemetry) |
| `METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `SERVICE_NAME` | `agent` | Service name for metrics |
| `METRICS_NAMESPACE` | `agent` | Metrics namespace prefix |
| `METRICS_COLLECT_RUNTIME` | `true` | Collect Go runtime metrics |

### Example Configuration

**Basic Configuration:**
```bash
export WORKER_PORT=9002
export LLM_MODE=mock
export POLICY_ALLOW_TOOLS="api.github.com,api.openai.com"
export SANDBOX_MEM_MB=8
export TASK_TIMEOUT=60s
export HYPOTHESES_DIR="/var/lib/agent/hypotheses"
export LOG_LEVEL=debug

# Metrics configuration
export METRICS_MODE=prom
export SERVICE_NAME=agent-worker
export METRICS_NAMESPACE=agent
export METRICS_COLLECT_RUNTIME=true

./worker
```

**OpenTelemetry Configuration:**
```bash
export WORKER_PORT=9002
export LLM_MODE=mock
export LOG_LEVEL=info

# OpenTelemetry metrics
export METRICS_MODE=otel
export SERVICE_NAME=agent-worker
export METRICS_NAMESPACE=agent
export METRICS_COLLECT_RUNTIME=true

./worker
```

## Usage

### Submitting a Task

Send a POST request to `/solve` with a JSON task:

**Via Router (recommended):**
```bash
curl -X POST http://localhost:9006/solve \
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
curl -X POST http://localhost:9004/solve \
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
curl http://localhost:9007/healthz
```

**Worker:**
```bash
curl http://localhost:9005/healthz
```

Response:
```json
{"status":"ok","service":"agent-worker"}
```

### Metrics

The system provides comprehensive metrics in both Prometheus and OpenTelemetry formats.

#### Accessing Metrics

**LLM Router:**
```bash
curl http://localhost:9001/metrics
```

**Router:**
```bash
curl http://localhost:9007/metrics
```

**Workers:**
```bash
curl http://localhost:9005/metrics  # Light worker
curl http://localhost:9003/metrics  # Heavy worker
```

#### Metrics Configuration

Configure metrics using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_MODE` | `prom` | Metrics mode: `prom` (Prometheus) or `otel` (OpenTelemetry) |
| `METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `SERVICE_NAME` | `agent` | Service name for metrics |
| `METRICS_NAMESPACE` | `agent` | Metrics namespace prefix |
| `METRICS_COLLECT_RUNTIME` | `true` | Collect Go runtime metrics |

#### Enabling OpenTelemetry Mode

To use OpenTelemetry instead of Prometheus:

```bash
export METRICS_MODE=otel
export SERVICE_NAME=agent-worker
export METRICS_NAMESPACE=agent
export METRICS_COLLECT_RUNTIME=true
./worker-bin
```

#### Available Metrics

##### Worker Metrics

**Task Processing:**
- `worker_task_received_total{worker_type,domain}` - Total tasks received
- `worker_task_completed_total{worker_type,domain,status}` - Tasks completed by status
- `worker_task_duration_seconds{worker_type,domain}` - Task execution duration (histogram)

**Solve Stages:**
- `worker_solve_stage_seconds{stage}` - Time spent in each solve stage (histogram)
  - Stages: `kb`, `llm`, `evolve`, `interpret`, `tests`

**Knowledge Base:**
- `worker_kb_hits_total` - KB cache hits
- `worker_kb_misses_total` - KB cache misses
- `worker_kb_artifacts_loaded` - Number of loaded artifacts (gauge)
- `worker_kb_save_artifact_total` - Artifacts saved to KB

**RAG/Vector Search:**
- `worker_rag_hits_total` - RAG search hits
- `worker_rag_search_total{backend}` - RAG searches performed
- `worker_rag_search_duration_seconds{backend}` - RAG search duration (histogram)
- `worker_rag_candidates_found{backend}` - Candidates found in search (histogram)

**Sandbox/Execution:**
- `worker_sandbox_exec_total{result}` - Sandbox executions by result
- `worker_sandbox_exec_seconds` - Sandbox execution duration (histogram)

**Policy & Security:**
- `worker_policy_denied_total{reason}` - Policy denials by reason

**Evolution & Testing:**
- `worker_mutations_total{kind}` - Mutations performed by type
- `worker_tests_run_total{result}` - Tests run by result
- `worker_tests_duration_seconds` - Test execution duration (histogram)

##### LLM Router Metrics

**Request Processing:**
- `llm_requests_total{provider,model,status,cache}` - LLM requests by provider/model
- `llm_request_duration_seconds{provider,model}` - Request duration (histogram)

**Token Usage:**
- `llm_tokens_input_total{provider,model}` - Input tokens consumed
- `llm_tokens_output_total{provider,model}` - Output tokens generated

**Cost Tracking:**
- `llm_cost_total{provider,model,currency}` - Total cost by provider/model

**Reliability:**
- `llm_retries_total{provider,model}` - Retry attempts
- `llm_circuit_open_total{provider,model}` - Circuit breaker activations

##### HTTP Metrics

**Request Metrics:**
- `http_requests_total{path,method,code}` - HTTP requests by path/method/status
- `http_request_duration_seconds{path,method}` - HTTP request duration (histogram)

#### Metrics Best Practices

##### Label Cardinality

**✅ Good Labels (Low Cardinality):**
- `worker_type`: `light`, `heavy`
- `domain`: `algorithms.sorting`, `data.structures`
- `provider`: `openai`, `anthropic`, `mock`
- `model`: `gpt-4`, `claude-3`, `mock`
- `status`: `ok`, `error`, `timeout`
- `stage`: `kb`, `llm`, `evolve`, `interpret`, `tests`

**❌ Bad Labels (High Cardinality):**
- `task_id`: Unique per task (avoid!)
- `user_id`: Unique per user (avoid!)
- `request_id`: Unique per request (avoid!)
- `timestamp`: Changes constantly (avoid!)

##### Required Labels

All metrics must include these mandatory labels:
- `service`: Service name (e.g., `agent-worker`, `agent-router`)
- `worker_type`: For worker metrics (`light`, `heavy`)
- `domain`: For task-related metrics
- `provider`: For LLM metrics
- `model`: For LLM metrics

##### Metric Types

**Counters** - For events that only increase:
- `worker_task_received_total`
- `llm_requests_total`
- `worker_kb_hits_total`

**Histograms** - For latencies and durations:
- `worker_task_duration_seconds`
- `llm_request_duration_seconds`
- `worker_solve_stage_seconds`

**Gauges** - For current state/size:
- `worker_kb_artifacts_loaded`
- `worker_memory_usage_bytes`

#### Example Queries

**Task Success Rate:**
```promql
rate(worker_task_completed_total{status="ok"}[5m]) / rate(worker_task_received_total[5m])
```

**Average Task Duration:**
```promql
histogram_quantile(0.5, rate(worker_task_duration_seconds_bucket[5m]))
```

**LLM Request Rate:**
```promql
rate(llm_requests_total[5m])
```

**KB Hit Rate:**
```promql
rate(worker_kb_hits_total[5m]) / (rate(worker_kb_hits_total[5m]) + rate(worker_kb_misses_total[5m]))
```

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

### Running Workers

```bash
# Run heavy worker (LLM+WASM+KB)
make run-heavy

# Run light worker (KB only)
make run-light

# Run router (capability-based routing)
make run-router

# Reindex artifacts for vector search
make reindex ARTIFACTS_DIR=./artifacts
```

## Artifact Knowledge Base

The system supports a unified artifact-based knowledge base that replaces Go skills with standardized artifacts containing WASM code and metadata.

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
  "sha256": "abc123...",
  "embedding_model": "text-embedding-3-small",
  "embedding": [0.1, 0.2, ...],
  "tests": [
    {
      "name": "sort_test_1",
      "input": "[3,1,2]",
      "oracle": "[1,2,3]",
      "checks": ["sorted_non_decreasing"],
      "weight": 1.0
    }
  ],
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Artifact Types

#### WASM Artifacts
- **Language**: `"wasm"`
- **Entry Point**: `"solve"` function
- **Code File**: `code.wasm` (WebAssembly bytecode)
- **SHA256**: Verified integrity checksum
- **Execution**: Sandboxed WASM runtime

#### Go Skill Artifacts (Migration)
- **Language**: `"go-skill"`
- **Entry Point**: Package function name (e.g., `"algorithms.Sort"`)
- **Code File**: None (compiled into binary)
- **SHA256**: Not applicable
- **Execution**: Direct Go function call

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

# Using Makefile
make reindex ARTIFACTS_DIR=./artifacts
```

#### Configuration Examples

**Local Development (Mock Embeddings)**
```bash
export ARTIFACTS_DIR=./artifacts
export EMBEDDINGS_MODE=mock
export VECTOR_BACKEND=memory
export INDEX_ON_START=true
./worker-bin
```

**Production (OpenAI + Qdrant)**
```bash
export ARTIFACTS_DIR=/var/lib/agent/artifacts
export EMBEDDINGS_MODE=openai
export EMBEDDINGS_MODEL=text-embedding-3-large
export VECTOR_BACKEND=qdrant
export QDRANT_URL=qdrant.example.com:6333
export QDRANT_API_KEY=your_api_key
export OPENAI_API_KEY=your_openai_key
./worker-bin
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
   curl -X POST http://localhost:9006/solve \
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
  - Router: http://localhost:9006
  - Light Worker: http://localhost:9004
  - Heavy Worker: http://localhost:9002

3. **With Nginx load balancer:**
   ```bash
   make docker-up-nginx
   ```
   - Access via: http://localhost (port 80)

### Service Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   Nginx         │    │   Router        │
│  (Port 80)      │───▶│  (Port 9006)    │
│  Load Balancer  │    │  Capability-    │
│                 │    │  Based Router   │
└─────────────────┘    └─────────────────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            ┌─────────────────┐    ┌─────────────────┐
            │  Light Worker   │    │  Heavy Worker   │
            │  (Port 9004)    │    │  (Port 9002)    │
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

#### Core Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_TYPE` | `heavy` | Worker type: `light` or `heavy` |
| `WORKER_PORT` | `9002` | Port for worker service |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `TASK_TIMEOUT` | `30s` | Default task timeout |
| `COMPLEXITY_THRESHOLD` | `5` | Complexity threshold for heavy worker routing |

#### Knowledge Base
| Variable | Default | Description |
|----------|---------|-------------|
| `ARTIFACTS_DIR` | `./artifacts` | Directory for artifact-based knowledge base |
| `HYPOTHESES_DIR` | `./hypotheses` | Directory for saved hypotheses (legacy) |
| `INDEX_ON_START` | `false` | Whether to index artifacts on worker startup |

#### LLM Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_MODE` | `mock` | LLM mode: `mock` or `real` |
| `SANDBOX_MEM_MB` | `4` | Memory limit for WASM sandbox |

#### Vector Search (RAG)
| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDINGS_MODE` | `mock` | Embeddings mode: `mock` or `openai` |
| `EMBEDDINGS_MODEL` | `text-embedding-3-small` | OpenAI embedding model |
| `VECTOR_BACKEND` | `memory` | Vector database backend: `memory` or `qdrant` |

#### Qdrant Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_URL` | `localhost:6333` | Qdrant server URL |
| `QDRANT_API_KEY` | - | Qdrant API key (optional) |

#### Metrics Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_MODE` | `prom` | Metrics mode: `prom` or `otel` |
| `METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `SERVICE_NAME` | `agent` | Service name for metrics |
| `METRICS_NAMESPACE` | `agent` | Metrics namespace prefix |
| `METRICS_COLLECT_RUNTIME` | `true` | Collect Go runtime metrics |

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
curl http://localhost:9006/caps

# Check if all workers are ready
curl http://localhost:9006/ready

# Check specific worker capabilities
curl http://localhost:9004/caps  # Light worker
curl http://localhost:9002/caps  # Heavy worker
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

## Documentation

- [Architecture Documentation](docs/architecture.md) - System architecture and design
- [API Usage Guide](docs/api-guide.md) - Comprehensive API usage examples
- [OpenAPI Specifications](docs/openapi/) - API specifications for all services
  - [Router API](docs/openapi/router.yaml)
  - [Worker API](docs/openapi/worker.yaml)
  - [LLM Router API](docs/openapi/llmrouter.yaml)

## Support

For issues and questions:
- Create an issue on GitHub
- Check the troubleshooting section
- Review the logs for error details
- Consult the [API Usage Guide](docs/api-guide.md) for detailed examples