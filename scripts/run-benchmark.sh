#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  DICT Benchmark Runner${NC}"
echo -e "${BLUE}========================================${NC}"

# Check for required commands
for cmd in docker k6; do
    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}Error: $cmd is required but not installed.${NC}"
        exit 1
    fi
done

# Function to wait for service health
wait_for_health() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}Waiting for $name to be healthy...${NC}"
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url/health" > /dev/null 2>&1; then
            echo -e "${GREEN}$name is healthy!${NC}"
            return 0
        fi
        sleep 1
        attempt=$((attempt + 1))
    done
    echo -e "${RED}$name failed to become healthy${NC}"
    return 1
}

# Function to run benchmark
run_benchmark() {
    local app_name=$1
    local base_url=$2
    
    echo -e "\n${BLUE}----------------------------------------${NC}"
    echo -e "${BLUE}  Running benchmark for: ${app_name}${NC}"
    echo -e "${BLUE}----------------------------------------${NC}"
    
    k6 run \
        --out experimental-prometheus-rw=http://localhost:9090/api/v1/write \
        -e BASE_URL="$base_url" \
        -e APP="$app_name" \
        "$PROJECT_DIR/k6/benchmark.test.js"
}

# Parse arguments
RUN_BUN=false
RUN_GO=false
CLEANUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --bun)
            RUN_BUN=true
            shift
            ;;
        --go)
            RUN_GO=true
            shift
            ;;
        --all)
            RUN_BUN=true
            RUN_GO=true
            shift
            ;;
        --cleanup)
            CLEANUP=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --bun      Run benchmark for Bun/Elysia app"
            echo "  --go       Run benchmark for Go app"
            echo "  --all      Run benchmark for both apps"
            echo "  --cleanup  Stop all services after benchmark"
            echo "  --help     Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Default to running all if no specific app selected
if [ "$RUN_BUN" = false ] && [ "$RUN_GO" = false ]; then
    RUN_BUN=true
    RUN_GO=true
fi

# Function to kill process on port
kill_port() {
    local port=$1
    echo -e "${YELLOW}Ensuring port $port is free...${NC}"
    lsof -ti :$port | xargs kill -9 2>/dev/null || true
    sleep 2
}

# Step 1: Start monitoring stack
echo -e "\n${YELLOW}Step 1: Starting monitoring stack...${NC}"
docker compose -f "$PROJECT_DIR/monitoring/docker-compose.yml" up -d

echo -e "${YELLOW}Waiting for Prometheus to start...${NC}"
sleep 5

# Step 2: Run Bun benchmark
if [ "$RUN_BUN" = true ]; then
    echo -e "\n${YELLOW}Step 2: Starting Bun/Elysia app...${NC}"
    kill_port 3000
    
    # Start infrastructure
    docker compose -f "$PROJECT_DIR/bun/docker-compose.yml" up -d mongo redis jaeger
    sleep 5
    
    # Start app with rate limiting disabled
    cd "$PROJECT_DIR/bun"
    RATE_LIMIT_ENABLED=false JWT_SECRET=benchmark-secret bun run src/index.ts &
    BUN_PID=$!
    cd "$PROJECT_DIR"
    
    # Wait for health
    wait_for_health "http://localhost:3000" "Bun/Elysia"
    
    # Run benchmark
    run_benchmark "bun" "http://localhost:3000"
    
    # Stop Bun app
    echo -e "${YELLOW}Stopping Bun/Elysia app...${NC}"
    kill $BUN_PID 2>/dev/null || true
    kill_port 3000
    docker compose -f "$PROJECT_DIR/bun/docker-compose.yml" down
    
    echo -e "\n${YELLOW}Cooldown period (10s)...${NC}"
    sleep 10
fi

# Step 3: Run Go benchmark
if [ "$RUN_GO" = true ]; then
    echo -e "\n${YELLOW}Step 3: Starting Go app...${NC}"
    kill_port 3000
    
    # Start infrastructure
    docker compose -f "$PROJECT_DIR/go/docker-compose.yml" up -d mongo redis jaeger
    sleep 5
    
    # Start app with rate limiting disabled
    cd "$PROJECT_DIR/go"
    RATE_LIMIT_ENABLED=false JWT_SECRET=benchmark-secret go run ./cmd/server &
    GO_PID=$!
    cd "$PROJECT_DIR"
    
    # Wait for health
    wait_for_health "http://localhost:3000" "Go"
    
    # Run benchmark
    run_benchmark "go" "http://localhost:3000"
    
    # Stop Go app
    echo -e "${YELLOW}Stopping Go app...${NC}"
    kill $GO_PID 2>/dev/null || true
    kill_port 3000
    docker compose -f "$PROJECT_DIR/go/docker-compose.yml" down
fi

# Step 4: Show results
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  Benchmark Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e ""
echo -e "View results in Grafana:"
echo -e "  ${BLUE}http://localhost:3001${NC}"
echo -e "  Login: admin / admin"
echo -e ""
echo -e "Prometheus metrics:"
echo -e "  ${BLUE}http://localhost:9090${NC}"

# Cleanup if requested
if [ "$CLEANUP" = true ]; then
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    docker compose -f "$PROJECT_DIR/monitoring/docker-compose.yml" down
    echo -e "${GREEN}All services stopped.${NC}"
fi

echo -e "\n${YELLOW}Tip: Run with --cleanup to stop all services when done.${NC}"
