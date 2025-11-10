# Changelog

All notable changes to the Go MuPDF wrapper project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.2] - 2025-11-10

### 🐛 Fixed
- **Critical**: Fixed `go get` workflow to use `go generate` for MuPDF setup
  - Converted `setup.go` from init-based to standalone script
  - CGO linking happens at compile-time, before init() runs
  - Users must now run `go generate ./pkg/mupdf` before building
  - Creates `generate.go` with `//go:generate go run setup.go` directive

### 📝 Changed
- `setup.go` is now a standalone script with `//go:build ignore`
- Added `generate.go` with go:generate directive for automatic setup
- Documentation updated with clear `go get` workflow using `go generate`

### ⚠️ Breaking Changes
- `go get` users must run `go generate` before building (see README)
- Workflow: `go get` → navigate to module dir → `go generate ./pkg/mupdf` → build

## [1.3.1] - 2025-11-10

### 🐛 Fixed
- Attempted to fix `go get` installation with init-based approach
  - MuPDF uses custom versions of dependencies (e.g., lcms2 multi-threaded fork)
  - These are incompatible with system libraries, requiring bundled submodules
  - `setup.go` clones MuPDF repository with `--recurse-submodules`
  - Builds with `USE_SYSTEM_LIBS=no` to ensure bundled dependencies are used

### 📝 Changed
- Reverted from tarball approach back to git clone with submodules
- Git is now required for installations (for submodule support)

### ⚠️ Note
- v1.3.1 init() approach doesn't work due to CGO compile-time requirements - use v1.3.2+

### ⚠️ Requirements
- **git** is now required for installation (to clone submodules)
- **make**, **gcc/clang** required for building
- System libraries (zlib, freetype, etc.) can be used but MuPDF's bundled versions are preferred

## [1.3.0] - 2025-11-10

### 🚀 Improved
- Automatic setup now runs in `pkg/mupdf/setup.go` init() function (works with `go get`)
- `build.go` deprecated (kept for reference only)

### ⚠️ Note
- v1.3.0 had issues with `go get` due to tarball approach - use v1.3.1 or later

## [1.2.7] - 2025-11-10

### 🐛 Fixed
- **Critical**: Fixed `libmupdf-third.a` not being built correctly in `build.go` for downstream consumers
  - The automatic build process now explicitly calls `make libs` with required flags
  - Added verification to ensure both `libmupdf.a` and `libmupdf-third.a` are created
  - Fixes "cannot find -lmupdf-third" linker errors when using `go get`
  - Build now uses: `USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release libs`
  - Added comprehensive error messages to help users troubleshoot build issues

### 📚 Documentation
- **Added**: `docs/BUILD_SYSTEM.md` - Comprehensive build system documentation
  - Explains how the build process works for different use cases
  - Documents the critical importance of `libmupdf-third.a`
  - Provides troubleshooting guide for common build issues
  - Details build flags and their purposes
- **Added**: `.cursor/rules/no-unsolicited-markdown.mdc` - Cursor rule preventing automatic markdown file generation

## [1.2.0] - 2024-11-09

### ✨ Added

#### 📦 Static Library Distribution System
- **Pre-built Library Distribution**: Complete system for building and distributing pre-compiled MuPDF libraries
  - `build-static-libs.sh` - Automated script to build distributable MuPDF packages
  - Platform-specific distribution packages (Linux, macOS, Windows - amd64, arm64)
  - Includes `libmupdf.a`, `libmupdf-third.a`, headers, and documentation
  - SHA256 checksums for package verification
  - Comprehensive usage documentation and README included in packages
  - Automatic CI/CD integration for release builds

#### ⚡ CI/CD Pipeline Optimization
- **Artifact-based Build System**: Revolutionary pipeline optimization reducing build times by 50-75%
  - `build-mupdf-artifact.sh` - Builds MuPDF once and creates reusable artifacts
  - `install-prebuilt-libs.sh` - Installs pre-built libraries with fallback to source build
  - Smart caching system keyed to MuPDF submodule version
  - Artifact sharing across all pipeline steps
  - **Performance Improvements**:
    - First run: 14-20 minutes (vs 30-36 minutes before) - **50-60% faster**
    - Cached run: 7-11 minutes (vs 30-36 minutes before) - **70-75% faster**
  - Applied to all pipelines: default, main, tags, and pull-requests
  - Parallel steps now truly parallel (no redundant MuPDF builds)

#### 🚀 Automatic Library Setup for Users
- **setup-mupdf.sh**: One-command setup script for developers
  - Automatic platform detection (Linux, macOS, Windows)
  - Downloads pre-built libraries from Bitbucket Downloads (< 1 minute)
  - Graceful fallback to source build if pre-built unavailable
  - Verifies library installation and integrity
  - Integrated with Makefile: `make setup` command
  - **User Experience**: Setup time reduced from 10-15 minutes to < 1 minute

#### 📚 Comprehensive Documentation
- **CI/CD Optimization Guide** (`docs/CI_CD_OPTIMIZATION.md`)
  - Architecture and flow diagrams
  - Performance metrics and benchmarks
  - Troubleshooting guide
  - Advanced usage and best practices
