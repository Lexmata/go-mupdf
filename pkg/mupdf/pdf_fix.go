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

// Fixed PDF page addition - creates the page object, links it into the page
// tree with pdf_insert_page, and returns a properly refcounted page handle
// obtained via pdf_load_page. The new page's index is returned via
// out_page_num.
pdf_page* go_mupdf_fixed_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, int *out_page_num, char **out_error) {
    pdf_page *page = NULL;
    pdf_obj *resources = NULL;
    pdf_obj *page_obj = NULL;
    fz_buffer *contents = NULL;
    *out_error = NULL;
    *out_page_num = -1;

    fz_var(page);
    fz_var(resources);
    fz_var(page_obj);
    fz_var(contents);

    fz_try(ctx) {
        // Create a new page dictionary
        fz_rect mediabox;
        mediabox.x0 = 0;
        mediabox.y0 = 0;
        mediabox.x1 = width;
        mediabox.y1 = height;

        // Create resources and an empty content stream
        resources = pdf_add_new_dict(ctx, doc, 0);
        contents = fz_new_buffer(ctx, 0);

        // Create the page object and link it into the page tree
        page_obj = pdf_add_page(ctx, doc, mediabox, rotate, resources, contents);
        pdf_insert_page(ctx, doc, -1, page_obj);

        // Load the freshly inserted page so we return a properly
        // refcounted fz_page-derived handle
        *out_page_num = pdf_count_pages(ctx, doc) - 1;
        page = pdf_load_page(ctx, doc, *out_page_num);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, resources);
        pdf_drop_obj(ctx, page_obj);
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
//   - error: An error if page creation fails or the dimensions are
//     invalid (negative or zero)
//
// Fixes addressed:
//   - Improved memory management for PDF objects
//   - Better error handling and recovery
//   - Corrected PDF page structure creation
//   - Enhanced finalizer safety
//
// Example:
//
//	page, err := writer.FixedAddPage(612, 792)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
// Deprecated: use [PDFWriter.AddPage]. FixedAddPage now shares the same
// pdf_add_page implementation as AddPage and offers no additional
// stability; it is retained for compatibility and will be removed in
// v2.0.0.
func (writer *PDFWriter) FixedAddPage(width, height float64) (*PDFPage, error) {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return nil, Error{message: "PDF writer is closed or invalid"}
	}

	if width <= 0 || height <= 0 {
		return nil, Error{message: "invalid page dimensions: width and height must be positive"}
	}

	var cError *C.char
	var cPageNum C.int
	page := C.go_mupdf_fixed_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cPageNum, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	if page == nil {
		return nil, Error{message: "failed to add page: C helper returned null page"}
	}

	result := &PDFPage{ctx: writer.ctx, doc: nil, writer: writer, page: page, num: int(cPageNum)}
	runtime.SetFinalizer(result, func(p *PDFPage) {
		p.Close()
	})

	return result, nil
}
