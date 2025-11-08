# Docker Testing Guide

This guide explains how to use Docker for local testing that simulates the CI/CD environment.

## Overview

The Docker testing setup allows you to:
- Test your code in an environment identical to CI/CD
- Quickly iterate without polluting your local environment
- Debug CI/CD failures locally
- Ensure consistent test results across different machines

## Quick Start

### Run Tests in Docker

The simplest way to run tests:

```bash
make docker-test
```

This command will:
1. Build the Docker image if it doesn't exist
2. Run all tests with race detection
3. Display the results

### Other Common Commands

```bash
# Build the Docker test image
make docker-build

# Run tests without rebuilding (faster)
make docker-quick

# Generate coverage report
make docker-coverage

# Open an interactive shell in the container
make docker-shell

# Clean up Docker image
make docker-clean
```

## Using the Docker Test Script Directly

The `scripts/docker-test.sh` script provides more options:

### Basic Commands

```bash
# Build the test image
./scripts/docker-test.sh build

# Run all tests
./scripts/docker-test.sh test

# Run tests in short mode (skips long-running tests)
./scripts/docker-test.sh test --short

# Force rebuild before testing
./scripts/docker-test.sh test --rebuild

# Run tests without rebuilding
./scripts/docker-test.sh quick

# Generate coverage report
./scripts/docker-test.sh coverage

# Open interactive shell
./scripts/docker-test.sh shell

# Clean up
./scripts/docker-test.sh clean
```

### Advanced Usage

```bash
# Build with no cache (clean build)
./scripts/docker-test.sh build --no-cache

# Run specific tests
./scripts/docker-test.sh test -run TestMergePDFs

# Run tests with verbose output
./scripts/docker-test.sh test -v

# Combine options
./scripts/docker-test.sh test --rebuild --short
```

## Docker Test Workflow

### Initial Setup

1. Build the Docker image:
   ```bash
   make docker-build
   ```

   This may take a few minutes the first time as it:
   - Downloads the Go 1.24 base image
   - Installs build dependencies
   - Initializes MuPDF submodule
   - Builds MuPDF from source
   - Downloads Go module dependencies

### Development Cycle

During development, use this workflow:

1. **Make code changes** in your local environment

2. **Quick test** without rebuilding:
   ```bash
   make docker-quick
   ```

   This mounts your local code into the container and runs tests.

3. **If you change dependencies** (go.mod), rebuild:
   ```bash
   make docker-test --rebuild
   ```

4. **Debug failures** interactively:
   ```bash
   make docker-shell
   # Inside the container:
   go test -v ./pkg/mupdf/ -run TestFailingTest
   ```

### Pre-Push Validation

Before pushing code, validate it matches CI/CD:

```bash
# Run full test suite with coverage
make docker-coverage

# Verify coverage is above threshold (80%)
# Check the output for the coverage percentage
```

## Docker Image Details

### Base Image

- **Image**: `golang:1.24`
- **OS**: Debian (latest stable)

### Installed Dependencies

- Build essentials (gcc, make, etc.)
- pkg-config
- MuPDF build dependencies:
  - libfreetype6-dev
  - libjpeg-dev
  - libpng-dev
  - zlib1g-dev
  - libjbig2dec-dev
  - libopenjp2-7-dev
  - libharfbuzz-dev
- unzip (for MuPDF build)
- git (for submodule operations)

### Environment Variables

```bash
GO111MODULE=on
CGO_ENABLED=1
GOOS=linux
GOARCH=amd64
GOTOOLCHAIN=auto
```

## Troubleshooting

### Image Build Failures

If the Docker build fails:

1. **Check Docker daemon**: Ensure Docker is running
   ```bash
   docker info
   ```

2. **Clean build**: Remove cache and rebuild
   ```bash
   ./scripts/docker-test.sh build --no-cache
   ```

3. **Check disk space**: Docker needs adequate space
   ```bash
   docker system df
   ```

### Test Failures That Pass Locally

If tests fail in Docker but pass locally:

1. **Check environment differences**:
   - Different Go versions?
   - Different MuPDF version?
   - Missing dependencies?

