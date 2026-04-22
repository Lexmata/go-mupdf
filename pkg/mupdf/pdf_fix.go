// Package mupdf - PDF Fix Module
//
// This file contains fixed implementations of PDF creation functions
// that address specific issues or limitations found in the standard
// implementations. These functions provide alternative approaches
// to PDF generation with improved reliability or functionality.
//
// Fix Functions:
//   - FixedAddPage: Alternative page addition with corrected memory management
//
// These functions are typically created to address bugs, improve
// performance, or provide workarounds for specific PDF creation
// scenarios. They may eventually replace the standard implementations
// once thoroughly tested and validated.
package mupdf

/*

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// Fixed PDF page addition
pdf_page* go_mupdf_fixed_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, char **out_error) {
    pdf_page *page = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        // Create a new page dictionary
        fz_rect mediabox;
        mediabox.x0 = 0;
        mediabox.y0 = 0;
        mediabox.x1 = width;
        mediabox.y1 = height;

        // Create resources and contents
        pdf_obj *resources = pdf_add_new_dict(ctx, doc, 0);
        fz_buffer *contents = fz_new_buffer(ctx, 0);

        // Add the page to the document
        pdf_obj *page_obj = pdf_add_page(ctx, doc, mediabox, rotate, resources, contents);

        // Create a new page structure
        page = fz_malloc_struct(ctx, pdf_page);
        page->obj = pdf_keep_obj(ctx, page_obj);
        page->doc = doc;

        // Clean up
        fz_drop_buffer(ctx, contents);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return page;
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// FixedAddPage adds a page using a corrected implementation.
//
// This function provides a fixed version of page addition that addresses
// specific issues found in the standard AddPage implementation. It uses
// improved memory management and more robust PDF object creation.
//
// Parameters:
//   - width: Page width in points (1/72 inch)
//   - height: Page height in points (1/72 inch)
//
// Returns:
//   - *PDFPage: A new page with corrected implementation
//   - error: An error if page creation fails
//
// Fixes addressed:
//   - Improved memory management for PDF objects
//   - Better error handling and recovery
//   - Corrected PDF page structure creation
//   - Enhanced finalizer safety
//
// This implementation may be more stable than the standard AddPage()
// method in certain scenarios, particularly those involving complex
// PDF structures or memory-constrained environments.
//
// Example:
//
//	// Use fixed implementation when standard method has issues
//	page, err := writer.FixedAddPage(612, 792)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	// Page created with improved implementation
func (writer *PDFWriter) FixedAddPage(width, height float64) (*PDFPage, error) {
	var cError *C.char
	page := C.go_mupdf_fixed_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	// Get the page number (it's the last page in the document)
	pageCount := int(C.pdf_count_pages(writer.ctx.ctx, writer.writer))
	pageNum := pageCount - 1

	// Ensure we have a valid page number
	if pageNum < 0 {
		pageNum = 0 // Default to the first page if we can't determine the correct number
	}

	result := &PDFPage{ctx: writer.ctx, doc: nil, page: page, num: pageNum}
	runtime.SetFinalizer(result, func(p *PDFPage) {
		p.Close()
	})

	return result, nil
}
