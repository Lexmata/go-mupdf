#!/bin/bash

# CI Setup Script for Go MuPDF Wrapper
# This script sets up the build environment for CI/CD pipelines
#
# NOTE: The exported Go environment variables only persist in your shell if
# this script is SOURCED, not executed:
#     source scripts/ci-setup.sh
# Running it directly (./scripts/ci-setup.sh) still installs dependencies and
# builds MuPDF, but the exports are lost when the script exits.
#
# NOTE: This script is not currently invoked by bitbucket-pipelines.yml --
# it is intended for local/manual environment setup.

set -e

echo "=== CI Setup for Go MuPDF Wrapper ==="

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install system dependencies based on OS
install_system_deps() {
    if command_exists apt-get; then
        echo "Installing dependencies with apt-get..."
        apt-get update
        apt-get install -y build-essential make gcc g++ libc6-dev pkg-config
        apt-get install -y libx11-dev libxext-dev libxrandr-dev libgl1-mesa-dev
        apt-get install -y git curl
    elif command_exists yum; then
        echo "Installing dependencies with yum..."
        yum update -y
        yum groupinstall -y "Development Tools"
        yum install -y libX11-devel libXext-devel libXrandr-devel mesa-libGL-devel
        yum install -y git curl
    elif command_exists apk; then
        echo "Installing dependencies with apk..."
        apk update
        apk add build-base make gcc g++ libc-dev pkgconfig
        apk add libx11-dev libxext-dev libxrandr-dev mesa-dev
        apk add git curl
    else
        echo "Warning: Unknown package manager. Please install build tools manually."
    fi
}

# Set up Go environment
setup_go_env() {
    echo "Setting up Go environment..."
    # Respect values already set by the caller (e.g. for cross-compilation)
    export GO111MODULE="${GO111MODULE:-on}"
    export CGO_ENABLED="${CGO_ENABLED:-1}"
    export GOOS="${GOOS:-linux}"
    export GOARCH="${GOARCH:-amd64}"
    
    # Verify Go installation
    go version
    
    echo "Go environment configured:"
    echo "  GO111MODULE=$GO111MODULE"
    echo "  CGO_ENABLED=$CGO_ENABLED"
    echo "  GOOS=$GOOS"
    echo "  GOARCH=$GOARCH"
}

# Build MuPDF library
build_mupdf() {
    echo "Building MuPDF library..."
    
    if [ ! -d "third_party/mupdf" ]; then
        echo "Error: MuPDF source directory not found at third_party/mupdf"
        exit 1
    fi
    
    cd third_party/mupdf
    
    # Check if already built
    if [ -f "build/release/libmupdf.a" ]; then
        echo "MuPDF library already built, skipping..."
        cd ../..
        return 0
    fi
    
    echo "Starting MuPDF build..."
    make -j$(nproc) libs USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release
    
    # Verify build
    if [ -f "build/release/libmupdf.a" ]; then
        echo "MuPDF library built successfully!"
        ls -la build/release/
    else
        echo "Error: MuPDF library build failed"
        exit 1
    fi
    
    cd ../..
}

# Verify build environment
verify_environment() {
    echo "Verifying build environment..."
    
    # Check Go
    if ! command_exists go; then
        echo "Error: Go not found in PATH"
        exit 1
    fi
    
    # Check GCC
    if ! command_exists gcc; then
        echo "Error: GCC not found in PATH"
        exit 1
    fi
    
    # Check make
    if ! command_exists make; then
        echo "Error: make not found in PATH"
        exit 1
    fi
    
    # Check MuPDF libraries
    if [ ! -f "third_party/mupdf/build/release/libmupdf.a" ]; then
        echo "Error: MuPDF library (libmupdf.a) not found"
        exit 1
    fi

    if [ ! -f "third_party/mupdf/build/release/libmupdf-third.a" ]; then
        echo "Error: MuPDF third-party library (libmupdf-third.a) not found"
        exit 1
    fi
    
    echo "Build environment verified successfully!"
}

# Main execution
main() {
    echo "Starting CI setup..."
    
    install_system_deps
    setup_go_env
    build_mupdf
    verify_environment
    
    echo "=== CI Setup Complete ==="
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi