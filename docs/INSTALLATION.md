# Installation Guide

This guide covers different installation methods for the Go MuPDF Wrapper.

## Important: Git Submodules Required

⚠️ **This library requires MuPDF to be built from source, which is provided as a git submodule.**

**The build script (`build.go`) automatically downloads the submodule if missing**, even when installed via `go get`. However, **git must be installed** on your system for this to work.

## Installation Methods

### Method 1: Clone with Submodules (Recommended)

This is the recommended method for all use cases:

```bash
# Clone with submodules
git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Build MuPDF (happens automatically via build.go)
go build ./pkg/mupdf/

# Or use the Makefile
make build
```

### Method 2: Clone Then Initialize Submodules

If you already cloned without submodules:

```bash
# Clone the repository
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Initialize submodules
git submodule update --init --recursive

# Build
go build ./pkg/mupdf/
```

### Method 3: Using go get (Automatic Submodule Download)

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

## Automatic Build Process

When you build the package, `build.go` automatically:

1. **Checks for MuPDF source** in `third_party/mupdf`
2. **Initializes submodules** if in a git repository
3. **Builds MuPDF** from source using `make`
4. **Verifies the build** before proceeding

This happens automatically when you run:
```bash
go build ./pkg/mupdf/
go test ./pkg/mupdf/
```

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

### Bitbucket Pipelines

The pipeline builds MuPDF from source:

```yaml
script:
  - apt-get install -y build-essential pkg-config libfreetype6-dev libjpeg-dev libpng-dev zlib1g-dev libjbig2dec-dev libopenjp2-7-dev libharfbuzz-dev
  - git submodule update --init --recursive
  - cd third_party/mupdf && make -j$(nproc) libs && cd ../..
  - go test ./pkg/mupdf/
```

### GitHub Actions

For source-built MuPDF:

```yaml
- uses: actions/checkout@v3
  with:
    submodules: recursive

- name: Install build dependencies
  run: |
    sudo apt-get update
    sudo apt-get install -y build-essential pkg-config libfreetype6-dev libjpeg-dev libpng-dev zlib1g-dev libjbig2dec-dev libopenjp2-7-dev libharfbuzz-dev

- name: Build and test
  run: |
    cd third_party/mupdf && make -j$(nproc) libs && cd ../..
    go test ./pkg/mupdf/
```

### Docker

The Dockerfile automatically handles submodules:

```dockerfile
RUN git submodule update --init --recursive
RUN cd third_party/mupdf && make -j$(nproc) libs
```

## Best Practices

1. **Always clone with submodules**: Use `--recurse-submodules` flag
2. **Use Makefile targets**: `make build` handles everything
3. **Check submodule status**: `git submodule status`
4. **Update submodules**: `git submodule update --remote` (when needed)

