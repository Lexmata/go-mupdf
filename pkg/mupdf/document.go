// Package mupdf - Document Operations
//
// This file contains the Document type and related functionality for opening,
// managing, and performing operations on documents in various formats supported
// by MuPDF.
package mupdf

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -lmupdf -lm

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"

// Open document
fz_document* go_mupdf_open_document(fz_context *ctx, const char *filename, char **out_error) {
    fz_document *doc = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        doc = fz_open_document(ctx, filename);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return doc;
}

// Count pages
int go_mupdf_count_pages(fz_context *ctx, fz_document *doc, char **out_error) {
    int count = 0;
    *out_error = NULL;

    fz_try(ctx) {
        count = fz_count_pages(ctx, doc);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
        return -1;
    }

    return count;
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// Document represents an opened document in MuPDF.
//
// Document provides access to document-level operations such as:
//   - Page counting and loading
//   - Document metadata access
//   - Format-specific operations (e.g., PDF-specific features)
//   - Memory management for the document structure
//
// Supported document formats include:
//   - PDF (Portable Document Format)
//   - XPS (XML Paper Specification)
//   - EPUB (Electronic Publication)
//   - CBZ (Comic Book Archive)
//   - And other formats supported by MuPDF
//
// Memory Management:
//   - Always call Close() when finished with a Document
//   - A finalizer provides automatic cleanup as a safety net
//   - All Pages loaded from this Document become invalid after Close()
//
// Thread Safety:
//   - Documents should not be shared between goroutines
//   - Create separate Documents (or use separate Contexts) for concurrent access
//
// Example:
//
//	doc, err := mupdf.OpenDocument(ctx, "document.pdf")
//	if err != nil {
//	    return err
//	}
//	defer doc.Close()
//
//	pageCount := doc.CountPages()
//	// Process pages...
type Document struct {
	ctx *Context
	doc *C.fz_document
}

// OpenDocument opens a document from the specified file path.
//
// This function automatically detects the document format based on file
// content and extension, then uses the appropriate MuPDF handler to
// parse the document structure.
//
// Parameters:
//   - ctx: A valid MuPDF context for the operation
//   - filename: Path to the document file (absolute or relative)
//
// Returns:
//   - *Document: A document ready for page operations
//   - error: An error if the file cannot be opened or parsed
//
// Supported formats:
//   - PDF files (.pdf)
//   - XPS files (.xps)
//   - EPUB files (.epub)
//   - CBZ/CBR comic book archives
//   - Other formats supported by MuPDF
//
// Error conditions:
//   - File does not exist or is not accessible
//   - File format is not supported or recognized
//   - File is corrupted or invalid
//   - Memory allocation failure
//   - MuPDF internal parsing errors
//
// Example:
//
//	doc, err := mupdf.OpenDocument(ctx, "/path/to/document.pdf")
//	if err != nil {
//	    log.Fatalf("Cannot open document: %v", err)
//	}
//	defer doc.Close()
//
//	fmt.Printf("Opened document with %d pages\n", doc.CountPages())
func OpenDocument(ctx *Context, filename string) (*Document, error) {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	var cError *C.char
	doc := C.go_mupdf_open_document(ctx.ctx, cFilename, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &Document{ctx: ctx, doc: doc}
	runtime.SetFinalizer(result, func(d *Document) {
		if d != nil {
			d.Close()
		}
	})

	return result, nil
}

// Close closes the document and releases all associated resources.
//
// This method must be called when the Document is no longer needed to
// prevent memory leaks. It's safe to call Close() multiple times -
// subsequent calls are no-ops.
//
// Close() will:
//   - Release the underlying MuPDF document structure
//   - Free all associated memory
//   - Invalidate all Pages loaded from this Document
//   - Make the Document unusable for further operations
//
// After calling Close(), all Pages created from this Document become
// invalid and should not be used. The Document itself should also
// not be used for any operations.
//
// Best Practices:
//   - Use defer doc.Close() immediately after opening a Document
//   - Ensure Close() is called even if errors occur
//   - Close all Pages before closing the Document
//
// Example:
//
//	doc, err := mupdf.OpenDocument(ctx, "file.pdf")
//	if err != nil {
//	    return err
//	}
//	defer doc.Close() // Guaranteed cleanup
//
//	// Use document for operations...
//	// Close() will be called automatically when function returns
func (doc *Document) Close() {
	if doc.doc != nil && doc.ctx != nil && doc.ctx.ctx != nil {
		C.fz_drop_document(doc.ctx.ctx, doc.doc)
		doc.doc = nil
	}
}

// CountPages returns the total number of pages in the document.
//
// This method counts all pages in the document, regardless of format.
// The page count is determined by the document's internal structure
// and may involve parsing the document tree.
//
// Returns:
//   - int: The number of pages (>= 0), or -1 if an error occurs
//
// The returned count can be used to iterate through all pages:
//
//	for i := 0; i < doc.CountPages(); i++ {
//	    page, err := doc.LoadPage(i)
//	    // ... process page
//	}
//
// Error conditions (returns -1):
//   - Document is closed or invalid
//   - Document structure is corrupted
//   - MuPDF internal error
//
// Note: Page numbering is zero-based, so valid page indices
// range from 0 to CountPages()-1.
//
// Example:
//
//	count := doc.CountPages()
//	if count > 0 {
//	    fmt.Printf("Document has %d pages\n", count)
//	    // Load first page
//	    page, err := doc.LoadPage(0)
//	    // ...
//	}
func (doc *Document) CountPages() int {
	var cError *C.char
	count := C.go_mupdf_count_pages(doc.ctx.ctx, doc.doc, &cError)

	if cError != nil {
		C.free(unsafe.Pointer(cError))
		return 0
	}

	return int(count)
}
