# Go MuPDF Wrapper - Best Practices

This guide covers best practices for using the Go MuPDF wrapper in production environments.

## Memory Management

### Always Use Defer for Cleanup

```go
// ✅ Good: Immediate defer after creation
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop() // Guaranteed cleanup

doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return err
}
defer doc.Close() // Guaranteed cleanup
```

```go
// ❌ Bad: Missing defer statements
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
// Missing defer - memory leak risk!

doc, err := mupdf.OpenDocument(ctx, filename)
// Missing defer - memory leak risk!
```

### Explicit Cleanup for Long-Running Operations

```go
// ✅ Good: Explicit cleanup in loops
for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        continue
    }
    
    // Process page...
    
    page.Close() // Explicit cleanup in loop
}
```

```go
// ❌ Bad: Relying only on defer in loops
for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        continue
    }
    defer page.Close() // Will accumulate until function ends!
    
    // Process page...
}
```

### Safe Resource Access

```go
// ✅ Good: Check before use
func processPage(page *mupdf.Page) error {
    if page == nil {
        return fmt.Errorf("page is nil")
    }
    
    bounds := page.Bound()
    // Process bounds...
    
    return nil
}
```

## Error Handling

### Wrap Errors with Context

```go
// ✅ Good: Descriptive error wrapping
doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return fmt.Errorf("failed to open document %q: %w", filename, err)
}

page, err := doc.LoadPage(pageNum)
if err != nil {
    return fmt.Errorf("failed to load page %d from document %q: %w", 
        pageNum, filename, err)
}
```

### Graceful Degradation

```go
// ✅ Good: Continue processing despite errors
func extractAllText(doc *mupdf.Document) ([]string, error) {
    var results []string
    var errors []error
    
    for i := 0; i < doc.CountPages(); i++ {
        text, err := extractPageText(doc, i)
        if err != nil {
            errors = append(errors, fmt.Errorf("page %d: %w", i, err))
            continue // Continue with other pages
        }
        results = append(results, text)
    }
    
    if len(results) == 0 {
        return nil, fmt.Errorf("failed to extract text from any page: %v", errors)
    }
    
    // Return partial results with warning
    if len(errors) > 0 {
        log.Printf("Warning: %d pages failed to process", len(errors))
    }
    
    return results, nil
}
```

## Performance Optimization

### Minimize Context Creation

```go
// ✅ Good: Reuse context
func processMultipleDocuments(filenames []string) error {
    ctx, err := mupdf.NewContext()
    if err != nil {
        return err
    }
    defer ctx.Drop()
    
    for _, filename := range filenames {
        if err := processDocument(ctx, filename); err != nil {
            log.Printf("Failed to process %s: %v", filename, err)
        }
    }
    
    return nil
}
```

```go
// ❌ Bad: Creating context for each operation
func processMultipleDocuments(filenames []string) error {
    for _, filename := range filenames {
        ctx, err := mupdf.NewContext() // Expensive!
        if err != nil {
            return err
        }
        processDocument(ctx, filename)
        ctx.Drop()
    }
    return nil
}
```

### Batch Processing

```go
// ✅ Good: Process pages in batches
func extractTextBatched(doc *mupdf.Document, batchSize int) ([]string, error) {
    pageCount := doc.CountPages()
    results := make([]string, 0, pageCount)
    
    for start := 0; start < pageCount; start += batchSize {
        end := start + batchSize
        if end > pageCount {
            end = pageCount
        }
        
        batch, err := processBatch(doc, start, end)
        if err != nil {
            return nil, err
        }
        
        results = append(results, batch...)
        
        // Optional: trigger GC between batches for large documents
        if start%100 == 0 {
            runtime.GC()
        }
    }
    
    return results, nil
}
```

## Concurrency

### One Context Per Goroutine

A `Context` is **not** thread-safe. MuPDF is initialised in single-threaded
mode (no locking primitives), so sharing a `Context` — or any Document, Page,
or PDFWriter derived from it — across goroutines is undefined behaviour. Give
each goroutine its own `Context`.

```go
// ✅ Good: Separate contexts for goroutines
func processDocumentsConcurrently(filenames []string) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(filenames))
    
    for _, filename := range filenames {
        wg.Add(1)
        go func(fname string) {
            defer wg.Done()
            
            // Create separate context for each goroutine
            ctx, err := mupdf.NewContext()
            if err != nil {
                errChan <- err
                return
            }
            defer ctx.Drop()
            
            if err := processDocument(ctx, fname); err != nil {
                errChan <- fmt.Errorf("failed to process %s: %w", fname, err)
            }
        }(filename)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errors []error
    for err := range errChan {
        errors = append(errors, err)
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("processing failed: %v", errors)
    }
    
    return nil
}
```

