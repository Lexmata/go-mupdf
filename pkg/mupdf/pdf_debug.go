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

// Improved PDF page addition - creates the page object, links it into the
// page tree with pdf_insert_page, and returns a properly refcounted page
// handle obtained via pdf_load_page. The new page's index is returned via
// out_page_num.
pdf_page* go_mupdf_improved_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, int *out_page_num, char **out_error) {
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

// DebugCountPages returns the number of pages in a PDF document for debugging.
// It returns 0 if the writer has been closed or is otherwise invalid.
func (writer *PDFWriter) DebugCountPages() int {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return 0
	}

	return int(C.go_mupdf_debug_pdf_count_pages(writer.ctx.ctx, writer.writer))
}

// ImprovedAddPage adds a new page to the PDF with improved implementation.
// It returns an error for invalid (negative or zero) dimensions.
//
// Deprecated: use [PDFWriter.AddPage]. ImprovedAddPage is retained for
// compatibility and will be removed in v2.0.0.
func (writer *PDFWriter) ImprovedAddPage(width, height float64) (*PDFPage, error) {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return nil, Error{message: "PDF writer is closed or invalid"}
	}

	if width <= 0 || height <= 0 {
		return nil, Error{message: "invalid page dimensions: width and height must be positive"}
	}

	var cError *C.char
	var cPageNum C.int
	page := C.go_mupdf_improved_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cPageNum, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	if page == nil {
		return nil, Error{message: "failed to add page: C helper returned null page"}
	}

	result := &PDFPage{ctx: writer.ctx, doc: nil, writer: writer, page: page, num: int(cPageNum)}
	runtime.SetFinalizer(result, func(p *PDFPage) {
		if p != nil {
			p.Close()
		}
	})

	return result, nil
}
