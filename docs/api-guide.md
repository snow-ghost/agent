# API Usage Guide

This guide provides comprehensive examples and best practices for using the Agent API.

## Table of Contents

- [Authentication](#authentication)
- [Router API](#router-api)
- [Worker API](#worker-api)
- [LLM Router API](#llm-router-api)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Authentication

The Agent API supports two authentication methods:

### API Key Authentication

```bash
curl -H "X-API-Key: your-api-key" \
     -H "Content-Type: application/json" \
     http://localhost:8080/solve
```

### Bearer Token Authentication

```bash
curl -H "Authorization: Bearer your-jwt-token" \
     -H "Content-Type: application/json" \
     http://localhost:8080/solve
```

### Getting API Keys

```bash
# Create a new API key
./agent-cli auth create-key --name "my-app" --rate-limit 1000

# List existing API keys
./agent-cli auth list-keys

# Revoke an API key
./agent-cli auth revoke-key --key "ak_..."
```

## Router API

The Router API is the main entry point for task processing.

### Base URL

- Development: `http://localhost:8080`
- Production: `https://api.agent.dev`

### Solve Task

**Endpoint**: `POST /solve`

**Description**: Routes and executes a task using the appropriate worker.

**Request Body**:

```json
{
  "id": "task-123",
  "domain": "algorithms",
  "spec": {
    "success_criteria": ["sorted_non_decreasing"],
    "props": {
      "type": "sort"
    },
    "metrics_weights": {
      "cases_passed": 1.0,
      "cases_total": 0.0
    }
  },
  "input": {
    "numbers": [3, 1, 2]
  },
  "flags": {
    "requires_sandbox": false,
    "max_complexity": 3
  },
  "budget": {
    "timeout": "30s",
    "max_cost": 0.01
  }
}
```

**Response**:

```json
{
  "success": true,
  "output": {
    "sorted_numbers": [1, 2, 3]
  },
  "metrics": {
    "cases_passed": 1,
    "cases_total": 1
  },
  "cost": 0.001,
  "duration": "150ms"
}
```

**Example with cURL**:

```bash
curl -X POST http://localhost:8080/solve \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "id": "sort-task-1",
    "domain": "algorithms",
    "spec": {
      "success_criteria": ["sorted_non_decreasing"],
      "props": {"type": "sort"},
      "metrics_weights": {"cases_passed": 1.0}
    },
    "input": {"numbers": [3, 1, 2]},
    "flags": {"requires_sandbox": false, "max_complexity": 3},
    "budget": {"timeout": "30s"}
  }'
```

### Health Check

**Endpoint**: `GET /health`

**Description**: Returns the health status of the router and its dependencies.

**Response**:

```json
{
  "status": "ok",
  "service": "agent-router",
  "light_worker": "http://localhost:8081",
  "heavy_worker": "http://localhost:8082",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Capabilities

**Endpoint**: `GET /caps`

**Description**: Returns the capabilities of all workers.

**Response**:

```json
{
  "light_worker": {
    "url": "http://localhost:8081",
    "capabilities": {
      "use_kb": true,
      "use_wasm": false,
      "use_llm": false
    }
  },
  "heavy_worker": {
    "url": "http://localhost:8082",
    "capabilities": {
      "use_kb": true,
      "use_wasm": true,
      "use_llm": true
    }
  },
  "routing_rules": {
    "requires_sandbox": "heavy",
    "max_complexity_threshold": 5,
    "high_complexity": "heavy",
    "default": "light"
  }
}
```

## Worker API

Workers handle the actual task execution.

### Base URLs

- Light Worker: `http://localhost:8081`
- Heavy Worker: `http://localhost:8082`

### Solve Task

**Endpoint**: `POST /solve`

**Description**: Executes a task using the worker's capabilities.

**Request/Response**: Same as Router API

**Example with cURL**:

```bash
# Direct call to light worker
curl -X POST http://localhost:8081/solve \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "id": "simple-task",
    "domain": "data_processing",
    "spec": {
      "success_criteria": ["completed"],
      "metrics_weights": {"completed": 1.0}
    },
    "input": {"data": "hello world"},
    "flags": {"requires_sandbox": false, "max_complexity": 2},
    "budget": {"timeout": "10s"}
  }'
```

### Get Worker Capabilities

**Endpoint**: `GET /caps`

**Description**: Returns the specific capabilities of this worker.

**Response**:

```json
{
  "worker_type": "light",
  "capabilities": {
    "use_kb": true,
    "use_wasm": false,
    "use_llm": false
  },
  "capabilities_string": "KB"
}
```

## LLM Router API

The LLM Router provides access to various LLM providers.

### Base URL

- Development: `http://localhost:8090`
- Production: `https://llm.agent.dev`

### Chat Completions

**Endpoint**: `POST /v1/chat/completions`

**Description**: Creates a chat completion using the best available LLM provider.

**Request Body**:

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Explain how to sort an array in Python."
    }
  ],
  "temperature": 0.7,
  "max_tokens": 150,
  "stream": false,
  "metadata": {
    "task_domain": "programming",
    "priority": "high"
  }
}
```

**Response**:

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-3.5-turbo",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "To sort an array in Python, you can use the built-in sorted() function..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 50,
    "total_tokens": 75
  },
  "cost": 0.001
}
```

**Example with cURL**:

```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "temperature": 0.7,
    "max_tokens": 100
  }'
```

### Streaming Chat Completions

**Endpoint**: `POST /v1/chat/completions`

**Description**: Creates a streaming chat completion.

**Request Body**:

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "Write a short story about a robot."
    }
  ],
  "stream": true,
  "temperature": 0.8
}
```

**Response** (Server-Sent Events):

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Once"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":" upon"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":" a time"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":""},"finish_reason":"stop"}]}
```

