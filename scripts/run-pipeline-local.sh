#!/bin/bash
# Script to run Bitbucket Pipelines locally using Docker
# This simulates the CI environment for debugging

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
IMAGE="golang:1.23"
STEP="test"
MOUNT_DIR=$(pwd)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --step)
            STEP="$2"
            shift 2
            ;;
        --image)
            IMAGE="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [--step STEP] [--image IMAGE]"
            echo ""
            echo "Options:"
            echo "  --step STEP    Pipeline step to run (lint, test, default: test)"
            echo "  --image IMAGE  Docker image to use (default: golang:1.23)"
            echo "  --help         Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 --step lint"
            echo "  $0 --step test"
            echo "  $0 --step test --image golang:1.24"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}=== Running Bitbucket Pipeline Locally ===${NC}"
echo -e "${BLUE}Image: ${IMAGE}${NC}"
echo -e "${BLUE}Step: ${STEP}${NC}"
echo -e "${BLUE}Directory: ${MOUNT_DIR}${NC}"
echo ""

# Function to run lint step
run_lint() {
    echo -e "${GREEN}Running lint step...${NC}"
    docker run --rm \
        -v "${MOUNT_DIR}:/workspace" \
        -w /workspace \
        "${IMAGE}" \
        bash -c "
            set -e
            apt-get update -qq
            apt-get install -y -qq libmupdf-dev pkg-config libfreetype6-dev libjpeg-dev libpng-dev zlib1g-dev libjbig2dec-dev libopenjp2-7-dev libharfbuzz-dev libgumbo-dev libmujs-dev > /dev/null

            go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
            export PATH=\$PATH:\$(go env GOPATH)/bin
            export GOTOOLCHAIN=local

            echo 'Checking code formatting with gofmt...'
            UNFORMATTED=\$(gofmt -l .)
            if [ -n \"\$UNFORMATTED\" ]; then
                echo '❌ The following files are not properly formatted:'
                echo \"\$UNFORMATTED\"
                exit 1
            else
                echo '✅ All files are properly formatted'
            fi

            export CGO_ENABLED=1
            export GOFLAGS=-buildvcs=false
            echo 'Running go vet...'
            go vet ./...

            echo 'Running golangci-lint...'
            golangci-lint run ./...

            if command -v staticcheck >/dev/null 2>&1; then
                echo 'Running staticcheck...'
                staticcheck ./...
            else
                echo 'Installing staticcheck...'
                go install honnef.co/go/tools/cmd/staticcheck@latest
                staticcheck ./...
            fi
        "
}

# Function to run test step
run_test() {
    echo -e "${GREEN}Running test step...${NC}"
    docker run --rm \
        -v "${MOUNT_DIR}:/workspace" \
        -w /workspace \
        "${IMAGE}" \
        bash -c "
            set -e
            apt-get update -qq
            apt-get install -y -qq libmupdf-dev pkg-config libfreetype6-dev libjpeg-dev libpng-dev zlib1g-dev libjbig2dec-dev libopenjp2-7-dev libharfbuzz-dev libgumbo-dev libmujs-dev > /dev/null
            ldconfig
            echo 'Checking installed libraries...'
            find /usr/lib -name '*harfbuzz*' 2>/dev/null | head -5
            find /usr/lib -name '*mupdf*' 2>/dev/null | head -5
            find /usr/lib -name '*extract*' 2>/dev/null | head -5
            pkg-config --libs harfbuzz 2>&1 || echo 'pkg-config harfbuzz failed'
            pkg-config --libs mupdf 2>&1 || echo 'pkg-config mupdf failed'
            export GO111MODULE=on
            export CGO_ENABLED=1
            export GOOS=linux
            export GOARCH=amd64
            export GOTOOLCHAIN=local
            echo 'Running tests with race detection...'
            go test -v -race ./...

            echo 'Running tests with coverage...'
            go test -v -race -coverprofile=coverage.out ./pkg/mupdf/

            echo 'Generating coverage report...'
            go tool cover -html=coverage.out -o coverage.html

            echo '✅ Tests completed successfully'
        "
}

# Run the requested step
case "${STEP}" in
    lint)
        run_lint
        ;;
    test)
        run_test
        ;;
    *)
        echo -e "${RED}Unknown step: ${STEP}${NC}"
        echo "Available steps: lint, test"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}✅ Pipeline step '${STEP}' completed successfully!${NC}"

