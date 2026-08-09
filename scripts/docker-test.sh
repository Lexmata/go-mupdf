#!/bin/bash
# Docker test runner script
# This script provides an easy way to build and run tests in Docker

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Docker image name
IMAGE_NAME="go-mupdf-test"
IMAGE_TAG="latest"

# Shared options (set by main from command-line flags)
VERBOSE=0
REBUILD=0
NO_CACHE=0
SHORT=0

# Source mounts: overlay only the Go sources onto the image's /workspace so
# the MuPDF libraries built into the image (third_party/...) stay visible.
# Mounting the whole repo over /workspace would shadow them.
DOCKER_MOUNTS=(
    -v "${PROJECT_ROOT}/pkg:/workspace/pkg"
    -v "${PROJECT_ROOT}/go.mod:/workspace/go.mod:ro"
    -v "${PROJECT_ROOT}/go.sum:/workspace/go.sum:ro"
)

# Host directory that receives files written by containers (e.g. coverage)
ARTIFACTS_DIR="${PROJECT_ROOT}/.docker-artifacts"

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    build       Build the Docker test image
    test        Run tests in Docker (builds image if needed)
    shell       Open a shell in the Docker container
    clean       Remove the Docker test image
    coverage    Run tests and generate coverage report (written to .docker-artifacts/)
    quick       Run tests without rebuilding (uses cached image)
    help        Show this help message

Options (may be placed before or after the command):
    -v, --verbose    Verbose output (go test -v; docker build --progress=plain)
    -r, --rebuild    Force rebuild of the Docker image before running the command
    -s, --short      Run tests in short mode (go test -short)
    --no-cache       Build Docker image without cache

Arguments after '--' are passed through to go test (test, quick, and
coverage commands), e.g.: $0 test -- -run TestDocument

Examples:
    $0 build                    # Build the test image
    $0 build --no-cache         # Build without Docker cache
    $0 test                     # Build (if needed) and run all tests
    $0 test --short             # Run quick tests only
    $0 test --rebuild           # Rebuild image, then run tests
    $0 quick                    # Run tests without rebuilding
    $0 coverage                 # Generate coverage report
    $0 shell                    # Open interactive shell

EOF
}

# Function to build Docker image
build_image() {
    local build_flags=()
    if [ "$NO_CACHE" = "1" ]; then
        build_flags+=(--no-cache)
    fi
    if [ "$VERBOSE" = "1" ]; then
        build_flags+=(--progress=plain)
    fi

    print_info "Building Docker test image..."
    cd "$PROJECT_ROOT"

    if docker build "${build_flags[@]}" -f Dockerfile.test -t "${IMAGE_NAME}:${IMAGE_TAG}" .; then
        print_info "Docker image built successfully: ${IMAGE_NAME}:${IMAGE_TAG}"
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to assemble go test flags from the shared options
go_test_flags() {
    local flags="-race"
    if [ "$VERBOSE" = "1" ]; then
        flags="$flags -v"
    fi
    if [ "$SHORT" = "1" ]; then
        flags="$flags -short"
    fi
    echo "$flags"
}

# Function to check if image exists
image_exists() {
    docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" > /dev/null 2>&1
}

# Function to run tests
run_tests() {
    # Keep the pass-through arguments as an array so quoted values such
    # as -run 'TestA|TestB' survive as single arguments.
    local test_args=("$@")

    if ! image_exists; then
        print_warn "Docker image not found. Building..."
        build_image
    fi

    print_info "Running tests in Docker..."
    cd "$PROJECT_ROOT"

    docker run --rm \
        "${DOCKER_MOUNTS[@]}" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        go test $(go_test_flags) "${test_args[@]}" ./pkg/mupdf/
}

# Function to run tests without rebuilding
quick_test() {
    local test_args=("$@")

    if ! image_exists; then
        print_error "Docker image not found. Run '$0 build' first or use '$0 test'"
        exit 1
    fi

    print_info "Running tests (using cached image)..."
    cd "$PROJECT_ROOT"

    docker run --rm \
        "${DOCKER_MOUNTS[@]}" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        go test $(go_test_flags) "${test_args[@]}" ./pkg/mupdf/
}

# Function to generate coverage report
run_coverage() {
    local test_args=("$@")

    if ! image_exists; then
        print_warn "Docker image not found. Building..."
        build_image
    fi

    print_info "Running tests with coverage..."
    cd "$PROJECT_ROOT"
    mkdir -p "$ARTIFACTS_DIR"

    docker run --rm \
        "${DOCKER_MOUNTS[@]}" \
        -v "${ARTIFACTS_DIR}:/artifacts" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        go test $(go_test_flags) -coverprofile=/artifacts/coverage.out "${test_args[@]}" ./pkg/mupdf/

    if [ -f "${ARTIFACTS_DIR}/coverage.out" ]; then
        docker run --rm \
            -v "${ARTIFACTS_DIR}:/artifacts" \
            -w /workspace \
            "${IMAGE_NAME}:${IMAGE_TAG}" \
            go tool cover -func=/artifacts/coverage.out | tail -1
    fi

    if [ -f "${ARTIFACTS_DIR}/coverage.out" ]; then
        print_info "Coverage report generated: .docker-artifacts/coverage.out"
        print_info "To view HTML report, run: go tool cover -html=.docker-artifacts/coverage.out"
    fi
}

# Function to open shell in container
run_shell() {
    if ! image_exists; then
        print_warn "Docker image not found. Building..."
        build_image
    fi

    print_info "Opening shell in Docker container..."
    cd "$PROJECT_ROOT"

    docker run --rm -it \
        "${DOCKER_MOUNTS[@]}" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        bash
}

# Function to clean up Docker image
clean_image() {
    print_info "Removing Docker test image..."
    docker rmi "${IMAGE_NAME}:${IMAGE_TAG}" 2>/dev/null || true
    print_info "Cleanup complete"
}

# Main script logic
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    # Parse shared options first (they may appear before or after the command)
    local positional=()
    while [ $# -gt 0 ]; do
        case "$1" in
            -v|--verbose)
                VERBOSE=1
                ;;
            -r|--rebuild)
                REBUILD=1
                ;;
            --no-cache)
                NO_CACHE=1
                ;;
            -s|--short)
                SHORT=1
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            --)
                # Everything after -- is passed through untouched (e.g. go test flags)
                shift
                positional+=("$@")
                break
                ;;
            -*)
                print_error "Unknown option: $1"
                echo ""
                usage
                exit 1
                ;;
            *)
                positional+=("$1")
                ;;
        esac
        shift
    done

    if [ ${#positional[@]} -eq 0 ]; then
        usage
        exit 0
    fi

    local command="${positional[0]}"
    local rest=("${positional[@]:1}")

    # --rebuild forces an image build before commands that use the image
    case "$command" in
        test|quick|coverage|shell)
            if [ "$REBUILD" = "1" ]; then
                build_image
            fi
            ;;
    esac

    case "$command" in
        build)
            build_image
            ;;
        test)
            run_tests "${rest[@]}"
            ;;
        quick)
            quick_test "${rest[@]}"
            ;;
        coverage)
            run_coverage "${rest[@]}"
            ;;
        shell)
            run_shell
            ;;
        clean)
            clean_image
            ;;
        help)
            usage
            ;;
        *)
            print_error "Unknown command: $command"
            echo ""
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
