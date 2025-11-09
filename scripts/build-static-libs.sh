#!/bin/bash
# Build static libraries for distribution
# This script builds MuPDF with all dependencies statically linked
# and creates distributable archives for different platforms

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MUPDF_DIR="$PROJECT_ROOT/third_party/mupdf"
DIST_DIR="$PROJECT_ROOT/dist"
BUILD_TYPE="${BUILD_TYPE:-release}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux*)
            os="linux"
            ;;
        darwin*)
            os="darwin"
            ;;
        mingw*|msys*|cygwin*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l)
            arch="arm"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Build MuPDF static libraries
build_mupdf() {
    log_info "Building MuPDF static libraries..."

    if [ ! -d "$MUPDF_DIR" ]; then
        log_error "MuPDF directory not found: $MUPDF_DIR"
        log_error "Please ensure git submodules are initialized: git submodule update --init --recursive"
        exit 1
    fi

    cd "$MUPDF_DIR"

    # Clean previous build
    log_info "Cleaning previous build..."
    make clean || true

    # Build MuPDF with all dependencies statically linked
    log_info "Compiling MuPDF (this may take several minutes)..."
    local nproc_count=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

    make -j"$nproc_count" \
        USE_SYSTEM_LIBS=no \
        HAVE_X11=no \
        HAVE_GLUT=no \
        build=$BUILD_TYPE \
        libs

    log_success "MuPDF built successfully"
}

# Create distribution package
create_distribution() {
    local platform="$1"
    local version="${VERSION:-dev}"

    log_info "Creating distribution package for $platform..."

    local dist_name="go-mupdf-${version}-${platform}"
    local dist_path="$DIST_DIR/$dist_name"

    # Clean and create distribution directory
    rm -rf "$dist_path"
    mkdir -p "$dist_path/lib"
    mkdir -p "$dist_path/include"
    mkdir -p "$dist_path/docs"

    # Copy static libraries
    log_info "Copying static libraries..."
    cp "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf.a" "$dist_path/lib/"
    cp "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf-third.a" "$dist_path/lib/"

    # Copy headers
    log_info "Copying header files..."
    cp -r "$MUPDF_DIR/include/mupdf" "$dist_path/include/"

    # Create README for distribution
    cat > "$dist_path/README.md" << 'EOF'
# go-mupdf Static Libraries

This package contains pre-built static libraries for go-mupdf.

## Contents

- `lib/libmupdf.a` - MuPDF core library
- `lib/libmupdf-third.a` - MuPDF third-party dependencies (statically linked)
- `include/mupdf/` - MuPDF header files

## Usage

### Option 1: Using with go-mupdf

1. Clone the go-mupdf repository:
   ```bash
   git clone https://bitbucket.org/lexmata/go-mupdf.git
   cd go-mupdf
   ```

2. Extract this package:
   ```bash
   tar -xzf go-mupdf-<version>-<platform>.tar.gz
   ```

3. Copy the libraries to the expected location:
   ```bash
   mkdir -p third_party/mupdf/build/release
   cp go-mupdf-<version>-<platform>/lib/*.a third_party/mupdf/build/release/
   cp -r go-mupdf-<version>-<platform>/include/mupdf third_party/mupdf/include/
   ```

4. Build your Go application:
   ```bash
   go build
   ```

### Option 2: Using in Your Own CGO Project

Add these flags to your CGO configuration:

```go
/*
#cgo CFLAGS: -I/path/to/extracted/include
#cgo LDFLAGS: -L/path/to/extracted/lib -lmupdf -lmupdf-third -lm
*/
import "C"
```

## Dependencies

These static libraries include all necessary dependencies:
- FreeType
- HarfBuzz
- JPEG
- JPEG XR
- OpenJPEG
- JBIG2Dec
- Zlib
- And more...

No additional system libraries are required beyond standard C libraries (libc, libm).

## Platform Notes

### Linux
- Requires: glibc 2.17+ (compatible with most modern Linux distributions)
- Standard math library (libm) is required but is part of glibc

### macOS
- Requires: macOS 10.13+ (High Sierra or later)
- Uses native system libraries

### Windows
- Requires: MinGW-w64 or similar for CGO
- Tested with Go 1.21+

## License

MuPDF is licensed under the GNU Affero General Public License (AGPL) v3.
Commercial licenses are available from Artifex Software.

See https://mupdf.com/licensing/ for more information.

## Support

For issues with the go-mupdf wrapper:
- Repository: https://bitbucket.org/lexmata/go-mupdf
- Issues: https://bitbucket.org/lexmata/go-mupdf/issues

For issues with MuPDF itself:
- Website: https://mupdf.com/
- Documentation: https://mupdf.readthedocs.io/
EOF

    # Copy version and license information
    if [ -f "$PROJECT_ROOT/VERSION" ]; then
        cp "$PROJECT_ROOT/VERSION" "$dist_path/"
    fi

    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        cp "$PROJECT_ROOT/LICENSE" "$dist_path/docs/"
    fi

    if [ -f "$MUPDF_DIR/COPYING" ]; then
        cp "$MUPDF_DIR/COPYING" "$dist_path/docs/LICENSE.MuPDF"
    fi

    # Create tarball
    log_info "Creating tarball..."
    cd "$DIST_DIR"
    tar -czf "${dist_name}.tar.gz" "$dist_name"

    # Create checksum
    log_info "Generating checksums..."
    sha256sum "${dist_name}.tar.gz" > "${dist_name}.tar.gz.sha256"

    # Cleanup temporary directory
    rm -rf "$dist_path"

    log_success "Distribution package created: $DIST_DIR/${dist_name}.tar.gz"

    # Print package info
    local size=$(du -h "$DIST_DIR/${dist_name}.tar.gz" | cut -f1)
    log_info "Package size: $size"
    log_info "SHA256: $(cat "$DIST_DIR/${dist_name}.tar.gz.sha256")"
}

