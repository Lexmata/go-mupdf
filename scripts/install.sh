#!/bin/bash
# Quick install script for go-mupdf
# Downloads pre-built libraries or falls back to source build

set -euo pipefail

echo "=== go-mupdf Quick Setup ==="
echo ""

# Detect platform
# Use Go's environment variables if available (respects GOOS/GOARCH for cross-compilation)
export GOOS=$(go env GOOS 2>/dev/null || uname -s | tr '[:upper:]' '[:lower:]')
export GOARCH=$(go env GOARCH 2>/dev/null || uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PLATFORM="${GOOS}-${GOARCH}"

echo "Platform: ${PLATFORM}"
echo ""

# Try to download pre-built libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if "${SCRIPT_DIR}/download-libs.sh"; then
    echo ""
    echo "✓ Setup complete! You can now build your Go project."
    echo ""
    echo "Example:"
    echo "  go build"
    echo ""
else
    echo ""
    echo "Pre-built libraries not available for your platform."
    echo "Building MuPDF from source (this will take 5-10 minutes)..."
    echo ""

    if [ -f "${SCRIPT_DIR}/setup-mupdf.sh" ]; then
        "${SCRIPT_DIR}/setup-mupdf.sh"
        echo ""
        echo "✓ Setup complete! MuPDF built from source."
    else
        echo "Error: setup-mupdf.sh not found. Are you in the go-mupdf project directory?"
        exit 1
    fi
fi

