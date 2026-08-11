#!/bin/bash
# Setup MuPDF libraries for go-mupdf
# This script ensures MuPDF libraries are available for building the Go wrapper

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MUPDF_DIR="$PROJECT_ROOT/third_party/mupdf"

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

# Check if libraries exist
check_libraries() {
    if [ -f "$MUPDF_DIR/build/release/libmupdf.a" ] && \
       [ -f "$MUPDF_DIR/build/release/libmupdf-third.a" ]; then
        return 0
    fi
    return 1
}

# Detect platform
detect_platform() {
    # Respects GOOS/GOARCH environment variables for cross-compilation
    # Falls back to uname for native builds
    local os="${GOOS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
    local arch="${GOARCH:-$(uname -m)}"
    
    case "$os" in
        linux*) os="linux" ;;
        darwin*) os="darwin" ;;
        mingw*|msys*|cygwin*|windows) os="windows" ;;
        *) os="unknown" ;;
    esac
    
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l|arm) arch="arm" ;;
        386|i386|i686) arch="386" ;;
        *) arch="unknown" ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest version from GitHub/Bitbucket
get_latest_version() {
    # Try to get from VERSION file
    if [ -f "$PROJECT_ROOT/VERSION" ]; then
        cat "$PROJECT_ROOT/VERSION"
        return
    fi
    
    # Try to get from git tags
    if command -v git >/dev/null 2>&1; then
        local version=$(cd "$PROJECT_ROOT" && git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -n "$version" ]; then
            echo "$version" | sed 's/^v//'
            return
        fi
    fi
    
    # Default version
    echo "1.1.0"
}

# Download pre-built libraries
download_prebuilt() {
    local platform="$1"
    local version="$2"
    
    log_info "Attempting to download pre-built libraries..."
    log_info "Platform: $platform"
    log_info "Version: $version"
    
    # Try GitHub Releases
    local base_url="https://github.com/Lexmata/go-mupdf/releases/download/v${version}"
    local filename="go-mupdf-${version}-${platform}.tar.gz"
    local url="${base_url}/${filename}"
    
    log_info "Downloading from: $url"
    
    local temp_dir=$(mktemp -d)
    
    if command -v wget >/dev/null 2>&1; then
        wget -O "$temp_dir/$filename" "$url" || {
            log_warn "Pre-built libraries not available for $platform v$version"
            rm -rf "$temp_dir"
            return 1
        }
    elif command -v curl >/dev/null 2>&1; then
        curl -f -L -o "$temp_dir/$filename" "$url" || {
            log_warn "Pre-built libraries not available for $platform v$version"
            rm -rf "$temp_dir"
            return 1
        }
    else
        log_error "Neither wget nor curl available for downloading"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Extract
    log_info "Extracting libraries..."
    tar -xzf "$temp_dir/$filename" -C "$temp_dir"
    
    local extracted_dir=$(find "$temp_dir" -maxdepth 1 -type d -name "go-mupdf-*" | head -1)
    
    if [ -z "$extracted_dir" ]; then
        log_error "Failed to find extracted directory"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Install libraries
    mkdir -p "$MUPDF_DIR/build/release"
    mkdir -p "$MUPDF_DIR/include"
    
    cp "$extracted_dir/lib/"*.a "$MUPDF_DIR/build/release/"
    cp -r "$extracted_dir/include/mupdf" "$MUPDF_DIR/include/"
    
    rm -rf "$temp_dir"
    
    if check_libraries; then
        log_success "Pre-built libraries installed successfully"
        return 0
    fi
    
    return 1
}

# Initialize submodule if needed
init_submodule() {
    if [ ! -d "$MUPDF_DIR" ] || [ ! -f "$MUPDF_DIR/Makefile" ]; then
        log_info "Initializing MuPDF submodule..."
        cd "$PROJECT_ROOT"
        
        if command -v git >/dev/null 2>&1; then
            git submodule update --init --recursive || {
                log_error "Failed to initialize submodule"
                return 1
            }
        else
            log_error "Git not available to initialize submodule"
            return 1
        fi
        
        log_success "Submodule initialized"
    fi
    return 0
}

# Build from source
build_from_source() {
    log_info "Building MuPDF from source..."
    log_warn "This may take 5-10 minutes on first build"
    
    if ! init_submodule; then
        return 1
    fi
    
    cd "$MUPDF_DIR"
    
    local nproc_count=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
    log_info "Building with $nproc_count parallel jobs..."
    
    make -j"$nproc_count" \
        USE_SYSTEM_LIBS=no \
        HAVE_X11=no \
        HAVE_GLUT=no \
        build=release \
        libs || {
        log_error "MuPDF build failed"
        return 1
    }
    
    cd "$PROJECT_ROOT"
    
    if check_libraries; then
        log_success "MuPDF built successfully"
        return 0
    fi
    
    return 1
}

