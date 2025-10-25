# Go MuPDF Wrapper - Architecture Overview

This document describes the architecture and design principles of the Go MuPDF wrapper.

## Overview

The Go MuPDF wrapper is designed as a safe, idiomatic Go interface to the MuPDF C library. It provides memory-safe operations, comprehensive error handling, and follows Go best practices.

## Design Principles

### 1. Memory Safety
- **Automatic cleanup** via Go finalizers as a safety net
- **Explicit resource management** with Close()/Drop() methods
- **Null pointer protection** throughout the codebase
- **Resource lifecycle tracking** in tests

### 2. Go Idioms
- **Standard error handling** with descriptive error messages
- **Interface segregation** with focused, single-purpose types
- **Clear ownership** of resources and dependencies
- **Proper package organization** with logical module separation

### 3. C Integration
- **Safe CGO usage** with proper error handling
- **C memory management** handled transparently
- **Type safety** between Go and C boundaries
- **Exception handling** from MuPDF C library

## Module Architecture

### Core Modules

```
pkg/mupdf/
├── mupdf.go           # Package documentation and overview
├── types.go           # Core types (Error, Rect)
├── context.go         # MuPDF context management
├── document.go        # Document operations
├── page.go           # Page operations
├── text.go           # Text extraction
├── pdf.go            # PDF-specific functionality
├── pdf_debug.go      # Debug utilities
├── pdf_fix.go        # Fixed implementations
├── pdf_simple.go     # Simplified implementations
└── test_helpers.go   # Testing utilities
```

### Dependency Graph

```
┌─────────────┐
│   mupdf.go  │ ── Package Documentation
└─────────────┘

┌─────────────┐
│   types.go  │ ── Core Types (Error, Rect)
└─────────────┘
       ▲
       │
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ context.go  │ ── │ document.go │ ── │   page.go   │
└─────────────┘    └─────────────┘    └─────────────┘
       ▲                   ▲                   ▲
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                    ┌─────────────┐
                    │   text.go   │
                    └─────────────┘
                           ▲
                           │
┌─────────────┐    ┌─────────────┐
│   pdf.go    │ ── │ pdf_*.go    │ ── Specialized PDF implementations
└─────────────┘    └─────────────┘
       ▲
       │
┌─────────────┐
│test_helpers │ ── Testing utilities
└─────────────┘
```

## Type Hierarchy

### Core Types

```go
// Foundation types
type Error struct        // MuPDF error representation
type Rect struct        // Geometric rectangle

// Resource management
type Context struct     // MuPDF execution context
```

### Document Types

```go
// Generic document handling
type Document struct    // Any supported document format
type Page struct       // Generic page from any document
type TextPage struct   // Extracted text from any page

// PDF-specific types  
type PDFDocument struct // PDF with extended functionality
type PDFPage struct    // PDF page with PDF-specific operations
type PDFWriter struct  // PDF document creation
type PDFObject struct  // PDF object tree manipulation
```

### Type Relationships

```
Context
├── Document (1:N)
│   ├── Page (1:N)
│   │   └── TextPage (1:1)
│   └── PDFDocument (1:1) [if PDF]
│       └── PDFPage (1:N)
└── PDFWriter (1:N)
    ├── PDFPage (1:N)
    └── PDFObject (1:N)
```

## Memory Management Model

### Resource Ownership

```go
// Parent-child relationships determine ownership
Context
├── owns Document instances
├── owns PDFWriter instances
└── manages C library context

Document
├── owns Page instances
├── references parent Context
└── manages C document structure

Page
├── owns TextPage instances
├── references parent Document/Context
└── manages C page structure
```

### Cleanup Strategy

1. **Explicit Cleanup (Recommended)**
```go
ctx, _ := mupdf.NewContext()
defer ctx.Drop()              // Explicit cleanup

doc, _ := mupdf.OpenDocument(ctx, "file.pdf")
defer doc.Close()             // Explicit cleanup
```

2. **Automatic Cleanup (Safety Net)**
```go
// Finalizers automatically cleanup unreferenced objects
runtime.SetFinalizer(obj, (*Type).cleanup)
```

3. **Cleanup Order**
```go
// Children must be cleaned before parents
TextPage.Close()    // First
Page.Close()        // Then
Document.Close()    // Then  
Context.Drop()      // Last
```

## Error Handling Architecture

### Error Types

