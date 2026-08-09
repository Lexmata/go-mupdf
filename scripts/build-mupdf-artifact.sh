#!/bin/bash
# Build MuPDF libraries and create artifact for CI/CD
# This creates a reusable artifact that can be shared across pipeline steps

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MUPDF_DIR="$PROJECT_ROOT/third_party/mupdf"
ARTIFACT_DIR="${ARTIFACT_DIR:-$PROJECT_ROOT/mupdf-artifacts}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

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

# Check if MuPDF source exists
check_mupdf_source() {
    if [ ! -d "$MUPDF_DIR" ]; then
        log_error "MuPDF directory not found: $MUPDF_DIR"
        log_error "Please run: git submodule update --init --recursive"
        exit 1
    fi

    if [ ! -f "$MUPDF_DIR/Makefile" ]; then
        log_error "MuPDF Makefile not found. Is the submodule properly initialized?"
        exit 1
    fi

    log_success "MuPDF source found"
}

# Build MuPDF libraries
build_mupdf() {
    log_info "Building MuPDF libraries..."

    cd "$MUPDF_DIR"

    # Clean previous build (only with --force; otherwise keep incremental state)
    if [ "$FORCE" = "1" ]; then
        log_info "Cleaning previous build..."
        make clean 2>/dev/null || true
    fi

    # Detect number of processors
    local nproc_count=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
    log_info "Building with $nproc_count parallel jobs..."

    # Build MuPDF with all dependencies statically linked
    make -j"$nproc_count" \
        USE_SYSTEM_LIBS=no \
        HAVE_X11=no \
        HAVE_GLUT=no \
        build=release \
        libs

    cd "$PROJECT_ROOT"

    # Verify libraries were built
    if [ ! -f "$MUPDF_DIR/build/release/libmupdf.a" ]; then
        log_error "libmupdf.a not built!"
        exit 1
    fi

    if [ ! -f "$MUPDF_DIR/build/release/libmupdf-third.a" ]; then
        log_error "libmupdf-third.a not built!"
        exit 1
    fi

    log_success "MuPDF built successfully"
}

# Create artifact package
create_artifact() {
    log_info "Creating artifact package..."

    # Wipe the artifact directory only with --force; otherwise refresh in place
    # so a restored CI cache is not destroyed
    if [ "$FORCE" = "1" ]; then
        rm -rf "$ARTIFACT_DIR"
    fi
    mkdir -p "$ARTIFACT_DIR/lib"
    mkdir -p "$ARTIFACT_DIR/include"

    # Copy libraries
    log_info "Copying libraries..."
    cp "$MUPDF_DIR/build/release/libmupdf.a" "$ARTIFACT_DIR/lib/"
    cp "$MUPDF_DIR/build/release/libmupdf-third.a" "$ARTIFACT_DIR/lib/"

    # Copy headers
    log_info "Copying headers..."
    cp -r "$MUPDF_DIR/include/mupdf" "$ARTIFACT_DIR/include/"

    # Create metadata
    cat > "$ARTIFACT_DIR/BUILD_INFO" << EOF
Build Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
Build Host: $(hostname)
Platform: $(uname -s)-$(uname -m)
MuPDF Commit: $(cd "$MUPDF_DIR" && git rev-parse HEAD 2>/dev/null || echo "unknown")
EOF

    # Create tarball
    log_info "Creating tarball..."
    cd "$PROJECT_ROOT"
    tar -czf mupdf-libs.tar.gz \
        third_party/mupdf/build/release/libmupdf.a \
        third_party/mupdf/build/release/libmupdf-third.a \
        third_party/mupdf/include/mupdf

    # Move tarball to artifact directory
    mv mupdf-libs.tar.gz "$ARTIFACT_DIR/"

    log_success "Artifact package created"
}

# Show artifact info
show_artifact_info() {
    log_info "Artifact Information"
    log_info "===================="

    if [ -f "$ARTIFACT_DIR/BUILD_INFO" ]; then
        cat "$ARTIFACT_DIR/BUILD_INFO"
    fi

    echo ""
    log_info "Library Sizes:"
    du -h "$ARTIFACT_DIR/lib/libmupdf.a"
    du -h "$ARTIFACT_DIR/lib/libmupdf-third.a"

    echo ""
    log_info "Tarball:"
    du -h "$ARTIFACT_DIR/mupdf-libs.tar.gz"

    echo ""
    log_info "Artifact contents:"
    ls -lh "$ARTIFACT_DIR/"

    echo ""
    log_info "Header files:"
    find "$ARTIFACT_DIR/include" -name "*.h" | wc -l | xargs echo "Total headers:"
}

# Check whether a previously built artifact (e.g. restored from the CI cache)
# is already valid and can be reused without rebuilding
check_cached_artifact() {
    if [ ! -f "$ARTIFACT_DIR/lib/libmupdf.a" ] || [ ! -f "$ARTIFACT_DIR/lib/libmupdf-third.a" ]; then
        return 1
    fi

    # Check library sizes (should be at least 1MB each)
    local mupdf_size=$(stat -f%z "$ARTIFACT_DIR/lib/libmupdf.a" 2>/dev/null || stat -c%s "$ARTIFACT_DIR/lib/libmupdf.a" 2>/dev/null)
    local third_size=$(stat -f%z "$ARTIFACT_DIR/lib/libmupdf-third.a" 2>/dev/null || stat -c%s "$ARTIFACT_DIR/lib/libmupdf-third.a" 2>/dev/null)

    if [ -z "$mupdf_size" ] || [ "$mupdf_size" -lt 1000000 ]; then
        log_warn "Cached libmupdf.a seems too small (< 1MB), rebuilding"
        return 1
    fi

    if [ -z "$third_size" ] || [ "$third_size" -lt 1000000 ]; then
        log_warn "Cached libmupdf-third.a seems too small (< 1MB), rebuilding"
        return 1
    fi

    # If a tarball is present it must be readable
    if [ -f "$ARTIFACT_DIR/mupdf-libs.tar.gz" ] && ! tar -tzf "$ARTIFACT_DIR/mupdf-libs.tar.gz" > /dev/null 2>&1; then
        log_warn "Cached tarball is corrupted, rebuilding"
        return 1
    fi

    return 0
}

