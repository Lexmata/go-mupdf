// Package mupdf - Page Operations
//
// This file contains the Page type and related functionality for loading,
// managing, and performing operations on individual pages within documents.
package mupdf

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -Wl,-Bdynamic -lmupdf -Wl,-Bstatic -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lharfbuzz -lgumbo -lmujs -lm

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"

// Load page
fz_page* go_mupdf_load_page(fz_context *ctx, fz_document *doc, int page_num, char **out_error) {
    fz_page *page = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        page = fz_load_page(ctx, doc, page_num);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return page;
}

// Get page bounds
fz_rect go_mupdf_bound_page(fz_context *ctx, fz_page *page, char **out_error) {
    fz_rect rect = {0, 0, 0, 0};
    *out_error = NULL;

    fz_try(ctx) {
        rect = fz_bound_page(ctx, page);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return rect;
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// Page represents a single page within a document.
//
// Page provides access to page-level operations such as:
//   - Bounding box calculation
//   - Text extraction
//   - Content rendering (future functionality)
//   - Page metadata access
//
// Pages are loaded from Documents and represent the actual content
// of a specific page number. Each Page maintains a reference to
// its parent Document and the MuPDF Context.
//
// Memory Management:
//   - Always call Close() when finished with a Page
//   - A finalizer provides automatic cleanup as a safety net
//   - Pages become invalid when their parent Document is closed
//
// Coordinate System:
//   - Uses PDF coordinate system (origin at bottom-left)
//   - Measurements are in points (1/72 inch)
//   - Y-axis increases upward
//
// Thread Safety:
//   - Pages should not be shared between goroutines
//   - Create separate Pages for concurrent access
//
// Example:
//
//	page, err := doc.LoadPage(0)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	bounds := page.Bound()
//	text, err := page.ExtractText()
//	// ...
type Page struct {
	ctx  *Context
	doc  *Document
	page *C.fz_page
	num  int
}

// LoadPage loads a specific page from the document by page number.
//
// Pages are loaded on-demand and provide access to page-specific
// operations like text extraction and bounds calculation. The page
// remains valid until either the Page is closed or the parent
// Document is closed.
//
// Parameters:
//   - pageNum: Zero-based page index (0 to CountPages()-1)
//
// Returns:
//   - *Page: A page ready for content operations
//   - error: An error if the page cannot be loaded
//
// Error conditions:
//   - Page number is out of range (< 0 or >= CountPages())
//   - Document is closed or invalid
//   - Page structure is corrupted
//   - Memory allocation failure
//   - MuPDF internal parsing errors
//
// Memory Management:
//   - The returned Page must be closed with Close()
//   - Pages become invalid when the Document is closed
//   - Use defer page.Close() for automatic cleanup
//
// Example:
//
//	// Load the first page
//	page, err := doc.LoadPage(0)
//	if err != nil {
//	    log.Printf("Cannot load page 0: %v", err)
//	    return
//	}
//	defer page.Close()
//
//	// Get page dimensions
//	bounds := page.Bound()
//	fmt.Printf("Page size: %.1fx%.1f points\n",
//	    bounds.X1-bounds.X0, bounds.Y1-bounds.Y0)
func (doc *Document) LoadPage(pageNum int) (*Page, error) {
	var cError *C.char
	page := C.go_mupdf_load_page(doc.ctx.ctx, doc.doc, C.int(pageNum), &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &Page{ctx: doc.ctx, doc: doc, page: page, num: pageNum}
	runtime.SetFinalizer(result, func(p *Page) {
		p.Close()
	})

	return result, nil
}

// Close closes the page and releases resources
func (page *Page) Close() {
	if page.page != nil && page.ctx != nil && page.ctx.ctx != nil {
		C.fz_drop_page(page.ctx.ctx, page.page)
		page.page = nil
	}
}

// Bound returns the page's bounding rectangle in the page's coordinate system.
//
// The bounding box represents the page's media box - the physical page
// dimensions that define the page size. This is typically used to
// determine the page dimensions for rendering or layout purposes.
//
// Returns:
//   - Rect: The page's bounding rectangle in points
//
// The returned Rect follows PDF coordinate conventions:
//   - (X0, Y0) is the bottom-left corner
//   - (X1, Y1) is the top-right corner
//   - Coordinates are in points (1/72 inch)
//   - Y-axis increases upward
//
// If an error occurs during bounds calculation, returns an empty Rect
// with all coordinates set to 0.
//
// Example:
//
//	bounds := page.Bound()
//	width := bounds.X1 - bounds.X0
//	height := bounds.Y1 - bounds.Y0
//
//	fmt.Printf("Page dimensions: %.1f x %.1f points\n", width, height)
//	fmt.Printf("Page size in inches: %.2f x %.2f\n",
//	    width/72.0, height/72.0)
//
//	// Check if page is portrait or landscape
//	if width > height {
//	    fmt.Println("Landscape orientation")
//	} else {
//	    fmt.Println("Portrait orientation")
//	}
func (page *Page) Bound() Rect {
	var cError *C.char
	rect := C.go_mupdf_bound_page(page.ctx.ctx, page.page, &cError)

	if cError != nil {
		C.free(unsafe.Pointer(cError))
		return Rect{}
	}

	return Rect{
		X0: float64(rect.x0),
		Y0: float64(rect.y0),
		X1: float64(rect.x1),
		Y1: float64(rect.y1),
	}
}
