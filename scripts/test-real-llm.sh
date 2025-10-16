#!/bin/bash

# Script to run tests with real LLM
# This script starts the LLM router and runs tests

set -e

echo "🚀 Starting Real LLM Test Setup"

# Check if API keys are set
if [ -z "$OPENAI_API_KEY" ] && [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "❌ Error: No API keys set. Please set OPENAI_API_KEY or ANTHROPIC_API_KEY"
    echo "   Example: export OPENAI_API_KEY=your_key_here"
    exit 1
fi

# Load environment variables
if [ -f .env ]; then
    echo "📋 Loading environment from .env file"
    export $(cat .env | grep -v '^#' | xargs)
fi

# Set default values
export LLMROUTER_PORT=${LLMROUTER_PORT:-9000}
export WORKER_PORT=${WORKER_PORT:-9002}
export LLM_ROUTER_URL=${LLM_ROUTER_URL:-http://localhost:9000}

echo "🔧 Configuration:"
echo "   LLM Router URL: $LLM_ROUTER_URL"
echo "   Worker Port: $WORKER_PORT"
echo "   API Keys: $(if [ -n "$OPENAI_API_KEY" ]; then echo "OpenAI ✓"; fi)$(if [ -n "$ANTHROPIC_API_KEY" ]; then echo " Anthropic ✓"; fi)"

# Build the project
echo "🔨 Building project..."
go build ./...

# Start LLM Router in background
echo "🌐 Starting LLM Router on port $LLMROUTER_PORT..."
go run ./cmd/llmrouter &
LLMROUTER_PID=$!

# Wait for LLM router to start
echo "⏳ Waiting for LLM Router to start..."
sleep 5

# Check if LLM router is running
if ! curl -s http://localhost:$LLMROUTER_PORT/healthz > /dev/null; then
    echo "❌ LLM Router failed to start"
    kill $LLMROUTER_PID 2>/dev/null || true
    exit 1
fi

echo "✅ LLM Router is running"

# Run the real LLM tests
echo "🧪 Running real LLM tests..."
go test -v ./worker/heavy -run "TestHeavySolve_WithRealLLM"

# Cleanup
echo "🧹 Cleaning up..."
kill $LLMROUTER_PID 2>/dev/null || true

echo "✅ Real LLM tests completed"

