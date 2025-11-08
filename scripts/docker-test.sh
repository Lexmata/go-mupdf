#!/bin/bash
# Docker testing script for Go MuPDF Wrapper
# This script builds and runs tests in a Docker container

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Go MuPDF Wrapper - Docker Testing ===${NC}"

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Parse command line arguments
BUILD_ONLY=false
RUN_SHELL=false
TEST_ARGS=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --shell)
            RUN_SHELL=true
            shift
            ;;
        --test-args)
            TEST_ARGS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--build-only] [--shell] [--test-args 'ARGS']"
            exit 1
            ;;
    esac
done

# Build Docker image
echo -e "${YELLOW}Building Docker image...${NC}"
docker build -t go-mupdf-test:latest .

if [ $? -ne 0 ]; then
    echo -e "${RED}Docker build failed!${NC}"
    exit 1
fi

echo -e "${GREEN}Docker image built successfully!${NC}"

if [ "$BUILD_ONLY" = true ]; then
    echo -e "${GREEN}Build complete. Exiting.${NC}"
    exit 0
fi

# Run container
if [ "$RUN_SHELL" = true ]; then
    echo -e "${YELLOW}Starting interactive shell...${NC}"
    docker run -it --rm \
        -v "$PROJECT_ROOT:/workspace" \
        -w /workspace \
        go-mupdf-test:latest \
        /bin/bash
else
    echo -e "${YELLOW}Running tests...${NC}"
    if [ -n "$TEST_ARGS" ]; then
        docker run --rm \
            -v "$PROJECT_ROOT:/workspace" \
            -w /workspace \
            go-mupdf-test:latest \
            go test ./pkg/mupdf/ -v $TEST_ARGS
    else
        docker run --rm \
            -v "$PROJECT_ROOT:/workspace" \
            -w /workspace \
            go-mupdf-test:latest \
            go test ./pkg/mupdf/ -v
    fi
fi

echo -e "${GREEN}=== Docker Testing Complete ===${NC}"

