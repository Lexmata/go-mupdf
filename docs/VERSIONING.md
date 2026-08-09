# Go MuPDF Wrapper - Versioning Strategy

This document outlines the versioning strategy and release process for the Go MuPDF wrapper project.

## Semantic Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html):

```
MAJOR.MINOR.PATCH
```

### Version Components

- **MAJOR**: Incompatible API changes that break backward compatibility
- **MINOR**: New functionality added in a backward-compatible manner
- **PATCH**: Backward-compatible bug fixes

### Examples

- `1.0.0` → `1.0.1`: Bug fixes (patch release)
- `1.0.0` → `1.1.0`: New features (minor release)
- `1.0.0` → `2.0.0`: Breaking changes (major release)

## Release Process

### 1. Version Planning

#### Major Releases (X.0.0)
- Breaking API changes
- Significant architectural changes
- Removal of deprecated features
- MuPDF major version updates

#### Minor Releases (X.Y.0)
- New features and capabilities
- New API methods or types
- Performance improvements
- Documentation enhancements
- MuPDF minor version updates

#### Patch Releases (X.Y.Z)
- Bug fixes
- Security patches
- Documentation corrections
- Build improvements

### 2. Release Workflow

#### Pre-Release
1. **Feature Freeze**: Complete all planned features
2. **Testing**: Run full test suite and manual testing
3. **Documentation**: Update all relevant documentation
4. **Changelog**: Update CHANGELOG.md with all changes

#### Release Creation
1. **Version Bump**: Update VERSION file
2. **Commit Changes**: Commit version and changelog updates
3. **Create Tag**: Create annotated and signed git tag
4. **Build Verification**: Verify builds work across platforms

#### Post-Release
1. **Announcement**: Announce release to community
2. **Documentation**: Update any external documentation
3. **Next Version**: Plan next release cycle

### 3. Branching Strategy

#### Main Branches
- **`main`**: Production-ready code, always stable
- **`develop`**: Integration branch for next release
- **`release/X.Y.Z`**: Release preparation branches

#### Feature Branches
- **`feature/description`**: New features and enhancements
- **`fix/description`**: Bug fixes and patches
- **`docs/description`**: Documentation updates

#### Release Branches
- **`release/X.Y.Z`**: Prepare specific release
- Used for final testing and bug fixes
- Merged to `main` and tagged for release

### 4. Git Tagging

#### Tag Format
- **Format**: `vX.Y.Z` (e.g., `v1.0.0`, `v1.2.3`)
- **Type**: Annotated tags with detailed messages
- **Signing**: Official releases are signed with GPG

#### Tag Creation
```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0: Description"

# Sign tag (for official releases)
git tag -s v1.0.0 -m "Release v1.0.0: Description"

# Push tags
git push origin v1.0.0
```

### 5. Go Module Versioning

#### Module Path
```
bitbucket.org/lexmata/go-mupdf
```

#### Import Paths
```go
// Main package
import "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"

// Version-specific (for major versions > 1)
import "bitbucket.org/lexmata/go-mupdf/v2/pkg/mupdf"
```

#### Go Version Compatibility
- **Minimum Go Version**: 1.19
- **Tested Versions**: 1.19, 1.20, 1.21, 1.22
- **CI Testing**: Latest 3 Go versions

### 6. Release Notes

#### Format
Each release includes:
- **Summary**: Brief description of release
- **Features**: New functionality added
- **Improvements**: Enhancements to existing features
- **Bug Fixes**: Issues resolved
- **Breaking Changes**: API changes requiring user action
- **Dependencies**: Updated dependencies
- **Migration Guide**: How to upgrade from previous version

#### Channels
- **CHANGELOG.md**: Detailed technical changelog
- **GitHub Releases**: User-friendly release notes
- **Documentation**: Version-specific documentation

### 7. Version Support

#### Long-Term Support (LTS)
- **Current**: Always supported with bug fixes
- **Previous Major**: Supported for critical security fixes
- **Older Versions**: Community support only

#### Security Updates
- **Critical**: Immediate patch releases
- **High**: Patch within 1 week
- **Medium**: Included in next scheduled release

### 8. Dependencies

#### MuPDF Version Policy
- **Major Releases**: May update MuPDF major version
- **Minor Releases**: May update MuPDF minor version
- **Patch Releases**: Only MuPDF patch updates

#### Go Version Policy
- **Support**: Latest 3 Go versions
- **Minimum**: Updated annually or as needed
- **Testing**: CI tests against all supported versions

### 9. Release Checklist

#### Pre-Release
- [ ] All tests pass
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] VERSION file updated
- [ ] Breaking changes documented
- [ ] Migration guide written (if needed)

#### Release
- [ ] Version commit created
- [ ] Git tag created and pushed
- [ ] Release notes published
- [ ] Documentation deployed
- [ ] Announcement posted

#### Post-Release
- [ ] Monitor for issues
- [ ] Update dependent projects
- [ ] Plan next release
- [ ] Archive old versions (if applicable)

### 10. Emergency Releases

#### Hotfix Process
1. **Create Hotfix Branch**: From latest release tag
2. **Fix Issue**: Implement minimal fix
3. **Test**: Verify fix works
4. **Release**: Create patch release immediately
5. **Backport**: Merge fix to develop branch

#### Security Releases
- **Priority**: Highest priority releases
- **Timeline**: Within 24-48 hours of discovery
- **Communication**: Security advisory with fix
- **Coordination**: Responsible disclosure process

## Version History

See [CHANGELOG.md](../CHANGELOG.md) for the full history. Highlights:

### v1.8.0 (2026-08-09)
- Fixed a use-after-free in `Document.AsPDFDocument` / `PDFDocument.Close`
- `PDFWriter.AddPage` links pages into the page tree and creates genuinely
  blank pages (no more placeholder text)
- Uniform closed-object handling across the API
- See the "Behavior Changes" section of the changelog before upgrading

### v1.0.0 (2024-10-25)
- Initial production-ready release
- Complete modular architecture
- Comprehensive documentation
- Memory-safe operations

## Tools and Automation

### Version Management
- **Conventional Commits**: Automated changelog generation
- **Semantic Release**: Automated version bumping
- **Git Hooks**: Pre-commit version validation

### CI/CD Integration
- **Automated Testing**: All supported Go versions
- **Release Builds**: Automated binary builds
- **Documentation**: Automated doc deployment
- **Security Scanning**: Automated vulnerability checks

This versioning strategy ensures predictable, professional releases while maintaining backward compatibility and clear communication with users.