```go
// Primary error type
type Error struct {
    message string  // From MuPDF C library
}

// Error sources
├── Context creation errors
├── Document parsing errors  
├── Page loading errors
├── Text extraction errors
├── PDF operation errors
└── Memory allocation errors
```

### Error Propagation

```go
// C library errors → Go Error type
C.fz_try(ctx) {
    // MuPDF operation
}
C.fz_catch(ctx) {
    error_msg := C.fz_caught_message(ctx)
    return Error{message: C.GoString(error_msg)}
}

// Go Error type → Standard Go error interface
func (e Error) Error() string {
    return e.message
}
```

## Concurrency Model

### Thread Safety

- **Context**: Thread-safe, can be shared between goroutines
- **Document**: Not thread-safe, don't share between goroutines  
- **Page**: Not thread-safe, don't share between goroutines
- **PDFWriter**: Not thread-safe, don't share between goroutines

### Recommended Patterns

```go
// Pattern 1: Separate contexts per goroutine
func processDocumentsConcurrently(files []string) {
    var wg sync.WaitGroup
    for _, file := range files {
        wg.Add(1)
        go func(filename string) {
            defer wg.Done()
            
            ctx, _ := mupdf.NewContext()  // Separate context
            defer ctx.Drop()
            
            processDocument(ctx, filename)
        }(file)
    }
    wg.Wait()
}

// Pattern 2: Shared context with synchronized access
func processWithSharedContext(files []string) {
    ctx, _ := mupdf.NewContext()
    defer ctx.Drop()
    
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    for _, file := range files {
        wg.Add(1)
        go func(filename string) {
            defer wg.Done()
            
            mu.Lock()
            doc, _ := mupdf.OpenDocument(ctx, filename)
            mu.Unlock()
            
            defer doc.Close()
            processDocument(doc)
        }(file)
    }
    wg.Wait()
}
```

## Testing Architecture

### Test Organization

```
tests/
├── Core Module Tests/
│   ├── context_test.go     # Context functionality
│   ├── document_test.go    # Document operations
│   ├── page_test.go        # Page operations
│   ├── text_test.go        # Text extraction
│   └── types_test.go       # Core types
├── PDF Module Tests/
│   ├── pdf_writer_test.go  # PDF creation
│   ├── pdf_document_test.go # PDF operations
│   ├── pdf_objects_test.go # PDF objects
│   └── pdf_modify_test.go  # PDF modifications
├── Infrastructure Tests/
│   ├── memory_test.go      # Memory management
│   ├── lifecycle_test.go   # Resource lifecycle
│   └── cleanup_test.go     # Cleanup validation
└── Quality Assurance/
    ├── boundary_test.go    # API boundaries
    ├── edge_test.go        # Edge cases
    ├── concurrent_test.go  # Concurrency
    └── stress_test.go      # Stress testing
```

### Test Strategy

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **Memory Tests**: Validate resource management
4. **Concurrency Tests**: Verify thread safety
5. **Stress Tests**: Test under load conditions

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Context creation | O(1) | One-time setup cost |
| Document opening | O(1) | File I/O dependent |
| Page loading | O(1) | Lazy loading |
| Page counting | O(1) | Cached after first call |
| Text extraction | O(n) | n = page content size |

### Memory Usage

| Component | Memory Pattern | Cleanup |
|-----------|----------------|---------|
| Context | Fixed allocation | Drop() |
| Document | Proportional to file size | Close() |
| Page | Proportional to page content | Close() |
| TextPage | Proportional to text content | Close() |

### Optimization Strategies

1. **Reuse contexts** for multiple operations
2. **Close pages immediately** in loops
3. **Process documents in batches** for large sets
4. **Use goroutines** for independent documents
5. **Force GC** periodically for long-running processes

## Future Architecture Considerations

### Potential Enhancements

1. **Streaming API** for very large documents
2. **Page caching** for frequently accessed pages
3. **Async operations** for I/O bound tasks
4. **Plugin architecture** for custom document formats
5. **Memory pooling** for high-throughput scenarios

### Extensibility Points

1. **Custom error types** for specific error conditions
2. **Middleware pattern** for operation hooks
3. **Event system** for monitoring and logging
4. **Configuration system** for runtime behavior
5. **Metrics collection** for performance monitoring

This architecture provides a solid foundation for PDF processing while maintaining safety, performance, and Go idioms.