# Recreate the tarball from a cached artifact if it is missing
recreate_tarball() {
    if [ -f "$ARTIFACT_DIR/mupdf-libs.tar.gz" ]; then
        return 0
    fi

    log_info "Tarball missing from cached artifact, recreating..."

    # Stage the cached libraries/headers into the source-tree layout the
    # tarball uses, then pack them
    mkdir -p "$MUPDF_DIR/build/release"
    cp "$ARTIFACT_DIR/lib/libmupdf.a" "$MUPDF_DIR/build/release/"
    cp "$ARTIFACT_DIR/lib/libmupdf-third.a" "$MUPDF_DIR/build/release/"
    if [ -d "$ARTIFACT_DIR/include/mupdf" ]; then
        mkdir -p "$MUPDF_DIR/include"
        cp -r "$ARTIFACT_DIR/include/mupdf" "$MUPDF_DIR/include/"
    fi

    cd "$PROJECT_ROOT"
    tar -czf mupdf-libs.tar.gz \
        third_party/mupdf/build/release/libmupdf.a \
        third_party/mupdf/build/release/libmupdf-third.a \
        third_party/mupdf/include/mupdf

    mv mupdf-libs.tar.gz "$ARTIFACT_DIR/"

    log_success "Tarball recreated"
}

# Verify artifact
verify_artifact() {
    log_info "Verifying artifact..."

    # Check libraries exist
    if [ ! -f "$ARTIFACT_DIR/lib/libmupdf.a" ]; then
        log_error "libmupdf.a missing from artifact"
        exit 1
    fi

    if [ ! -f "$ARTIFACT_DIR/lib/libmupdf-third.a" ]; then
        log_error "libmupdf-third.a missing from artifact"
        exit 1
    fi

    # Check tarball exists
    if [ ! -f "$ARTIFACT_DIR/mupdf-libs.tar.gz" ]; then
        log_error "Tarball missing from artifact"
        exit 1
    fi

    # Check tarball can be extracted
    local temp_dir=$(mktemp -d)
    if ! tar -tzf "$ARTIFACT_DIR/mupdf-libs.tar.gz" > /dev/null 2>&1; then
        log_error "Tarball is corrupted"
        rm -rf "$temp_dir"
        exit 1
    fi
    rm -rf "$temp_dir"

    # Check library sizes (should be at least 1MB each)
    local mupdf_size=$(stat -f%z "$ARTIFACT_DIR/lib/libmupdf.a" 2>/dev/null || stat -c%s "$ARTIFACT_DIR/lib/libmupdf.a" 2>/dev/null)
    local third_size=$(stat -f%z "$ARTIFACT_DIR/lib/libmupdf-third.a" 2>/dev/null || stat -c%s "$ARTIFACT_DIR/lib/libmupdf-third.a" 2>/dev/null)

    if [ "$mupdf_size" -lt 1000000 ]; then
        log_error "libmupdf.a seems too small (< 1MB)"
        exit 1
    fi

    if [ "$third_size" -lt 1000000 ]; then
        log_error "libmupdf-third.a seems too small (< 1MB)"
        exit 1
    fi

    log_success "Artifact verified successfully"
}

# Main execution
main() {
    log_info "MuPDF Artifact Builder"
    log_info "======================"

    check_mupdf_source

    # Reuse a valid cached artifact (e.g. restored from the CI cache) instead
    # of rebuilding; use --force to rebuild from scratch
    if [ "$FORCE" != "1" ] && check_cached_artifact; then
        log_info "Using cached MuPDF artifact"
        recreate_tarball
        verify_artifact
        show_artifact_info

        log_success "Cached artifact ready!"
        log_info "Artifact location: $ARTIFACT_DIR"
        log_info "Tarball: $ARTIFACT_DIR/mupdf-libs.tar.gz"
        exit 0
    fi

    build_mupdf
    create_artifact
    verify_artifact
    show_artifact_info

    log_success "Artifact build complete!"
    log_info "Artifact location: $ARTIFACT_DIR"
    log_info "Tarball: $ARTIFACT_DIR/mupdf-libs.tar.gz"
}

# Help
show_help() {
    cat << EOF
MuPDF Artifact Builder

Builds MuPDF libraries and creates a reusable artifact package for CI/CD.

Usage: $0 [OPTIONS]

Options:
    --artifact-dir DIR     Output directory for artifacts (default: mupdf-artifacts)
    --force                Force a clean rebuild, discarding any cached artifact
    --help                 Show this help message

Environment Variables:
    ARTIFACT_DIR          Output directory for artifacts

The artifact includes:
- libmupdf.a (MuPDF core library)
- libmupdf-third.a (third-party dependencies)
- MuPDF header files
- Tarball for easy distribution

Examples:
    $0                              # Build and create artifact
    $0 --artifact-dir ./my-artifact # Use custom output directory
    ARTIFACT_DIR=/tmp/mupdf $0     # Use environment variable

EOF
}

# Parse arguments
FORCE=0

while [[ $# -gt 0 ]]; do
    case $1 in
        --artifact-dir)
            ARTIFACT_DIR="$2"
            shift 2
            ;;
        --force)
            FORCE=1
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main
main

