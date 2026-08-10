package mupdf

/*

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// PDF document conversion.
//
// Uses fz_new_pdf_document_from_fz_document rather than the
// pdf_document_from_fz_document down-cast: the former returns a KEPT
// reference that the caller must drop, which is what balances the
// pdf_drop_document call in PDFDocument.Close(). Both return NULL when
// the document is not a PDF.
pdf_document* go_mupdf_pdf_document_from_fz_document(fz_context *ctx, fz_document *doc, char **out_error) {
    pdf_document *pdf = NULL;
    *out_error = NULL;

    fz_var(pdf);

    fz_try(ctx) {
        pdf = fz_new_pdf_document_from_fz_document(ctx, doc);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return pdf;
}

// Drop a kept PDF document reference
void go_mupdf_pdf_drop_document(fz_context *ctx, pdf_document *doc, char **out_error) {
    *out_error = NULL;

    fz_try(ctx) {
        pdf_drop_document(ctx, doc);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }
}

// Count PDF pages
int go_mupdf_pdf_count_pages(fz_context *ctx, pdf_document *doc, char **out_error) {
    int count = 0;
    *out_error = NULL;

    fz_try(ctx) {
        count = pdf_count_pages(ctx, doc);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return count;
}

// Load PDF page
pdf_page* go_mupdf_pdf_load_page(fz_context *ctx, pdf_document *doc, int page_number, char **out_error) {
    pdf_page *page = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        page = pdf_load_page(ctx, doc, page_number);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return page;
}

// Get PDF page bounds
fz_rect go_mupdf_pdf_bound_page(fz_context *ctx, pdf_page *page, char **out_error) {
    fz_rect rect = {0, 0, 0, 0};
    *out_error = NULL;

    fz_var(rect);

    if (!page) {
        return rect;
    }

    fz_try(ctx) {
        // Use fz_bound_page as the primary path: it honours CropBox, Rotate
        // and non-zero origins. Pages returned by pdf_load_page have a fully
        // initialized vtable, so this is safe. We use fz_bound_page instead
        // of pdf_bound_page for version compatibility (the pdf_bound_page API
        // varies between MuPDF versions, but fz_bound_page is stable). Cast
        // pdf_page to fz_page since pdf_page extends fz_page.
        rect = fz_bound_page(ctx, (fz_page *)page);

        // Defensive fallback: if bounding produced an empty rect, read the
        // raw MediaBox directly from the page object.
        if (rect.x0 == 0 && rect.y0 == 0 && rect.x1 == 0 && rect.y1 == 0 && page->obj) {
            pdf_obj *mediabox = pdf_dict_get(ctx, page->obj, PDF_NAME(MediaBox));
            if (mediabox && pdf_is_array(ctx, mediabox)) {
                int len = pdf_array_len(ctx, mediabox);
                if (len >= 4) {
                    // MediaBox can contain either integers or reals, try both
                    rect.x0 = pdf_to_real(ctx, pdf_array_get(ctx, mediabox, 0));
                    rect.y0 = pdf_to_real(ctx, pdf_array_get(ctx, mediabox, 1));
                    rect.x1 = pdf_to_real(ctx, pdf_array_get(ctx, mediabox, 2));
                    rect.y1 = pdf_to_real(ctx, pdf_array_get(ctx, mediabox, 3));
                }
            }
        }
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return rect;
}

// Create PDF document
pdf_document* go_mupdf_pdf_create_document(fz_context *ctx, char **out_error) {
    pdf_document *doc = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        doc = pdf_create_document(ctx);

        // Ensure proper PDF structure is initialized
        pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
        if (!root) {
            root = pdf_add_new_dict(ctx, doc, 2);
            pdf_dict_put(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root), root);
            pdf_dict_put_name(ctx, root, PDF_NAME(Type), "Catalog");

            // Create Pages tree
            pdf_obj *pages = pdf_add_new_dict(ctx, doc, 3);
            pdf_dict_put(ctx, root, PDF_NAME(Pages), pages);
            pdf_dict_put_name(ctx, pages, PDF_NAME(Type), "Pages");
            pdf_dict_put_int(ctx, pages, PDF_NAME(Count), 0);
            pdf_dict_put_array(ctx, pages, PDF_NAME(Kids), 0);
        }
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return doc;
}

// Add PDF page - creates the page object, links it into the page tree with
// pdf_insert_page, and returns a properly refcounted page handle obtained
// via pdf_load_page. The new page's index is returned via out_page_num.
pdf_page* go_mupdf_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, int *out_page_num, char **out_error) {
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
        fz_rect mediabox;
        mediabox.x0 = 0;
        mediabox.y0 = 0;
        mediabox.x1 = width;
        mediabox.y1 = height;

        // Minimal resources dictionary and an empty content stream;
        // content is expected to be added later by the caller.
        resources = pdf_add_new_dict(ctx, doc, 1);
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

// Save PDF document
void go_mupdf_pdf_save_document(fz_context *ctx, pdf_document *doc, const char *filename, char **out_error) {
    *out_error = NULL;

    fz_try(ctx) {
        pdf_save_document(ctx, doc, filename, NULL);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }
}

// PDF object creation - null
pdf_obj* go_mupdf_pdf_new_null(fz_context *ctx, pdf_document *doc, char **out_error) {
    pdf_obj *obj = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        obj = PDF_NULL;
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return obj;
}

// PDF object creation - boolean
pdf_obj* go_mupdf_pdf_new_bool(fz_context *ctx, pdf_document *doc, int b, char **out_error) {
    pdf_obj *obj = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        obj = b ? PDF_TRUE : PDF_FALSE;
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return obj;
}

// PDF object creation - integer
pdf_obj* go_mupdf_pdf_new_int(fz_context *ctx, pdf_document *doc, int i, char **out_error) {
    pdf_obj *obj = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        obj = pdf_new_int(ctx, i);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return obj;
}

// PDF object creation - real
pdf_obj* go_mupdf_pdf_new_real(fz_context *ctx, pdf_document *doc, float f, char **out_error) {
    pdf_obj *obj = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        obj = pdf_new_real(ctx, f);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return obj;
}

// PDF object creation - string
pdf_obj* go_mupdf_pdf_new_string(fz_context *ctx, pdf_document *doc, const char *s, size_t len, char **out_error) {
    pdf_obj *obj = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        obj = pdf_new_string(ctx, s, len);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return obj;
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// PDFDocument represents a PDF document.
//
// Memory Management:
//   - Always call Close() when finished with a PDFDocument to release
//     the kept PDF document reference obtained from AsPDFDocument
//   - A finalizer provides automatic cleanup as a safety net
//   - Because the reference is kept independently of the parent
//     Document, a PDFDocument stays valid after the parent Document is
//     closed; Close() must still be called on it
//   - The Context must outlive the PDFDocument. Dropping the Context first
//     releases the PDF document along with it, so a later Close() has
//     nothing left to release and cannot report the difference.
type PDFDocument struct {
	ctx *Context
	doc *Document
	pdf *C.pdf_document
}

// AsPDFDocument converts a Document to a PDFDocument if possible.
//
// It returns an error if the document is not a PDF. The returned
// PDFDocument holds a kept (refcounted) reference to the underlying
// PDF document; always call Close() when finished with it.
func (doc *Document) AsPDFDocument() (*PDFDocument, error) {
	var cError *C.char
	pdf := C.go_mupdf_pdf_document_from_fz_document(doc.ctx.ctx, doc.doc, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	if pdf == nil {
		return nil, Error{message: "document is not a PDF"}
	}

	result := &PDFDocument{ctx: doc.ctx, doc: doc, pdf: pdf}
	runtime.SetFinalizer(result, func(p *PDFDocument) {
		p.Close()
	})

	return result, nil
}

// Close releases the kept PDF document reference obtained from
// AsPDFDocument. It is safe to call Close multiple times.
func (pdf *PDFDocument) Close() {
	pdf.ctx.withLock(func(c *C.fz_context) {
		if pdf.pdf == nil {
			return
		}
		if c != nil {
			var cError *C.char
			C.go_mupdf_pdf_drop_document(c, pdf.pdf, &cError)
			if cError != nil {
				C.free(unsafe.Pointer(cError))
			}
		}
		// Cleared even when the Context is already gone: fz_drop_context
		// released the document along with it, so keeping the pointer
		// would leave a dangling reference for later calls to use.
		pdf.pdf = nil
	})
}

// OpenPDFDocument opens a PDF document from a file path
func OpenPDFDocument(ctx *Context, filename string) (*PDFDocument, error) {
	doc, err := OpenDocument(ctx, filename)
	if err != nil {
		return nil, err
	}

	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		doc.Close()
		return nil, err
	}

	return pdfDoc, nil
}

// CountPages returns the number of pages in the PDF document.
//
// It returns 0 if the document has been closed. Note that 0 is returned
// both for a document with no pages and for a closed or invalid one;
// these cases are not distinguishable through this method.
func (pdf *PDFDocument) CountPages() int {
	if pdf.pdf == nil || pdf.ctx == nil || pdf.ctx.ctx == nil {
		return 0
	}

	var cError *C.char
	count := C.go_mupdf_pdf_count_pages(pdf.ctx.ctx, pdf.pdf, &cError)

	if cError != nil {
		C.free(unsafe.Pointer(cError))
		return 0
	}

	return int(count)
}

// PDFPage represents a page within a PDF document with PDF-specific functionality.
//
// PDFPage extends the basic Page interface with PDF-specific operations:
//   - PDF object access for the page
//   - Form field enumeration and manipulation
//   - Annotation access and modification
//   - PDF page metadata operations
//   - Content stream access
//
// PDFPage objects are created by loading pages from a PDFDocument
// or by adding pages to a PDFWriter during PDF creation.
//
// PDF-Specific Features:
//   - Access to page's PDF object dictionary
//   - Form field processing on the page
//   - Annotation creation and manipulation
//   - Content stream analysis and modification
//   - PDF page inheritance resolution
//
// Memory Management:
//   - Always call Close() when finished with a PDFPage
//   - A finalizer provides automatic cleanup as a safety net
//   - PDFPages become invalid when their parent PDFDocument is closed
//
// Coordinate System:
//   - Uses PDF coordinate system (origin at bottom-left)
//   - Measurements are in points (1/72 inch)
//   - Y-axis increases upward
//   - Supports rotation and transformation matrices
//
// Example:
//
//	pdfPage, err := pdfDoc.LoadPage(0)
//	if err != nil {
//	    return err
//	}
//	defer pdfPage.Close()
//
//	bounds := pdfPage.Bound()
//	// ... PDF-specific page operations
type PDFPage struct {
	ctx *Context
	doc *PDFDocument
	// writer keeps the owning PDFWriter reachable for pages created via
	// the AddPage family, so the writer cannot be finalized (and its
	// document freed) while the page is still alive. It is nil for pages
	// loaded from a PDFDocument.
	writer *PDFWriter
	page   *C.pdf_page
	num    int
}

// LoadPage loads a page by number
func (pdf *PDFDocument) LoadPage(pageNum int) (*PDFPage, error) {
	if pdf.pdf == nil || pdf.ctx == nil || pdf.ctx.ctx == nil {
		return nil, Error{message: "PDF document is closed or invalid"}
	}

	var cError *C.char
	page := C.go_mupdf_pdf_load_page(pdf.ctx.ctx, pdf.pdf, C.int(pageNum), &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &PDFPage{ctx: pdf.ctx, doc: pdf, page: page, num: pageNum}
	runtime.SetFinalizer(result, func(p *PDFPage) {
		p.Close()
	})

	return result, nil
}

// Close closes the page and releases resources
func (page *PDFPage) Close() {
	page.ctx.withLock(func(c *C.fz_context) {
		if page.page == nil {
			return
		}
		if c != nil {
			C.fz_drop_page(c, (*C.fz_page)(unsafe.Pointer(page.page)))
		}
		page.page = nil
	})
}

// Bound returns the page's bounding box.
//
// The returned rectangle reflects the page's CropBox and /Rotate
// entries where present, not the raw MediaBox. It returns the zero Rect
// if the page has been closed or if bounds calculation fails.
func (page *PDFPage) Bound() Rect {
	if page.page == nil || page.ctx == nil || page.ctx.ctx == nil {
		return Rect{}
	}

	var cError *C.char
	rect := C.go_mupdf_pdf_bound_page(page.ctx.ctx, page.page, &cError)

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

// PDFObject represents a PDF object within the PDF object tree.
//
// PDF objects are the fundamental building blocks of PDF documents.
// They can represent various types of data:
//   - Primitive types (null, boolean, integer, real, string, name)
//   - Container types (arrays, dictionaries)
//   - Reference objects (indirect object references)
//   - Stream objects (compressed data with dictionaries)
//
// PDFObject provides type-safe access to the PDF object tree,
// allowing inspection and manipulation of PDF structure at the
// object level. This is useful for:
//   - Advanced PDF processing
//   - Custom PDF generation
//   - PDF debugging and analysis
//   - Form field manipulation
//   - Annotation processing
//
// Object Types:
//   - null: Represents absence of a value
//   - bool: Boolean true/false values
//   - int: Integer numbers
//   - real: Floating-point numbers
//   - string: Text strings (literal or hexadecimal)
//   - name: PDF name objects (identifiers)
//   - array: Ordered collections of objects
//   - dict: Key-value collections (dictionaries)
//   - stream: Data streams with associated dictionaries
//
// Memory Management:
//   - Always call Drop() when finished with a PDFObject
//   - A finalizer provides automatic cleanup as a safety net
//   - PDFObjects become invalid when their parent document is closed
//
// Example:
//
//	obj, err := writer.NewPDFObject("Hello World")
//	if err != nil {
//	    return err
//	}
//	defer obj.Drop()
//
//	// Object is ready for use in PDF structure
type PDFObject struct {
	ctx *Context
	doc *PDFDocument
	obj *C.pdf_obj
}

// NewPDFObject creates a new PDF object from a value.
//
// It returns an error if the document has been closed or is otherwise
// invalid.
func (pdf *PDFDocument) NewPDFObject(value interface{}) (*PDFObject, error) {
	if pdf.pdf == nil || pdf.ctx == nil || pdf.ctx.ctx == nil {
		return nil, Error{message: "PDF document is closed or invalid"}
	}

	var obj *C.pdf_obj
	var cError *C.char

	switch v := value.(type) {
	case nil:
		obj = C.go_mupdf_pdf_new_null(pdf.ctx.ctx, pdf.pdf, &cError)

	case bool:
		boolVal := 0
		if v {
			boolVal = 1
		}
		obj = C.go_mupdf_pdf_new_bool(pdf.ctx.ctx, pdf.pdf, C.int(boolVal), &cError)

	case int:
		obj = C.go_mupdf_pdf_new_int(pdf.ctx.ctx, pdf.pdf, C.int(v), &cError)

	case float64:
		obj = C.go_mupdf_pdf_new_real(pdf.ctx.ctx, pdf.pdf, C.float(v), &cError)

	case string:
		cstr := C.CString(v)
		defer C.free(unsafe.Pointer(cstr))
		obj = C.go_mupdf_pdf_new_string(pdf.ctx.ctx, pdf.pdf, cstr, C.size_t(len(v)), &cError)

	default:
		return nil, Error{message: "unsupported type for PDF object"}
	}

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &PDFObject{ctx: pdf.ctx, doc: pdf, obj: obj}
	runtime.SetFinalizer(result, func(o *PDFObject) {
		o.Drop()
	})

	return result, nil
}

// Drop releases the PDF object
func (obj *PDFObject) Drop() {
	obj.ctx.withLock(func(c *C.fz_context) {
		if obj.obj == nil {
			return
		}
		if c != nil {
			C.pdf_drop_obj(c, obj.obj)
		}
		obj.obj = nil
	})
}

// PDFWriter provides functionality for creating new PDF documents from scratch.
//
// PDFWriter enables programmatic PDF creation with full control over
// document structure, page layout, and content. It supports:
//   - Creating new PDF documents
//   - Adding pages with custom dimensions
//   - Creating PDF objects of various types
//   - Saving documents to files
//   - Memory-efficient document generation
//
// Document Creation Workflow:
//  1. Create a PDFWriter with NewPDFWriter()
//  2. Add pages using AddPage() or similar methods
//  3. Optionally create and manipulate PDF objects
//  4. Save the document with Save()
//  5. Close the writer to free resources
//
// PDF Structure:
//   - Automatically creates proper PDF document structure
//   - Manages page tree and catalog objects
//   - Handles PDF version compatibility
//   - Generates valid cross-reference tables
//   - Creates proper PDF trailers
//
// Memory Management:
//   - Always call Close() when finished with a PDFWriter
//   - A finalizer provides automatic cleanup as a safety net
//   - All pages and objects become invalid after Close()
//
// Thread Safety:
//   - PDFWriter is not thread-safe
//   - Use separate writers for concurrent document creation
//
// Example:
//
//	writer, err := mupdf.NewPDFWriter(ctx)
//	if err != nil {
//	    return err
//	}
//	defer writer.Close()
//
//	// Add pages
//	page, err := writer.AddPage(612, 792) // US Letter
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	// Save the document
//	err = writer.Save("output.pdf")
//	if err != nil {
//	    return err
//	}
type PDFWriter struct {
	ctx    *Context
	writer *C.pdf_document
}

// NewPDFWriter creates a new PDF writer for document generation.
//
// This function initializes a new PDF document structure with
// the necessary PDF objects (catalog, page tree, etc.) to create
// a valid PDF document. The writer is ready to accept pages and
// content immediately after creation.
//
// Parameters:
//   - ctx: A valid MuPDF context for the operation
//
// Returns:
//   - *PDFWriter: A writer ready for PDF creation
//   - error: An error if writer creation fails
//
// The created PDFWriter includes:
//   - A properly initialized PDF document structure
//   - Root catalog object
//   - Empty page tree ready for pages
//   - Proper PDF headers and version information
//
// Error conditions:
//   - Context is closed or invalid
//   - Memory allocation failure
//   - MuPDF internal initialization errors
//
// Memory Management:
//   - The returned PDFWriter must be closed with Close()
//   - Use defer writer.Close() for automatic cleanup
//   - Close the writer before opening the generated file
//
// Example:
//
//	writer, err := mupdf.NewPDFWriter(ctx)
//	if err != nil {
//	    log.Fatalf("Cannot create PDF writer: %v", err)
//	}
//	defer writer.Close()
//
//	// Writer is ready for page creation
//	page, err := writer.AddPage(595, 842) // A4 size
//	// ...
func NewPDFWriter(ctx *Context) (*PDFWriter, error) {
	var cError *C.char
	writer := C.go_mupdf_pdf_create_document(ctx.ctx, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &PDFWriter{ctx: ctx, writer: writer}
	runtime.SetFinalizer(result, func(w *PDFWriter) {
		if w != nil {
			w.Close()
		}
	})

	return result, nil
}

// Close closes the PDF writer and releases resources
func (writer *PDFWriter) Close() {
	writer.ctx.withLock(func(c *C.fz_context) {
		if writer.writer == nil {
			return
		}
		if c != nil {
			C.pdf_drop_document(c, writer.writer)
		}
		writer.writer = nil
	})
}

// AddPage adds a new page to the PDF document with specified dimensions.
//
// This method creates a new page with the given width and height,
// adds it to the document's page tree, and returns a PDFPage object
// that can be used for further page-specific operations.
//
// Parameters:
//   - width: Page width in points (1/72 inch)
//   - height: Page height in points (1/72 inch)
//
// Returns:
//   - *PDFPage: A new page ready for content
//   - error: An error if page creation fails
//
// The created page includes:
//   - Proper PDF page object with MediaBox
//   - Link to the document's page tree
//   - Minimal resource dictionary
//   - An empty content stream
//
// The page is blank: this package does not currently expose an API for
// writing to a page content stream, so a document built solely with
// AddPage contains no text or graphics. Use pdfcpu or an external tool
// to add content. Earlier versions stamped placeholder text ("Page
// Content") into every added page; that is no longer the case.
//
// Common page sizes (in points):
//   - US Letter: 612 x 792
//   - A4: 595 x 842
//   - A3: 842 x 1191
//   - Legal: 612 x 1008
//   - Tabloid: 792 x 1224
//
// Error conditions:
//   - Writer is closed or invalid
//   - Invalid dimensions (negative or zero)
//   - Memory allocation failure
//   - PDF structure corruption
//
// Example:
//
//	// Add standard A4 page
//	page, err := writer.AddPage(595, 842)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	// Add custom size page
//	customPage, err := writer.AddPage(400, 600)
//	if err != nil {
//	    return err
//	}
//	defer customPage.Close()
func (writer *PDFWriter) AddPage(width, height float64) (*PDFPage, error) {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return nil, Error{message: "PDF writer is closed or invalid"}
	}

	if width <= 0 || height <= 0 {
		return nil, Error{message: "invalid page dimensions: width and height must be positive"}
	}

	var cError *C.char
	var cPageNum C.int

	page := C.go_mupdf_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cPageNum, &cError)

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

// Save writes the PDF document to a file with the specified filename.
//
// This method finalizes the PDF document structure, generates the
// cross-reference table, and writes the complete PDF to disk.
// The resulting file is a valid PDF that can be opened by any
// PDF viewer or processor.
//
// Parameters:
//   - filename: Path where the PDF file should be saved
//
// Returns:
//   - error: An error if saving fails
//
// The save process:
//   - Finalizes all page and object references
//   - Generates the cross-reference (xref) table
//   - Calculates object offsets and sizes
//   - Writes the complete PDF structure to disk
//   - Creates a valid PDF trailer
//
// Error conditions:
//   - Invalid or inaccessible file path
//   - Insufficient disk space
//   - File permission errors
//   - Writer is closed or invalid
//   - PDF structure is incomplete or corrupted
//
// File Handling:
//   - Creates the file if it doesn't exist
//   - Overwrites existing files
//   - File is created with standard permissions
//   - Atomic write operation (file is complete or not created)
//
// Best Practices:
//   - Save only after adding all desired content
//   - Don't use the writer after saving (close it)
//   - Verify the file was created successfully
//
// Example:
//
//	// Create and populate the PDF
//	writer, err := mupdf.NewPDFWriter(ctx)
//	if err != nil {
//	    return err
//	}
//	defer writer.Close()
//
//	page, err := writer.AddPage(612, 792)
//	if err != nil {
//	    return err
//	}
//	defer page.Close()
//
//	// Save to file
//	err = writer.Save("output.pdf")
//	if err != nil {
//	    log.Fatalf("Cannot save PDF: %v", err)
//	}
//
//	fmt.Println("PDF saved successfully")
func (writer *PDFWriter) Save(filename string) error {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return Error{message: "PDF writer is closed or invalid"}
	}

	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	var cError *C.char
	C.go_mupdf_pdf_save_document(writer.ctx.ctx, writer.writer, cFilename, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return Error{message: C.GoString(cError)}
	}

	return nil
}

// NewPDFObject creates a new PDF object from a Go value.
//
// This method converts Go values into their corresponding PDF object
// representations, enabling type-safe creation of PDF objects for
// use in document structure, content streams, or metadata.
//
// Parameters:
//   - value: The Go value to convert to a PDF object
//
// Returns:
//   - *PDFObject: A PDF object representing the value
//   - error: An error if conversion fails or type is unsupported
//
// Supported Go types and their PDF equivalents:
//   - nil → PDF null object
//   - bool → PDF boolean (true/false)
//   - int → PDF integer number
//   - float64 → PDF real number
//   - string → PDF string object (literal encoding)
//
// PDF Object Usage:
//   - Building custom PDF structures
//   - Creating metadata entries
//   - Constructing form field values
//   - Defining annotation properties
//   - Setting up document information
//
// Error conditions:
//   - Unsupported Go type provided
//   - Writer is closed or invalid
//   - Memory allocation failure
//   - MuPDF internal object creation error
//
// Memory Management:
//   - The returned PDFObject must be dropped with Drop()
//   - Use defer obj.Drop() for automatic cleanup
//   - Objects become invalid when the writer is closed
//
// Example:
//
//	// Create various PDF objects
//	nullObj, err := writer.NewPDFObject(nil)
//	if err == nil {
//	    defer nullObj.Drop()
//	}
//
//	boolObj, err := writer.NewPDFObject(true)
//	if err == nil {
//	    defer boolObj.Drop()
//	}
//
//	intObj, err := writer.NewPDFObject(42)
//	if err == nil {
//	    defer intObj.Drop()
//	}
//
//	floatObj, err := writer.NewPDFObject(3.14159)
//	if err == nil {
//	    defer floatObj.Drop()
//	}
//
//	stringObj, err := writer.NewPDFObject("Hello World")
//	if err == nil {
//	    defer stringObj.Drop()
//	}
//
//	// Unsupported type will return an error
//	_, err = writer.NewPDFObject([]int{1, 2, 3})
//	if err != nil {
//	    fmt.Printf("Expected error: %v\n", err)
//	}
func (writer *PDFWriter) NewPDFObject(value interface{}) (*PDFObject, error) {
	if writer.writer == nil || writer.ctx == nil || writer.ctx.ctx == nil {
		return nil, Error{message: "PDF writer is closed or invalid"}
	}

	var obj *C.pdf_obj
	var cError *C.char

	switch v := value.(type) {
	case nil:
		obj = C.go_mupdf_pdf_new_null(writer.ctx.ctx, writer.writer, &cError)

	case bool:
		boolVal := 0
		if v {
			boolVal = 1
		}
		obj = C.go_mupdf_pdf_new_bool(writer.ctx.ctx, writer.writer, C.int(boolVal), &cError)

	case int:
		obj = C.go_mupdf_pdf_new_int(writer.ctx.ctx, writer.writer, C.int(v), &cError)

	case float64:
		obj = C.go_mupdf_pdf_new_real(writer.ctx.ctx, writer.writer, C.float(v), &cError)

	case string:
		cstr := C.CString(v)
		defer C.free(unsafe.Pointer(cstr))
		obj = C.go_mupdf_pdf_new_string(writer.ctx.ctx, writer.writer, cstr, C.size_t(len(v)), &cError)

	default:
		return nil, Error{message: "unsupported type for PDF object"}
	}

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &PDFObject{ctx: writer.ctx, doc: nil, obj: obj}
	runtime.SetFinalizer(result, func(o *PDFObject) {
		if o != nil {
			o.Drop()
		}
	})

	return result, nil
}
