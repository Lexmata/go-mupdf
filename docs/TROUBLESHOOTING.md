# Go MuPDF Wrapper - Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using the Go MuPDF wrapper.

## Build Issues

### Problem: `fatal error: 'mupdf/fitz.h' file not found`

**Cause**: MuPDF headers are not found during compilation.

**Solution**:
```bash
# 1. Ensure MuPDF submodule is initialized
git submodule update --init --recursive

# 2. Build MuPDF
cd third_party/mupdf
make clean && make
cd ../..

# 3. Clear Go build cache
go clean -cache

# 4. Try building again
go build ./pkg/mupdf/
```

### Problem: `undefined reference to 'pdf_*'` linker errors

**Cause**: MuPDF libraries are not built or not found.

**Solution**:
```bash
# Check if libraries exist
ls third_party/mupdf/build/release/

# If missing, rebuild MuPDF
cd third_party/mupdf
make clean && make
cd ../..

# Verify libraries are created
ls -la third_party/mupdf/build/release/*.a
```

### Problem: Build fails on Windows

**Cause**: Missing build tools or incorrect compiler setup.

**Solution for MSYS2/MinGW**:
```bash
# Install required packages
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-make

# Use mingw32-make instead of make
cd third_party/mupdf
mingw32-make
cd ../..
```

**Solution for Visual Studio**:
```cmd
# Open Developer Command Prompt for VS
cd third_party\mupdf
nmake
cd ..\..
```

## Runtime Issues

### Problem: Segmentation fault on Context creation

**Symptoms**:
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Diagnosis**:
```go
// Add debug information
func debugContextCreation() {
    fmt.Printf("MuPDF Version: %s\n", mupdf.GetVersion())
    
    ctx, err := mupdf.NewContext()
    if err != nil {
        fmt.Printf("Context creation failed: %v\n", err)
        return
    }
    defer ctx.Drop()
    
    fmt.Println("Context created successfully")
}
```

**Solutions**:
1. Rebuild MuPDF with debug symbols:
```bash
cd third_party/mupdf
make clean && make debug
cd ../..
```

2. Check for threading issues:
```go
// Don't share contexts between goroutines
func safeConcurrentUsage() {
    var wg sync.WaitGroup
    
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            // Create separate context for each goroutine
            ctx, err := mupdf.NewContext()
            if err != nil {
                log.Printf("Failed to create context: %v", err)
                return
            }
            defer ctx.Drop()
            
            // Use context...
        }()
    }
    
    wg.Wait()
}
```

### Problem: Memory leaks or high memory usage

**Diagnosis**:
```go
import "runtime"

func monitorMemory() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    fmt.Printf("Alloc = %d KB", bToKb(m.Alloc))
    fmt.Printf("TotalAlloc = %d KB", bToKb(m.TotalAlloc))
    fmt.Printf("Sys = %d KB", bToKb(m.Sys))
    fmt.Printf("NumGC = %v\n", m.NumGC)
}

func bToKb(b uint64) uint64 {
    return b / 1024
}
```

**Solutions**:
1. Ensure proper cleanup:
```go
// ✅ Good: Always cleanup resources
func processDocument(filename string) error {
    ctx, err := mupdf.NewContext()
    if err != nil {
        return err
    }
    defer ctx.Drop() // Essential!
    
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        return err
    }
    defer doc.Close() // Essential!
    
    // Process pages...
    return nil
}
```

2. Force garbage collection in loops:
```go
func processLargeDocuments(filenames []string) error {
    for i, filename := range filenames {
        if err := processDocument(filename); err != nil {
            return err
        }
        
        // Force GC every 10 documents
        if i%10 == 0 {
            runtime.GC()
        }
    }
    return nil
}
```

### Problem: "Document has 0 pages" when it should have content

**Cause**: Document parsing failed or file is corrupted.

**Diagnosis**:
```go
func diagnoseDocument(filename string) {
    ctx, err := mupdf.NewContext()
    if err != nil {
        fmt.Printf("Context error: %v\n", err)
        return
    }
    defer ctx.Drop()
    
    // Check if file exists and is readable
    info, err := os.Stat(filename)
    if err != nil {
        fmt.Printf("File access error: %v\n", err)
        return
    }
    fmt.Printf("File size: %d bytes\n", info.Size())
    
    // Try to open document
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        fmt.Printf("Document open error: %v\n", err)
        return
    }
    defer doc.Close()
    
    pageCount := doc.CountPages()
    fmt.Printf("Page count: %d\n", pageCount)
    
    // Try PDF-specific opening
    pdfDoc, err := mupdf.OpenPDFDocument(ctx, filename)
    if err != nil {
        fmt.Printf("PDF open error: %v\n", err)
        return
    }
    
    pdfPageCount := pdfDoc.CountPages()
    fmt.Printf("PDF page count: %d\n", pdfPageCount)
}
```

**Solutions**:
1. Validate file format:
```go
func validatePDFFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Check PDF magic number
    header := make([]byte, 4)
    _, err = file.Read(header)
    if err != nil {
        return err
    }
    
    if string(header) != "%PDF" {
        return fmt.Errorf("not a valid PDF file")
    }
    
    return nil
}
```

2. Try alternative opening methods:
```go
func tryMultipleOpenMethods(ctx *mupdf.Context, filename string) error {
    // Method 1: Regular document opening
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err == nil {
        defer doc.Close()
        if doc.CountPages() > 0 {
            fmt.Println("Opened successfully with OpenDocument")
            return nil
        }
    }
    
    // Method 2: PDF-specific opening
    pdfDoc, err := mupdf.OpenPDFDocument(ctx, filename)
    if err == nil {
        if pdfDoc.CountPages() > 0 {
            fmt.Println("Opened successfully with OpenPDFDocument")
            return nil
        }
    }
    
    return fmt.Errorf("failed to open document with any method")
}
```

## Performance Issues

### Problem: Slow document processing

**Diagnosis**:
```go
import "time"

func benchmarkProcessing(filename string) {
    start := time.Now()
    
    ctx, err := mupdf.NewContext()
    if err != nil {
        log.Fatal(err)
    }
    defer ctx.Drop()
    
    contextTime := time.Since(start)
    fmt.Printf("Context creation: %v\n", contextTime)
    
    docStart := time.Now()
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        log.Fatal(err)
    }
    defer doc.Close()
    
    docTime := time.Since(docStart)
    fmt.Printf("Document opening: %v\n", docTime)
    
    pageCount := doc.CountPages()
    fmt.Printf("Page count: %d\n", pageCount)
    
    // Benchmark page loading
    pageStart := time.Now()
    for i := 0; i < pageCount; i++ {
        page, err := doc.LoadPage(i)
        if err != nil {
            continue
        }
        page.Close()
    }
    pageTime := time.Since(pageStart)
    fmt.Printf("All pages loading: %v (avg: %v per page)\n", 
        pageTime, pageTime/time.Duration(pageCount))
}
```

**Solutions**:
1. Process pages selectively:
```go
func processSelectedPages(doc *mupdf.Document, pageNumbers []int) error {
    for _, pageNum := range pageNumbers {
        if pageNum < 0 || pageNum >= doc.CountPages() {
            continue
        }
        
        page, err := doc.LoadPage(pageNum)
        if err != nil {
            continue
        }
        
        // Process specific page
        processPage(page)
        page.Close()
    }
    return nil
}
```

2. Use goroutines for independent operations:
```go
func processDocumentsConcurrently(filenames []string) {
    sem := make(chan struct{}, runtime.NumCPU()) // Limit concurrency
    var wg sync.WaitGroup
    
    for _, filename := range filenames {
        wg.Add(1)
        go func(fname string) {
            defer wg.Done()
            sem <- struct{}{} // Acquire
            defer func() { <-sem }() // Release
            
            processDocument(fname)
        }(filename)
    }
    
    wg.Wait()
}
```

## Environment Issues

### Problem: Different behavior between development and production

**Check Go version**:
```bash
go version
```

**Check build tags and environment**:
```bash
go env GOOS GOARCH CGO_ENABLED
```

**Ensure consistent builds**:
```go
// Add build information to your binary
package main

import (
    "fmt"
    "runtime"
    "runtime/debug"
)

func printBuildInfo() {
    if info, ok := debug.ReadBuildInfo(); ok {
        fmt.Printf("Go version: %s\n", info.GoVersion)
        fmt.Printf("Module: %s\n", info.Main.Path)
        
        for _, setting := range info.Settings {
            if setting.Key == "CGO_ENABLED" {
                fmt.Printf("CGO_ENABLED: %s\n", setting.Value)
            }
        }
    }
    
    fmt.Printf("Runtime OS: %s\n", runtime.GOOS)
    fmt.Printf("Runtime Arch: %s\n", runtime.GOARCH)
}
```

### Problem: Test failures in CI/CD

**Add CI debugging**:
```bash
# In your CI script
echo "=== Environment ==="
go env
echo "=== MuPDF Build ==="
ls -la third_party/mupdf/build/release/
echo "=== Test Run ==="
go test -v ./pkg/mupdf/
```

**Skip problematic tests in CI**:
```go
func TestSomething(t *testing.T) {
    if os.Getenv("CI") != "" {
        t.Skip("Skipping in CI environment")
    }
    
    // Test implementation...
}
```

## Getting Help

### Enable Debug Mode

```bash
export MUPDF_DEBUG=1
go test ./pkg/mupdf/ -v
```

### Create Minimal Reproduction

```go
package main

import (
    "fmt"
    "log"
    "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)

func main() {
    fmt.Printf("MuPDF Version: %s\n", mupdf.GetVersion())
    
    ctx, err := mupdf.NewContext()
    if err != nil {
        log.Fatalf("Context creation failed: %v", err)
    }
    defer ctx.Drop()
    
    // Add your failing code here
    
    fmt.Println("Test completed successfully")
}
```

### Collect System Information

```bash
# Create a bug report with system info
echo "=== System Info ===" > bug_report.txt
uname -a >> bug_report.txt
echo "" >> bug_report.txt

echo "=== Go Info ===" >> bug_report.txt
go version >> bug_report.txt
go env >> bug_report.txt
echo "" >> bug_report.txt

echo "=== MuPDF Build ===" >> bug_report.txt
ls -la third_party/mupdf/build/release/ >> bug_report.txt
echo "" >> bug_report.txt

echo "=== Error Log ===" >> bug_report.txt
go test ./pkg/mupdf/ -v 2>&1 >> bug_report.txt
```

### Common Command Patterns

```bash
# Clean rebuild everything
make clean -C third_party/mupdf
go clean -cache
git submodule update --init --recursive
cd third_party/mupdf && make && cd ../..
go test ./pkg/mupdf/

# Debug build
cd third_party/mupdf
make clean && make debug
cd ../..
go test ./pkg/mupdf/ -v

# Memory debugging (if available)
go test ./pkg/mupdf/ -race
go test ./pkg/mupdf/ -msan

# Verbose test output
go test ./pkg/mupdf/ -v -run "TestSpecificFunction"
```

If you're still experiencing issues after trying these solutions, please create a minimal reproduction case and include your system information when seeking help.