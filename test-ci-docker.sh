#!/bin/bash
# Test script to simulate CI/CD pipeline in Docker

set -e

echo "=== CI/CD Pipeline Test ==="

# Use golang:1.24 image like CI/CD
docker run --rm -v "$(pwd):/workspace" -w /workspace golang:1.24 bash -c '
set -e

echo "=== Installing build dependencies ==="
apt-get update -qq
apt-get install -y -qq build-essential pkg-config libfreetype6-dev libjpeg-dev libpng-dev zlib1g-dev libjbig2dec-dev libopenjp2-7-dev libharfbuzz-dev >/dev/null 2>&1

echo "=== Initializing git submodules ==="
git config --global --add safe.directory /workspace
git submodule update --init --recursive || echo "Submodule init failed, trying direct clone"

echo "=== Building MuPDF from source ==="
if [ ! -d "third_party/mupdf" ]; then
    echo "MuPDF submodule not found, building will be handled by build.go"
else
    echo "Building MuPDF from source..."
    cd third_party/mupdf
    make -j$(nproc) libs
    cd ../..
    echo "Verifying MuPDF libraries..."
    ls -la third_party/mupdf/build/release/*.a || exit 1
fi

echo "=== Setting up Go environment ==="
export GO111MODULE=on
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
export GOTOOLCHAIN=auto

echo "=== Downloading Go modules ==="
go mod download

echo "=== Running go vet ==="
go vet ./pkg/... ./cmd/... || exit 1

echo "=== Running tests with race detection ==="
go test -v -race ./pkg/mupdf/ || exit 1

echo "=== Running tests with coverage ==="
go test -v -race -coverprofile=coverage.out ./pkg/mupdf/ || exit 1

echo "=== Generating coverage report ==="
go tool cover -func=coverage.out | tail -1

echo "=== All tests passed! ==="
'

