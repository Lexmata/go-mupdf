# macOS Builds Guide

This document explains how to build go-mupdf on macOS and enable automated macOS builds.

## CI/CD Status

The repository uses **GitHub Actions** for automated builds:

- ✅ **Linux AMD64** - Automated via GitHub Actions
- ✅ **Linux ARM64** - Automated via GitHub Actions
- ⚠️ **Darwin AMD64** - Can be added to GitHub Actions workflow if needed
- ⚠️ **Darwin ARM64** - Can be added to GitHub Actions workflow if needed

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

## Automated Builds with GitHub Actions

GitHub Actions provides managed macOS runners. To add macOS builds to the workflow, extend `.github/workflows/release.yml` with macOS-specific jobs.


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

## GitHub Actions for macOS Builds

The repository uses GitHub Actions for automated builds. To add macOS support, extend `.github/workflows/release.yml`:

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

### CI/CD

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

