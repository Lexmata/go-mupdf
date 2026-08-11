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

# Artifacts are published as GitHub Release assets on the Lexmata mirror. The
# repo is private, so asset downloads require a token: either `gh` already
# authenticated, or GITHUB_TOKEN/GH_TOKEN in the environment (CI injects this).
GH_REPO="${GO_MUPDF_GH_REPO:-Lexmata/go-mupdf}"
ASSET="go-mupdf-${VERSION}-${PLATFORM}.tar.gz"
TAG="v${VERSION}"
GH_TOKEN_VALUE="${GITHUB_TOKEN:-${GH_TOKEN:-}}"

echo "========================================"
echo "Downloading pre-built MuPDF libraries"
echo "========================================"
echo "Target OS:           ${TARGET_OS}"
echo "Target Architecture: ${TARGET_ARCH}"
echo "Platform:            ${PLATFORM}"
echo "Version:             ${VERSION}"
echo "Repo:                ${GH_REPO}"
echo "Asset:               ${ASSET} (release ${TAG})"
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

if command -v gh >/dev/null 2>&1; then
    # gh handles private-repo auth and the asset-id lookup in one step.
    GH_TOKEN="${GH_TOKEN_VALUE}" gh release download "${TAG}" \
        --repo "${GH_REPO}" --pattern "${ASSET}" --output "mupdf-libs.tar.gz" \
        || download_failed "gh release download failed"
elif command -v curl >/dev/null 2>&1; then
    [ -n "${GH_TOKEN_VALUE}" ] || download_failed "no GITHUB_TOKEN/GH_TOKEN set and gh unavailable; cannot read private release assets"
    # Resolve the asset's API URL from the release metadata, then fetch it with
    # Accept: application/octet-stream (the documented way to download a
    # private-repo asset — the browser_download_url 404s without a session).
    # GitHub returns pretty-printed JSON: within each asset object the "url"
    # field precedes "name", so track the last-seen url and emit it when the
    # matching name line appears. No jq dependency (build image lacks it).
    ASSET_URL=$(curl -fsSL \
        -H "Authorization: Bearer ${GH_TOKEN_VALUE}" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/${GH_REPO}/releases/tags/${TAG}" \
        | while IFS= read -r line; do
              case "$line" in
                  *'"url":'*'/releases/assets/'*)
                      last_url=$(printf '%s' "$line" | sed -n 's/.*"url": *"\([^"]*\)".*/\1/p') ;;
                  *'"name":'*)
                      name=$(printf '%s' "$line" | sed -n 's/.*"name": *"\([^"]*\)".*/\1/p')
                      [ "$name" = "${ASSET}" ] && { printf '%s\n' "$last_url"; break; } ;;
              esac
          done)
    [ -n "${ASSET_URL}" ] || download_failed "asset ${ASSET} not found in release ${TAG}"
    curl -fL \
        -H "Authorization: Bearer ${GH_TOKEN_VALUE}" \
        -H "Accept: application/octet-stream" \
        -o "mupdf-libs.tar.gz" "${ASSET_URL}" \
        || download_failed "asset download failed"
else
    echo "Error: neither gh nor curl found. Please install one of them."
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
    # Record which target platform these libraries were installed for
    echo "${PLATFORM}" > "${PLATFORM_MARKER}"
    echo "✓ Successfully downloaded and installed MuPDF libraries"
    echo "  Location: ${THIRD_PARTY}"
else
    echo "Error: Installation failed - missing required libraries"
    exit 1
fi

