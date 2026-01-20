#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Start Docker services in detached mode
echo "Starting Docker services..."
docker compose up -d

# Function to open URL based on OS
open_url() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        open "$1"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v xdg-open &> /dev/null; then
            xdg-open "$1"
        else
            echo "Could not detect web browser to open $1"
        fi
    else
        echo "Unsupported OS for opening browser automatically: $OSTYPE"
    fi
}

echo "Waiting for services to initialize..."
sleep 5

echo "Opening Grafana..."
open_url "http://localhost:3001"

echo "Opening Jaeger..."
open_url "http://localhost:16686"

echo "Opening OpenAPI (Swagger UI)..."
open_url "http://localhost:3000/swagger/"

# Attach to logs
echo "Attaching to logs..."
docker compose logs -f
