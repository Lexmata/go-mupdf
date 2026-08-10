# Getting Started with Go MuPDF Wrapper

This guide will help you get up and running with the Go MuPDF wrapper quickly and efficiently.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.19 or later** - [Download Go](https://golang.org/dl/)
- **C compiler** - GCC, Clang, or MSVC
- **Make build system** - For building MuPDF
- **Git** - For cloning repositories and submodules

### Platform-Specific Requirements

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get update
sudo apt-get install build-essential pkg-config git
```

#### macOS
```bash
# Install Xcode command line tools
xcode-select --install

# Optional: Install via Homebrew
brew install git make
```

#### Windows
- Install [MSYS2](https://www.msys2.org/) or [MinGW-w64](https://www.mingw-w64.org/)
- Or use [Visual Studio Build Tools](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)

## Installation

### Step 1: Clone the Repository

```bash
git clone https://bitbucket.org/lexmata/go-mupdf.git
cd go-mupdf
```

### Step 2: Initialize MuPDF Submodule

```bash
git submodule update --init --recursive
```

This downloads the MuPDF source code that the wrapper depends on.

### Step 3: Build MuPDF

```bash
cd third_party/mupdf
make
cd ../..
```

**Note:** This step may take several minutes as it compiles the entire MuPDF library.

### Step 4: Verify Installation

```bash
go build ./pkg/mupdf/
go test ./pkg/mupdf/ -run "TestGetVersion" -v
```

If successful, you should see output similar to:
```
=== RUN   TestGetVersion
    context_test.go:15: MuPDF version: 1.27.2
--- PASS: TestGetVersion (0.00s)
PASS
```

## Your First Program

Let's create a simple program that opens a PDF and displays basic information:

### Create main.go

```go
package main

import (
    "fmt"
    "log"
    "os"

    "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: go run main.go <pdf-file>")
    }
    
    filename := os.Args[1]
    
    // Step 1: Create a MuPDF context
    ctx, err := mupdf.NewContext()
    if err != nil {
        log.Fatalf("Failed to create context: %v", err)
    }
    defer ctx.Drop() // Always cleanup
    
    // Step 2: Open the document
    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        log.Fatalf("Failed to open document: %v", err)
    }
    defer doc.Close() // Always cleanup
    
    // Step 3: Get document information
    pageCount := doc.CountPages()
    fmt.Printf("Document: %s\n", filename)
    fmt.Printf("Pages: %d\n", pageCount)
    fmt.Printf("MuPDF Version: %s\n", mupdf.GetVersion())
    
    // Step 4: Process first page
    if pageCount > 0 {
        page, err := doc.LoadPage(0)
        if err != nil {
            log.Printf("Failed to load first page: %v", err)
            return
        }
        defer page.Close()
        
        // Get page dimensions
        bounds := page.Bound()
        width := bounds.X1 - bounds.X0
        height := bounds.Y1 - bounds.Y0
        
        fmt.Printf("First page size: %.1f x %.1f points\n", width, height)
        fmt.Printf("First page size: %.2f x %.2f inches\n", width/72, height/72)
        
        // Extract text from first page
        text, err := page.ExtractText()
        if err != nil {
            log.Printf("Failed to extract text: %v", err)
            return
        }
        defer text.Close()
        
        content := text.String()
        if len(content) > 0 {
            fmt.Printf("Text preview (first 100 chars): %.100s...\n", content)
        } else {
            fmt.Println("No text found on first page")
        }
    }
}
```

### Test Your Program

```bash
# Create a test PDF or use any existing PDF file
go run main.go path/to/your/document.pdf
```

Expected output:
```
Document: path/to/your/document.pdf
Pages: 5
MuPDF Version: 1.27.2
First page size: 612.0 x 792.0 points
First page size: 8.50 x 11.00 inches
Text preview (first 100 chars): This is the beginning of the document text content that was extract...
```

## Core Concepts

### 1. Context Management

The `Context` is the foundation of all MuPDF operations:

```go
// Always create a context first
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop() // Essential for memory management
```

**Key Points:**
- Required for all operations
- **Not** thread-safe: a `Context` must never be shared across goroutines.
  Create one `Context` per goroutine.
- Documents, Pages, and every other object created from a `Context` inherit
  that restriction — they belong to the goroutine that owns the `Context`.
- Always call `Drop()` when finished
- Use `defer` for automatic cleanup

### 2. Document Lifecycle

Documents represent opened files and must be properly managed:

```go
doc, err := mupdf.OpenDocument(ctx, "file.pdf")
if err != nil {
    return err
}
defer doc.Close() // Always cleanup documents

