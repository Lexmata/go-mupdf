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
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -lmupdf -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lharfbuzz -lgumbo -lextract -lmujs -lm

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// Simple PDF page addition that manually manages the page tree
pdf_page* go_mupdf_simple_add_page(fz_context *ctx, pdf_document *doc, float width, float height, char **out_error) {
    pdf_page *page = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        // Get root and pages objects
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
        pdf_obj *pages = pdf_dict_get(ctx, root, PDF_NAME(Pages));
        pdf_obj *kids = pdf_dict_get(ctx, pages, PDF_NAME(Kids));

        // Create the page object
        pdf_obj *page_obj = pdf_add_new_dict(ctx, doc, 5);
        pdf_dict_put_name(ctx, page_obj, PDF_NAME(Type), "Page");
        pdf_dict_put(ctx, page_obj, PDF_NAME(Parent), pages);

        // Set media box
        pdf_obj *mediabox = pdf_new_array(ctx, doc, 4);
        pdf_array_push_int(ctx, mediabox, 0);
        pdf_array_push_int(ctx, mediabox, 0);
        pdf_array_push_int(ctx, mediabox, (int)width);
        pdf_array_push_int(ctx, mediabox, (int)height);
        pdf_dict_put(ctx, page_obj, PDF_NAME(MediaBox), mediabox);

        // Create simple resources
        pdf_obj *resources = pdf_add_new_dict(ctx, doc, 1);
        pdf_obj *procset = pdf_new_array(ctx, doc, 2);
        pdf_array_push_name(ctx, procset, "PDF");
        pdf_array_push_name(ctx, procset, "Text");
        pdf_dict_put(ctx, resources, PDF_NAME(ProcSet), procset);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Resources), resources);

        // Create simple content stream
        fz_buffer *contents = fz_new_buffer(ctx, 0);
        fz_append_string(ctx, contents, "BT /F1 12 Tf 50 750 Td (Hello World) Tj ET");
        pdf_obj *contents_obj = pdf_add_stream(ctx, doc, contents, NULL, 0);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Contents), contents_obj);

        // Add page to kids array
        pdf_array_push(ctx, kids, page_obj);

        // Update page count
        int count = pdf_dict_get_int(ctx, pages, PDF_NAME(Count));
        pdf_dict_put_int(ctx, pages, PDF_NAME(Count), count + 1);

        // Create the page structure
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
//   - Content stream creation with sample text
//   - Page count updates in the Pages object
//
// This implementation provides:
//   - Complete control over PDF structure
//   - Educational insight into PDF internals
//   - Debugging capabilities for PDF issues
//   - Alternative when higher-level functions fail
//
// The created page includes a simple "Hello World" content stream
// as a demonstration of content creation.
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
//	// Contains sample "Hello World" text content
func (writer *PDFWriter) SimpleAddPage(width, height float64) (*PDFPage, error) {
	var cError *C.char
	page := C.go_mupdf_simple_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	if page == nil {
		return nil, Error{message: "Simple add page returned null"}
	}

	// Get the current page count
	pageCount := int(C.pdf_count_pages(writer.ctx.ctx, writer.writer))
	pageNum := pageCount - 1

	if pageNum < 0 {
		pageNum = 0
	}

	result := &PDFPage{ctx: writer.ctx, doc: nil, page: page, num: pageNum}
	runtime.SetFinalizer(result, func(p *PDFPage) {
		if p != nil {
			p.Close()
		}
	})

	return result, nil
}
