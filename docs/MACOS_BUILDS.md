# macOS Builds Guide

This document explains how to build go-mupdf on macOS. **Automated builds are now enabled** for Bitbucket Premium users!

## ✅ Automated Builds (Enabled)

**Pre-built libraries are automatically generated** for darwin-amd64 and darwin-arm64 on every release tag! Just run:

```bash
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf
make setup  # Downloads pre-built libraries instantly!
```

No manual build needed unless you're on an unsupported platform or want to customize the build.

## Quick Start (Manual Build)

### On macOS (Intel or Apple Silicon)

```bash
# Clone the repository
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf

# Install dependencies
brew install make gcc pkg-config

# Build MuPDF libraries
make setup

# Verify build
go build ./pkg/mupdf/

# Create distribution package
./scripts/build-static-libs.sh
```

## Automated Builds with Bitbucket Pipelines

### Requirements

- **Bitbucket Premium or above** ✅ (You have this!)
- macOS runner allocation in your workspace settings

### Current Status

✅ **macOS builds are ENABLED and running!**

The pipeline automatically builds for both:
- **darwin-amd64** - Intel Macs (using `macos-12-xcode-14`)
- **darwin-arm64** - Apple Silicon (using `macos-14-xcode-15`)

All builds run in parallel on tagged releases and upload to Bitbucket Downloads.

### Pipeline Configuration for macOS

#### Darwin AMD64 (Intel Macs)

```yaml
- step:
    name: Build Static Library Distribution (Darwin AMD64)
    image: macos-12-xcode-14
    caches:
      - go
    script:
      - brew install make gcc pkg-config
      - git submodule update --init --recursive
      - export TAG_NAME=$BITBUCKET_TAG
      - export VERSION=${TAG_NAME#v}
      - export GOOS=darwin
      - export GOARCH=amd64
      - export CGO_ENABLED=1
      - echo "Building static library distribution for darwin-amd64 version: $VERSION"
      - ./scripts/build-static-libs.sh
      - echo "=== Darwin AMD64 Distribution Package Created ==="
      - ls -lh dist/
      - cat dist/*.sha256
      - echo "===================================="
    after-script:
      - |
        TOKEN="${BITBUCKET_DOWNLOADS_TOKEN:-$BITBUCKET_API_TOKEN}"
        USERNAME="${BITBUCKET_USERNAME}"
        if [ ! -z "$TOKEN" ] && [ ! -z "$USERNAME" ]; then
          echo "Uploading Darwin AMD64 distribution packages to Bitbucket Downloads..."
          for file in dist/*.tar.gz dist/*.sha256; do
            if [ -f "$file" ]; then
              filename=$(basename "$file")
              echo "Uploading $filename..."
              curl -X POST \
                "https://api.bitbucket.org/2.0/repositories/$BITBUCKET_REPO_OWNER/$BITBUCKET_REPO_SLUG/downloads" \
                -u "$USERNAME:$TOKEN" \
                -F "files=@$file"
            fi
          done
        fi
    artifacts:
      - dist/*.tar.gz
      - dist/*.sha256
```

#### Darwin ARM64 (Apple Silicon)

```yaml
- step:
    name: Build Static Library Distribution (Darwin ARM64)
    image: macos-14-xcode-15
    caches:
      - go
    script:
      - brew install make gcc pkg-config
      - git submodule update --init --recursive
      - export TAG_NAME=$BITBUCKET_TAG
      - export VERSION=${TAG_NAME#v}
      - export GOOS=darwin
      - export GOARCH=arm64
      - export CGO_ENABLED=1
      - echo "Building static library distribution for darwin-arm64 version: $VERSION"
      - ./scripts/build-static-libs.sh
      - echo "=== Darwin ARM64 Distribution Package Created ==="
      - ls -lh dist/
      - cat dist/*.sha256
      - echo "===================================="
    after-script:
      - |
        TOKEN="${BITBUCKET_DOWNLOADS_TOKEN:-$BITBUCKET_API_TOKEN}"
        USERNAME="${BITBUCKET_USERNAME}"
        if [ ! -z "$TOKEN" ] && [ ! -z "$USERNAME" ]; then
          echo "Uploading Darwin ARM64 distribution packages to Bitbucket Downloads..."
          for file in dist/*.tar.gz dist/*.sha256; do
            if [ -f "$file" ]; then
              filename=$(basename "$file")
              echo "Uploading $filename..."
              curl -X POST \
                "https://api.bitbucket.org/2.0/repositories/$BITBUCKET_REPO_OWNER/$BITBUCKET_REPO_SLUG/downloads" \
                -u "$USERNAME:$TOKEN" \
                -F "files=@$file"
            fi
          done
        fi
    artifacts:
      - dist/*.tar.gz
      - dist/*.sha256
```

