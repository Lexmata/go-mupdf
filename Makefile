# Makefile for Go MuPDF Wrapper
# Provides convenient build targets

.PHONY: help setup build test clean mupdf-build docker-build docker-test docker-quick docker-coverage docker-shell docker-clean dist dist-clean

# Default: build with source-built MuPDF
help:
	@echo "Available targets:"
	@echo ""
	@echo "Setup:"
	@echo "  make setup           - Setup MuPDF libraries (download or build)"
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
	@echo ""
	@echo "Distribution:"
	@echo "  make dist            - Build static libraries and create distribution package"
	@echo "  make dist-clean      - Clean distribution artifacts"

# Setup MuPDF libraries (download pre-built or build from source)
setup:
	@echo "Setting up MuPDF libraries..."
	@./scripts/setup-mupdf.sh

# Build MuPDF from source
mupdf-build:
	@echo "Building MuPDF from source..."
	@if [ ! -d "third_party/mupdf" ]; then \
		echo "Error: third_party/mupdf not found. Run: git submodule update --init --recursive"; \
		exit 1; \
	fi
	cd third_party/mupdf && make -j$$(nproc) libs

# Build with source-built MuPDF
build: setup
	@echo "Building Go wrapper..."
	go build ./pkg/mupdf/

# Test with source-built MuPDF
test: setup
	@echo "Running tests..."
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

# Distribution targets
dist:
	@echo "Building static library distribution package..."
	@./scripts/build-static-libs.sh

dist-clean:
	@echo "Cleaning distribution artifacts..."
	@./scripts/build-static-libs.sh --clean