# Show usage
show_usage() {
    cat << EOF
MuPDF Setup Script

This script sets up MuPDF libraries for go-mupdf by:
1. Checking if libraries already exist
2. Downloading pre-built libraries for your platform
3. Building from source if pre-built unavailable

Usage: $0 [OPTIONS]

Options:
    --download-only    Only try downloading, don't build from source
    --build-only       Skip download, build from source
    --force            Force rebuild even if libraries exist
    --help             Show this help message

Environment Variables:
    MUPDF_PLATFORM     Override platform detection (e.g., linux-amd64)
    MUPDF_VERSION      Override version detection

Examples:
    $0                          # Auto setup (recommended)
    $0 --download-only         # Only download pre-built
    $0 --build-only            # Always build from source
    $0 --force                 # Force rebuild

EOF
}

# Report that existing libraries are usable and exit successfully
report_existing_libraries() {
    log_success "MuPDF libraries already available"

    local mupdf_size=$(du -h "$MUPDF_DIR/build/release/libmupdf.a" 2>/dev/null | cut -f1)
    local third_size=$(du -h "$MUPDF_DIR/build/release/libmupdf-third.a" 2>/dev/null | cut -f1)

    log_info "libmupdf.a: $mupdf_size"
    log_info "libmupdf-third.a: $third_size"
    log_success "Setup complete! You can now build go-mupdf"
}

# Main setup logic
main() {
    log_info "go-mupdf MuPDF Setup"
    log_info "===================="

    # Detect platform
    local platform="${MUPDF_PLATFORM:-$(detect_platform)}"
    local marker="$MUPDF_DIR/build/release/.platform"

    # Check if already set up (for the right target platform)
    if check_libraries && [ "$FORCE" != "1" ]; then
        local existing=""
        if [ -f "$marker" ]; then
            existing=$(cat "$marker")
        fi

        if [ -n "$existing" ]; then
            if [ "$existing" = "$platform" ]; then
                report_existing_libraries
                return 0
            fi
            log_warn "Existing libraries are for $existing, need $platform — reinstalling"
            rm -f "$MUPDF_DIR/build/release/libmupdf.a" "$MUPDF_DIR/build/release/libmupdf-third.a" "$marker"
        else
            # Legacy install without a platform marker: inspect the archive itself
            local target_os="${platform%%-*}"
            local expected=""
            case "${platform##*-}" in
                amd64) expected="Advanced Micro Devices X86-64" ;;
                arm64) expected="AArch64" ;;
            esac
            local machine=""
            if [ "$target_os" = "linux" ] && [ -n "$expected" ] && command -v readelf >/dev/null 2>&1; then
                local member=$(ar t "$MUPDF_DIR/build/release/libmupdf.a" 2>/dev/null | head -1)
                machine=$(ar p "$MUPDF_DIR/build/release/libmupdf.a" "$member" 2>/dev/null | readelf -h /dev/stdin 2>/dev/null | awk -F: '/Machine/ {gsub(/^ +/,"",$2); print $2}')
            fi
            if [ -n "$machine" ] && [ "$machine" != "$expected" ]; then
                log_warn "Existing libraries are for $machine, need $expected ($platform) — reinstalling"
                rm -f "$MUPDF_DIR/build/release/libmupdf.a" "$MUPDF_DIR/build/release/libmupdf-third.a"
            else
                if [ -z "$machine" ]; then
                    log_warn "No platform marker found and architecture could not be verified; assuming libraries match $platform"
                fi
                report_existing_libraries
                return 0
            fi
        fi
    fi

    local version="${MUPDF_VERSION:-$(get_latest_version)}"

    log_info "Detected platform: $platform"
    log_info "Version: $version"

    # Try to download pre-built libraries
    if [ "$BUILD_ONLY" != "1" ]; then
        if download_prebuilt "$platform" "$version"; then
            echo "$platform" > "$marker"
            log_success "Setup complete! You can now build go-mupdf"
            return 0
        fi
    fi
    
    # Fall back to building from source
    if [ "$DOWNLOAD_ONLY" == "1" ]; then
        log_error "Download-only mode, but pre-built libraries not available"
        log_info "Try running without --download-only to build from source"
        return 1
    fi
    
    log_info "Pre-built libraries not available, building from source..."
    
    if build_from_source; then
        echo "$platform" > "$marker"
        log_success "Setup complete! You can now build go-mupdf"
        return 0
    fi
    
    log_error "Setup failed"
    log_error "Please check the error messages above and:"
    log_error "1. Ensure you have build tools installed (make, gcc/clang)"
    log_error "2. Check that git submodules are initialized"
    log_error "3. See docs/INSTALLATION.md for detailed instructions"
    return 1
}

# Parse arguments
DOWNLOAD_ONLY=0
BUILD_ONLY=0
FORCE=0

while [[ $# -gt 0 ]]; do
    case $1 in
        --download-only)
            DOWNLOAD_ONLY=1
            shift
            ;;
        --build-only)
            BUILD_ONLY=1
            shift
            ;;
        --force)
            FORCE=1
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Run main
main
exit $?

