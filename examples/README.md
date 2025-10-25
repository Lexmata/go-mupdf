# Go MuPDF Examples

This directory contains practical examples demonstrating how to use the Go MuPDF wrapper.

## Available Examples

### 1. Basic Usage (`basic-usage/`)
**File**: `basic-usage/main.go`

Demonstrates fundamental operations:
- Creating a MuPDF context
- Creating a simple PDF document
- Adding pages to a PDF
- Reading existing PDF files
- Text extraction

**Run the example**:
```bash
cd examples/basic-usage
go mod tidy
go run main.go
```

## Running Examples

### Prerequisites
Ensure you have the Go MuPDF wrapper installed:

```bash
# Option 1: Install via go get
go get bitbucket.org/lexmata/go-mupdf@latest

# Option 2: For local development
# Use the replace directive in go.mod (already included in examples)
```

### System Dependencies
Make sure you have the required system dependencies installed:

**Linux (Ubuntu/Debian)**:
```bash
sudo apt-get update
sudo apt-get install build-essential gcc g++ libc6-dev pkg-config
```

**macOS**:
```bash
xcode-select --install
```

**Windows (MinGW)**:
```bash
# Using MSYS2
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-make
```

### Local Development
If you're developing locally and want to test against the local version:

1. **Ensure MuPDF is built**:
   ```bash
   cd ../../third_party/mupdf
   make
   cd ../../examples/basic-usage
   ```

2. **Run with local module**:
   ```bash
   # The go.mod already includes a replace directive for local development
   go run main.go
   ```

## Example Output

When you run the basic usage example, you should see:

```
Go MuPDF Basic Usage Example
============================
✅ MuPDF context created successfully

📄 Creating a simple PDF...
✅ Created PDF with one page
   Page size: 612.0 x 792.0
✅ PDF creation example completed

🎉 Example completed successfully!
```

## Adding Your Own Examples

To create a new example:

1. **Create a new directory**:
   ```bash
   mkdir examples/my-example
   cd examples/my-example
   ```

2. **Initialize go.mod**:
   ```bash
   go mod init my-example
   ```

3. **Add dependency**:
   ```bash
   go get bitbucket.org/lexmata/go-mupdf@latest
   # Or for local development:
   # Add: replace bitbucket.org/lexmata/go-mupdf => ../..
   ```

4. **Create main.go**:
   ```go
   package main
   
   import (
       "fmt"
       "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
   )
   
   func main() {
       // Your example code here
   }
   ```

## Common Patterns

### Error Handling
Always check errors and properly clean up resources:

```go
ctx, err := mupdf.NewContext()
if err != nil {
    log.Fatal("Failed to create context:", err)
}
defer ctx.Drop() // Always clean up!
```

### Resource Management
Use defer for automatic cleanup:

```go
doc, err := mupdf.OpenDocument(ctx, "file.pdf")
if err != nil {
    return err
}
defer doc.Close() // Ensures cleanup even if function returns early
```

### Concurrent Operations
Create separate contexts for concurrent operations:

```go
// Don't share contexts between goroutines
go func() {
    ctx, err := mupdf.NewContext()
    if err != nil {
        return
    }
    defer ctx.Drop()
    
    // Use ctx in this goroutine only
}()
```

## Troubleshooting

### Common Issues

**"mupdf/fitz.h not found"**:
- Ensure MuPDF library is built: `cd third_party/mupdf && make`
- For go get installation, the library should build automatically

**"undefined reference to pdf_*"**:
- MuPDF libraries not properly linked
- Try rebuilding: `cd third_party/mupdf && make clean && make`

**CGO compilation errors**:
- Ensure you have a C compiler installed
- Check that CGO_ENABLED=1 (default)

### Getting Help

1. Check the main documentation: `docs/`
2. Review troubleshooting guide: `docs/TROUBLESHOOTING.md`
3. Run tests to verify installation: `go test ./pkg/mupdf/`
4. Check the examples work: `cd examples/basic-usage && go run main.go`

---

*These examples are designed to help you get started quickly with Go MuPDF. For production use, always implement proper error handling and resource management.*