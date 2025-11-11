#!/bin/bash
# Download pre-built MuPDF libraries to project third_party directory
# This script is called by install.sh or manually by users

set -euo pipefail

# Detect platform
GOOS=$(go env GOOS 2>/dev/null || uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=$(go env GOARCH 2>/dev/null || uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PLATFORM="${GOOS}-${GOARCH}"

# Project directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
THIRD_PARTY="${PROJECT_ROOT}/third_party/mupdf"
BUILD_DIR="${THIRD_PARTY}/build/release"
INCLUDE_DIR="${THIRD_PARTY}/include"

# Check if already set up
if [ -f "${BUILD_DIR}/libmupdf.a" ] && [ -f "${BUILD_DIR}/libmupdf-third.a" ] && [ -d "${INCLUDE_DIR}/mupdf" ]; then
    echo "✓ Pre-built libraries already installed at ${THIRD_PARTY}"
    exit 0
fi

# Get version from VERSION file or git tag
if [ -f "${PROJECT_ROOT}/VERSION" ]; then
    VERSION=$(cat "${PROJECT_ROOT}/VERSION")
else
    VERSION=$(git -C "${PROJECT_ROOT}" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "1.3.2")
fi

DOWNLOAD_URL="https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-${VERSION}-${PLATFORM}.tar.gz"

echo "Downloading pre-built MuPDF libraries for ${PLATFORM}..."
echo "URL: ${DOWNLOAD_URL}"

# Create temporary directory for download
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

cd "${TEMP_DIR}"

# Download with curl or wget
if command -v curl >/dev/null 2>&1; then
    if ! curl -f -L -o "mupdf-libs.tar.gz" "${DOWNLOAD_URL}"; then
        echo "⚠ Download failed. Pre-built libraries not available for ${PLATFORM}."
        echo "   Falling back to source build..."
        exit 1
    fi
elif command -v wget >/dev/null 2>&1; then
    if ! wget -O "mupdf-libs.tar.gz" "${DOWNLOAD_URL}"; then
        echo "⚠ Download failed. Pre-built libraries not available for ${PLATFORM}."
        echo "   Falling back to source build..."
        exit 1
    fi
else
    echo "Error: Neither curl nor wget found. Please install one of them."
    exit 1
fi

# Extract
echo "Extracting libraries..."
tar -xzf "mupdf-libs.tar.gz"

# Get the extracted directory name (should be go-mupdf-VERSION-PLATFORM)
EXTRACTED_DIR=$(ls -d go-mupdf-* 2>/dev/null | head -1)
if [ -z "$EXTRACTED_DIR" ]; then
    echo "Error: Could not find extracted directory"
    exit 1
fi

# Create target directories
mkdir -p "${BUILD_DIR}"
mkdir -p "${INCLUDE_DIR}"

# Copy libraries and headers
echo "Installing to ${THIRD_PARTY}..."
cp "${EXTRACTED_DIR}/lib/"*.a "${BUILD_DIR}/"
cp -r "${EXTRACTED_DIR}/include/mupdf" "${INCLUDE_DIR}/"

# Verify
if [ -f "${BUILD_DIR}/libmupdf.a" ] && [ -f "${BUILD_DIR}/libmupdf-third.a" ]; then
    echo "✓ Successfully downloaded and installed MuPDF libraries"
    echo "  Location: ${THIRD_PARTY}"
else
    echo "Error: Installation failed - missing required libraries"
    exit 1
fi