# Verify libraries
verify_libraries() {
    local platform="$1"

    log_info "Verifying built libraries..."

    if [ ! -f "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf.a" ]; then
        log_error "libmupdf.a not found!"
        exit 1
    fi

    if [ ! -f "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf-third.a" ]; then
        log_error "libmupdf-third.a not found!"
        exit 1
    fi

    # Check library sizes
    local mupdf_size=$(du -h "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf.a" | cut -f1)
    local third_size=$(du -h "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf-third.a" | cut -f1)

    log_info "libmupdf.a size: $mupdf_size"
    log_info "libmupdf-third.a size: $third_size"

    # List symbols (basic verification)
    log_info "Checking for key symbols..."
    if command -v nm >/dev/null 2>&1; then
        if nm "$MUPDF_DIR/build/$BUILD_TYPE/libmupdf.a" 2>/dev/null | grep -q "fz_new_context"; then
            log_success "Key MuPDF symbols found"
        else
            log_warn "Could not verify MuPDF symbols (this may be normal on some platforms)"
        fi
    fi

    log_success "Library verification completed"
}

# Main execution
main() {
    log_info "go-mupdf Static Library Builder"
    log_info "================================"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                log_info "Cleaning dist directory..."
                rm -rf "$DIST_DIR"
                log_success "Cleaned"
                exit 0
                ;;
            --build-only)
                BUILD_ONLY=1
                ;;
            --skip-build)
                SKIP_BUILD=1
                ;;
            --help)
                cat << EOF
Usage: $0 [OPTIONS]

Build static libraries for distribution.

Options:
    --clean         Clean the dist directory and exit
    --build-only    Only build libraries, don't create distribution
    --skip-build    Skip building, only create distribution (assumes libraries exist)
    --help          Show this help message

Environment Variables:
    BUILD_TYPE      Build type: release (default) or debug
    VERSION         Version string for the distribution package

Examples:
    $0                          # Build and create distribution
    $0 --build-only            # Only build MuPDF
    BUILD_TYPE=debug $0        # Build debug version
    VERSION=1.1.0 $0           # Build with specific version
EOF
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                log_info "Use --help for usage information"
                exit 1
                ;;
        esac
        shift
    done

    # Detect platform
    PLATFORM=$(detect_platform)
    log_info "Detected platform: $PLATFORM"
    log_info "Build type: $BUILD_TYPE"

    # Build MuPDF
    if [ -z "$SKIP_BUILD" ]; then
        build_mupdf
        verify_libraries "$PLATFORM"
    else
        log_warn "Skipping build (--skip-build specified)"
    fi

    # Create distribution
    if [ -z "$BUILD_ONLY" ]; then
        # Get version from VERSION file if not set
        if [ -z "$VERSION" ] && [ -f "$PROJECT_ROOT/VERSION" ]; then
            VERSION=$(cat "$PROJECT_ROOT/VERSION")
            log_info "Using version from VERSION file: $VERSION"
        fi

        create_distribution "$PLATFORM"

        log_success "Build completed successfully!"
        log_info "Distribution package: $DIST_DIR/go-mupdf-${VERSION:-dev}-${PLATFORM}.tar.gz"
    else
        log_info "Build-only mode: distribution package not created"
    fi
}

# Run main function
main "$@"