// Document operations...
pageCount := doc.CountPages()
```

**Key Points:**
- Documents become invalid when closed
- All pages from a document become invalid when document is closed
- Always call `Close()` when finished

### 3. Page Operations

Pages are loaded on-demand from documents:

```go
page, err := doc.LoadPage(0) // Zero-based indexing
if err != nil {
    return err
}
defer page.Close() // Always cleanup pages

// Page operations...
bounds := page.Bound()
text, err := page.ExtractText()
```

**Key Points:**
- Pages are zero-indexed (0 to CountPages()-1)
- Pages become invalid when their document is closed
- Always call `Close()` when finished

### 4. Memory Management

The wrapper uses a combination of explicit cleanup and automatic finalizers:

```go
// Explicit cleanup (recommended)
ctx, _ := mupdf.NewContext()
defer ctx.Drop()

doc, _ := mupdf.OpenDocument(ctx, "file.pdf")
defer doc.Close()

page, _ := doc.LoadPage(0)
defer page.Close()

text, _ := page.ExtractText()
defer text.Close()
```

**Best Practices:**
- Always use `defer` for cleanup
- Don't access objects after closing them
- Finalizers provide safety but explicit cleanup is better
- Close objects in reverse order of creation

## Common Patterns

### Processing All Pages

```go
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop()

doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return err
}
defer doc.Close()

for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        log.Printf("Failed to load page %d: %v", i, err)
        continue
    }
    
    // Process page...
    fmt.Printf("Processing page %d\n", i+1)
    
    page.Close() // Close immediately if processing many pages
}
```

### Text Extraction from All Pages

```go
var allText strings.Builder

for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        continue
    }
    
    text, err := page.ExtractText()
    if err != nil {
        page.Close()
        continue
    }
    
    allText.WriteString(fmt.Sprintf("=== Page %d ===\n", i+1))
    allText.WriteString(text.String())
    allText.WriteString("\n\n")
    
    text.Close()
    page.Close()
}

fmt.Println("Complete document text:")
fmt.Println(allText.String())
```

### Creating a PDF

```go
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop()

writer, err := mupdf.NewPDFWriter(ctx)
if err != nil {
    return err
}
defer writer.Close()

// Add multiple pages
for i := 0; i < 3; i++ {
    page, err := writer.AddPage(612, 792) // US Letter size
    if err != nil {
        return err
    }
    page.Close() // Close immediately after adding
}

// Save the PDF
err = writer.Save("output.pdf")
if err != nil {
    return err
}

fmt.Println("PDF created successfully!")
```

## Error Handling

Always check and handle errors appropriately:

```go
// Good error handling
doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return fmt.Errorf("failed to open document %q: %w", filename, err)
}
defer doc.Close()

// Check for specific conditions
if doc.CountPages() == 0 {
    return fmt.Errorf("document %q has no pages", filename)
}

// Handle page loading errors gracefully
page, err := doc.LoadPage(pageNum)
if err != nil {
    log.Printf("Warning: could not load page %d: %v", pageNum, err)
    return // or continue with next page
}
defer page.Close()
```

## Next Steps

Now that you have the basics working:

1. **Explore the [API Reference](API_REFERENCE.md)** for detailed function documentation
2. **Check out [Examples](EXAMPLES.md)** for more advanced usage patterns
3. **Read [Best Practices](BEST_PRACTICES.md)** for production-ready code
4. **See [Troubleshooting](TROUBLESHOOTING.md)** if you encounter issues

## Quick Reference

### Essential Imports
```go
import "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
```

### Basic Workflow
1. Create Context → 2. Open Document → 3. Load Pages → 4. Extract/Process → 5. Cleanup

### Memory Management
- Use `defer` for all cleanup functions
- Don't access closed objects
- Close in reverse order: Page → Document → Context

### Error Handling
- Always check errors from functions that return them
- Use `fmt.Errorf` with `%w` verb for error wrapping
- Handle errors gracefully for robust applications