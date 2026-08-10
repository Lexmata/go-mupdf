# 🚀 Automated Release Process

## Overview

This document describes the automated release process for the Go MuPDF wrapper project. The system provides comprehensive automation for creating, building, testing, and publishing releases.

## 🎯 Release Workflow

### 1. Automated Pipeline Triggers
- **Tag Creation**: Any tag matching `v*` pattern (e.g., `v1.0.0`, `v1.1.0-rc.1`)
- **Comprehensive Testing**: Full test suite with coverage reporting
- **Quality Gates**: Code formatting, linting, and static analysis
- **Artifact Generation**: Binaries, documentation, and source archives
- **Automatic Upload**: Release artifacts to Bitbucket Downloads

### 2. Release Artifacts Generated
- **Static Library Packages**: `dist/go-mupdf-<version>-linux-amd64.tar.gz` and `dist/go-mupdf-<version>-linux-arm64.tar.gz` (pre-built MuPDF libraries and headers; version has no `v` prefix)
- **Checksums**: A per-file `.tar.gz.sha256` alongside each package
- **Coverage Data**: `release-coverage.out` from the release test run
- **Upload**: Packages and checksums are uploaded to Bitbucket Downloads

## 🛠️ Using the Release Script

### Interactive Release Creation
```bash
# Simple release with explicit version
./scripts/release.sh 1.4.7

# Auto-increment the version
./scripts/release.sh --major    # Increment major version (X.0.0)
./scripts/release.sh --minor    # Increment minor version (x.Y.0)
./scripts/release.sh --patch    # Increment patch version (x.y.Z)
./scripts/release.sh --pre      # Create pre-release (adds -rc.N suffix)

# Test release process (dry run)
./scripts/release.sh --dry-run 1.4.7

# Get help
./scripts/release.sh --help
```

### Release Script Features
- ✅ **Pre-flight Checks**: Branch validation, clean working directory
- ✅ **Quality Assurance**: Full test suite, coverage analysis, formatting
- ✅ **Version Management**: Semantic version validation and updates
- ✅ **Changelog Updates**: Automatic changelog generation
- ✅ **Git Operations**: Tag creation and repository push
- ✅ **Pipeline Trigger**: Automatic CI/CD pipeline initiation

## 📋 Manual Release Process (Alternative)

If you prefer manual control:

### Step 1: Prepare Release
```bash
# Ensure clean state
git status
git checkout main
git pull origin main

# Run tests
./scripts/test-runner.sh --benchmarks

# Update version
echo "1.1.0" > VERSION
```

### Step 2: Update Documentation
```bash
# Update CHANGELOG.md with new version
# Add release notes and changes
vim CHANGELOG.md
```

### Step 3: Create and Push Tag
```bash
# Commit version changes
git add VERSION CHANGELOG.md
git commit -m "Release 1.1.0"

# Create annotated tag
git tag -a v1.1.0 -m "Release 1.1.0

Complete feature set with:
- [List key features/changes]
- [Notable improvements]
- [Bug fixes]

See CHANGELOG.md for full details."

# Push changes and tag
git push origin main
git push origin v1.1.0
```

## 🔄 Pipeline Execution Details

### What Happens When You Create a Tag

1. **Trigger Detection**
   - Bitbucket detects new tag matching `v*`
   - Release pipeline automatically starts
   - Environment variables set (TAG_NAME, COMMIT, etc.)

2. **Environment Setup**
   - System dependencies installed
   - Go environment configured
   - MuPDF library built (or loaded from cache)

3. **Quality Assurance**
   - Complete test suite execution
   - Race condition detection
   - Code coverage analysis
   - Static analysis (go vet)
   - Code formatting validation

4. **Artifact Building**
   ```bash
   # Directory structure created (for tag v1.4.7):
   dist/
   ├── go-mupdf-1.4.7-linux-amd64.tar.gz         # Static library package (amd64)
   ├── go-mupdf-1.4.7-linux-amd64.tar.gz.sha256  # Checksum for the amd64 package
   ├── go-mupdf-1.4.7-linux-arm64.tar.gz         # Static library package (arm64)
   └── go-mupdf-1.4.7-linux-arm64.tar.gz.sha256  # Checksum for the arm64 package
   ```
   Note: the version in the filenames has no `v` prefix (it is derived from the tag with the prefix stripped).

5. **Artifact Upload**
   - Upload to Bitbucket Downloads section
   - Requires BITBUCKET_USERNAME and BITBUCKET_DOWNLOADS_TOKEN repository variables
   - Files available for public download (also kept as pipeline artifacts)

## 📦 Release Artifacts

### Generated Files
| Artifact | Description | Use Case |
|----------|-------------|----------|
| `go-mupdf-1.4.7-linux-amd64.tar.gz` | Pre-built MuPDF static libraries and headers (amd64) | Skipping MuPDF compilation |
| `go-mupdf-1.4.7-linux-amd64.tar.gz.sha256` | Checksum for the amd64 package | Security verification |
| `go-mupdf-1.4.7-linux-arm64.tar.gz` | Pre-built MuPDF static libraries and headers (arm64) | Skipping MuPDF compilation |
| `go-mupdf-1.4.7-linux-arm64.tar.gz.sha256` | Checksum for the arm64 package | Security verification |

