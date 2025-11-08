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
    coverage    Run tests and generate coverage report
    quick       Run tests without rebuilding (uses cached image)
    help        Show this help message

Options:
    -v, --verbose    Show verbose output
    -r, --rebuild    Force rebuild of Docker image
    -s, --short      Run tests in short mode
    --no-cache       Build Docker image without cache

Examples:
    $0 build                    # Build the test image
    $0 test                     # Build and run all tests
    $0 test --short             # Run quick tests only
    $0 quick                    # Run tests without rebuilding
    $0 coverage                 # Generate coverage report
    $0 shell                    # Open interactive shell

EOF
}

# Function to build Docker image
build_image() {
    local no_cache=""
    if [ "$1" = "--no-cache" ]; then
        no_cache="--no-cache"
    fi

    print_info "Building Docker test image..."
    cd "$PROJECT_ROOT"

    docker build $no_cache -f Dockerfile.test -t "${IMAGE_NAME}:${IMAGE_TAG}" .

    if [ $? -eq 0 ]; then
        print_info "Docker image built successfully: ${IMAGE_NAME}:${IMAGE_TAG}"
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to check if image exists
image_exists() {
    docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" > /dev/null 2>&1
}

# Function to run tests
run_tests() {
    local test_args=""
    local short_mode=""

    while [ $# -gt 0 ]; do
        case "$1" in
            -s|--short)
                short_mode="-short"
                shift
                ;;
            *)
                test_args="$test_args $1"
                shift
                ;;
        esac
    done

    if ! image_exists; then
        print_warn "Docker image not found. Building..."
        build_image
    fi

    print_info "Running tests in Docker..."
    cd "$PROJECT_ROOT"

    docker run --rm \
        -v "$(pwd):/workspace" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        go test -v -race $short_mode $test_args ./pkg/mupdf/
}

# Function to run tests without rebuilding
quick_test() {
    if ! image_exists; then
        print_error "Docker image not found. Run '$0 build' first or use '$0 test'"
        exit 1
    fi

    print_info "Running tests (using cached image)..."
    cd "$PROJECT_ROOT"

    docker run --rm \
        -v "$(pwd):/workspace" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        go test -v -race ./pkg/mupdf/
}

# Function to generate coverage report
run_coverage() {
    if ! image_exists; then
        print_warn "Docker image not found. Building..."
        build_image
    fi

    print_info "Running tests with coverage..."
    cd "$PROJECT_ROOT"

    docker run --rm \
        -v "$(pwd):/workspace" \
        -w /workspace \
        "${IMAGE_NAME}:${IMAGE_TAG}" \
        bash -c "go test -v -race -coverprofile=coverage.out ./pkg/mupdf/ && go tool cover -func=coverage.out | tail -1"

    if [ -f "coverage.out" ]; then
        print_info "Coverage report generated: coverage.out"
        print_info "To view HTML report, run: go tool cover -html=coverage.out"
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
        -v "$(pwd):/workspace" \
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

    local command="$1"
    shift

    case "$command" in
        build)
            build_image "$@"
            ;;
        test)
            local rebuild=""
            while [ $# -gt 0 ]; do
                case "$1" in
                    -r|--rebuild)
                        rebuild="yes"
                        shift
                        ;;
                    *)
                        break
                        ;;
                esac
            done

            if [ "$rebuild" = "yes" ]; then
                build_image
            fi

            run_tests "$@"
            ;;
        quick)
            quick_test
            ;;
        coverage)
            run_coverage
            ;;
        shell)
            run_shell
            ;;
        clean)
            clean_image
            ;;
        help|--help|-h)
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
