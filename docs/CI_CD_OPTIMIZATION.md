# CI/CD Optimization Guide

This document explains the CI/CD optimization strategy for building and reusing MuPDF libraries across pipeline steps.

## Overview

Building MuPDF from source takes 5-10 minutes per build. To optimize the CI/CD pipeline, we:

1. **Build once**: MuPDF is built once at the start of the pipeline
2. **Cache**: Artifacts are cached between pipeline runs
3. **Share**: Artifacts are shared across all pipeline steps
4. **Reuse**: All steps use pre-built libraries instead of building from source

## Architecture

### Pipeline Flow

```
Pipeline Start
     |
     v
[Build MuPDF Libraries] ← Cached (mupdf-libs cache)
     |
     ├─ Creates: mupdf-artifacts/
     ├─ Contains: libmupdf.a, libmupdf-third.a, headers
     └─ Artifact shared with all subsequent steps
     |
     v
[Parallel Steps]
     |
     ├─> [Lint Code] ← Uses pre-built libs
     ├─> [Run Tests] ← Uses pre-built libs
     ├─> [Build Release] ← Uses pre-built libs
     └─> [Distribution] ← Builds fresh for packaging
```

### Key Components

1. **build-mupdf-artifact.sh**
   - Builds MuPDF libraries from source
   - Creates reusable artifact package
   - Generates tarball for distribution

2. **install-prebuilt-libs.sh**
   - Installs pre-built libraries
   - Falls back to source build if needed
   - Verifies installation

3. **Pipeline Cache**
   - Cache key: `third_party/mupdf/.git/HEAD`
   - Updates when MuPDF submodule changes
   - Persists across pipeline runs

4. **Artifact Sharing**
   - `mupdf-artifacts/` shared between steps
   - Contains libraries and headers
   - Automatically downloaded by steps

## Performance Impact

### Before Optimization

| Step | Build Time |
|------|------------|
| Lint Code | 10-12 min (includes MuPDF build) |
| Run Tests | 10-12 min (includes MuPDF build) |
| Build Release | 10-12 min (includes MuPDF build) |
| **Total** | **30-36 min** |

### After Optimization

| Step | Build Time |
|------|------------|
| Build MuPDF Libraries (once) | 8-10 min |
| Lint Code | 1-2 min |
| Run Tests | 3-5 min |
| Build Release | 2-3 min |
| **Total** | **14-20 min** |

**Time saved**: ~50-60% reduction (16-20 minutes)

### With Cache (Subsequent Runs)

When the MuPDF cache is warm:

| Step | Build Time |
|------|------------|
| Build MuPDF Libraries (cached) | 30-60 sec |
| Lint Code | 1-2 min |
| Run Tests | 3-5 min |
| Build Release | 2-3 min |
| **Total** | **7-11 min** |

**Time saved**: ~70-75% reduction (23-28 minutes)

## How It Works

### 1. Building Artifacts

The `build-mupdf` step runs first:

```yaml
- step: &build-mupdf
    name: Build MuPDF Libraries
    caches:
      - mupdf-libs
    script:
      - git submodule update --init --recursive
      - ./scripts/build-mupdf-artifact.sh
    artifacts:
      - mupdf-artifacts/**
```

This step:
- Initializes the MuPDF submodule
- Builds `libmupdf.a` and `libmupdf-third.a`
- Copies headers to artifact directory
- Creates `mupdf-libs.tar.gz` tarball
- Saves everything to `mupdf-artifacts/`
- Stores in cache for future runs

### 2. Using Artifacts

All subsequent steps use the pre-built libraries:

```yaml
- step: *lint
    name: Lint Code
    caches:
      - go
      - mupdf-libs
    script:
      - ./scripts/install-prebuilt-libs.sh
      # ... rest of lint commands
```

The install script:
1. Checks if libraries already exist
2. Tries to install from `mupdf-artifacts/` (from artifact or cache)
3. Verifies installation
4. Falls back to source build if needed

### 3. Cache Strategy

The cache is keyed to the MuPDF submodule state:

```yaml
caches:
  mupdf-libs:
    key:
      files:
        - third_party/mupdf/.git/HEAD
    path: mupdf-artifacts
```

**Cache invalidation**:
- Automatically updates when MuPDF version changes
- Rebuilds when submodule is updated
- Persists across pipeline runs on same MuPDF version

### 4. Artifact Flow

```
Step 1: build-mupdf
  ↓
  Produces: mupdf-artifacts/ (artifact)
  ↓
  Saved to: cache (mupdf-libs)

Step 2: lint (parallel)
  ↓
  Downloads: mupdf-artifacts/ (from artifact)
  ↓
  Restores: cache (mupdf-libs) if available
  ↓
  Installs: ./scripts/install-prebuilt-libs.sh

Step 3: test (parallel)
  ↓
  Downloads: mupdf-artifacts/ (from artifact)
  ↓
  Restores: cache (mupdf-libs) if available
  ↓
  Installs: ./scripts/install-prebuilt-libs.sh
```

## Benefits

### 1. Faster Pipelines

- **50-60% faster** on first run
- **70-75% faster** with cache
- Parallel steps run truly in parallel

### 2. Consistent Builds

- Same libraries used across all steps
- No version drift between steps
- Reproducible builds

### 3. Resource Efficiency

- Build once, use many times
- Less CPU usage
- Lower costs

### 4. Better DX

- Faster feedback on PRs
- Quicker deploys
- Less waiting time

## Local Development

Developers can also use the pre-built artifacts:

### Build Artifacts Locally

```bash
# Build MuPDF and create artifact
./scripts/build-mupdf-artifact.sh

# Artifact created in: mupdf-artifacts/
```

### Use Pre-built Artifacts

```bash
# Install from artifact directory
./scripts/install-prebuilt-libs.sh

# Or specify custom location
ARTIFACT_DIR=/path/to/artifacts ./scripts/install-prebuilt-libs.sh
```

### Force Source Build

```bash
# Skip artifacts and build from source
./scripts/install-prebuilt-libs.sh --force-build
```

## Troubleshooting

### Cache Not Working

**Symptoms**: MuPDF rebuilds every time

**Solutions**:
```bash
# Check cache key
cat third_party/mupdf/.git/HEAD

# Verify submodule is initialized
git submodule status

# Clear cache in Bitbucket Pipelines settings
# Repository Settings → Pipelines → Caches → Clear cache: mupdf-libs
```

### Artifact Not Found

**Symptoms**: `install-prebuilt-libs.sh` falls back to source build

**Solutions**:
```bash
# Check artifact directory exists
ls -la mupdf-artifacts/

# Verify artifact contents
ls -la mupdf-artifacts/lib/

# Check if artifact is being shared
# In pipeline YAML, ensure:
artifacts:
  - mupdf-artifacts/**
```

### Libraries Corrupted

**Symptoms**: Link errors or test failures

**Solutions**:
```bash
# Rebuild artifact
rm -rf mupdf-artifacts/
./scripts/build-mupdf-artifact.sh

# Verify libraries
./scripts/install-prebuilt-libs.sh --force-build
go test ./pkg/mupdf/
```

### Cache Too Large

**Symptoms**: Slow cache restore

**Solutions**:
```bash
# Check cache size
du -sh mupdf-artifacts/

# Expected size: ~25-35 MB compressed

# If too large, check for:
# - Debug symbols (should use release build)
# - Duplicate files
# - Unnecessary includes
```

## Advanced Usage

### Custom Cache Key

To cache based on different criteria:

```yaml
caches:
  mupdf-libs:
    key:
      files:
        - third_party/mupdf/Makefile
        - scripts/build-mupdf-artifact.sh
    path: mupdf-artifacts
```

### Multiple Platforms

Build artifacts for multiple platforms:

```yaml
- step:
    name: Build MuPDF (Linux AMD64)
    script:
      - ./scripts/build-mupdf-artifact.sh --artifact-dir mupdf-linux-amd64
    artifacts:
      - mupdf-linux-amd64/**

- step:
    name: Build MuPDF (Linux ARM64)
    image: arm64v8/golang:1.24
    script:
      - ./scripts/build-mupdf-artifact.sh --artifact-dir mupdf-linux-arm64
    artifacts:
      - mupdf-linux-arm64/**
```

### Artifact Versioning

Include version in artifact:

```bash
# Build with version tag
VERSION=1.1.0 ./scripts/build-mupdf-artifact.sh

# Creates: mupdf-artifacts/ with BUILD_INFO containing version
```

## Monitoring

### Check Pipeline Performance

```bash
# View pipeline duration
# Bitbucket → Repository → Pipelines → Click on pipeline run
# Compare "Build time" before and after optimization
```

### Cache Hit Rate

```bash
# In pipeline logs, look for:
# "Restoring mupdf-libs cache" → Cache hit
# "Building MuPDF from source" → Cache miss
```

### Artifact Size

```bash
# In build-mupdf step logs:
cat mupdf-artifacts/BUILD_INFO
du -h mupdf-artifacts/lib/
du -h mupdf-artifacts/mupdf-libs.tar.gz
```

## Best Practices

### 1. Keep MuPDF Version Stable

```bash
# Update submodule deliberately
cd third_party/mupdf
git fetch --all
git checkout v1.23.9
cd ../..
git add third_party/mupdf
git commit -m "chore: update MuPDF to v1.23.9"
```

### 2. Monitor Cache Size

```bash
# Periodically check artifact size
./scripts/build-mupdf-artifact.sh
du -sh mupdf-artifacts/

# Keep under 50MB for fast restore
```

### 3. Test with Fresh Cache

```bash
# Simulate cache miss locally
rm -rf mupdf-artifacts/
./scripts/install-prebuilt-libs.sh

# Should fall back to source build gracefully
```

### 4. Verify Artifacts

```bash
# Always verify after building
./scripts/build-mupdf-artifact.sh
./scripts/install-prebuilt-libs.sh
go test ./pkg/mupdf/ -v
```

## Migration Guide

### For Existing Pipelines

If migrating from a non-optimized pipeline:

1. **Add build-mupdf step**:
   ```yaml
   - step: *build-mupdf
   ```

2. **Add cache to all steps**:
   ```yaml
   caches:
     - go
     - mupdf-libs  # Add this
   ```

3. **Replace build commands**:
   ```yaml
   # Old:
   - cd third_party/mupdf && make -j$(nproc) libs && cd ../..
   
   # New:
   - ./scripts/install-prebuilt-libs.sh
   ```

4. **Test the pipeline**:
   - Run on a test branch first
   - Verify all steps complete successfully
   - Check time savings in pipeline logs

### Rollback Plan

If issues occur, rollback by:

1. Remove `- step: *build-mupdf`
2. Remove `- mupdf-libs` from caches
3. Replace `./scripts/install-prebuilt-libs.sh` with original build commands

## Future Improvements

### Planned

- [ ] Multi-arch support (ARM64, etc.)
- [ ] Parallel artifact builds
- [ ] Cross-compilation support
- [ ] CDN hosting for public artifacts
- [ ] Automated artifact uploads to releases

### Under Consideration

- [ ] Docker image with pre-built libraries
- [ ] Pre-built library distribution packages
- [ ] GitHub Actions optimization
- [ ] GitLab CI optimization

## Support

For issues or questions:

- **Repository**: https://bitbucket.org/lexmata/go-mupdf
- **Issues**: https://bitbucket.org/lexmata/go-mupdf/issues
- **Documentation**: https://bitbucket.org/lexmata/go-mupdf/src/main/docs/

---

**Last Updated**: 2024
**Pipeline Version**: v2.0 (Optimized)
**MuPDF Version**: 1.23.x

