#!/bin/bash

# Test Runner Script for Go MuPDF Wrapper
# This script runs comprehensive tests for the project

set -e
set -o pipefail

echo "=== Go MuPDF Test Runner ==="

# Configuration
COVERAGE_FILE="coverage.out"
COVERAGE_HTML="coverage.html"
BENCHMARK_FILE="benchmark.out"

# Function to run tests with coverage
run_tests_with_coverage() {
    echo "Running tests with coverage..."
    
    # Run all tests with race detection and coverage
    if go test -v -race -coverprofile="$COVERAGE_FILE" ./...; then
        echo "✅ All tests passed!"
    else
        echo "❌ Some tests failed!"
        exit 1
    fi
    
    # Generate coverage report
    if [ -f "$COVERAGE_FILE" ]; then
        echo "Generating coverage report..."
        go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"
        go tool cover -func="$COVERAGE_FILE"
        
        # Extract coverage percentage
        COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}')
        echo "Total coverage: $COVERAGE"
    fi
}

# Function to run benchmarks
run_benchmarks() {
    echo "Running benchmarks..."
    
    # Run benchmarks for the main package
    if go test -v -bench=. -benchmem ./pkg/mupdf/ | tee "$BENCHMARK_FILE"; then
        echo "✅ Benchmarks completed successfully!"
    else
        echo "⚠️  Some benchmarks failed or encountered issues"
    fi
}

# Function to run specific test categories
run_test_categories() {
    echo "Running categorized tests..."
    
    # Run unit tests
    echo "📋 Running unit tests..."
    go test -v -short ./pkg/mupdf/
    
    # Run integration tests (non-short)
    echo "🔗 Running integration tests..."
    go test -v -run="Integration" ./pkg/mupdf/
    
    # Run memory tests
    echo "🧠 Running memory tests..."
    go test -v -run="Memory" ./pkg/mupdf/
    
    # Run concurrent tests
    echo "🔄 Running concurrent tests..."
    go test -v -run="Concurrent" ./pkg/mupdf/
    
    # Run stress tests
    echo "💪 Running stress tests..."
    go test -v -run="Stress" ./pkg/mupdf/
}

# Function to check code quality
check_code_quality() {
    echo "Checking code quality..."
    
    # Check code formatting
    echo "📝 Checking code formatting..."
    UNFORMATTED=$(gofmt -l .)
    if [ -n "$UNFORMATTED" ]; then
        echo "❌ The following files are not properly formatted:"
        echo "$UNFORMATTED"
        exit 1
    else
        echo "✅ All files are properly formatted"
    fi
    
    # Run go vet
    echo "🔍 Running go vet..."
    if go vet ./...; then
        echo "✅ go vet passed"
    else
        echo "❌ go vet found issues"
        exit 1
    fi
    
    # Run staticcheck if available
    if command -v staticcheck >/dev/null 2>&1; then
        echo "🔬 Running staticcheck..."
        if staticcheck ./...; then
            echo "✅ staticcheck passed"
        else
            echo "❌ staticcheck found issues"
            exit 1
        fi
    else
        echo "⚠️  staticcheck not available, skipping"
    fi
    
    # Check for TODO comments
    echo "📝 Checking for TODO comments..."
    TODO_COUNT=$(grep -r "TODO" --include="*.go" . | wc -l || true)
    if [ "$TODO_COUNT" -gt 0 ]; then
        echo "📋 Found $TODO_COUNT TODO comments:"
        grep -rn "TODO" --include="*.go" .
    else
        echo "✅ No TODO comments found"
    fi
}

# Function to validate dependencies
validate_dependencies() {
    echo "Validating dependencies..."
    
    # Check for vulnerabilities
    if command -v govulncheck >/dev/null 2>&1; then
        echo "🔒 Running vulnerability check..."
        govulncheck ./...
    else
        echo "⚠️  govulncheck not available, skipping vulnerability check"
    fi
    
    # Verify go.mod
    echo "📦 Verifying go.mod..."
    if go mod verify; then
        echo "✅ go.mod verified"
    else
        echo "❌ go.mod verification failed"
        exit 1
    fi
    
    # Check for unused dependencies
    if command -v go-mod-outdated >/dev/null 2>&1; then
        echo "📦 Checking for outdated dependencies..."
        go list -u -m all | go-mod-outdated
    fi
}

# Function to clean up test artifacts
cleanup() {
    echo "Cleaning up test artifacts..."
    
    # Remove temporary test files
    find . -name "tmp_*" -delete 2>/dev/null || true
    find . -name "*.tmp" -delete 2>/dev/null || true
    
    echo "✅ Cleanup completed"
}

# Main test execution
main() {
    echo "Starting comprehensive test suite..."
    
    # Parse command line arguments
    RUN_BENCHMARKS=false
    RUN_QUALITY_CHECKS=true
    RUN_CATEGORIZED=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --benchmarks)
                RUN_BENCHMARKS=true
                shift
                ;;
            --no-quality)
                RUN_QUALITY_CHECKS=false
                shift
                ;;
            --categorized)
                RUN_CATEGORIZED=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                echo "Usage: $0 [--benchmarks] [--no-quality] [--categorized]"
                exit 1
                ;;
        esac
    done
    
    # Run tests
    run_tests_with_coverage
    
    if [ "$RUN_CATEGORIZED" = true ]; then
        run_test_categories
    fi
    
    if [ "$RUN_BENCHMARKS" = true ]; then
        run_benchmarks
    fi
    
    if [ "$RUN_QUALITY_CHECKS" = true ]; then
        check_code_quality
    fi
    
    validate_dependencies
    cleanup
    
    echo "=== Test Suite Complete ==="
    echo "📊 Coverage report: $COVERAGE_HTML"
    if [ "$RUN_BENCHMARKS" = true ]; then
        echo "⚡ Benchmark results: $BENCHMARK_FILE"
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi