# Build System Documentation

This document explains how the go-mupdf build system works, particularly for downstream consumers using `go get`.

## Overview

The go-mupdf library wraps the MuPDF C library using CGO. This requires pre-compiled C libraries to be available before the Go package can be built. The build system handles this in several ways depending on how the package is being used.

## Build Methods

### 1. Direct Repository Clone (Development)

When cloning the repository directly:

```bash
git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf
make setup
go build ./pkg/mupdf/
```

The `make setup` command runs `scripts/install.sh` which:
1. Runs `scripts/download-libs.sh` to download pre-built libraries from Bitbucket Downloads (if available for your platform)
2. Falls back to `scripts/setup-mupdf.sh` to build from source if the download fails
3. Source builds use the correct flags: `make libs USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release`

### 2. Using `go get` (Downstream Consumers)

When users install the package with `go get`:

```bash
go get bitbucket.org/lexmata/go-mupdf
```

The `pkg/mupdf/setup.go` file's `init()` function automatically runs before CGO compilation. This function:

1. Checks if `libmupdf.a` and `libmupdf-third.a` already exist
2. If not, downloads MuPDF tarball from GitHub releases (https://github.com/ArtifexSoftware/mupdf/archive/refs/tags/1.27.2.tar.gz)
3. Extracts the tarball to `third_party/mupdf`
4. Builds MuPDF libraries with the correct flags
5. Verifies both `libmupdf.a` and `libmupdf-third.a` were created

**Benefits of this approach:**
- ✅ No git required (only wget/curl or Go's http package)
- ✅ Smaller download (no git history)
- ✅ Faster extraction
- ✅ Specific version pinning
- ✅ Works reliably with `go get`

### 3. Pre-built Library Distribution

For the fastest setup, users can download pre-built libraries:

```bash
wget https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-<version>-<platform>.tar.gz
tar -xzf go-mupdf-<version>-<platform>.tar.gz
# Copy to expected location
```

See [Static Library Distribution Guide](STATIC_LIBRARY_DISTRIBUTION.md) for details.

## Critical Build Requirements

### The `libs` Target

The MuPDF build MUST use the `libs` target, not the default `all` target:

```bash
make libs  # ✅ Correct - builds libmupdf.a and libmupdf-third.a
make       # ❌ Wrong - may not build libmupdf-third.a correctly
```

### Required Build Flags

These flags are critical for proper static linking:

- **`USE_SYSTEM_LIBS=no`**: Ensures all dependencies are statically linked into `libmupdf-third.a`
  - Without this flag, the build may attempt to use system libraries, resulting in an incomplete or missing `libmupdf-third.a`

- **`HAVE_X11=no`**: Disables X11 GUI dependencies (not needed for library usage)

- **`HAVE_GLUT=no`**: Disables GLUT GUI dependencies (not needed for library usage)

- **`build=release`**: Build optimized release version (vs debug)

### Complete Build Command

```bash
make -j$(nproc) \
    USE_SYSTEM_LIBS=no \
    HAVE_X11=no \
    HAVE_GLUT=no \
    build=release \
    libs
```

## The `libmupdf-third.a` Issue

### What is `libmupdf-third.a`?

This library contains all third-party dependencies statically compiled:
- FreeType (font rendering)
- HarfBuzz (text shaping)
- JPEG, PNG (image formats)
- OpenJPEG (JPEG 2000)
- JBIG2Dec (JBIG2 compression)
- Zlib (compression)
- And more...

### Why It's Required

The CGO linker flags in `pkg/mupdf/cgo_flags.go` explicitly link against it:

```go
// #cgo LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third ...
```

Without `libmupdf-third.a`, the Go build fails with:

```
/usr/bin/ld: cannot find -lmupdf-third
```

### Common Causes of Missing `libmupdf-third.a`

1. **Wrong Build Target**: Running `make` instead of `make libs`
2. **Missing Flags**: Not using `USE_SYSTEM_LIBS=no`
3. **Incomplete Build**: Build failed or was interrupted
4. **Wrong MuPDF Version**: Very old versions may not have this target

### Fix Applied

The `build.go` file was updated to:

1. Explicitly use `make libs` target
2. Include all required build flags
3. Verify both libraries are created after build
4. Provide clear error messages if `libmupdf-third.a` is missing

## Build Process Flow

### For `go get` Users

```
User runs: go get bitbucket.org/lexmata/go-mupdf
    │
    ├─> Go downloads source code
    │
    ├─> Go attempts to build pkg/mupdf
    │   │
    │   ├─> pkg/mupdf/setup.go init() runs before CGO compilation
    │   │   │
    │   │   ├─> Check if libraries exist
    │   │   │   └─> If yes, skip to CGO compilation
    │   │   │
    │   │   ├─> Check if third_party/mupdf/Makefile exists
    │   │   │   └─> If not, download tarball from GitHub
    │   │   │       ├─> Download: https://github.com/ArtifexSoftware/mupdf/archive/refs/tags/1.27.2.tar.gz
    │   │   │       ├─> Extract to third_party/
    │   │   │       └─> Rename mupdf-1.27.2 to mupdf
    │   │   │
    │   │   └─> Build MuPDF libraries
    │   │       ├─> Run: make -j<N> USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release libs
    │   │       └─> Verify libmupdf.a and libmupdf-third.a exist
    │   │
    │   └─> CGO compiles with libraries
    │       └─> Links against libmupdf.a and libmupdf-third.a
    │
    └─> Success!
```

### For Development

```
Developer clones repository
    │
    ├─> git clone --recurse-submodules (gets submodule automatically)
    │
    ├─> Run: make setup
    │   │
    │   └─> scripts/install.sh
    │       ├─> Run scripts/download-libs.sh (download pre-built libraries from Bitbucket)
    │       └─> Fall back to scripts/setup-mupdf.sh (build from source) if download fails
    │
    └─> go build ./pkg/mupdf/
```

## Verification

To verify the build system is working correctly:

1. **Check libraries exist**:
```bash
ls -lh third_party/mupdf/build/release/
# Should show:
# libmupdf.a        (typically 50-60 MB)
# libmupdf-third.a  (typically 8-10 MB)
```

2. **Test build**:
```bash
go clean -cache
go build ./pkg/mupdf/
```

3. **Run tests**:
```bash
go test ./pkg/mupdf/ -v
```

## Debugging Build Issues

### Issue: `cannot find -lmupdf-third`

**Cause**: The `libmupdf-third.a` library was not built.

**Solution**:
1. Clean and rebuild:
```bash
cd third_party/mupdf
make clean
make -j$(nproc) USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release libs
cd ../..
```

2. Verify both libraries exist:
```bash
ls -lh third_party/mupdf/build/release/*.a
```

### Issue: Build fails with "git: command not found"

**Cause**: Git is not installed (needed to download MuPDF submodule).

**Solution**: Install git:
```bash
# Ubuntu/Debian
sudo apt-get install git

# macOS
xcode-select --install

# Windows
# Download from https://git-scm.com/
```

### Issue: Build fails with "make: command not found"

**Cause**: Build tools not installed.

**Solution**: Install build tools:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS
xcode-select --install
```

## Build Performance

Typical build times:

- **From Source (First Time)**: 5-10 minutes
  - Highly parallelized with `-j$(nproc)`
  - Depends on CPU cores and speed

- **From Pre-built Libraries**: < 1 minute
  - Download + extraction only
  - No compilation needed

- **Cached (Incremental)**: < 30 seconds
  - Only rebuilds changed files

## CI/CD Integration

The CI/CD pipeline uses a three-tier approach:

1. **Build Once**: MuPDF is built once per pipeline run
2. **Artifact Sharing**: Built libraries are shared across all pipeline steps
3. **Caching**: Libraries are cached, keyed on `.gitmodules` and the `mupdf.lock` pin file

This reduces build time by 50-75% compared to building in every step.

See [CI/CD Optimization Guide](CI_CD_OPTIMIZATION.md) for details.

## References

- **MuPDF Build System**: See `third_party/mupdf/Makefile` and `third_party/mupdf/Makerules`
- **Build Scripts**:
  - `build.go` - Automatic build for `go get`
  - `scripts/setup-mupdf.sh` - Setup script for development
  - `scripts/build-static-libs.sh` - Distribution package builder
  - `scripts/install-prebuilt-libs.sh` - Pre-built library installer
- **CGO Configuration**: `pkg/mupdf/cgo_flags.go`

