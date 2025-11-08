#!/bin/bash
# Build script for using system-installed MuPDF libraries
# This is useful for CI/CD environments that install libmupdf-dev

set -e

echo "=== Building with System MuPDF Libraries ==="

# Check if system MuPDF is available
if ! pkg-config --exists mupdf 2>/dev/null && [ ! -f /usr/include/mupdf/fitz.h ]; then
    echo "Warning: System MuPDF not found. Installing..."
    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y libmupdf-dev
    else
        echo "Error: Cannot install MuPDF. Please install libmupdf-dev manually."
        exit 1
    fi
fi

# Verify installation
if [ -f /usr/include/mupdf/fitz.h ]; then
    echo "✅ System MuPDF headers found"
else
    echo "❌ System MuPDF headers not found at /usr/include/mupdf/fitz.h"
    exit 1
fi

# Build with system_mupdf tag
echo "Building with system_mupdf build tag..."
go build -tags system_mupdf ./pkg/mupdf/

echo "✅ Build complete with system MuPDF libraries"