- **Maintainer Guide** (`docs/MAINTAINER_GUIDE.md`)
  - Release process documentation
  - Distribution package building
  - Multi-platform builds
  - Testing procedures
- **Static Library Distribution Guide** (`docs/STATIC_LIBRARY_DISTRIBUTION.md`)
  - Building distribution packages
  - Platform support details
  - Using pre-built libraries
  - Hosting options and CI/CD integration

### 🔧 Improved

#### Build System
- **Makefile Enhancement**: Added `make setup` target
  - `make build` now depends on `make setup` for automatic library management
  - `make test` now depends on `make setup` for automatic library management
  - `make dist` - Build static library distribution packages
  - `make dist-clean` - Clean distribution artifacts

#### Installation Process
- **Updated README**: Comprehensive installation guide with 4 options
  - **Option 1**: Quick Setup (recommended) - `make setup` < 1 minute
  - **Option 2**: Pre-built static libraries (manual download)
  - **Option 3**: Using in your own project (`go get` workflow)
  - **Option 4**: Manual build with submodules
- **Better User Experience**: Clear instructions for all use cases

#### CI/CD Pipeline
- **Optimized Pipeline Configuration**: Restructured `bitbucket-pipelines.yml`
  - New `build-mupdf` step runs once at pipeline start
  - All subsequent steps use pre-built artifacts
  - Smart caching with `mupdf-libs` cache
  - Applied to tags pipeline for automatic release distribution building

### 📊 Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| **Local Setup** | 10-15 min | < 1 min | **10-15x faster** |
| **CI/CD (first run)** | 30-36 min | 14-20 min | **50-60% faster** |
| **CI/CD (cached)** | 30-36 min | 7-11 min | **70-75% faster** |
| **User go get** | Manual build | Auto download | **Much easier** |

### 🎯 Benefits

- **Faster Development**: Developers can start contributing in < 1 minute
- **Faster CI/CD**: Pipelines run 2-3x faster with caching
- **Better DX**: One-command setup with automatic library management
- **Easier Distribution**: Pre-built libraries available for download
- **Consistent Builds**: Same libraries across all environments
- **Lower Costs**: Reduced CI/CD resource consumption

## [1.1.0] - 2024-11-08

### ✨ Added

#### 🔧 Build System Improvements
- **Automatic Submodule Download**: Enhanced `build.go` to automatically download MuPDF git submodule when missing
  - Works seamlessly with `go get` installations (no manual submodule initialization needed)
  - Parses `.gitmodules` to extract submodule URL and branch information
  - Falls back to direct `git clone` when standard git submodule commands fail
  - Provides clear error messages with helpful installation guidance
  - Supports both git repository and module cache environments

#### 📦 PDFCPU Integration
- **Complete PDFCPU Integration**: Integrated `github.com/pdfcpu/pdfcpu` library for advanced PDF manipulation
  - **PDF Merging**: `MergePDFs()` - Combine multiple PDF files into one
  - **PDF Splitting**: `SplitPDF()` - Split PDFs by page ranges into multiple files
  - **PDF Encryption**: `EncryptPDF()` - Add password protection with user/owner passwords and permissions
  - **PDF Decryption**: `DecryptPDF()` - Remove password protection from encrypted PDFs
  - **PDF Watermarking**: `AddWatermark()` - Add text or image watermarks to PDFs
  - **PDF Validation**: `ValidatePDF()` - Verify PDF structure and integrity
  - **PDF Optimization**: `OptimizePDF()` - Compress and optimize PDF file size
  - **Page Rotation**: `RotatePages()` - Rotate specific pages or page ranges
  - **Page Extraction**: `ExtractPages()` - Extract specific pages to a new PDF
  - **PDF Metadata**: `GetPDFInfo()` - Retrieve comprehensive PDF information and metadata
- **Configuration Support**: `PDFCPUConfig` struct for fine-grained control over PDF operations
- **Comprehensive Testing**: Full test coverage for all PDFCPU functions with edge case testing

#### 📚 Documentation Updates
- **Installation Guide**: Updated `docs/INSTALLATION.md` with automatic submodule download information
- **README Updates**: Enhanced installation instructions highlighting `go get` support
- **Build Documentation**: Clarified build process and submodule handling

### 🔧 Changed
- **Build Process**: `build.go` now automatically handles submodule initialization for better user experience
- **Installation Method**: `go get` is now the recommended installation method (previously required manual cloning)
- **Error Messages**: Improved error messages in build script with clear guidance for troubleshooting

### 🐛 Fixed
- **Test Suite**: Fixed `ExampleGetVersion` test to match current MuPDF version (1.26.3)
- **PDFCPU Tests**: Fixed `TestSplitPDF` to correctly handle pdfcpu file naming patterns
- **PDFCPU Tests**: Fixed `TestEncryptPDF` to properly handle password requirements
- **PDFCPU Tests**: Fixed `TestGetPDFInfo` to support encrypted PDFs with password configuration

