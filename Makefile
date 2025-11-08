# Makefile for Go MuPDF Wrapper
# Provides convenient build targets

.PHONY: help build test clean mupdf-build

# Default: build with source-built MuPDF
help:
	@echo "Available targets:"
	@echo "  make build          - Build with source-built MuPDF"
	@echo "  make test           - Run tests with source-built MuPDF"
	@echo "  make mupdf-build     - Build MuPDF from source"
	@echo "  make clean          - Clean build artifacts"

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
	@if [ -d "third_party/mupdf/build" ]; then \
		cd third_party/mupdf && make clean; \
	fi