## Production Patterns

### Structured Error Types

```go
// Define custom error types for better error handling
type ProcessingError struct {
    Filename string
    Page     int
    Op       string
    Err      error
}

func (e *ProcessingError) Error() string {
    if e.Page >= 0 {
        return fmt.Sprintf("failed to %s page %d of %s: %v", 
            e.Op, e.Page, e.Filename, e.Err)
    }
    return fmt.Sprintf("failed to %s %s: %v", e.Op, e.Filename, e.Err)
}

func (e *ProcessingError) Unwrap() error { return e.Err }
```

### Configuration and Settings

```go
type ProcessingConfig struct {
    MaxPages        int
    TextExtractionEnabled bool
    ConcurrentPages int
    TempDir        string
    LogLevel       string
}

func ProcessDocumentWithConfig(filename string, config ProcessingConfig) error {
    ctx, err := mupdf.NewContext()
    if err != nil {
        return err
    }
    defer ctx.Drop()
    
    // Apply configuration...
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        return err
    }
    defer doc.Close()
    
    pageCount := doc.CountPages()
    if config.MaxPages > 0 && pageCount > config.MaxPages {
        pageCount = config.MaxPages
    }
    
    // Process according to configuration...
    return nil
}
```

### Logging and Monitoring

```go
import "log/slog"

func processDocumentWithLogging(ctx *mupdf.Context, filename string) error {
    logger := slog.With("document", filename)
    logger.Info("Starting document processing")
    
    start := time.Now()
    defer func() {
        logger.Info("Document processing completed", 
            "duration", time.Since(start))
    }()
    
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        logger.Error("Failed to open document", "error", err)
        return err
    }
    defer doc.Close()
    
    pageCount := doc.CountPages()
    logger.Info("Document opened", "pages", pageCount)
    
    for i := 0; i < pageCount; i++ {
        if err := processPage(doc, i); err != nil {
            logger.Warn("Page processing failed", 
                "page", i, "error", err)
            continue
        }
    }
    
    return nil
}
```

## Security Considerations

### Input Validation

```go
func validateAndProcessDocument(filename string) error {
    // Validate file exists and is readable
    info, err := os.Stat(filename)
    if err != nil {
        return fmt.Errorf("cannot access file: %w", err)
    }
    
    // Check file size limits
    const maxFileSize = 100 * 1024 * 1024 // 100MB
    if info.Size() > maxFileSize {
        return fmt.Errorf("file too large: %d bytes (max %d)", 
            info.Size(), maxFileSize)
    }
    
    // Check file extension
    if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
        return fmt.Errorf("unsupported file type: %s", 
            filepath.Ext(filename))
    }
    
    return processDocument(filename)
}
```

### Resource Limits

```go
func processWithLimits(filename string) error {
    // Set processing timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    done := make(chan error, 1)
    go func() {
        done <- processDocument(filename)
    }()
    
    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        return fmt.Errorf("processing timeout: %w", ctx.Err())
    }
}
```

## Testing Best Practices

### Test Helpers

```go
func createTestPDF(t *testing.T, pageCount int) string {
    t.Helper()
    
    ctx, err := mupdf.NewContext()
    require.NoError(t, err)
    defer ctx.Drop()
    
    writer, err := mupdf.NewPDFWriter(ctx)
    require.NoError(t, err)
    defer writer.Close()
    
    for i := 0; i < pageCount; i++ {
        page, err := writer.AddPage(612, 792)
        require.NoError(t, err)
        page.Close()
    }
    
    tempFile := filepath.Join(t.TempDir(), "test.pdf")
    err = writer.Save(tempFile)
    require.NoError(t, err)
    
    return tempFile
}

func TestDocumentProcessing(t *testing.T) {
    pdfPath := createTestPDF(t, 3)
    
    // Test your function
    err := processDocument(pdfPath)
    assert.NoError(t, err)
}
```

### Mock and Stub Patterns

```go
// Interface for testability
type DocumentProcessor interface {
    ProcessDocument(filename string) error
}

type MuPDFProcessor struct {
    config ProcessingConfig
}

func (p *MuPDFProcessor) ProcessDocument(filename string) error {
    // Implementation using MuPDF
    return nil
}

// Mock for testing
type MockProcessor struct {
    shouldFail bool
}

func (m *MockProcessor) ProcessDocument(filename string) error {
    if m.shouldFail {
        return fmt.Errorf("mock error")
    }
    return nil
}
```

Remember: Always prioritize correctness and safety over performance optimizations. Profile your application to identify actual bottlenecks before optimizing.