### Embeddings

**Endpoint**: `POST /v1/embeddings`

**Description**: Creates embeddings for the given input text.

**Request Body**:

```json
{
  "model": "text-embedding-ada-002",
  "input": "The quick brown fox jumps over the lazy dog"
}
```

**Response**:

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.1, 0.2, 0.3, ...]
    }
  ],
  "model": "text-embedding-ada-002",
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 0,
    "total_tokens": 9
  },
  "cost": 0.0001
}
```

**Example with cURL**:

```bash
curl -X POST http://localhost:8090/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "Hello world"
  }'
```

### List Models

**Endpoint**: `GET /v1/models`

**Description**: Lists all available models across providers.

**Response**:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-3.5-turbo",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai",
      "capabilities": {
        "chat_completion": true,
        "embeddings": false
      },
      "cost_per_token": {
        "prompt": 0.0015,
        "completion": 0.002
      }
    }
  ]
}
```

## Error Handling

### Error Response Format

All errors follow a consistent format:

```json
{
  "error": "InvalidInput",
  "message": "Invalid task data provided",
  "code": 400,
  "type": "InvalidInput"
}
```

### Common Error Types

| Error Type | HTTP Code | Description |
|------------|-----------|-------------|
| `InvalidInput` | 400 | Invalid request data |
| `Unauthorized` | 401 | Invalid API key or token |
| `RateLimit` | 429 | Rate limit exceeded |
| `ServiceUnavailable` | 503 | Service temporarily unavailable |
| `Internal` | 500 | Internal server error |

### Error Handling Example

```python
import requests
import json

def solve_task(task_data, api_key):
    url = "http://localhost:8080/solve"
    headers = {
        "Content-Type": "application/json",
        "X-API-Key": api_key
    }
    
    try:
        response = requests.post(url, json=task_data, headers=headers)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.HTTPError as e:
        if e.response.status_code == 401:
            print("Authentication failed. Check your API key.")
        elif e.response.status_code == 429:
            print("Rate limit exceeded. Please wait and try again.")
        else:
            error_data = e.response.json()
            print(f"Error {error_data['code']}: {error_data['message']}")
        raise
    except requests.exceptions.RequestException as e:
        print(f"Request failed: {e}")
        raise
```

## Best Practices

### 1. Task Design

- **Use appropriate complexity levels**: Simple tasks should use light workers
- **Set realistic timeouts**: Don't set timeouts too low or too high
- **Include proper success criteria**: Define clear success metrics
- **Use meaningful task IDs**: Include context in task identifiers

### 2. Error Handling

- **Always check response status**: Don't assume requests will succeed
- **Implement retries**: Use exponential backoff for transient failures
- **Handle rate limits**: Respect rate limits and implement backoff
- **Log errors appropriately**: Include context for debugging

### 3. Performance

- **Use appropriate workers**: Route tasks to the right worker type
- **Batch requests when possible**: Combine multiple operations
- **Cache results**: Cache frequently used data
- **Monitor costs**: Track LLM usage and costs

### 4. Security

- **Protect API keys**: Never expose API keys in client code
- **Use HTTPS in production**: Always use encrypted connections
- **Validate inputs**: Sanitize all user inputs
- **Implement proper authentication**: Use strong API keys

## Examples

### Python Client

