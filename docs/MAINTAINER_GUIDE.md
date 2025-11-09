# Maintainer Guide

Quick reference for project maintainers.

## Release Process

### 1. Prepare Release

```bash
# Ensure develop is up to date
git checkout develop
git pull origin develop

# Create release branch
git checkout -b release/X.Y.Z

# Update VERSION file
echo "X.Y.Z" > VERSION

# Update CHANGELOG.md
# Add release notes under ## [X.Y.Z] - YYYY-MM-DD

# Commit changes
git add VERSION CHANGELOG.md
git commit -m "chore: prepare release v X.Y.Z"
```

### 2. Merge to Main and Tag

```bash
# Merge to main
git checkout main
git merge release/X.Y.Z

# Create and push tag
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags

# Merge back to develop
git checkout develop
git merge release/X.Y.Z
git push origin develop

# Clean up release branch
git branch -d release/X.Y.Z
```

### 3. Verify CI/CD

After pushing the tag, the CI/CD pipeline will automatically:

1. Run all tests
2. Build release artifacts
3. **Build static library distribution packages**
4. Upload to Bitbucket Downloads (if configured)

Check the pipeline at:
https://bitbucket.org/lexmata/go-mupdf/addon/pipelines/home

### 4. Distribution Packages

The pipeline automatically creates:

- `go-mupdf-X.Y.Z-linux-amd64.tar.gz` - Static libraries for Linux
- `go-mupdf-X.Y.Z-linux-amd64.tar.gz.sha256` - Checksum
- Release artifacts (source, binaries, docs)

These are available in:
- Pipeline artifacts
- Bitbucket Downloads (if `BITBUCKET_DOWNLOADS_TOKEN` is configured)

### 5. Manual Distribution Build (Optional)

To build distribution packages locally:

```bash
# Build for current platform
make dist

# Or with specific version
VERSION=1.1.0 make dist

# Output in dist/ directory
ls -lh dist/
```

## Building Distribution Packages

### Quick Build

```bash
# Build distribution for current platform
make dist
```

### Advanced Options

```bash
# Build with specific version
VERSION=1.1.0 ./scripts/build-static-libs.sh

# Build debug version
BUILD_TYPE=debug ./scripts/build-static-libs.sh

# Only build libraries (no tarball)
./scripts/build-static-libs.sh --build-only

# Only create tarball (libraries exist)
./scripts/build-static-libs.sh --skip-build

# Clean distribution artifacts
make dist-clean
# or
./scripts/build-static-libs.sh --clean
```

### Multi-Platform Builds

To build for multiple platforms, use platform-specific machines or CI/CD:

```bash
# On Linux AMD64
make dist  # Creates go-mupdf-X.Y.Z-linux-amd64.tar.gz

# On macOS ARM64
make dist  # Creates go-mupdf-X.Y.Z-darwin-arm64.tar.gz

# Upload all to Bitbucket Downloads or release page
```

## Hosting Distribution Packages

### Option 1: Bitbucket Downloads (Automated)

Configure the `BITBUCKET_DOWNLOADS_TOKEN` environment variable in Bitbucket Pipelines:

1. Go to Repository Settings → Pipelines → Repository variables
2. Add variable: `BITBUCKET_DOWNLOADS_TOKEN`
3. Set value to your Bitbucket App Password with Downloads permission
4. On release tag push, packages are automatically uploaded

### Option 2: Manual Upload to Bitbucket

```bash
# Upload manually using curl
curl -X POST \
  "https://api.bitbucket.org/2.0/repositories/lexmata/go-mupdf/downloads" \
  -H "Authorization: Bearer $TOKEN" \
  -F "files=@dist/go-mupdf-1.1.0-linux-amd64.tar.gz"
```

### Option 3: GitHub Releases

If mirroring to GitHub:

```bash
gh release create v1.1.0 \
  dist/go-mupdf-1.1.0-*.tar.gz \
  dist/go-mupdf-1.1.0-*.sha256 \
  --title "Release v1.1.0" \
  --notes-file CHANGELOG.md
```

## Testing Distribution Packages

Before releasing, test the distribution package:

```bash
# Build distribution
make dist

# Extract to test location
mkdir -p /tmp/test-dist
cd /tmp/test-dist
tar -xzf ~/go-mupdf/dist/go-mupdf-1.1.0-linux-amd64.tar.gz

# Verify contents
ls -la go-mupdf-1.1.0-linux-amd64/
cat go-mupdf-1.1.0-linux-amd64/README.md

# Test with a Go project
mkdir test-project && cd test-project
go mod init test
go get bitbucket.org/lexmata/go-mupdf@latest

# Copy libraries
mkdir -p third_party/mupdf/build/release third_party/mupdf/include
cp ~/go-mupdf/dist/go-mupdf-1.1.0-linux-amd64/lib/*.a third_party/mupdf/build/release/
cp -r ~/go-mupdf/dist/go-mupdf-1.1.0-linux-amd64/include/mupdf third_party/mupdf/include/

# Create test program
cat > main.go << 'EOF'
package main
import (
    "fmt"
    "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)
func main() {
    ctx, _ := mupdf.NewContext()
    defer ctx.Drop()
    fmt.Println("Success!")
}
EOF

# Build
go build
./test
```

## Docker Testing

Test in CI/CD-like environment:

```bash
# Build Docker test image
make docker-build

# Run tests
make docker-test

# Quick test (no rebuild)
make docker-quick

# Interactive debugging
make docker-shell
```

## Common Tasks

### Update MuPDF Version

```bash
cd third_party/mupdf
git fetch --all
git checkout <new-version-tag>
cd ../..
git add third_party/mupdf
git commit -m "chore: update MuPDF to <version>"
```

### Run Benchmarks

```bash
go test ./pkg/mupdf/ -bench=. -benchmem -run=^$
```

### Check Coverage

```bash
go test ./pkg/mupdf/ -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Lint Code

```bash
# Format
go fmt ./...

# Vet
go vet ./pkg/mupdf/

# Lint (if golangci-lint installed)
golangci-lint run
```

## CI/CD Configuration

### Required Secrets

For full CI/CD functionality, configure these secrets in Bitbucket Pipelines:

- `CODECOV_TOKEN` - For code coverage reporting
- `BITBUCKET_DOWNLOADS_TOKEN` - For automatic release uploads

### Pipeline Triggers

- **Main branch**: Full test suite + coverage
- **Develop branch**: Standard tests
- **Feature branches**: Basic validation
- **Pull requests**: Quick tests + race detection
- **Tags (v*)**: Full release pipeline + distribution build

## Troubleshooting

### Distribution Build Fails

```bash
# Clean and rebuild
make dist-clean
git submodule update --init --recursive
make dist
```

### Pipeline Fails on Tag

Common issues:

1. **MuPDF submodule not initialized**: Fixed automatically by pipeline
2. **VERSION file mismatch**: Ensure VERSION file matches tag
3. **Build dependencies missing**: Pipeline installs automatically

### Library Size Issues

```bash
# Check library sizes
ls -lh third_party/mupdf/build/release/*.a

# Expected sizes:
# libmupdf.a: 8-12 MB
# libmupdf-third.a: 15-20 MB
```

## Support

- **Repository**: https://bitbucket.org/lexmata/go-mupdf
- **Issues**: https://bitbucket.org/lexmata/go-mupdf/issues
- **Wiki**: https://bitbucket.org/lexmata/go-mupdf/wiki