2. **Debug interactively**:
   ```bash
   make docker-shell
   # Run tests manually to see detailed output
   go test -v ./pkg/mupdf/ -run TestFailingTest
   ```

3. **Compare environments**:
   ```bash
   # Local
   go version

   # Docker
   make docker-shell
   go version
   ```

### Slow Build Times

Docker builds cache layers for efficiency, but sometimes you need a clean build:

1. **First build is always slow**: MuPDF compilation takes time (~2-5 minutes)

2. **Subsequent builds are fast**: Docker caches layers

3. **Force clean build** if needed:
   ```bash
   make docker-clean
   make docker-build --no-cache
   ```

### Container Keeps Running

If a container doesn't exit:

```bash
# List running containers
docker ps

# Stop a container
docker stop <container-id>

# Remove all stopped containers
docker container prune
```

## CI/CD Simulation

The Docker setup exactly matches the CI/CD environment:

### Bitbucket Pipelines

The Dockerfile.test uses the same:
- Base image: `golang:1.24`
- Dependencies installation
- Environment variables
- Build commands
- Test commands

### Local Testing = CI/CD Testing

Running `make docker-test` locally gives you the same results as Bitbucket Pipelines will produce.

## Performance Tips

1. **Use `docker-quick` for rapid iteration**: Skips image rebuild

2. **Build once, test many**: The image persists between runs

3. **Mount local code**: Changes are reflected immediately without rebuild

4. **Clean up periodically**:
   ```bash
   # Remove old images
   docker image prune

   # Remove test image
   make docker-clean
   ```

## Integration with CI/CD

The Docker setup is designed to match CI/CD exactly:

### Dockerfile.test vs bitbucket-pipelines.yml

Both:
- Use `golang:1.24` base image
- Install identical dependencies
- Set the same environment variables
- Build MuPDF from source
- Run the same test commands

### Debugging CI/CD Failures

When CI/CD fails:

1. **Reproduce locally**:
   ```bash
   make docker-test
   ```

2. **If it passes locally**, the issue is environmental:
   - Check CI/CD logs for differences
   - Verify git submodule initialization
   - Check for missing dependencies

3. **If it fails locally**, debug interactively:
   ```bash
   make docker-shell
   # Investigate inside the container
   ```

## Best Practices

1. **Always test before pushing**:
   ```bash
   make docker-coverage
   ```

2. **Keep Docker image updated**:
   ```bash
   # After changing dependencies
   make docker-clean
   make docker-build
   ```

3. **Use short mode for quick checks**:
   ```bash
   ./scripts/docker-test.sh test --short
   ```

4. **Generate coverage reports regularly**:
   ```bash
   make docker-coverage
   ```

5. **Clean up old images**:
   ```bash
   docker image prune -a
   ```

## Examples

### Example 1: Quick Development Cycle

```bash
# Initial build
make docker-build

# Make changes to code
vim pkg/mupdf/pdfcpu.go

# Quick test
make docker-quick

# More changes
vim pkg/mupdf/pdfcpu_test.go

# Quick test again
make docker-quick
```

### Example 2: Debugging Test Failure

```bash
# Test fails
make docker-test
# FAIL: TestMergePDFs

# Open shell to debug
make docker-shell

# Inside container
go test -v ./pkg/mupdf/ -run TestMergePDFs
# See detailed output

# Test with more verbosity
go test -v -race ./pkg/mupdf/ -run TestMergePDFs

# Exit container
exit

# Fix the issue locally, then test again
make docker-quick
```

### Example 3: Pre-Release Validation

```bash
# Full test suite with coverage
make docker-coverage

# Check output shows >80% coverage
# If passed, ready to push

# Clean up
make docker-clean
```

## Summary

The Docker testing setup provides:
- ✅ Consistent environment matching CI/CD
- ✅ Fast iteration with cached builds
- ✅ Easy debugging with interactive shells
- ✅ Coverage reporting
- ✅ Integration with Makefile

Use `make docker-test` as your go-to validation before pushing code!