## Manual Distribution Creation

### Build Script Usage

```bash
# On macOS AMD64
export GOOS=darwin
export GOARCH=amd64
export TAG_NAME=v1.4.0
./scripts/build-static-libs.sh

# On macOS ARM64 (Apple Silicon)
export GOOS=darwin
export GOARCH=arm64
export TAG_NAME=v1.4.0
./scripts/build-static-libs.sh
```

### Upload to Bitbucket Downloads

```bash
# Set credentials
export BITBUCKET_USERNAME="your-username"
export BITBUCKET_DOWNLOADS_TOKEN="your-app-password"
export REPO_OWNER="lexmata"
export REPO_SLUG="go-mupdf"

# Upload distribution
for file in dist/*.tar.gz dist/*.sha256; do
  filename=$(basename "$file")
  echo "Uploading $filename..."
  curl -X POST \
    "https://api.bitbucket.org/2.0/repositories/$REPO_OWNER/$REPO_SLUG/downloads" \
    -u "$BITBUCKET_USERNAME:$BITBUCKET_DOWNLOADS_TOKEN" \
    -F "files=@$file"
done
```

## GitHub Actions Alternative

If Bitbucket macOS runners are not available, you can use GitHub Actions for macOS builds:

### `.github/workflows/macos-release.yml`

```yaml
name: Build macOS Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-macos:
    strategy:
      matrix:
        include:
          - os: macos-12
            goarch: amd64
          - os: macos-14
            goarch: arm64

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install dependencies
        run: brew install make gcc pkg-config

      - name: Build distribution
        env:
          GOOS: darwin
          GOARCH: ${{ matrix.goarch }}
          TAG_NAME: ${{ github.ref_name }}
        run: ./scripts/build-static-libs.sh

      - name: Upload to Bitbucket
        env:
          BITBUCKET_USERNAME: ${{ secrets.BITBUCKET_USERNAME }}
          BITBUCKET_TOKEN: ${{ secrets.BITBUCKET_TOKEN }}
        run: |
          for file in dist/*.tar.gz dist/*.sha256; do
            curl -X POST \
              "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/downloads" \
              -u "$BITBUCKET_USERNAME:$BITBUCKET_TOKEN" \
              -F "files=@$file"
          done
```

## Troubleshooting

### Common Issues

**"Command line tools not found"**
```bash
xcode-select --install
```

**"make: command not found"**
```bash
brew install make
# Use gmake instead of make on macOS
```

**"pkg-config not found"**
```bash
brew install pkg-config
```

**CGO compilation errors**
```bash
# Ensure Xcode Command Line Tools are installed
xcode-select --print-path

# If needed, reset to default
sudo xcode-select --reset
```

### Build Performance

- **Intel Macs**: ~8-10 minutes for full build
- **Apple Silicon**: ~5-7 minutes for full build
- **Pre-built downloads**: <10 seconds (when available)

## Cost Considerations

### Bitbucket Pipelines

- **macOS runners** cost more build minutes than Linux
- Check current pricing: https://bitbucket.org/product/pricing
- Consider using manual builds or GitHub Actions as alternatives

### GitHub Actions

- **macOS runners** are available on free tier (limited minutes)
- Better for open-source projects
- Can sync artifacts back to Bitbucket

## Future Improvements

- [ ] Enable automated macOS builds when runners are available
- [ ] Universal binaries (combined AMD64 + ARM64)
- [ ] Code signing for macOS binaries
- [ ] Notarization for macOS packages

---

**Last Updated**: 2025-11-10

