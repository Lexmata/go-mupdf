#!/bin/bash
# Script to run Bitbucket Pipelines locally using Docker
# This simulates the CI environment for debugging
#
# Reproducible pipeline steps: lint, test (both build MuPDF from
# third_party/mupdf via scripts/setup-mupdf.sh, matching CI).
# NOT reproducible here: the build-mupdf artifact steps and the dist/release
# packaging steps, which depend on Bitbucket artifact/download infrastructure.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values (IMAGE matches bitbucket-pipelines.yml)
IMAGE="golang:1.24"
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
            echo "  --image IMAGE  Docker image to use (default: golang:1.24)"
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
            # Build MuPDF from the vendored source, exactly like CI does
            # (the project links against third_party/mupdf/build/release, not system mupdf)
            ./scripts/setup-mupdf.sh

            go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
            export PATH=\$PATH:\$(go env GOPATH)/bin
            export GOTOOLCHAIN=auto

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
            # Build MuPDF from the vendored source, exactly like CI does
            # (the project links against third_party/mupdf/build/release, not system mupdf)
            ./scripts/setup-mupdf.sh
            echo 'Checking built MuPDF libraries...'
            ls -la third_party/mupdf/build/release/*.a | head -3 || echo 'MuPDF libraries not found'
            export GO111MODULE=on
            export CGO_ENABLED=1
            export GOTOOLCHAIN=auto
            export GOFLAGS=-buildvcs=false
            echo 'Running tests with race detection...'
            go test -v -race ./pkg/mupdf/

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

