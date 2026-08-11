# Installation Guide

This guide covers different installation methods for the Go MuPDF Wrapper.

## Quick Start ⚡

The **easiest and fastest** way to get started:

```bash
# Clone the repository
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Automatic setup (downloads pre-built libraries or builds from source)
make setup

# Build and test
make build
make test
```

The `make setup` command automatically:
1. ✅ Downloads pre-built MuPDF libraries from GitHub Releases (if available for your platform)
2. ✅ Falls back to building from source if pre-built unavailable
3. ✅ Caches libraries for faster subsequent builds

**No manual submodule initialization required!**

## Installation Methods

### Method 1: Automatic Setup (Recommended)

This method automatically downloads pre-built libraries when possible:

```bash
# Clone the repository
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Automatic library setup
make setup

# Build the Go wrapper
make build
```

**What happens during setup:**
- Detects your platform (linux-amd64, darwin-arm64, etc.)
- Downloads pre-built MuPDF libraries from GitHub Releases
- Extracts libraries to `third_party/mupdf/build/release`
- If pre-built unavailable, automatically builds from source

**Advantages:**
- ⚡ Fast: Pre-built libraries install in seconds
- 🎯 Reliable: Known working configuration for your platform
- 💾 Efficient: Reuses cached builds from CI/CD
- 🔄 Automatic: Falls back to source build if needed

### Method 2: Clone with Submodules (Traditional)

If you prefer the traditional approach:

```bash
# Clone with submodules
git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Setup and build
make setup
make build
```

### Method 3: Manual Script Execution

For more control over the setup process:

```bash
# Clone the repository
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Run setup script directly
./scripts/setup-mupdf.sh

# Build
go build ./pkg/mupdf/
```

**Script options:**
```bash
# Download only (don't build from source)
./scripts/setup-mupdf.sh --download-only

# Build from source only (skip download)
./scripts/setup-mupdf.sh --build-only

# Force rebuild even if libraries exist
./scripts/setup-mupdf.sh --force
```

### Method 4: Using go get (Legacy)

✅ **The build script automatically downloads the MuPDF submodule when needed.**

```bash
# Install the library
go get bitbucket.org/lexmata/go-mupdf@latest

# Build your project (submodule will be downloaded automatically)
go build ./your-project
```

**Requirements**:
- Git must be installed on your system
- Network access to GitHub (for MuPDF submodule)
- The build script will automatically:
  1. Parse `.gitmodules` to find submodule URL and branch
  2. Clone the MuPDF repository to `third_party/mupdf`
  3. Build MuPDF from source
  4. Compile the Go package

**Note**: The first build may take longer as it downloads and compiles MuPDF (~5-10 minutes depending on your system).

### Method 4: Using Go Modules with Replace Directive

For local development, you can use a replace directive:

```go
// go.mod
module your-project

require (
    bitbucket.org/lexmata/go-mupdf v1.0.0
)

replace bitbucket.org/lexmata/go-mupdf => ../go-mupdf
```

Then clone the repository properly:

```bash
git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git ../go-mupdf
```

## Why Submodules Are Required

MuPDF is a large C library that:
- Must be built from source for this wrapper
- Is versioned separately from the Go wrapper
- Requires specific build configuration
- Is too large to include directly in the repository

The git submodule approach ensures:
- Consistent MuPDF version across environments
- Proper version control
- Smaller repository size
- Clear dependency management

## How Library Setup Works

The setup process intelligently handles MuPDF libraries with multiple fallback methods:

### Priority Order

1. **Check Existing Libraries** ✅
   - If libraries already exist in `third_party/mupdf/build/release`, skip setup

2. **Download Pre-built from GitHub Releases** 🚀
   - Automatically detects your platform (linux-amd64, darwin-arm64, etc.)
   - Downloads matching release from GitHub Releases
   - Extracts and installs in seconds
   - **Available for:** linux-amd64 and linux-arm64 only — macOS and Windows fall back to building from source automatically

3. **Build from Source** 🔨
   - Falls back if download fails or platform not supported
   - Initializes git submodule automatically
   - Compiles MuPDF with optimized settings
   - Takes 5-10 minutes on first build

### Platform Detection

The scripts automatically detect:
- **Operating System**: Linux, macOS, Windows
- **Architecture**: amd64 (x86_64), arm64 (aarch64), arm
- **Version**: From VERSION file or git tags

Example detected platforms:
- `linux-amd64` - Linux on Intel/AMD 64-bit (pre-built libraries available)
- `linux-arm64` - Linux on ARM 64-bit (pre-built libraries available)
- `darwin-arm64` - macOS on Apple Silicon (builds from source)
- `darwin-amd64` - macOS on Intel (builds from source)

### Download URLs

Pre-built libraries are downloaded from:
```
https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-{version}-{platform}.tar.gz
```

Example:
```
https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-1.2.6-linux-amd64.tar.gz
```

These are automatically built and uploaded by the CI/CD pipeline on each release.

## Troubleshooting

### Error: "MuPDF submodule not found"

**Cause**: Submodules weren't initialized.

**Solution**:
```bash
git submodule update --init --recursive
```

### Error: "go get doesn't fetch submodules"

**Cause**: `go get` doesn't support git submodules.

**Solution**: Use `git clone --recurse-submodules` instead.

### Error: "Failed to initialize git submodules"

**Cause**: Not in a git repository or git is not available.

**Solution**:
- Ensure you cloned the repository (not just downloaded a zip)
- Verify git is installed: `git --version`
- Check you're in the repository root: `ls -la .git`

### Error: "MuPDF build failed"

**Cause**: Missing build dependencies.

**Solution**: Install required packages:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential gcc g++ make pkg-config \
    libharfbuzz-dev libfreetype6-dev libjpeg-dev libpng-dev \
    zlib1g-dev libjbig2dec-dev libopenjp2-7-dev
```

## CI/CD Considerations

### GitHub Actions

Using pre-built libraries:

```yaml
- uses: actions/checkout@v3

- name: Setup MuPDF Libraries
  run: ./scripts/setup-mupdf.sh

- name: Build and test
  run: |
    make build
    make test
```

The setup script will automatically download pre-built libraries for the platform.

### Docker

Optimized Dockerfile using pre-built libraries:

```dockerfile
# Copy setup script
COPY scripts/setup-mupdf.sh /app/scripts/

# Setup libraries (downloads pre-built if available)
RUN ./scripts/setup-mupdf.sh

# Build Go wrapper
RUN go build ./pkg/mupdf/
```

## Best Practices

1. **Use `make setup`** for automatic library installation
2. **Prefer pre-built libraries** for faster development iteration
3. **Use Makefile targets**: `make build` and `make test` handle dependencies
4. **Cache in CI/CD**: Use pipeline cache for MuPDF libraries
5. **Clean builds**: `make clean` removes cached libraries

## Advanced Configuration

### Force Source Build

To always build from source instead of downloading:

```bash
# Environment variable
export SKIP_DOWNLOAD=1
./scripts/setup-mupdf.sh

# Or use script flag
./scripts/install-prebuilt-libs.sh --skip-download
```

### Use Specific Version

```bash
# Download specific version
export MUPDF_VERSION=1.2.6
./scripts/setup-mupdf.sh
```

### Override Platform Detection

```bash
# Force specific platform
export MUPDF_PLATFORM=linux-amd64
./scripts/setup-mupdf.sh
```

## Related Documentation

- **GitHub Actions**: See `.github/workflows/release.yml` for the CI/CD workflow
- **Build System**: See `docs/BUILD_SYSTEM.md` for build details

