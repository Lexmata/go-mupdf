# Makefile for Go MuPDF Wrapper
# Provides convenient build targets for different configurations

.PHONY: help build build-system test test-system clean mupdf-build

# Default: build with source-built MuPDF
help:
	@echo "Available targets:"
	@echo "  make build          - Build with source-built MuPDF (default)"
	@echo "  make build-system   - Build with system MuPDF libraries"
	@echo "  make test           - Run tests with source-built MuPDF"
	@echo "  make test-system    - Run tests with system MuPDF"
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

# Build with source-built MuPDF (default)
build: mupdf-build
	@echo "Building Go wrapper with source-built MuPDF..."
	go build ./pkg/mupdf/

# Build with system MuPDF
build-system:
	@echo "Building Go wrapper with system MuPDF libraries..."
	@if [ ! -f /usr/include/mupdf/fitz.h ]; then \
		echo "Warning: System MuPDF not found. Install with: sudo apt-get install libmupdf-dev"; \
	fi
	go build -tags system_mupdf ./pkg/mupdf/

# Test with source-built MuPDF (default)
test: mupdf-build
	@echo "Running tests with source-built MuPDF..."
	go test ./pkg/mupdf/ -v

# Test with system MuPDF
test-system:
	@echo "Running tests with system MuPDF libraries..."
	@if [ ! -f /usr/include/mupdf/fitz.h ]; then \
		echo "Warning: System MuPDF not found. Install with: sudo apt-get install libmupdf-dev"; \
	fi
	go test -tags system_mupdf ./pkg/mupdf/ -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	go clean -cache
	@if [ -d "third_party/mupdf/build" ]; then \
		cd third_party/mupdf && make clean; \
	fi

