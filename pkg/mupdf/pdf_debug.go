// Package mupdf - PDF Debug Module
//
// This file contains debugging and development utilities for PDF creation
// and manipulation. These functions are primarily used for testing,
// development, and troubleshooting PDF generation issues.
//
// Debug Functions:
//   - DebugCountPages: Internal page counting with debug information
//   - ImprovedAddPage: Enhanced page addition with better error handling
//
// These functions may use alternative implementations or provide
// additional debugging information compared to the main API functions.
// They are intended for development use and may have different
// performance characteristics or stability guarantees.
package mupdf

/*

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// Debug function to check page count
int go_mupdf_debug_pdf_count_pages(fz_context *ctx, pdf_document *doc) {
    return pdf_count_pages(ctx, doc);
}

// Improved PDF page addition
pdf_page* go_mupdf_improved_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, char **out_error) {
    pdf_page *page = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        // Create a new page dictionary
        fz_rect mediabox;
        mediabox.x0 = 0;
        mediabox.y0 = 0;
        mediabox.x1 = width;
        mediabox.y1 = height;

        // Get the root
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
        if (!root) {
            root = pdf_add_new_dict(ctx, doc, 1);
            pdf_dict_put(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root), root);
        }

        // Ensure we have a Pages tree
        pdf_obj *pages = pdf_dict_get(ctx, root, PDF_NAME(Pages));
        if (!pages) {
            pages = pdf_add_new_dict(ctx, doc, 1);
            pdf_dict_put(ctx, root, PDF_NAME(Pages), pages);
            pdf_dict_put_int(ctx, pages, PDF_NAME(Count), 0);
            pdf_dict_put_array(ctx, pages, PDF_NAME(Kids), 0);
        }

        // Create resources and contents
        pdf_obj *resources = pdf_add_new_dict(ctx, doc, 0);
        fz_buffer *contents = fz_new_buffer(ctx, 0);

        // Add the page to the document
        pdf_obj *page_obj = pdf_add_page(ctx, doc, mediabox, rotate, resources, contents);

        // Load the page from the document to ensure it's properly added
        int page_count = pdf_count_pages(ctx, doc);
        if (page_count > 0) {
            page = pdf_load_page(ctx, doc, page_count - 1);
        }

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
	"unsafe"
)

// DebugCountPages returns the number of pages in a PDF document for debugging
func (writer *PDFWriter) DebugCountPages() int {
	return int(C.go_mupdf_debug_pdf_count_pages(writer.ctx.ctx, writer.writer))
}

// ImprovedAddPage adds a new page to the PDF with improved implementation
func (writer *PDFWriter) ImprovedAddPage(width, height float64) (*PDFPage, error) {
	var cError *C.char
	page := C.go_mupdf_improved_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cError)

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
	return result, nil
}