### 🔄 Dependencies
- **Added**: `github.com/pdfcpu/pdfcpu v0.11.1` - Pure Go PDF library for advanced manipulation
- **MuPDF**: 1.26.3 (included as git submodule, automatically downloaded if missing)

## [1.0.0] - 2024-10-25

### 🎉 Initial Release - Production Ready

This marks the first stable release of the Go MuPDF wrapper, representing a complete, production-ready PDF processing library.

#### ✨ Added
- **Core PDF Operations**: Complete document opening, reading, and processing capabilities
- **PDF Creation**: Full PDF document creation with multiple page addition methods
- **Text Extraction**: Comprehensive text extraction from all supported document formats
- **Memory Management**: Safe, automatic resource cleanup with finalizers and explicit methods
- **Error Handling**: Robust error handling with descriptive messages and recovery
- **Thread Safety**: Safe concurrent operations with proper context management

#### 🏗️ Architecture
- **Modular Design**: Clean separation into 6 focused modules (types, context, document, page, text, PDF operations)
- **Professional Structure**: Well-organized codebase following Go best practices
- **Comprehensive APIs**: Complete coverage of MuPDF functionality through idiomatic Go interfaces

#### 📚 Documentation
- **API Reference** (736 lines): Complete documentation for all public APIs with examples
- **Getting Started Guide** (405 lines): Installation, setup, and tutorial for new users
- **Best Practices** (458 lines): Production-ready patterns and optimization techniques
- **Troubleshooting Guide** (522 lines): Comprehensive problem-solving and debugging
- **Architecture Documentation** (345 lines): System design and internal structure
- **Contributing Guide** (453 lines): Development setup and contribution guidelines
- **Examples** (98 lines): Practical usage patterns and code samples
- **Navigation Guide** (139 lines): Documentation organization and quick reference

#### 🧪 Testing
- **81.8% Test Coverage**: Comprehensive testing across all functionality
- **123+ Test Functions**: Thorough testing of all features and edge cases
- **21 Organized Test Files**: Clear test organization mirroring module structure
- **Multiple Test Categories**: Unit, integration, stress, concurrency, and memory tests
- **Quality Assurance**: Boundary testing, edge cases, and error condition validation

#### 🔧 Quality & Standards
- **Lint-Free Code**: All code passes gofmt, go vet, and staticcheck
- **Memory Safety**: Comprehensive null pointer protection and resource management
- **Performance Optimized**: Efficient memory allocation and resource usage
- **Professional Standards**: Enterprise-ready code quality and organization

#### 🎯 Features
- **Multiple Document Formats**: PDF, XPS, EPUB, CBZ, and other formats supported by MuPDF
- **PDF Creation Methods**: Standard AddPage, SimpleAddPage, ImprovedAddPage, and FixedAddPage
- **Page Size Support**: US Letter, A4, A3, Legal, and custom page dimensions
- **Text Processing**: Complete text extraction with UTF-8 support and layout preservation
- **Resource Management**: Automatic and explicit cleanup options for all resource types

#### 🚀 Platform Support
- **Cross-Platform**: Linux, macOS, and Windows support
- **Go Version**: Compatible with Go 1.19 and later
- **MuPDF Version**: Built against MuPDF 1.26.3
- **C Integration**: Safe CGO usage with proper error handling

#### 📊 Project Statistics
- **Source Files**: 10 focused modules (6,927 lines of code)
- **Test Files**: 21 comprehensive test modules
- **Documentation**: 8 professional guides (3,156 lines)
- **API Coverage**: 100% of public APIs documented with examples

### 🔄 Dependencies
- **MuPDF**: 1.26.3 (included as git submodule)
- **Go**: 1.19 or later
- **Build Tools**: C compiler (GCC/Clang/MSVC), Make

### 🎯 Breaking Changes
- This is the initial release, no breaking changes from previous versions

### 🔒 Security
- Memory-safe operations with comprehensive bounds checking
- Resource cleanup prevents memory leaks and resource exhaustion
- Safe handling of malformed or malicious PDF files through MuPDF's robust parser

### 📈 Performance
- Efficient memory management with automatic cleanup
- Optimized resource allocation patterns
- Support for concurrent operations with separate contexts
- High-performance text extraction and document processing

---

## Version Schema

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version when making incompatible API changes
- **MINOR** version when adding functionality in a backwards compatible manner
- **PATCH** version when making backwards compatible bug fixes

### Release Branches
- `main`: Production-ready releases
- `develop`: Integration branch for new features
- `release/x.y.z`: Release preparation branches

### Git Tags
- Format: `vX.Y.Z` (e.g., `v1.0.0`)
- Annotated tags with release notes
- Signed tags for official releases

[Unreleased]: https://bitbucket.org/lexmata/go-mupdf/compare/v1.1.0..HEAD
[1.1.0]: https://bitbucket.org/lexmata/go-mupdf/src/v1.1.0
[1.0.0]: https://bitbucket.org/lexmata/go-mupdf/src/v1.0.0