// Package mupdf provides a comprehensive Go wrapper for the MuPDF library,
// enabling powerful PDF processing capabilities with robust memory management.
//
// MuPDF is a lightweight PDF, XPS, and E-book viewer and toolkit written in portable C.
// This Go wrapper provides safe, idiomatic Go interfaces to MuPDF's core functionality
// including document parsing, page rendering, text extraction, and PDF creation.
//
// # Key Features
//
//   - Memory-safe operations with automatic cleanup via finalizers
//   - Thread-safe concurrent operations
//   - Comprehensive error handling and recovery
//   - Support for PDF reading, writing, and manipulation
//   - High-performance text extraction and page processing
//
// # Basic Usage
//
// The typical workflow involves creating a Context, opening a Document,
// and then performing operations on individual Pages:
//
//	ctx, err := mupdf.NewContext()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ctx.Drop()
//
//	doc, err := mupdf.OpenDocument(ctx, "example.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer doc.Close()
//
//	pageCount := doc.CountPages()
//	for i := 0; i < pageCount; i++ {
//	    page, err := doc.LoadPage(i)
//	    if err != nil {
//	        continue
//	    }
//	    defer page.Close()
//
//	    // Extract text from the page
//	    text, err := page.ExtractText()
//	    if err == nil {
//	        fmt.Println(text.String())
//	        text.Close()
//	    }
//	}
//
// # Memory Management
//
// This wrapper implements comprehensive memory management to prevent leaks:
//
//   - All resources have explicit Close() or Drop() methods
//   - Finalizers provide automatic cleanup as a safety net
//   - Null pointer checks prevent segmentation faults
//   - Resource lifecycle is clearly documented
//
// # Thread Safety
//
// MuPDF contexts are thread-safe, but individual documents and pages
// should not be shared between goroutines without proper synchronization.
// Create separate contexts for concurrent operations when needed.
//
// # Error Handling
//
// All operations that can fail return an error following Go conventions.
// The Error type provides detailed error messages from the underlying
// MuPDF library. Always check errors and handle them appropriately.
//
// # File Organization
//
// The package is organized into logical modules:
//
//   - types.go: Core types and data structures (Error, Rect)
//   - context.go: Context management and library initialization
//   - document.go: Document opening and management
//   - page.go: Page loading and operations
//   - text.go: Text extraction functionality
//   - pdf.go: PDF-specific operations and creation
//   - pdf_*.go: Specialized PDF implementations (debug, fix, simple)
//   - test_helpers.go: Testing utilities and helpers
package mupdf