```python
import requests
import json
from typing import Dict, Any

class AgentClient:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            "Content-Type": "application/json",
            "X-API-Key": api_key
        })
    
    def solve_task(self, task: Dict[str, Any]) -> Dict[str, Any]:
        """Solve a task using the router."""
        response = self.session.post(
            f"{self.base_url}/solve",
            json=task
        )
        response.raise_for_status()
        return response.json()
    
    def create_chat_completion(self, messages: list, model: str = "gpt-3.5-turbo") -> Dict[str, Any]:
        """Create a chat completion using the LLM router."""
        response = self.session.post(
            f"{self.base_url}/v1/chat/completions",
            json={
                "model": model,
                "messages": messages,
                "temperature": 0.7
            }
        )
        response.raise_for_status()
        return response.json()

# Usage
client = AgentClient("http://localhost:8080", "your-api-key")

# Solve a sorting task
task = {
    "id": "sort-1",
    "domain": "algorithms",
    "spec": {
        "success_criteria": ["sorted_non_decreasing"],
        "props": {"type": "sort"},
        "metrics_weights": {"cases_passed": 1.0}
    },
    "input": {"numbers": [3, 1, 2]},
    "flags": {"requires_sandbox": False, "max_complexity": 3},
    "budget": {"timeout": "30s"}
}

result = client.solve_task(task)
print(f"Task result: {result}")

# Create a chat completion
messages = [
    {"role": "user", "content": "Explain sorting algorithms"}
]
completion = client.create_chat_completion(messages)
print(f"Chat completion: {completion['choices'][0]['message']['content']}")
```

### JavaScript Client

```javascript
class AgentClient {
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
    }
    
    async solveTask(task) {
        const response = await fetch(`${this.baseUrl}/solve`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': this.apiKey
            },
            body: JSON.stringify(task)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        return await response.json();
    }
    
    async createChatCompletion(messages, model = 'gpt-3.5-turbo') {
        const response = await fetch(`${this.baseUrl}/v1/chat/completions`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': this.apiKey
            },
            body: JSON.stringify({
                model,
                messages,
                temperature: 0.7
            })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        return await response.json();
    }
}

// Usage
const client = new AgentClient('http://localhost:8080', 'your-api-key');

// Solve a task
const task = {
    id: 'sort-1',
    domain: 'algorithms',
    spec: {
        success_criteria: ['sorted_non_decreasing'],
        props: { type: 'sort' },
        metrics_weights: { cases_passed: 1.0 }
    },
    input: { numbers: [3, 1, 2] },
    flags: { requires_sandbox: false, max_complexity: 3 },
    budget: { timeout: '30s' }
};

client.solveTask(task)
    .then(result => console.log('Task result:', result))
    .catch(error => console.error('Error:', error));

// Create a chat completion
const messages = [
    { role: 'user', content: 'Explain sorting algorithms' }
];

client.createChatCompletion(messages)
    .then(completion => console.log('Chat completion:', completion.choices[0].message.content))
    .catch(error => console.error('Error:', error));
```

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type AgentClient struct {
    BaseURL string
    APIKey  string
    client  *http.Client
}

func NewAgentClient(baseURL, apiKey string) *AgentClient {
    return &AgentClient{
        BaseURL: baseURL,
        APIKey:  apiKey,
        client:  &http.Client{},
    }
}

func (c *AgentClient) SolveTask(task interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(task)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", c.BaseURL+"/solve", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", c.APIKey)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    return result, err
}

// Usage
func main() {
    client := NewAgentClient("http://localhost:8080", "your-api-key")
    
    task := map[string]interface{}{
        "id":     "sort-1",
        "domain": "algorithms",
        "spec": map[string]interface{}{
            "success_criteria": []string{"sorted_non_decreasing"},
            "props":           map[string]string{"type": "sort"},
            "metrics_weights": map[string]float64{"cases_passed": 1.0},
        },
        "input": map[string]interface{}{
            "numbers": []int{3, 1, 2},
        },
        "flags": map[string]interface{}{
            "requires_sandbox": false,
            "max_complexity":   3,
        },
        "budget": map[string]interface{}{
            "timeout": "30s",
        },
    }
    
    result, err := client.SolveTask(task)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Task result: %+v\n", result)
}
```

## Monitoring and Debugging

### Health Checks

```bash
# Check router health
curl http://localhost:8080/health

# Check worker health
curl http://localhost:8081/health
curl http://localhost:8082/health

# Check LLM router health
curl http://localhost:8090/health
```

### Metrics

```bash
# Get Prometheus metrics
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics
curl http://localhost:8082/metrics
curl http://localhost:8090/metrics
```

### Logs

```bash
# View logs with Docker Compose
docker-compose logs -f router
docker-compose logs -f light-worker
docker-compose logs -f heavy-worker
docker-compose logs -f llmrouter
```

### Tracing

Access Jaeger UI at `http://localhost:16686` to view distributed traces and debug request flows.
