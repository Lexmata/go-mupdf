#!/bin/bash
# Install pre-built MuPDF libraries from artifacts or cache
# This script can work with local artifacts, downloaded artifacts, or pre-built distributions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ARTIFACT_DIR="${ARTIFACT_DIR:-$PROJECT_ROOT/mupdf-artifacts}"
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

# Check if MuPDF libraries already exist
check_existing_libs() {
    if [ -f "$MUPDF_DIR/build/release/libmupdf.a" ] && [ -f "$MUPDF_DIR/build/release/libmupdf-third.a" ]; then
        log_info "MuPDF libraries already exist"
        return 0
    fi
    return 1
}

# Detect platform
# Respects GOOS/GOARCH environment variables for cross-compilation
# Falls back to uname for native builds
detect_platform() {
    # Detect OS - prefer GOOS if set
    local os="${GOOS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
    
    # Detect architecture - prefer GOARCH if set
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

# Record which target platform the installed libraries were built for
write_platform_marker() {
    mkdir -p "$MUPDF_DIR/build/release"
    detect_platform > "$MUPDF_DIR/build/release/.platform"
}

# Get version
get_version() {
    # Try VERSION file
    if [ -f "$PROJECT_ROOT/VERSION" ]; then
        cat "$PROJECT_ROOT/VERSION"
        return
    fi
    
    # Try git tags
    if command -v git >/dev/null 2>&1; then
        local version=$(cd "$PROJECT_ROOT" && git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -n "$version" ]; then
            echo "$version" | sed 's/^v//'
            return
        fi
    fi
    
    echo "latest"
}

# Download from Bitbucket
download_from_bitbucket() {
    local platform="$1"
    local version="$2"
    
    log_info "Attempting to download from Bitbucket Downloads..."
    log_info "Platform: $platform, Version: $version"
    
    local base_url="https://bitbucket.org/lexmata/go-mupdf/downloads"
    local filename="go-mupdf-${version}-${platform}.tar.gz"
    local url="${base_url}/${filename}"
    
    log_info "URL: $url"
    
    local temp_dir=$(mktemp -d)
    local success=0
    
    # Try to download
    if command -v curl >/dev/null 2>&1; then
        log_info "Downloading with curl..."
        if curl -f -L -o "$temp_dir/$filename" "$url" 2>/dev/null; then
            success=1
        fi
    elif command -v wget >/dev/null 2>&1; then
        log_info "Downloading with wget..."
        if wget -q -O "$temp_dir/$filename" "$url" 2>/dev/null; then
            success=1
        fi
    else
        log_warn "Neither curl nor wget available"
        rm -rf "$temp_dir"
        return 1
    fi
    
    if [ $success -eq 0 ]; then
        log_warn "Download failed for $platform v$version"
        rm -rf "$temp_dir"
        return 1
    fi
    
    log_success "Downloaded successfully"
    
    # Extract and install
    log_info "Extracting..."
    tar -xzf "$temp_dir/$filename" -C "$temp_dir"
    
    local extracted_dir=$(find "$temp_dir" -maxdepth 1 -type d -name "go-mupdf-*" | head -1)
    
    if [ -z "$extracted_dir" ]; then
        log_error "Failed to find extracted directory"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Install
    mkdir -p "$MUPDF_DIR/build/release"
    mkdir -p "$MUPDF_DIR/include"
    
    cp "$extracted_dir/lib/"*.a "$MUPDF_DIR/build/release/"
    cp -r "$extracted_dir/include/mupdf" "$MUPDF_DIR/include/"
    
    rm -rf "$temp_dir"
    
    if check_existing_libs; then
        write_platform_marker
        log_success "Installed from Bitbucket Downloads"
        return 0
    fi
    
    return 1
}

# Install from artifact directory
install_from_artifact() {
    log_info "Installing MuPDF libraries from artifact..."

    if [ ! -d "$ARTIFACT_DIR" ]; then
        log_error "Artifact directory not found: $ARTIFACT_DIR"
        return 1
    fi

    # Check for tarball
    if [ -f "$ARTIFACT_DIR/mupdf-libs.tar.gz" ]; then
        log_info "Extracting MuPDF libraries from tarball..."
        tar -xzf "$ARTIFACT_DIR/mupdf-libs.tar.gz" -C "$PROJECT_ROOT"

        if check_existing_libs; then
            write_platform_marker
            log_success "MuPDF libraries installed from tarball"
            return 0
        fi
    fi

    # Check for direct library files
    if [ -f "$ARTIFACT_DIR/lib/libmupdf.a" ]; then
        log_info "Copying MuPDF libraries from artifact directory..."
        mkdir -p "$MUPDF_DIR/build/release"
        cp "$ARTIFACT_DIR/lib/"*.a "$MUPDF_DIR/build/release/"

        # Copy headers if available
        if [ -d "$ARTIFACT_DIR/include/mupdf" ]; then
            mkdir -p "$MUPDF_DIR/include"
            cp -r "$ARTIFACT_DIR/include/mupdf" "$MUPDF_DIR/include/"
        fi

        if check_existing_libs; then
            write_platform_marker
            log_success "MuPDF libraries installed from artifact directory"
            return 0
        fi
    fi

    log_warn "No usable artifacts found in $ARTIFACT_DIR"
    return 1
}

# Install from distribution package
install_from_distribution() {
    local dist_file="$1"

    if [ ! -f "$dist_file" ]; then
        log_error "Distribution file not found: $dist_file"
        return 1
    fi

    log_info "Installing from distribution: $dist_file"

    local temp_dir=$(mktemp -d)
    tar -xzf "$dist_file" -C "$temp_dir"

    # Find the extracted directory
    local extracted_dir=$(find "$temp_dir" -maxdepth 1 -type d -name "go-mupdf-*" | head -1)

    if [ -z "$extracted_dir" ]; then
        log_error "Failed to find extracted directory"
        rm -rf "$temp_dir"
        return 1
    fi

    # Copy libraries
    mkdir -p "$MUPDF_DIR/build/release"
    cp "$extracted_dir/lib/"*.a "$MUPDF_DIR/build/release/"

    # Copy headers
    mkdir -p "$MUPDF_DIR/include"
    cp -r "$extracted_dir/include/mupdf" "$MUPDF_DIR/include/"

    rm -rf "$temp_dir"

    if check_existing_libs; then
        write_platform_marker
        log_success "MuPDF libraries installed from distribution"
        return 0
    fi

    return 1
}

# Build from source as fallback
build_from_source() {
    log_warn "Building MuPDF from source (this may take several minutes)..."

    if [ ! -d "$MUPDF_DIR" ]; then
        log_error "MuPDF source directory not found: $MUPDF_DIR"
        log_error "Run: git submodule update --init --recursive"
        return 1
    fi

    cd "$MUPDF_DIR"

    local nproc_count=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

    make -j"$nproc_count" \
        USE_SYSTEM_LIBS=no \
        HAVE_X11=no \
        HAVE_GLUT=no \
        build=release \
        libs

    cd "$PROJECT_ROOT"

    if check_existing_libs; then
        write_platform_marker
        log_success "MuPDF built from source successfully"
        return 0
    fi

    log_error "Failed to build MuPDF from source"
    return 1
}

# Verify installed libraries
verify_libraries() {
    log_info "Verifying MuPDF libraries..."

    if [ ! -f "$MUPDF_DIR/build/release/libmupdf.a" ]; then
        log_error "libmupdf.a not found"
        return 1
    fi

    if [ ! -f "$MUPDF_DIR/build/release/libmupdf-third.a" ]; then
        log_error "libmupdf-third.a not found"
        return 1
    fi

    local mupdf_size=$(du -h "$MUPDF_DIR/build/release/libmupdf.a" | cut -f1)
    local third_size=$(du -h "$MUPDF_DIR/build/release/libmupdf-third.a" | cut -f1)

    log_info "libmupdf.a: $mupdf_size"
    log_info "libmupdf-third.a: $third_size"

    # Basic sanity check - files should be at least 1MB
    local mupdf_size_bytes=$(stat -f%z "$MUPDF_DIR/build/release/libmupdf.a" 2>/dev/null || stat -c%s "$MUPDF_DIR/build/release/libmupdf.a" 2>/dev/null)

    if [ "$mupdf_size_bytes" -lt 1000000 ]; then
        log_error "libmupdf.a seems too small (< 1MB), may be corrupted"
        return 1
    fi

    local third_size_bytes=$(stat -f%z "$MUPDF_DIR/build/release/libmupdf-third.a" 2>/dev/null || stat -c%s "$MUPDF_DIR/build/release/libmupdf-third.a" 2>/dev/null)

    if [ "$third_size_bytes" -lt 1000000 ]; then
        log_error "libmupdf-third.a seems too small (< 1MB), may be corrupted"
        return 1
    fi

    log_success "MuPDF libraries verified successfully"
    return 0
}

# Main installation logic
main() {
    log_info "MuPDF Pre-built Library Installer"
    log_info "=================================="

    # Check if libraries already exist (and match the target platform)
    local platform=$(detect_platform)
    local marker="$MUPDF_DIR/build/release/.platform"

    if check_existing_libs; then
        local existing=""
        if [ -f "$marker" ]; then
            existing=$(cat "$marker")
        fi

        if [ -n "$existing" ]; then
            if [ "$existing" = "$platform" ]; then
                log_success "MuPDF libraries already installed"
                verify_libraries
                exit 0
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
                log_success "MuPDF libraries already installed"
                verify_libraries
                exit 0
            fi
        fi
    fi

    # Try installation methods in order of preference
    local success=0

    # 1. Try to install from artifact directory
    if [ -d "$ARTIFACT_DIR" ]; then
        if install_from_artifact; then
            success=1
        fi
    fi

    # 2. Try to install from specified distribution file
    if [ $success -eq 0 ] && [ -n "$DIST_FILE" ] && [ -f "$DIST_FILE" ]; then
        if install_from_distribution "$DIST_FILE"; then
            success=1
        fi
    fi

    # 3. Try to download from Bitbucket
    if [ $success -eq 0 ] && [ "$SKIP_DOWNLOAD" != "1" ]; then
        local version=$(get_version)
        
        log_info "No local artifacts found, trying Bitbucket Downloads..."
        if download_from_bitbucket "$platform" "$version"; then
            success=1
        fi
    fi

    # 4. Fall back to building from source
    if [ $success -eq 0 ]; then
        log_warn "No pre-built libraries available, falling back to source build"
        if build_from_source; then
            success=1
        fi
    fi

    if [ $success -eq 0 ]; then
        log_error "Failed to install MuPDF libraries"
        exit 1
    fi

    # Verify the installation
    if ! verify_libraries; then
        log_error "Library verification failed"
        exit 1
    fi

    log_success "MuPDF installation complete!"
}

# Help text
show_help() {
    cat << EOF
MuPDF Pre-built Library Installer

Usage: $0 [OPTIONS]

Options:
    --artifact-dir DIR     Directory containing pre-built artifacts (default: mupdf-artifacts)
    --dist-file FILE       Install from a specific distribution tarball
    --skip-download        Skip downloading from Bitbucket Downloads
    --force-build          Skip artifacts and build from source
    --help                 Show this help message

Environment Variables:
    ARTIFACT_DIR           Directory containing pre-built artifacts
    DIST_FILE             Distribution tarball to install from
    SKIP_DOWNLOAD         Set to "1" to skip downloading from Bitbucket

Examples:
    $0                                          # Auto-detect and install
    $0 --artifact-dir ./artifacts              # Use specific artifact directory
    $0 --dist-file dist/go-mupdf-1.1.0.tar.gz # Install from distribution
    $0 --skip-download                         # Don't download, build if needed
    $0 --force-build                           # Force build from source

This script attempts to install MuPDF libraries in the following order:
1. From artifact directory (ARTIFACT_DIR or --artifact-dir)
2. From distribution tarball (DIST_FILE or --dist-file)
3. Download from Bitbucket Downloads (unless --skip-download)
4. Build from source as fallback

EOF
}

# Parse arguments
FORCE_BUILD=0
SKIP_DOWNLOAD=${SKIP_DOWNLOAD:-0}

while [[ $# -gt 0 ]]; do
    case $1 in
        --artifact-dir)
            ARTIFACT_DIR="$2"
            shift 2
            ;;
        --dist-file)
            DIST_FILE="$2"
            shift 2
            ;;
        --skip-download)
            SKIP_DOWNLOAD=1
            shift
            ;;
        --force-build)
            FORCE_BUILD=1
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

# Force build mode
if [ $FORCE_BUILD -eq 1 ]; then
    log_info "Force build mode enabled"
    build_from_source
    verify_libraries
    exit $?
fi

# Run main installation
main

