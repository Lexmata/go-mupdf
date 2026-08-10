// Package mupdf - PDF Simple Module
//
// This file contains simplified implementations of PDF creation functions
// that manually manage PDF document structure. These functions provide
// direct control over PDF object creation and page tree management.
//
// Simple Functions:
//   - SimpleAddPage: Manual page tree management with direct object creation
//
// These functions bypass some of MuPDF's higher-level abstractions
// to provide more direct control over PDF generation. They are useful
// for debugging, education, or cases where precise control over PDF
// structure is required.
//
// The simple implementations manually create and manage:
//   - PDF page objects and dictionaries
//   - Page tree structure (Pages, Kids arrays)
//   - Resource dictionaries and content streams
//   - Cross-references and object relationships
package mupdf

/*

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// Simple PDF page addition that manually manages the page tree. After the
// page object is linked into the tree, the page is loaded via pdf_load_page
// so the returned handle is a properly refcounted fz_page-derived page.
// The new page's index is returned via out_page_num.
pdf_page* go_mupdf_simple_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int *out_page_num, char **out_error) {
    pdf_page *page = NULL;
    pdf_obj *page_obj = NULL;
    pdf_obj *mediabox = NULL;
    pdf_obj *resources = NULL;
    pdf_obj *procset = NULL;
    pdf_obj *contents_obj = NULL;
    fz_buffer *contents = NULL;
    *out_error = NULL;
    *out_page_num = -1;

    fz_var(page);
    fz_var(page_obj);
    fz_var(mediabox);
    fz_var(resources);
    fz_var(procset);
    fz_var(contents_obj);
    fz_var(contents);

    fz_try(ctx) {
        // Get root and pages objects
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
        pdf_obj *pages = pdf_dict_get(ctx, root, PDF_NAME(Pages));
        pdf_obj *kids = pdf_dict_get(ctx, pages, PDF_NAME(Kids));

        // Create the page object
        page_obj = pdf_add_new_dict(ctx, doc, 5);
        pdf_dict_put_name(ctx, page_obj, PDF_NAME(Type), "Page");
        pdf_dict_put(ctx, page_obj, PDF_NAME(Parent), pages);

        // Set media box (reals, so fractional point sizes are preserved)
        mediabox = pdf_new_array(ctx, doc, 4);
        pdf_array_push_real(ctx, mediabox, 0);
        pdf_array_push_real(ctx, mediabox, 0);
        pdf_array_push_real(ctx, mediabox, width);
        pdf_array_push_real(ctx, mediabox, height);
        pdf_dict_put(ctx, page_obj, PDF_NAME(MediaBox), mediabox);

        // Create simple resources
        resources = pdf_add_new_dict(ctx, doc, 1);
        procset = pdf_new_array(ctx, doc, 2);
        pdf_array_push_name(ctx, procset, "PDF");
        pdf_array_push_name(ctx, procset, "Text");
        pdf_dict_put(ctx, resources, PDF_NAME(ProcSet), procset);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Resources), resources);

        // Create an empty content stream; content can be added later
        contents = fz_new_buffer(ctx, 0);
        contents_obj = pdf_add_stream(ctx, doc, contents, NULL, 0);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Contents), contents_obj);

        // Add page to kids array
        pdf_array_push(ctx, kids, page_obj);

        // Update page count
        int count = pdf_dict_get_int(ctx, pages, PDF_NAME(Count));
        pdf_dict_put_int(ctx, pages, PDF_NAME(Count), count + 1);

        // Load the freshly inserted page so we return a properly
        // refcounted fz_page-derived handle
        *out_page_num = count;
        page = pdf_load_page(ctx, doc, count);
    }
    fz_always(ctx) {
        pdf_drop_obj(ctx, page_obj);
        pdf_drop_obj(ctx, mediabox);
        pdf_drop_obj(ctx, resources);
        pdf_drop_obj(ctx, procset);
        pdf_drop_obj(ctx, contents_obj);
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

// SimpleAddPage adds a page using direct PDF object manipulation.
//
// This function provides a simplified approach to page creation that
// manually manages the PDF document structure, including the page tree,
// resource dictionaries, and content streams. It bypasses MuPDF's
// higher-level page creation functions for direct control.
//
// Parameters:
//   - width: Page width in points (1/72 inch)
//   - height: Page height in points (1/72 inch)
//
// Returns:
//   - *PDFPage: A new page created with manual object management
//   - error: An error if page creation fails
//
// Manual Operations Performed:
//   - Direct PDF object dictionary creation
//   - Manual page tree management (Kids array updates)
//   - Resource dictionary construction
//   - Empty content stream creation
//   - Page count updates in the Pages object
//
// This implementation provides:
//   - Complete control over PDF structure
//   - Educational insight into PDF internals
//   - Debugging capabilities for PDF issues
//   - Alternative when higher-level functions fail
//
// The created page includes an empty content stream; content can be
// added later by the caller. An error is returned for invalid
// (negative or zero) dimensions.
//
// Example:
//
//	// Use simple method for direct PDF control
//	page, err := writer.SimpleAddPage(595, 842)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	// Page created with manual PDF object management
//
// Deprecated: use [PDFWriter.AddPage]. SimpleAddPage produces an
// equivalent page via manual object manipulation and is retained for
// compatibility and debugging; it will be removed in v2.0.0.
func (writer *PDFWriter) SimpleAddPage(width, height float64) (*PDFPage, error) {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return nil, Error{message: "PDF writer is closed or invalid"}
	}

	if width <= 0 || height <= 0 {
		return nil, Error{message: "invalid page dimensions: width and height must be positive"}
	}

	var cError *C.char
	var cPageNum C.int
	page := C.go_mupdf_simple_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), &cPageNum, &cError)

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
