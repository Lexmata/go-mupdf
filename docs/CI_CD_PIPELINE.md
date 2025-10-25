# CI/CD Pipeline Documentation

## Overview

This document describes the Bitbucket Pipelines CI/CD setup for the Go MuPDF Wrapper project. The pipeline is designed to handle the complexities of building a Go project with native C dependencies (MuPDF library).

## Pipeline Structure

### Default Pipeline
Runs on all branches except `main` and for pull requests:
- Builds MuPDF native library
- Runs Go tests with race detection
- Generates coverage reports
- Caches build artifacts for faster subsequent runs

### Main Branch Pipeline
Enhanced pipeline for the main branch:
- All default pipeline steps
- Code quality checks (formatting, vet, staticcheck)
- Comprehensive test suite including benchmarks
- Extended coverage analysis

### Tag Pipeline (v*)
Release pipeline for version tags:
- All main branch checks
- Multi-platform build preparation
- Release artifact generation
- Automated release preparation

### Pull Request Pipeline
Focused on validation:
- Quick build and test cycle
- Code formatting verification
- Race condition detection
- Coverage analysis for changed code

## Key Features

### 1. Native Library Handling
- **Caching**: MuPDF library builds are cached to reduce build time
- **Dependencies**: Automated installation of system dependencies
- **Verification**: Build verification steps ensure library integrity

### 2. Comprehensive Testing
- **Race Detection**: All tests run with `-race` flag
- **Coverage**: Detailed coverage reports with HTML output
- **Categorized Tests**: Support for different test categories (unit, integration, memory, stress)
- **Benchmarks**: Performance benchmarking on main branch

### 3. Code Quality
- **Formatting**: Enforced code formatting with `gofmt`
- **Static Analysis**: `go vet` and optional `staticcheck`
- **Vulnerability Scanning**: Optional `govulncheck` integration
- **Dependency Validation**: `go mod verify` and outdated dependency checks

### 4. Performance Optimization
- **Parallel Builds**: Uses all available CPU cores
- **Caching Strategy**: Go modules and MuPDF builds are cached
- **Selective Testing**: Different test suites for different pipeline types

## Configuration Files

### bitbucket-pipelines.yml
Main pipeline configuration with four distinct workflows:

```yaml
# Key sections:
- Default: Basic build and test
- Main branch: Enhanced quality checks
- Tags: Release preparation
- Pull requests: Fast validation
```

### scripts/ci-setup.sh
Environment setup script that:
- Installs system dependencies
- Configures Go environment
- Builds MuPDF library
- Verifies build environment

### scripts/test-runner.sh
Comprehensive test runner with options for:
- Coverage analysis
- Benchmark execution
- Code quality checks
- Categorized test execution

## Environment Requirements

### System Dependencies
```bash
# Required packages:
- build-essential, make, gcc, g++
- libc6-dev, pkg-config
- libx11-dev, libxext-dev, libxrandr-dev
- libgl1-mesa-dev (for OpenGL support)
```

### Go Configuration
```bash
GO111MODULE=on
CGO_ENABLED=1
GOOS=linux
GOARCH=amd64
```

## Usage

### Running Locally
```bash
# Set up environment
./scripts/ci-setup.sh

# Run tests
./scripts/test-runner.sh

# Run with benchmarks
./scripts/test-runner.sh --benchmarks

# Run categorized tests
./scripts/test-runner.sh --categorized

# Skip quality checks
./scripts/test-runner.sh --no-quality
```

### Pipeline Triggers

| Trigger | Pipeline | Purpose |
|---------|----------|---------|
| Push to feature branch | Default | Basic validation |
| Push to main | Main Branch | Full quality checks |
| Pull request | PR Pipeline | Fast validation |
| Tag (v*) | Release | Release preparation |

## Artifacts

### Generated Artifacts
- `coverage.html`: HTML coverage report
- `coverage.out`: Coverage data file
- `benchmark.out`: Benchmark results (main branch)
- `releases/`: Release binaries (tags only)

### Artifact Retention
- Coverage reports: Available for download
- Build logs: Retained per Bitbucket settings
- Release artifacts: Permanent for tagged releases

## Caching Strategy

### Go Modules Cache
- **Key**: Go version + go.sum hash
- **Path**: `$GOPATH/pkg/mod`
- **Benefits**: Faster dependency resolution

### MuPDF Build Cache
- **Key**: MuPDF source hash
- **Path**: `third_party/mupdf/build/`
- **Benefits**: Avoids expensive native compilation

## Performance Metrics

### Typical Build Times
- **Cold build**: 5-8 minutes (includes MuPDF compilation)
- **Warm build**: 2-3 minutes (with caches)
- **PR validation**: 3-4 minutes
- **Release build**: 8-12 minutes

### Resource Usage
- **Memory**: 2GB allocated
- **CPU**: All available cores for compilation
- **Storage**: ~500MB for caches

## Troubleshooting

### Common Issues

#### MuPDF Build Failures
```bash
# Symptoms: Library not found errors
# Solution: Clear cache and rebuild
rm -rf third_party/mupdf/build/
./scripts/ci-setup.sh
```

#### CGO Compilation Errors
```bash
# Symptoms: C compiler errors
# Solution: Verify system dependencies
apt-get install build-essential gcc g++
```

#### Test Failures
```bash
# Symptoms: Specific test failures
# Solution: Run categorized tests
./scripts/test-runner.sh --categorized
```

### Debug Mode
Enable debug output by setting:
```bash
export MUPDF_DEBUG=1
export CGO_CFLAGS="-g -O0"
```

## Security Considerations

### Dependency Security
- Regular vulnerability scanning with `govulncheck`
- Automated dependency updates
- Secure handling of build artifacts

### Build Security
- No secrets in pipeline configuration
- Minimal required permissions
- Isolated build environments

## Maintenance

### Regular Tasks
1. **Monthly**: Review and update dependencies
2. **Quarterly**: Evaluate pipeline performance
3. **Per Release**: Update documentation
4. **As Needed**: Optimize cache strategies

### Monitoring
- Build success rates
- Average build times
- Cache hit rates
- Test coverage trends

## Future Enhancements

### Planned Improvements
1. **Multi-platform Builds**: Cross-compilation support
2. **Parallel Testing**: Matrix builds for different Go versions
3. **Security Scanning**: Enhanced vulnerability detection
4. **Performance Tracking**: Historical benchmark comparisons

### Integration Opportunities
1. **Code Coverage Services**: Codecov integration
2. **Quality Gates**: SonarQube integration
3. **Release Automation**: Automated changelog generation
4. **Notification Systems**: Slack/Teams integration

## Support

For pipeline issues:
1. Check build logs in Bitbucket
2. Review this documentation
3. Run scripts locally for debugging
4. Contact the development team

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2024 | Initial pipeline setup |

---

*This documentation is maintained alongside the pipeline configuration and should be updated with any changes.*