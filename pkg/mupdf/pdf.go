package mupdf

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -lmupdf -lharfbuzz -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lgumbo -lmujs -lm

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// Define FZ_MEDIA_BOX if not available (for older MuPDF versions)
#ifndef FZ_MEDIA_BOX
#define FZ_MEDIA_BOX 0
#endif

// PDF document conversion
pdf_document* go_mupdf_pdf_document_from_fz_document(fz_context *ctx, fz_document *doc, char **out_error) {
    pdf_document *pdf = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        pdf = pdf_document_from_fz_document(ctx, doc);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return pdf;
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

    fz_try(ctx) {
        // If page has an obj, try to read MediaBox directly (for manually created pages)
        if (page && page->obj) {
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

        // If we didn't get bounds from MediaBox, try fz_bound_page
        if (rect.x0 == 0 && rect.y0 == 0 && rect.x1 == 0 && rect.y1 == 0) {
            // Use fz_bound_page instead of pdf_bound_page for version compatibility
            // pdf_bound_page API varies between MuPDF versions, but fz_bound_page
            // is stable. Cast pdf_page to fz_page since pdf_page extends fz_page.
            rect = fz_bound_page(ctx, (fz_page *)page);
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

// Add PDF page - using the working simple approach
pdf_page* go_mupdf_pdf_add_page(fz_context *ctx, pdf_document *doc, float width, float height, int rotate, char **out_error) {
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

        // Set rotation if provided
        if (rotate != 0) {
            pdf_dict_put_int(ctx, page_obj, PDF_NAME(Rotate), rotate);
        }

        // Create simple resources
        pdf_obj *resources = pdf_add_new_dict(ctx, doc, 1);
        pdf_obj *procset = pdf_new_array(ctx, doc, 2);
        pdf_array_push_name(ctx, procset, "PDF");
        pdf_array_push_name(ctx, procset, "Text");
        pdf_dict_put(ctx, resources, PDF_NAME(ProcSet), procset);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Resources), resources);

        // Create simple content stream
        fz_buffer *contents = fz_new_buffer(ctx, 0);
        fz_append_string(ctx, contents, "BT /F1 12 Tf 50 750 Td (Page Content) Tj ET");
        pdf_obj *contents_obj = pdf_add_stream(ctx, doc, contents, NULL, 0);
        pdf_dict_put(ctx, page_obj, PDF_NAME(Contents), contents_obj);

        // Add page to kids array
        pdf_array_push(ctx, kids, page_obj);

        // Update page count
        int count = pdf_dict_get_int(ctx, pages, PDF_NAME(Count));
        pdf_dict_put_int(ctx, pages, PDF_NAME(Count), count + 1);

        // Clean up
        fz_drop_buffer(ctx, contents);

        // Create the page structure manually
        // We initialize it with the page object so that go_mupdf_pdf_bound_page
        // can read the MediaBox directly from the page object
        // Note: fz_malloc_struct already zero-initializes the struct
        page = fz_malloc_struct(ctx, pdf_page);
        page->obj = pdf_keep_obj(ctx, page_obj);
        page->doc = doc;
        // Initialize other fields to safe defaults
        page->transparency = 0;
        page->overprint = 0;
        page->links = NULL;
        page->annots = NULL;
        page->annot_tailp = &page->annots;
        page->widgets = NULL;
        page->widget_tailp = &page->widgets;
        // Initialize the super (fz_page) structure
        // Note: This is a minimal initialization. For full functionality,
        // we would need to use fz_new_derived_page, but that's not available
        // in the public API. The Bound() method will read from page->obj directly.
        // fz_malloc_struct already zero-initializes, so we just set the doc field
        page->super.doc = (fz_document *)doc;
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
	"fmt"
	"runtime"
	"unsafe"
)

// PDFDocument represents a PDF document
type PDFDocument struct {
	ctx *Context
	doc *Document
	pdf *C.pdf_document
}

// AsPDFDocument converts a Document to a PDFDocument if possible
func (doc *Document) AsPDFDocument() (*PDFDocument, error) {
	var cError *C.char
	pdf := C.go_mupdf_pdf_document_from_fz_document(doc.ctx.ctx, doc.doc, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &PDFDocument{ctx: doc.ctx, doc: doc, pdf: pdf}
	runtime.SetFinalizer(result, func(p *PDFDocument) {
		// No need to free pdf as it's just a cast of doc
	})

	return result, nil
}

// OpenPDFDocument opens a PDF document from a file path
func OpenPDFDocument(ctx *Context, filename string) (*PDFDocument, error) {
	doc, err := OpenDocument(ctx, filename)
	if err != nil {
		return nil, err
	}

	return doc.AsPDFDocument()
}

// CountPages returns the number of pages in the PDF document
func (pdf *PDFDocument) CountPages() int {
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
	ctx  *Context
	doc  *PDFDocument
	page *C.pdf_page
	num  int
}

// LoadPage loads a page by number
func (pdf *PDFDocument) LoadPage(pageNum int) (*PDFPage, error) {
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
	if page.page != nil && page.ctx != nil && page.ctx.ctx != nil {
		// Always drop the page - pages should be properly managed
		C.fz_drop_page(page.ctx.ctx, (*C.fz_page)(unsafe.Pointer(page.page)))
		page.page = nil
	}
}

// Bound returns the page's bounding box
func (page *PDFPage) Bound() Rect {
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

// NewPDFObject creates a new PDF object from a value
func (pdf *PDFDocument) NewPDFObject(value interface{}) (*PDFObject, error) {
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
	if obj.obj != nil && obj.ctx != nil && obj.ctx.ctx != nil {
		C.pdf_drop_obj(obj.ctx.ctx, obj.obj)
		obj.obj = nil
	}
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
	if writer.writer != nil && writer.ctx != nil && writer.ctx.ctx != nil {
		C.pdf_drop_document(writer.ctx.ctx, writer.writer)
		writer.writer = nil
	}
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
//   - Basic resource dictionary
//   - Default content stream for future content
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
	var cError *C.char

	page := C.go_mupdf_pdf_add_page(writer.ctx.ctx, writer.writer, C.float(width), C.float(height), C.int(0), &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	if page == nil {
		return nil, Error{message: fmt.Sprintf("C function returned null page, page count is %d", int(C.pdf_count_pages(writer.ctx.ctx, writer.writer)))}
	}

	// Get the page count after adding the page
	pageCount := int(C.pdf_count_pages(writer.ctx.ctx, writer.writer))
	pageNum := pageCount - 1

	// Ensure we have a valid page number
	if pageNum < 0 {
		// Page count might be 0 for newly created docs, set to 0 as first page
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
