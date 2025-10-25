# Contributing to Go MuPDF Wrapper

Thank you for your interest in contributing! This guide will help you get started with contributing to the Go MuPDF wrapper project.

## Table of Contents

- [Development Setup](#development-setup)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Documentation](#documentation)

## Development Setup

### Prerequisites

- Go 1.19 or later
- C compiler (GCC, Clang, or MSVC)
- Make build system
- Git with submodule support

### Setting Up Development Environment

1. **Fork and clone the repository**:
```bash
git clone https://github.com/your-username/go-mupdf.git
cd go-mupdf
git submodule update --init --recursive
```

2. **Build MuPDF**:
```bash
cd third_party/mupdf
make
cd ../..
```

3. **Verify setup**:
```bash
go build ./pkg/mupdf/
go test ./pkg/mupdf/ -v
```

4. **Install development tools**:
```bash
# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Code formatting
go install golang.org/x/tools/cmd/goimports@latest

# Documentation
go install golang.org/x/tools/cmd/godoc@latest
```

## Code Standards

### Go Style Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use [gofmt](https://golang.org/cmd/gofmt/) for formatting
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use meaningful names for variables, functions, and types

### Code Organization

- Keep modules focused and cohesive
- Maintain clear separation of concerns
- Follow the established file organization pattern:
  - `*_test.go` files should mirror source file names
  - Group related functionality in logical modules
  - Keep public APIs minimal and well-documented

### Documentation Standards

All public APIs must include comprehensive godoc comments:

```go
// NewContext creates a new MuPDF execution context.
//
// This initializes the MuPDF library state and registers document handlers
// for supported file formats. The context manages memory allocation and 
// error handling for all subsequent operations.
//
// Returns:
//   - *Context: A new context ready for use
//   - error: An error if context creation fails
//
// Example:
//
//	ctx, err := mupdf.NewContext()
//	if err != nil {
//	    return err
//	}
//	defer ctx.Drop()
func NewContext() (*Context, error) {
    // Implementation...
}
```

### Memory Management

- Always provide explicit cleanup methods (Close(), Drop())
- Include finalizers as safety nets
- Add null pointer checks in all cleanup functions
- Document resource lifecycle clearly

```go
// ✅ Good: Proper resource management
func (page *Page) Close() {
    if page.page != nil && page.ctx != nil && page.ctx.ctx != nil {
        C.fz_drop_page(page.ctx.ctx, page.page)
        page.page = nil
    }
}

// ✅ Good: Finalizer as safety net
result := &Page{ctx: ctx, page: page}
runtime.SetFinalizer(result, func(p *Page) {
    if p != nil {
        p.Close()
    }
})
```

### Error Handling

- Always return descriptive errors
- Use error wrapping with `fmt.Errorf` and `%w` verb
- Handle C library errors properly

```go
// ✅ Good: Descriptive error handling
doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return fmt.Errorf("failed to open document %q: %w", filename, err)
}

// ✅ Good: C error handling
if cError != nil {
    defer C.free(unsafe.Pointer(cError))
    return Error{message: C.GoString(cError)}
}
```

## Testing Guidelines

### Test Organization

Tests should be organized to mirror the source code structure:

- **Core module tests**: `context_test.go`, `document_test.go`, etc.
- **Infrastructure tests**: `memory_test.go`, `lifecycle_test.go`, etc.
- **Quality assurance**: `edge_test.go`, `concurrent_test.go`, etc.

### Writing Tests

1. **Test naming**: Use descriptive test names
```go
func TestContext_Drop_MultipleCallsSafe(t *testing.T) { ... }
func TestDocument_LoadPage_InvalidPageNumber(t *testing.T) { ... }
```

2. **Test structure**: Follow Arrange-Act-Assert pattern
```go
func TestDocument_CountPages_ValidDocument(t *testing.T) {
    // Arrange
    ctx, err := NewContext()
    require.NoError(t, err)
    defer ctx.Drop()
    
    pdfPath := createTestPDF(t, 3) // Helper creates 3-page PDF
    
    // Act
    doc, err := OpenDocument(ctx, pdfPath)
    require.NoError(t, err)
    defer doc.Close()
    
    count := doc.CountPages()
    
    // Assert
    assert.Equal(t, 3, count)
}
```

3. **Use test helpers**: Leverage existing helpers or create new ones
```go
func TestSomething(t *testing.T) {
    requireMuPDF(t)  // Skip if MuPDF not available
    
    ctx := createTestContext(t)  // Helper with cleanup
    pdfPath := createTestPDF(t, pageCount)  // Helper creates test PDF
    
    // Test implementation...
}
```

4. **Test edge cases**:
```go
func TestDocument_LoadPage_EdgeCases(t *testing.T) {
    ctx := createTestContext(t)
    doc := openTestDocument(t, ctx)
    
    testCases := []struct {
        name     string
        pageNum  int
        wantErr  bool
    }{
        {"negative_page", -1, true},
        {"zero_page", 0, false},
        {"last_page", doc.CountPages()-1, false},
        {"beyond_last", doc.CountPages(), true},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            page, err := doc.LoadPage(tc.pageNum)
            if tc.wantErr {
                assert.Error(t, err)
                assert.Nil(t, page)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, page)
                page.Close()
            }
        })
    }
}
```

### Test Coverage

- Aim for >80% test coverage
- Include both positive and negative test cases
- Test error conditions and edge cases
- Validate memory management and cleanup

```bash
# Check coverage
go test ./pkg/mupdf/ -cover

# Generate detailed coverage report
go test ./pkg/mupdf/ -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Benchmarks

Include benchmarks for performance-critical operations:

```go
func BenchmarkDocument_CountPages(b *testing.B) {
    ctx, _ := NewContext()
    defer ctx.Drop()
    
    doc := openTestDocument(b, ctx)
    defer doc.Close()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = doc.CountPages()
    }
}
```

## Pull Request Process

### Before Submitting

1. **Run all tests**:
```bash
go test ./pkg/mupdf/ -v
go test ./pkg/mupdf/ -race  # Check for race conditions
```

2. **Check code quality**:
```bash
golangci-lint run ./pkg/mupdf/
gofmt -d .
goimports -d .
```

3. **Verify documentation**:
```bash
godoc -http=:6060  # Check generated docs
```

4. **Update coverage**:
```bash
go test ./pkg/mupdf/ -cover
# Ensure coverage doesn't decrease significantly
```

### PR Guidelines

1. **Clear description**: Explain what the PR does and why
2. **Link issues**: Reference related issues with "Fixes #123"
3. **Small focused changes**: Keep PRs focused and reviewable
4. **Tests included**: Add tests for new functionality
5. **Documentation updated**: Update docs for API changes

### PR Template

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] New tests added for new functionality
- [ ] Coverage maintained or improved

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or breaking changes documented)
```

