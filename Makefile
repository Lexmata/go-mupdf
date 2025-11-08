# Makefile for Go MuPDF Wrapper
# Provides convenient build targets

.PHONY: help build test clean mupdf-build docker-build docker-test docker-quick docker-coverage docker-shell docker-clean

# Default: build with source-built MuPDF
help:
	@echo "Available targets:"
	@echo ""
	@echo "Local Development:"
	@echo "  make build           - Build with source-built MuPDF"
	@echo "  make test            - Run tests with source-built MuPDF"
	@echo "  make mupdf-build     - Build MuPDF from source"
	@echo "  make clean           - Clean build artifacts"
	@echo ""
	@echo "Docker Testing (CI/CD simulation):"
	@echo "  make docker-build    - Build Docker test image"
	@echo "  make docker-test     - Run tests in Docker"
	@echo "  make docker-quick    - Run tests without rebuilding"
	@echo "  make docker-coverage - Generate coverage report in Docker"
	@echo "  make docker-shell    - Open shell in Docker container"
	@echo "  make docker-clean    - Remove Docker test image"

# Build MuPDF from source
mupdf-build:
	@echo "Building MuPDF from source..."
	@if [ ! -d "third_party/mupdf" ]; then \
		echo "Error: third_party/mupdf not found. Run: git submodule update --init --recursive"; \
		exit 1; \
	fi
	cd third_party/mupdf && make -j$$(nproc) libs

# Build with source-built MuPDF
build: mupdf-build
	@echo "Building Go wrapper with source-built MuPDF..."
	go build ./pkg/mupdf/

# Test with source-built MuPDF
test: mupdf-build
	@echo "Running tests with source-built MuPDF..."
	go test ./pkg/mupdf/ -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	go clean -cache
	rm -f coverage.out
	@if [ -d "third_party/mupdf/build" ]; then \
		cd third_party/mupdf && make clean; \
	fi

# Docker testing targets
docker-build:
	@./scripts/docker-test.sh build

docker-test:
	@./scripts/docker-test.sh test

docker-quick:
	@./scripts/docker-test.sh quick

docker-coverage:
	@./scripts/docker-test.sh coverage

docker-shell:
	@./scripts/docker-test.sh shell

docker-clean:
	@./scripts/docker-test.sh clean

