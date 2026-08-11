#!/bin/bash
# Download pre-built MuPDF libraries to project third_party directory
# This script is called by install.sh or manually by users

set -euo pipefail

# Detect platform
# Respects GOOS/GOARCH environment variables for cross-compilation
# Falls back to uname for native builds
detect_target_os() {
    # If GOOS is set (cross-compilation), use it
    if [ -n "${GOOS:-}" ]; then
        echo "$GOOS"
        return
    fi
    
    # Otherwise detect from uname
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux*)   echo "linux" ;;
        darwin*)  echo "darwin" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *)        echo "$os" ;;
    esac
}

detect_target_arch() {
    # If GOARCH is set (cross-compilation), use it
    if [ -n "${GOARCH:-}" ]; then
        case "$GOARCH" in
            amd64)   echo "amd64" ;;
            arm64)   echo "arm64" ;;
            arm)     echo "arm" ;;
            386)     echo "386" ;;
            *)       echo "$GOARCH" ;;
        esac
        return
    fi
    
    # Otherwise detect from uname
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)     echo "amd64" ;;
        aarch64|arm64)    echo "arm64" ;;
        armv7l)           echo "arm" ;;
        i386|i686)        echo "386" ;;
        *)                echo "$arch" ;;
    esac
}

TARGET_OS=$(detect_target_os)
TARGET_ARCH=$(detect_target_arch)
PLATFORM="${TARGET_OS}-${TARGET_ARCH}"

# Project directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
THIRD_PARTY="${PROJECT_ROOT}/third_party/mupdf"
BUILD_DIR="${THIRD_PARTY}/build/release"
INCLUDE_DIR="${THIRD_PARTY}/include"

# Check if already set up (and built for the right target platform)
PLATFORM_MARKER="${BUILD_DIR}/.platform"
if [ -f "${BUILD_DIR}/libmupdf.a" ] && [ -f "${BUILD_DIR}/libmupdf-third.a" ] && [ -d "${INCLUDE_DIR}/mupdf" ]; then
    if [ -f "${PLATFORM_MARKER}" ]; then
        EXISTING_PLATFORM=$(cat "${PLATFORM_MARKER}")
        if [ "${EXISTING_PLATFORM}" = "${PLATFORM}" ]; then
            echo "✓ MuPDF libraries already present at ${THIRD_PARTY}"
            exit 0
        fi
        echo "⚠ Existing libraries are for ${EXISTING_PLATFORM}, need ${PLATFORM} — reinstalling"
        rm -f "${BUILD_DIR}/libmupdf.a" "${BUILD_DIR}/libmupdf-third.a" "${PLATFORM_MARKER}"
    else
        # Legacy install without a platform marker: we cannot trust the arch of
        # what's on disk. The old check sniffed a single archive member and
        # assumed a match when readelf couldn't read it — exactly the blind spot
        # that let a mixed-arch archive install cleanly. Rather than re-derive a
        # trustworthy arch from an untagged install, treat it as stale and
        # reinstall; a fresh download always writes the marker below.
        echo "⚠ Existing libraries have no platform marker; reinstalling to guarantee ${PLATFORM}"
        rm -f "${BUILD_DIR}/libmupdf.a" "${BUILD_DIR}/libmupdf-third.a"
    fi
fi

# Get version from VERSION file or git tag
if [ -f "${PROJECT_ROOT}/VERSION" ]; then
    VERSION=$(cat "${PROJECT_ROOT}/VERSION")
else
    VERSION=$(git -C "${PROJECT_ROOT}" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "1.3.2")
fi

# Artifacts are published as GitHub Release assets on the Lexmata mirror, which
# is public — so an asset downloads from its stable release-download URL with a
# plain unauthenticated GET, no token and no API/JSON lookup required. Override
# the repo with GO_MUPDF_GH_REPO if the mirror ever moves.
GH_REPO="${GO_MUPDF_GH_REPO:-Lexmata/go-mupdf}"
ASSET="go-mupdf-${VERSION}-${PLATFORM}.tar.gz"
TAG="v${VERSION}"
DOWNLOAD_URL="https://github.com/${GH_REPO}/releases/download/${TAG}/${ASSET}"

echo "========================================"
echo "Downloading pre-built MuPDF libraries"
echo "========================================"
echo "Target OS:           ${TARGET_OS}"
echo "Target Architecture: ${TARGET_ARCH}"
echo "Platform:            ${PLATFORM}"
echo "Version:             ${VERSION}"
echo "URL:                 ${DOWNLOAD_URL}"
echo "========================================"

# Create temporary directory for download
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT

cd "${TEMP_DIR}"

download_failed() {
    echo "⚠ Download failed: $1"
    echo "   Pre-built libraries not available for ${PLATFORM} at ${GH_REPO}@${TAG}."
    echo "   Falling back to source build..."
    exit 1
}

fetch() {  # $1 = url, $2 = output path
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        echo "Error: neither curl nor wget found. Please install one of them." >&2
        exit 1
    fi
}

fetch "${DOWNLOAD_URL}" "mupdf-libs.tar.gz" \
    || download_failed "could not download ${ASSET}"
fetch "${DOWNLOAD_URL}.sha256" "mupdf-libs.tar.gz.sha256" \
    || download_failed "could not download ${ASSET}.sha256"

# Verify the download against its published checksum before trusting it. Compare
# the hash fields directly rather than `sha256sum -c`, since the .sha256 names
# the original tarball (go-mupdf-<ver>-<platform>.tar.gz), not our local name.
EXPECTED_SHA=$(awk '{print $1}' "mupdf-libs.tar.gz.sha256" 2>/dev/null)
ACTUAL_SHA=$(sha256sum "mupdf-libs.tar.gz" | awk '{print $1}')
if [ -z "${EXPECTED_SHA}" ] || [ "${EXPECTED_SHA}" != "${ACTUAL_SHA}" ]; then
    download_failed "checksum mismatch (expected ${EXPECTED_SHA:-none}, got ${ACTUAL_SHA})"
fi
echo "✓ Checksum verified: ${ACTUAL_SHA}"

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
    # Record which target platform these libraries were installed for
    echo "${PLATFORM}" > "${PLATFORM_MARKER}"
    echo "✓ Successfully downloaded and installed MuPDF libraries"
    echo "  Location: ${THIRD_PARTY}"
else
    echo "Error: Installation failed - missing required libraries"
    exit 1
fi

