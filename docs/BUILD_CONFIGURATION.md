# Build Configuration Guide

This document explains how the build system handles MuPDF library dependencies and how to configure builds for different environments.

## Overview

The Go MuPDF wrapper supports two build modes:

1. **Source-Built MuPDF** (default) - Builds MuPDF from source in `third_party/mupdf`
2. **System MuPDF** - Uses system-installed MuPDF libraries (via `libmupdf-dev` package)

## Build Modes

### Source-Built MuPDF (Default)

This is the recommended approach for:
- Docker containers
- Local development
- Consistent builds across different Ubuntu/Debian versions
- When you need a specific MuPDF version

**How it works:**
- MuPDF is built from source in `third_party/mupdf`
- CGO flags point to: `${SRCDIR}/../../third_party/mupdf/include` (headers)
- Libraries are linked from: `${SRCDIR}/../../third_party/mupdf/build/release`

**Building:**
```bash
# Build MuPDF from source first
cd third_party/mupdf
make -j$(nproc) libs
cd ../..

# Build Go wrapper (uses source-built MuPDF by default)
go build ./pkg/mupdf/
```

### System MuPDF

This mode is used in CI/CD environments that install `libmupdf-dev`:
- Bitbucket Pipelines
- GitHub Actions with system packages
- Environments with pre-installed MuPDF

**How it works:**
- Uses system-installed MuPDF libraries
- CGO flags point to: `/usr/include` (headers)
- Libraries are linked via system paths

**Building:**
```bash
# Install system MuPDF (Ubuntu/Debian)
sudo apt-get install libmupdf-dev

# Build with system_mupdf tag
go build -tags system_mupdf ./pkg/mupdf/
```

## Build Tags

The build system uses Go build tags to switch between modes:

- **Default (no tag)**: Uses source-built MuPDF
- **`system_mupdf` tag**: Uses system-installed MuPDF

### Using Build Tags

```bash
# Source-built (default)
go build ./pkg/mupdf/

# System libraries
go build -tags system_mupdf ./pkg/mupdf/

# Run tests with system libraries
go test -tags system_mupdf ./pkg/mupdf/
```

## Docker Configuration

The Dockerfile is configured to use source-built MuPDF by default:

```dockerfile
# MuPDF is built from source during image creation
RUN cd third_party/mupdf && make -j$(nproc) libs
```

This ensures:
- Consistent MuPDF version regardless of Ubuntu package availability
- Same build across different Ubuntu versions
- No dependency on system package versions

## CI/CD Configuration

### Bitbucket Pipelines

The pipeline uses system MuPDF libraries:

```yaml
script:
  - apt-get install -y libmupdf-dev
  - go test -tags system_mupdf ./pkg/mupdf/
```

### GitHub Actions

Example configuration:

```yaml
- name: Install MuPDF
  run: |
    sudo apt-get update
    sudo apt-get install -y libmupdf-dev

- name: Run tests
  run: go test -tags system_mupdf ./pkg/mupdf/
```

## Troubleshooting

### Problem: "fatal error: 'mupdf/fitz.h' file not found"

**Solution for source-built mode:**
```bash
# Ensure MuPDF is built
cd third_party/mupdf
make -j$(nproc) libs
cd ../..

# Verify headers exist
ls third_party/mupdf/include/mupdf/fitz.h
```

**Solution for system mode:**
```bash
# Install system package
sudo apt-get install libmupdf-dev

# Verify headers exist
ls /usr/include/mupdf/fitz.h
```

### Problem: "undefined reference to 'pdf_*'"

**Solution for source-built mode:**
```bash
# Verify libraries are built
ls third_party/mupdf/build/release/libmupdf.a
ls third_party/mupdf/build/release/libmupdf-third.a

# Rebuild if missing
cd third_party/mupdf
make clean
make -j$(nproc) libs
```

**Solution for system mode:**
```bash
# Verify system libraries exist
ldconfig -p | grep mupdf

# Reinstall if missing
sudo apt-get install --reinstall libmupdf-dev
```

### Problem: Different MuPDF versions in Docker vs CI/CD

**Solution:** Use source-built MuPDF in both environments:

1. Update CI/CD to build from source:
```yaml
script:
  - git submodule update --init --recursive
  - cd third_party/mupdf && make -j$(nproc) libs && cd ../..
  - go test ./pkg/mupdf/
```

2. Or use system MuPDF in Docker:
```dockerfile
RUN apt-get install -y libmupdf-dev
ENV BUILD_TAGS=system_mupdf
```

## Environment Variables

You can control the build mode via environment variables in some cases:

```bash
# Force source-built mode (default)
unset BUILD_TAGS

# Force system mode
export BUILD_TAGS=system_mupdf
go build -tags $BUILD_TAGS ./pkg/mupdf/
```

## Best Practices

1. **Docker/Containers**: Always use source-built MuPDF for consistency
2. **CI/CD**: Use system MuPDF if available, otherwise build from source
3. **Local Development**: Use source-built MuPDF to match Docker environment
4. **Production**: Use source-built MuPDF for reproducible builds

## Migration Guide

### Migrating from System to Source-Built

1. Remove system package dependencies from CI/CD
2. Add MuPDF build step to CI/CD
3. Update Dockerfile to build from source (already done)
4. Remove `-tags system_mupdf` from build commands

### Migrating from Source-Built to System

1. Install system packages: `apt-get install libmupdf-dev`
2. Add `-tags system_mupdf` to all build commands
3. Update CI/CD to install system packages
4. Remove MuPDF build steps from CI/CD

## Verification

Check which mode is being used:

```bash
# Check CGO flags (source-built)
go build -n ./pkg/mupdf/ 2>&1 | grep CFLAGS

# Check CGO flags (system)
go build -tags system_mupdf -n ./pkg/mupdf/ 2>&1 | grep CFLAGS
```

The output should show:
- Source-built: `-I.../third_party/mupdf/include`
- System: `-I/usr/include`

