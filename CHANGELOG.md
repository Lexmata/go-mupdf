# Changelog

All notable changes to the Go MuPDF wrapper project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.9.1] - 2026-08-10

### 🐛 Bug Fixes
- **CI**: Regenerated `mupdf.lock`, which had gone stale in 1.9.0 (the submodule
  pointer moved to MuPDF 1.27.2 but the separate cache-key pin didn't), causing
  `check-mupdf-lock.sh` to fail "Build MuPDF Libraries" on every branch. The
  1.9.0 tag's release pipeline failed before producing any artifacts — use
  1.9.1, not 1.9.0.

## [1.9.0] - 2026-08-10

### 🔄 Dependencies
- **MuPDF**: Updated from 1.26.3 to 1.27.2
  - See [MuPDF CHANGES](third_party/mupdf/CHANGES) for the upstream changelog
  - Verified `AddBookmarks` against an ObjStm-compressed PDF (descending-page-order
    siblings, `qpdf --check`, byte-identical text extraction, outline round-trip) on
    both 1.26.3 and 1.27.2 — both pass identically, confirming this bump does not
    change behavior for that path

## [1.8.1] - 2026-08-10

Supersedes 1.8.0. The 1.8.0 tag exists but its release build failed before
publishing any artifacts, so 1.8.1 is the first usable 1.8.x release. Use
1.8.1 rather than 1.8.0.

### 🐛 Bug Fixes

- Fixed a data race between the GC finalizer goroutine and `Context.Drop`.
  Cleanup reaches MuPDF from two goroutines — the one that owns the object,
  and the finalizer goroutine running the documented safety net — and every
  `Close`/`Drop` tested `ctx.ctx` without synchronisation while
  `Context.Drop` wrote it. Beyond the reported race, a finalizer could pass
  a context that `Drop` was midway through freeing to MuPDF.

  `Context` now carries a mutex, and all seven cleanup paths (`Document`,
  `PDFDocument`, `Page`, `PDFPage`, `TextPage`, `PDFWriter`, `PDFObject`) go
  through a `withLock` helper that makes the "still alive?" check and the
  use of the context a single atomic step. Holding the lock across the cgo
  call also prevents two goroutines from entering MuPDF at once, which
  matters because MuPDF is built here in single-threaded mode.

  The `PDFDocument` case was a regression introduced in 1.8.0, which gave
  `PDFDocument` a finalizer that actually frees (previously a no-op). The
  other six were the same latent pattern and are fixed with the same
  mechanism.

- Cleanup now clears its object's pointer even when the `Context` was
  dropped first. `fz_drop_context` already released the object, so keeping
  the pointer left a dangling reference for later calls to pick up.

### ✨ Added

- `pkg/mupdf/finalizer_race_test.go` — regression coverage for the above.
  Verified by reverting the fix: the deterministic test reports
  `DATA RACE ... (*Context).Drop()` against each of the affected types, and
  passes once the fix is restored.

## [1.8.0] - 2026-08-09

Minor version bump rather than a patch: this release changes several
observable behaviours. Every change below is intentional, but code written
against 1.4.x may need adjusting.

### ⚠️ Behavior Changes

- **`PDFWriter.AddPage` now creates a genuinely blank page.** Previously every
  added page was stamped with the placeholder text `"Page Content"`, which was
  never real document content. Pages built with `AddPage` now contain an empty
  content stream, so `ExtractText().String()` returns `""` for them. This
  package does not expose an API for writing to a page content stream; use
  pdfcpu or an external tool to add content.
- **Pages added by `AddPage` are linked into the document page tree.** Pages are
  now created via `pdf_add_page` + `pdf_insert_page` instead of a hand-built
  `pdf_page` struct, so `CountPages()` and other page-tree readers see them.
  Documents saved by 1.4.x could report a page count that disagreed with the
  number of `AddPage` calls.
- **`Document.AsPDFDocument` now returns an error for non-PDF documents**
  instead of a `*PDFDocument` wrapping a null pointer. Callers that ignored the
  error and checked for `nil` still work; callers that ignored both will now see
  the error.
- **`Document.AsPDFDocument` returns an independently owned `*PDFDocument`.**
  It now takes its own reference to the underlying document (via
  `fz_new_pdf_document_from_fz_document` rather than the non-keeping
  `pdf_document_from_fz_document` down-cast). Consequences:
  - The returned `*PDFDocument` **must** be closed with `Close()`, or the
    document is leaked. A finalizer is a safety net, not a substitute.
  - Closing the `*PDFDocument` no longer invalidates the parent `Document`, and
    closing the parent no longer frees the `*PDFDocument`. In 1.4.x, closing
    either one freed the shared document — a use-after-free in the other.
  - Covered by a regression test in `pkg/mupdf/pdf_refcount_test.go`.
- **`PDFWriter.AddPage` rejects non-positive dimensions.** `width` and `height`
  must both be `> 0`; previously zero or negative values produced a page with a
  degenerate MediaBox.
- **`AddPage`, `ImprovedAddPage`, `FixedAddPage`, `SimpleAddPage`,
  `NewPDFObject`, `DebugCountPages`, `Document.LoadPage`, and `Page.Bound` now
  check for a closed receiver** and return an error (or the zero value) instead
  of dereferencing a freed pointer. See the "Closed Objects" section of the
  package documentation for the uniform convention.
- **`ExtractPages` merges multiple page ranges into the single output file.**
  pdfcpu writes one single-page PDF per extracted page; those are now combined
  in ascending page order into `outputPath`. Previously the output depended on
  filesystem glob ordering and could contain only part of the requested range.
  Extraction also uses a private temporary directory, so concurrent calls
  sharing an output directory no longer merge each other's pages.
- **`DefaultPDFCPUConfig()` returns a populated configuration**
  (`model.NewDefaultConfiguration()`) rather than an empty struct, and each
  operation now works on a private copy of the caller's configuration. pdfcpu
  mutates the configuration it is handed (`api.Merge` sets both `Cmd` and
  `ValidationMode`), so a shared `*PDFCPUConfig` could previously have its
  settings silently altered between calls.
- **`Page.Bound` reports the effective page box** (CropBox intersected with
  MediaBox, as MuPDF computes it) and falls back to reading the raw MediaBox —
  including real-valued entries — only when bounding yields an empty rect.
  MediaBoxes with non-integer coordinates were previously truncated.

### 🐛 Bug Fixes

- Fixed a use-after-free in `AsPDFDocument`/`PDFDocument.Close` (see above).
- Added `fz_var` declarations for locals assigned inside `fz_try` blocks and
  read afterwards (`pdf`, `buf`, `rect`); without them their values are
  indeterminate after a MuPDF longjmp.
- `TextPage.String` clamps the buffer length before converting to a Go string
  instead of silently truncating lengths that exceed `C.int` range.
- `EncryptPDF` no longer downgrades to RC4-40 when handed a zero-valued
  `EncryptKeyLength`; it normalizes to AES-256.
- `GetPDFInfo` returns an error instead of panicking when the PDF has no
  cross-reference table.
- `RotatePages`' documented signature in README now matches the code
  (`pageRanges` precedes `rotation`).
- `scripts/release.sh`: `version_greater` now implements semver pre-release
  precedence. `sort -V` orders `1.4.7` before `1.4.7-rc.1`, the inverse of
  semver, so `--pre` from a plain release previously produced a version that
  ranked *below* the shipped one; it now bumps the patch first
  (`1.4.7` → `1.4.8-rc.1`). Breaking-change commit forms (`feat!:`, `fix!:`) are
  no longer misfiled under "Changed" in the generated changelog.
- `.githooks/pre-push`: when the pushed range cannot be determined (new branch,
  shallow clone, no stdin) the test gate now runs the suite unconditionally
  instead of skipping it — it fails closed.
- `scripts/docker-test.sh`: pass-through arguments after `--` are kept as an
  array, so quoted values such as `-run 'TestA|TestB'` survive as single
  arguments; `quick` and `coverage` accept pass-through arguments too.

### 🔒 Security

- Codecov uploads no longer pipe an unpinned remote script into a shell. The new
  `scripts/upload-coverage.sh` downloads a pinned uploader release and verifies
  its SHA-256 checksum before executing it.

### ✨ Added

- `scripts/check-mupdf-lock.sh` verifies that `mupdf.lock` matches the MuPDF
  submodule pointer. The pin is part of the CI cache key for the pre-built
  libraries, and a stale pin silently serves libraries built from a different
  MuPDF commit. Now enforced by both the pre-commit hook and the `build-mupdf`
  pipeline step.
- `pkg/mupdf/pdf_refcount_test.go` — regression coverage for the
  `AsPDFDocument`/`Close` reference balance, exercised both explicitly and
  through the finalizer.

### 📚 Documentation

- Corrected the thread-safety guidance in `docs/GETTING_STARTED.md` and
  `docs/BEST_PRACTICES.md`: a `Context` is **not** thread-safe (MuPDF is
  initialised in single-threaded mode), so each goroutine needs its own.
- README now states the real minimum Go version (1.24), documents the
  `AsPDFDocument` lifetime and `AddPage` dimension constraints, and lists the
  current test files.
- `pkg/mupdf/mupdf.go` gained a "Closed Objects" section describing the
  package-wide convention for operations on closed objects.
- Marked `ImprovedAddPage`, `FixedAddPage`, and `SimpleAddPage` as deprecated in
  favour of `AddPage`, and removed the unsupported claim that `FixedAddPage`
  was more stable than it.

## [1.7.2] - 2026-04-28

### 🐛 Bug Fixes
- **MPDF-99: `AddBookmarks` now writes nested children and sibling chains correctly.**
  The outline iterator was misdriven: `fz_outline_iterator_insert` auto-advances
  past the inserted item, but `insertOutlineItems` called `_next` again (skipping
  a slot) and `_down` from the already-advanced position (silently failing to
  descend). As a result only the first top-level sibling was persisted, and all
  children at every level were dropped; mupdf then logged `warning: repaired
  broken tree structure in outline` when reading the PDF back. Fix: after
  inserting an item with children, `_prev` back onto the item before `_down`;
  after `_up` from children, skip the redundant `_next` (since `_insert`'s
  auto-advance already positioned us for the next sibling). Regression tests
  added for nested children round-trip, multiple top-level siblings, and
  alphabetical outline extraction.

## [1.4.7] - 2025-11-20

### 🐛 Bug Fixes
- **Fixed cross-compilation library download for ARM64 in CI/CD** (CRITICAL)
  - Download scripts now respect `GOOS` and `GOARCH` environment variables
  - When cross-compiling with `GOARCH=arm64`, correct ARM64 libraries are downloaded
  - Falls back to `uname` detection for normal (non-cross-compile) usage
  - Fixes issue where CI/CD ARM64 builds downloaded wrong architecture libraries
  - **Impact**: ARM64 cross-compilation builds in CI/CD will now work correctly
  - Affected scripts: `download-libs.sh`, `install-prebuilt-libs.sh`, `setup-mupdf.sh`, `install.sh`

## [1.4.6] - 2025-11-11

### 🐛 Bug Fixes
- **Fixed platform detection in build script for cross-compilation** (CRITICAL)
  - `build-static-libs.sh` now respects `GOOS` and `GOARCH` environment variables
  - Previously used `uname` which always returned host architecture (amd64)
  - **This caused ARM64 builds to create and upload `linux-amd64` packages, overwriting the real AMD64 build**
  - ARM64 packages were being uploaded with wrong architecture name
  - Now correctly creates `linux-arm64` packages when `GOARCH=arm64` is set
  - Both AMD64 and ARM64 builds will now upload correctly without conflicts

## [1.4.5] - 2025-11-11

### 🐛 Bug Fixes
- **Fixed architecture detection in download script** (CRITICAL)
  - `download-libs.sh` now correctly detects host architecture using `uname -m`
  - Previously used `go env GOARCH` which could be affected by cross-compilation env vars
  - Fixes issue where ARM64 libraries were downloaded on x86_64/amd64 systems
  - Added diagnostic output showing detected OS, architecture, and platform
  - **Impact**: Users with `GOARCH` or `GOOS` env vars set will now get correct libraries
  - Resolves linker errors: `skipping incompatible libmupdf.a`

### ♻️ Removed
- **Removed Darwin (macOS) cross-compiled builds**
  - OSXCross builds removed from pipeline
  - macOS users will build from source automatically (5-10 minutes)
  - Linux AMD64 and ARM64 pre-built libraries remain available
  - Does not affect functionality - source build works perfectly

## [1.4.4] - 2025-11-11

### ✨ Added
- **Cross-compiled Darwin builds!** macOS binaries now build automatically
  - Using OSXCross toolchain (`crazymax/osxcross:latest`) to cross-compile from Linux
  - darwin-amd64 and darwin-arm64 now fully automated
  - No need for macOS runners or GitHub Actions
  - All four major platforms now have instant downloads!
  - Builds run in parallel alongside Linux builds (~12 minutes total)

## [1.4.3] - 2025-11-11

### ♻️ Removed
- **Removed redundant "Build Release Artifacts" step**
  - Source archives available directly from Bitbucket/Git
  - Documentation accessible in repository
  - Focus on core value: pre-built MuPDF libraries
  - Saves ~5-8 minutes per release
  - Reduces artifact storage and complexity

## [1.4.2] - 2025-11-11

### 🐛 Fixed
- **Parallel Builds!** All four platform builds now run simultaneously
  - Reduced total build time from ~40 minutes to ~15 minutes
  - AMD64, ARM64, Darwin AMD64, and Darwin ARM64 all build in parallel
  - Fixed YAML indentation to properly configure parallel execution

## [1.4.1] - 2025-11-11

### ✨ Added
- **Full Multi-Platform Support!** All major platforms now have automated builds
  - ✅ **linux-amd64** - Automated builds with pre-built libraries
  - ✅ **linux-arm64** - Automated cross-compilation builds
  - ✅ **darwin-amd64** - Automated Intel Mac builds (Bitbucket Premium)
  - ✅ **darwin-arm64** - Automated Apple Silicon builds (Bitbucket Premium)
  - Added comprehensive macOS build documentation (`docs/MACOS_BUILDS.md`)

### 📝 Changed
- Enabled native macOS runners in CI/CD pipeline (requires Bitbucket Premium)
- All four major platforms now support instant library downloads
- Updated platform support documentation across README and docs

## [1.4.0] - 2025-11-10

### 🚀 Improved
- **Simplified Installation!** Pre-built libraries auto-download on setup
  - `make setup` tries pre-built download first, falls back to source build
  - Downloads from CI/CD artifacts to project's `third_party/mupdf/`
  - Takes <10 seconds for supported platforms (vs 5-10 minutes for source build)
  - Graceful fallback for unsupported platforms

### ✨ Added
- `scripts/download-libs.sh` - Downloads pre-built libraries from Bitbucket CI/CD
- `scripts/install.sh` - Smart setup script (download first, build as fallback)

### 📝 Changed
- `make setup` now intelligently tries download before building
- Updated README with clearer setup instructions
- Documented that go-mupdf requires one-time setup like other Go+CGO projects
- Updated year references from 2024 to 2025

### 💡 Why This Approach?
- ✅ Instant setup for linux/amd64 (most common platform)
- ✅ No repository bloat (libraries downloaded, not committed)
- ✅ Works with both cloned repos and go get workflows
- ✅ Automatic fallback for unsupported platforms/developers
- ✅ Leverages existing CI/CD infrastructure

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