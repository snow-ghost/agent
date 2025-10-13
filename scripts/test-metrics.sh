#!/bin/bash

# Test script for metrics integration tests
# This script starts the services and runs metrics tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting metrics integration tests...${NC}"

# Check if services are running
check_service() {
    local url=$1
    local name=$2
    
    echo -e "${YELLOW}Checking $name at $url...${NC}"
    if curl -s "$url" > /dev/null; then
        echo -e "${GREEN}✓ $name is running${NC}"
        return 0
    else
        echo -e "${RED}✗ $name is not running${NC}"
        return 1
    fi
}

# Wait for services to be ready
wait_for_services() {
    local max_attempts=30
    local attempt=0
    
    echo -e "${YELLOW}Waiting for services to be ready...${NC}"
    
    while [ $attempt -lt $max_attempts ]; do
        if check_service "http://localhost:9000/healthz" "LLM Router" && \
           check_service "http://localhost:9004/healthz" "Light Worker" && \
           check_service "http://localhost:9006/healthz" "Router"; then
            echo -e "${GREEN}All services are ready!${NC}"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -e "${YELLOW}Attempt $attempt/$max_attempts - waiting 2 seconds...${NC}"
        sleep 2
    done
    
    echo -e "${RED}Services failed to start within timeout${NC}"
    return 1
}

# Run metrics tests
run_metrics_tests() {
    echo -e "${YELLOW}Running metrics integration tests...${NC}"
    
    # Set environment variables for tests
    export E2E=1
    export ROUTER_URL=http://localhost:9006
    export LLMROUTER_URL=http://localhost:9000
    export WORKER_URL=http://localhost:9004
    export WORKER_METRICS_URL=http://localhost:9005
    export LLMROUTER_METRICS_URL=http://localhost:9001
    export ARTIFACTS_DIR=./artifacts
    
    # Run the metrics tests
    if go test -v ./testkit -run "TestMetrics"; then
        echo -e "${GREEN}✓ Metrics tests passed!${NC}"
        return 0
    else
        echo -e "${RED}✗ Metrics tests failed!${NC}"
        return 1
    fi
}

# Main execution
main() {
    echo -e "${YELLOW}Metrics Integration Test Runner${NC}"
    echo "=================================="
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ]; then
        echo -e "${RED}Error: Please run this script from the project root directory${NC}"
        exit 1
    fi
    
    # Wait for services to be ready
    if ! wait_for_services; then
        echo -e "${RED}Failed to start services. Please ensure they are running:${NC}"
        echo "  - LLM Router: http://localhost:9000"
        echo "  - Light Worker: http://localhost:9004" 
        echo "  - Router: http://localhost:9006"
        exit 1
    fi
    
    # Run the tests
    if run_metrics_tests; then
        echo -e "${GREEN}All metrics tests completed successfully!${NC}"
        exit 0
    else
        echo -e "${RED}Metrics tests failed!${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
