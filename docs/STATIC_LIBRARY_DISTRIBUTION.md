# Static Library Distribution

This document describes how to build and distribute static libraries for go-mupdf, allowing users to use pre-compiled MuPDF libraries instead of building from source.

## Overview

The static library distribution includes:

- `libmupdf.a` - MuPDF core library
- `libmupdf-third.a` - All third-party dependencies (statically linked)
- MuPDF header files
- Usage documentation

This allows users to:
- Skip the lengthy MuPDF compilation process
- Use go-mupdf without installing build dependencies
- Deploy applications with pre-built libraries

## Building Distribution Packages

### Quick Start

Build a distribution package for your current platform:

```bash
make dist
```

This will:
1. Build MuPDF with all dependencies statically linked
2. Create a tarball with libraries and headers
3. Generate checksums
4. Save everything to `dist/go-mupdf-<version>-<platform>.tar.gz`

### Manual Build

You can also use the script directly for more control:

```bash
# Build with default settings
./scripts/build-static-libs.sh

# Build with specific version
VERSION=1.1.0 ./scripts/build-static-libs.sh

# Build debug version
BUILD_TYPE=debug ./scripts/build-static-libs.sh

# Only build libraries (don't create tarball)
./scripts/build-static-libs.sh --build-only

# Only create tarball (skip building)
./scripts/build-static-libs.sh --skip-build

# Clean distribution artifacts
./scripts/build-static-libs.sh --clean
# or
make dist-clean
```

## Platform Support

### Supported Platforms

The build script automatically detects and creates packages for:

- **Linux**
  - `linux-amd64` (x86_64)
  - `linux-arm64` (aarch64)
  - `linux-arm` (armv7l)

- **macOS**
  - `darwin-amd64` (Intel)
  - `darwin-arm64` (Apple Silicon)

- **Windows**
  - `windows-amd64` (with MinGW-w64)

### Building for Multiple Platforms

To build for multiple platforms, you'll need to:

1. **Use Cross-Compilation** (advanced)
2. **Use Platform-Specific Machines** (recommended)
3. **Use CI/CD** (automated)

Example multi-platform build script:

```bash
#!/bin/bash
# Build on different machines or Docker containers

platforms=(
  "linux:ubuntu:latest"
  "macos:macos-12"
  "windows:windows-2022"
)

for platform in "${platforms[@]}"; do
  IFS=':' read -r os image <<< "$platform"
  echo "Building for $os..."
  # Use platform-specific build commands
done
```

## Distribution Package Contents

Each distribution tarball contains:

```
go-mupdf-1.1.0-linux-amd64/
├── lib/
│   ├── libmupdf.a           # MuPDF core library (~8-12 MB)
│   └── libmupdf-third.a     # Third-party dependencies (~15-20 MB)
├── include/
│   └── mupdf/               # All MuPDF headers
│       ├── fitz.h
│       ├── pdf.h
│       └── ...
├── docs/
│   ├── LICENSE.MuPDF        # MuPDF AGPL license
│   └── LICENSE              # go-mupdf license
├── VERSION                  # Version information
└── README.md               # Usage instructions
```

### Library Sizes

Typical sizes (may vary by platform):

| Component | Size (Release) | Size (Debug) |
|-----------|---------------|--------------|
| libmupdf.a | 8-12 MB | 25-35 MB |
| libmupdf-third.a | 15-20 MB | 40-60 MB |
| Headers | ~1 MB | ~1 MB |
| **Total (compressed)** | **~10-15 MB** | **~30-40 MB** |

## Using Pre-built Libraries

### For go-mupdf Users

1. Download the distribution package for your platform
2. Extract to your project:

```bash
# Download
wget https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-1.1.0-linux-amd64.tar.gz

# Extract
tar -xzf go-mupdf-1.1.0-linux-amd64.tar.gz

# Install to go-mupdf expected location
cd go-mupdf
mkdir -p third_party/mupdf/build/release third_party/mupdf/include
cp ../go-mupdf-1.1.0-linux-amd64/lib/*.a third_party/mupdf/build/release/
cp -r ../go-mupdf-1.1.0-linux-amd64/include/mupdf third_party/mupdf/include/

# Build your application (no MuPDF compilation needed!)
go build
```

### For Custom CGO Projects

If you're using MuPDF directly in your own CGO project:

```go
package main

/*
#cgo CFLAGS: -I/path/to/go-mupdf-1.1.0-linux-amd64/include
#cgo LDFLAGS: -L/path/to/go-mupdf-1.1.0-linux-amd64/lib -lmupdf -lmupdf-third -lm
#include <mupdf/fitz.h>
*/
import "C"

func main() {
    // Your MuPDF code here
}
```

## CI/CD Integration

### Bitbucket Pipelines

Add distribution building to your pipeline:

```yaml
pipelines:
  tags:
    'v*':
      - step:
          name: Build Distribution Packages
          image: golang:1.23
          script:
            - apt-get update && apt-get install -y build-essential
            - git submodule update --init --recursive
            - make dist
          artifacts:
            - dist/*.tar.gz
            - dist/*.sha256
```

### GitHub Actions

```yaml
name: Build Distribution
on:
  release:
    types: [created]

jobs:
  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          submodules: recursive
      - name: Build distribution
        run: make dist
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: linux-amd64
          path: dist/*.tar.gz
```