### Download and Verification
```bash
# Download artifacts (replace 1.4.7 with the released version)
curl -L -O "https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-1.4.7-linux-amd64.tar.gz"
curl -L -O "https://bitbucket.org/lexmata/go-mupdf/downloads/go-mupdf-1.4.7-linux-amd64.tar.gz.sha256"

# Verify checksum
sha256sum -c go-mupdf-1.4.7-linux-amd64.tar.gz.sha256

# Extract and use
tar -xzf go-mupdf-1.4.7-linux-amd64.tar.gz
ls go-mupdf-1.4.7-linux-amd64/lib/   # libmupdf.a, libmupdf-third.a
```

## 🔧 Configuration

### Environment Variables (Optional)
```bash
# For automatic upload to Bitbucket Downloads
export BITBUCKET_DOWNLOADS_TOKEN="your-access-token"

# Pipeline automatically sets these:
# BITBUCKET_TAG          - The git tag name
# BITBUCKET_COMMIT       - The commit hash
# BITBUCKET_REPO_OWNER   - Repository owner
# BITBUCKET_REPO_SLUG    - Repository name
```

### Creating Download Token
1. Go to Bitbucket Settings → App Passwords
2. Create new app password with permissions:
   - ✅ **Repositories: Write** (for downloads)
   - ✅ **Repositories: Admin** (if needed)
3. Add token to repository variables:
   - Repository Settings → Repository variables
   - Name: `BITBUCKET_DOWNLOADS_TOKEN`
   - Value: Your app password
   - Secured: ✅ Yes

## 📊 Release Metrics

### Build Performance
- **Cold Build**: ~8-12 minutes (includes MuPDF compilation)
- **Warm Build**: ~5-8 minutes (with MuPDF cache)
- **Test Execution**: ~2-3 minutes
- **Artifact Generation**: ~1-2 minutes

### Quality Gates
- **Test Coverage**: Minimum 70% (the measured coverage is recorded in the release tag message)
- **Code Quality**: Must pass go vet and gofmt
- **Build Success**: All platforms must build successfully
- **Documentation**: Must include updated changelog

## 🎯 Release Types

### Stable Releases
```bash
# Format: vMAJOR.MINOR.PATCH
v1.0.0    # Major release
v1.1.0    # Minor release
v1.0.1    # Patch release
```

### Pre-releases
```bash
# Format: vMAJOR.MINOR.PATCH-PRERELEASE
v1.1.0-rc.1     # Release candidate
v1.1.0-beta.1   # Beta release
v1.1.0-alpha.1  # Alpha release
```

## 🚨 Troubleshooting

### Common Issues

#### Pipeline Failure
```bash
# Check pipeline logs
# Go to: Repository → Pipelines → Failed build

# Common causes:
# 1. Test failures
# 2. Code formatting issues
# 3. MuPDF build problems
# 4. Missing dependencies
```

#### Tag Creation Issues
```bash
# Tag already exists
git tag -d v1.1.0              # Delete local tag
git push origin --delete v1.1.0  # Delete remote tag

# Wrong tag format
# Must match: v[0-9]+.[0-9]+.[0-9]+
```

#### Upload Failures
```bash
# Check if BITBUCKET_DOWNLOADS_TOKEN is set
# Verify token has correct permissions
# Check repository variables in Bitbucket settings
```

### Recovery Procedures

#### Failed Release
```bash
# If pipeline fails after tag creation:
1. Fix the issue in code
2. Commit the fix
3. Delete and recreate the tag
4. Push the new tag

# Commands:
git tag -d v1.1.0
git push origin --delete v1.1.0
# Make fixes, commit
git tag -a v1.1.0 -m "Release 1.1.0 (fixed)"
git push origin v1.1.0
```

#### Rollback Release
```bash
# To rollback a release:
1. Delete the git tag
2. Revert version changes
3. Update changelog if needed

# Commands:
git tag -d v1.1.0
git push origin --delete v1.1.0
git revert HEAD  # If version was committed
```

## 📈 Best Practices

### Before Creating Release
- [ ] All tests pass locally
- [ ] Documentation is up to date
- [ ] CHANGELOG.md has been updated
- [ ] Version number follows semantic versioning
- [ ] Branch is up to date with remote

### Release Planning
- **Major Releases**: Plan for breaking changes, update migration docs
- **Minor Releases**: Focus on new features and improvements
- **Patch Releases**: Bug fixes and security updates only

### Communication
- **Internal**: Update team channels, project status
- **External**: Announce on relevant forums, update documentation
- **Users**: Consider sending notifications for major releases

## 🔮 Future Enhancements

### Planned Improvements
1. **Multi-platform Builds**: Windows and macOS binaries
2. **Docker Images**: Container releases for easy deployment
3. **GitHub Integration**: Mirror releases to GitHub
4. **Package Managers**: Homebrew, Chocolatey formulas
5. **Automated Testing**: Cross-platform compatibility testing

### Integration Opportunities
1. **Release Notes Enhancement**: Auto-generate from commit messages
2. **Notification Systems**: Slack/Teams integration
3. **Metrics Collection**: Download statistics and usage analytics
4. **Security Scanning**: Automated vulnerability assessment

---

## 📞 Support

For release process issues:
1. Check pipeline logs in Bitbucket
2. Review this documentation
3. Test locally with `./scripts/release.sh --dry-run`
4. Contact the development team

**Remember**: The release process is designed to be safe and reversible. Don't hesitate to test with dry-runs and pre-releases before creating stable releases.

---

*Last updated: 2024 - This process is continuously improved based on team feedback and experience.*