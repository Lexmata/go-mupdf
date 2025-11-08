# Changelog

All notable changes to the Go MuPDF wrapper project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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