## Hosting Distribution Packages

### Option 1: Bitbucket Downloads

Upload to Bitbucket's Downloads section:

1. Go to your repository on Bitbucket
2. Navigate to Downloads in the sidebar
3. Upload the `.tar.gz` and `.sha256` files
4. Users can download directly from Bitbucket

### Option 2: GitHub Releases

Attach to release tags:

```bash
# Using GitHub CLI
gh release create v1.1.0 \
  dist/go-mupdf-1.1.0-*.tar.gz \
  dist/go-mupdf-1.1.0-*.sha256 \
  --title "Release v1.1.0" \
  --notes "Release notes here"
```

### Option 3: CDN / Object Storage

Upload to S3, Google Cloud Storage, etc.:

```bash
# AWS S3 example
aws s3 cp dist/ s3://your-bucket/go-mupdf/releases/v1.1.0/ \
  --recursive \
  --include "*.tar.gz" \
  --include "*.sha256"
```

## Verification

### Verify Downloads

Users should verify checksums before using:

```bash
# Download checksum
wget https://example.com/go-mupdf-1.1.0-linux-amd64.tar.gz.sha256

# Verify
sha256sum -c go-mupdf-1.1.0-linux-amd64.tar.gz.sha256
```

### Verify Library Contents

Check that libraries contain expected symbols:

```bash
# Extract
tar -xzf go-mupdf-1.1.0-linux-amd64.tar.gz
cd go-mupdf-1.1.0-linux-amd64

# List symbols in libmupdf.a
nm lib/libmupdf.a | grep fz_new_context

# Check dependencies in libmupdf-third.a
nm lib/libmupdf-third.a | grep -E "(FT_Init|jpeg_start|png_create)"
```

## Licensing Considerations

### Important Notes

1. **MuPDF is AGPL-licensed**: Users must comply with AGPL v3 terms
2. **Commercial licenses available**: From Artifex Software for non-AGPL use
3. **Attribution required**: Include MuPDF license with distributions
4. **Source availability**: AGPL requires source code availability

### License Files Included

Each distribution package includes:

- `docs/LICENSE` - go-mupdf license
- `docs/LICENSE.MuPDF` - MuPDF AGPL license
- `README.md` - License information and links

### For Commercial Use

Users requiring commercial licenses should:

1. Contact Artifex Software: https://mupdf.com/licensing/
2. Obtain appropriate commercial license
3. Follow Artifex's terms for distribution

## Troubleshooting

### Build Fails on macOS

**Issue**: Missing command line tools

```bash
# Install Xcode command line tools
xcode-select --install
```

### Build Fails on Linux

**Issue**: Missing build dependencies

```bash
# Install build essentials
sudo apt-get update
sudo apt-get install -y build-essential pkg-config
```

### Distribution Package Too Large

**Issue**: Debug symbols included

```bash
# Build release version (default)
BUILD_TYPE=release make dist

# Strip debug symbols (if needed)
strip -S lib/*.a
```

### Missing Symbols at Link Time

**Issue**: Library built for wrong architecture

```bash
# Check library architecture
file lib/libmupdf.a

# Verify it matches your system
uname -m
```

### Permission Denied on Script

**Issue**: Script not executable

```bash
chmod +x scripts/build-static-libs.sh
```

## Best Practices

### For Maintainers

1. **Build for all supported platforms** on each release
2. **Test each distribution package** before publishing
3. **Generate and publish checksums** for verification
4. **Update version numbers** consistently
5. **Document platform-specific notes** in release notes

### For Users

1. **Always verify checksums** after downloading
2. **Check library architecture** matches your system
3. **Review license terms** before distribution
4. **Keep libraries updated** with security patches
5. **Report issues** to the repository

## Advanced Topics

### Custom Build Flags

Modify `scripts/build-static-libs.sh` to add custom flags:

```bash
# Example: Enable/disable specific features
make -j"$nproc_count" \
    USE_SYSTEM_LIBS=no \
    HAVE_X11=no \
    HAVE_GLUT=no \
    HAVE_PTHREAD=yes \
    build=$BUILD_TYPE \
    libs
```

### Optimizing Library Size

Techniques to reduce size:

1. **Strip debug symbols**: `strip -S lib/*.a`
2. **Use release build**: `BUILD_TYPE=release`
3. **Disable unused features**: Edit MuPDF Makefile
4. **Link-time optimization**: Add `-flto` to CFLAGS

### Multi-Architecture Builds

Build for multiple architectures on same platform:

```bash
# Example: Build for both amd64 and arm64 on macOS
# (requires Rosetta 2 or cross-compilation setup)

# For Intel
arch -x86_64 make dist

# For Apple Silicon  
arch -arm64 make dist
```

## Support

For issues with static library distribution:

- **Repository**: https://bitbucket.org/lexmata/go-mupdf
- **Issues**: https://bitbucket.org/lexmata/go-mupdf/issues
- **Documentation**: https://bitbucket.org/lexmata/go-mupdf/src/main/docs/

For MuPDF-specific questions:

- **Website**: https://mupdf.com/
- **Documentation**: https://mupdf.readthedocs.io/
- **Forum**: https://discord.gg/TZGKzzMf (MuPDF Discord)