## Issue Reporting

### Bug Reports

Use the bug report template:

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Create context with '...'
2. Open document '...'
3. Call method '...'
4. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g. Ubuntu 20.04]
- Go version: [e.g. 1.19.2]
- MuPDF version: [from GetVersion()]

**Additional context**
Any other context about the problem.
```

### Feature Requests

Use the feature request template:

```markdown
**Is your feature request related to a problem?**
A clear description of what the problem is.

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Any alternative solutions or features you've considered.

**Additional context**
Any other context or screenshots about the feature request.
```

## Documentation

### API Documentation

- All public functions and types must have godoc comments
- Include examples in documentation
- Document error conditions and edge cases
- Specify parameter constraints and return value meanings

### Guides and Tutorials

When adding new features, consider updating:

- `docs/GETTING_STARTED.md` for basic usage
- `docs/EXAMPLES.md` for practical examples
- `docs/API_REFERENCE.md` for detailed API docs
- `docs/BEST_PRACTICES.md` for recommended patterns

### README Updates

Keep the main README.md updated when:

- Adding new major features
- Changing installation procedures
- Updating test organization
- Modifying file structure

## Development Workflow

### Branch Strategy

- `main`: Stable, production-ready code
- `develop`: Integration branch for new features
- `feature/xyz`: Feature development branches
- `fix/xyz`: Bug fix branches

### Commit Messages

Follow conventional commit format:

```
type(scope): brief description

Longer description if needed

Fixes #123
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

Examples:
```
feat(pdf): add support for PDF object creation
fix(memory): prevent segfault in Page.Close()
docs(api): update Context documentation with examples
test(edge): add tests for invalid page numbers
```

### Release Process

1. Update version numbers
2. Update CHANGELOG.md
3. Run full test suite
4. Create release tag
5. Update documentation

## Getting Help

- Check existing issues and documentation first
- Ask questions in GitHub discussions
- For complex changes, open an issue for discussion before implementation
- Join the community chat (if available)

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributor graphs

Thank you for contributing to the Go MuPDF wrapper!