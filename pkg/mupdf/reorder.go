package mupdf

/*
#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>

static int safe_reorder_pages(fz_context *ctx, const char *input, const char *output,
	int *page_order, int count)
{
	pdf_document *doc = NULL;
	pdf_document *out_doc = NULL;
	pdf_write_options opts = {0};
	int result = 0;

	fz_try(ctx)
	{
		doc = pdf_open_document(ctx, input);
		out_doc = pdf_create_document(ctx);

		for (int i = 0; i < count; i++)
		{
			pdf_graft_page(ctx, out_doc, -1, doc, page_order[i]);
		}

		pdf_save_document(ctx, out_doc, output, &opts);
	}
	fz_always(ctx)
	{
		if (out_doc) pdf_drop_document(ctx, out_doc);
		if (doc) pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		result = -1;
	}

	return result;
}

static int safe_page_count(fz_context *ctx, const char *path)
{
	int count = -1;
	pdf_document *doc = NULL;
	fz_try(ctx)
	{
		doc = pdf_open_document(ctx, path);
		count = pdf_count_pages(ctx, doc);
	}
	fz_always(ctx)
	{
		if (doc) pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		count = -1;
	}
	return count;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// ReorderPages writes a new PDF to outputPath containing pages from inputPath
// rearranged according to pageOrder. Each element is a 0-based page index.
func ReorderPages(ctx *Context, inputPath, outputPath string, pageOrder []int) error {
	if len(pageOrder) == 0 {
		return fmt.Errorf("go-mupdf: ReorderPages: pageOrder must not be empty")
	}

	cInput := C.CString(inputPath)
	defer C.free(unsafe.Pointer(cInput))
	cOutput := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cOutput))

	cOrder := (*C.int)(C.malloc(C.size_t(len(pageOrder)) * C.size_t(unsafe.Sizeof(C.int(0)))))
	defer C.free(unsafe.Pointer(cOrder))

	orderSlice := unsafe.Slice(cOrder, len(pageOrder))
	for i, p := range pageOrder {
		orderSlice[i] = C.int(p)
	}

	ret := C.safe_reorder_pages(ctx.ctx, cInput, cOutput, cOrder, C.int(len(pageOrder)))
	if ret != 0 {
		return fmt.Errorf("go-mupdf: ReorderPages failed")
	}
	return nil
}

// PageCount returns the number of pages in a PDF.
func PageCount(ctx *Context, path string) (int, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	count := C.safe_page_count(ctx.ctx, cPath)
	if count < 0 {
		return 0, fmt.Errorf("go-mupdf: PageCount: failed to open %s", path)
	}
	return int(count), nil
}
