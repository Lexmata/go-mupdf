package mupdf

/*
#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// CGo cannot access C bitfields directly.
static void set_outline_is_open(fz_outline *o, int val) { o->is_open = val; }

// mupdf uses setjmp/longjmp for error handling. Any mupdf function that
// can throw MUST be called inside fz_try/fz_catch or the process aborts.
// CGo cannot use fz_try macros directly, so we wrap the operations in C.

typedef struct {
	const char *title;
	const char *uri;
	int is_open;
} go_outline_item;

typedef struct {
	go_outline_item *items;
	int count;
	int *child_counts;    // number of children per item
	int **child_indices;  // indices into a flat items array for children
} go_outline_flat;

// Insert a single item at the current iterator position.
static int safe_insert_item(fz_context *ctx, fz_outline_iterator *iter,
	const char *title, const char *uri, int is_open)
{
	int ret = -1;
	fz_outline_item item = {0};
	item.title = (char *)title;
	item.uri = (char *)uri;
	item.is_open = is_open;

	fz_try(ctx)
		ret = fz_outline_iterator_insert(ctx, iter, &item);
	fz_catch(ctx)
		ret = -1;

	return ret;
}

static int safe_iter_next(fz_context *ctx, fz_outline_iterator *iter)
{
	int ret = -1;
	fz_try(ctx)
		ret = fz_outline_iterator_next(ctx, iter);
	fz_catch(ctx)
		ret = -1;
	return ret;
}

static int safe_iter_down(fz_context *ctx, fz_outline_iterator *iter)
{
	int ret = -1;
	fz_try(ctx)
		ret = fz_outline_iterator_down(ctx, iter);
	fz_catch(ctx)
		ret = -1;
	return ret;
}

static int safe_iter_up(fz_context *ctx, fz_outline_iterator *iter)
{
	int ret = -1;
	fz_try(ctx)
		ret = fz_outline_iterator_up(ctx, iter);
	fz_catch(ctx)
		ret = -1;
	return ret;
}

static int safe_iter_delete(fz_context *ctx, fz_outline_iterator *iter)
{
	int ret = -1;
	fz_try(ctx)
		ret = fz_outline_iterator_delete(ctx, iter);
	fz_catch(ctx)
		ret = -1;
	return ret;
}

// Open a PDF, get its outline iterator, or return NULL on error.
static pdf_document *safe_pdf_open(fz_context *ctx, const char *path)
{
	pdf_document *doc = NULL;
	fz_try(ctx)
		doc = pdf_open_document(ctx, path);
	fz_catch(ctx)
		doc = NULL;
	return doc;
}

static fz_outline_iterator *safe_pdf_outline_iter(fz_context *ctx, pdf_document *doc)
{
	fz_outline_iterator *iter = NULL;
	fz_try(ctx)
		iter = pdf_new_outline_iterator(ctx, doc);
	fz_catch(ctx)
		iter = NULL;
	return iter;
}

static int safe_pdf_save(fz_context *ctx, pdf_document *doc, const char *path)
{
	int ret = 0;
	pdf_write_options opts = {0};
	fz_try(ctx)
		pdf_save_document(ctx, doc, path, &opts);
	fz_catch(ctx)
		ret = -1;
	return ret;
}

static void safe_pdf_drop(fz_context *ctx, pdf_document *doc)
{
	fz_try(ctx)
		pdf_drop_document(ctx, doc);
	fz_catch(ctx)
		;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// OutlineItem represents a bookmark/outline entry in a PDF.
type OutlineItem struct {
	Title    string
	Page     int // 0-based page index
	IsOpen   bool
	Children []OutlineItem
}

// AddBookmarks opens a PDF, replaces its outline with the given items,
// and saves to outputPath. Unlike pdfcpu, mupdf does not require
// bookmarks to be sorted by page number.
func AddBookmarks(ctx *Context, inputPath, outputPath string, items []OutlineItem) error {
	if len(items) == 0 {
		return fmt.Errorf("go-mupdf: AddBookmarks: items must not be empty")
	}

	cInput := C.CString(inputPath)
	defer C.free(unsafe.Pointer(cInput))
	cOutput := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cOutput))

	doc := C.safe_pdf_open(ctx.ctx, cInput)
	if doc == nil {
		return fmt.Errorf("go-mupdf: failed to open PDF: %s", inputPath)
	}
	defer C.safe_pdf_drop(ctx.ctx, doc)

	// Clear existing outlines
	iter := C.safe_pdf_outline_iter(ctx.ctx, doc)
	if iter != nil {
		for {
			ret := C.safe_iter_delete(ctx.ctx, iter)
			if ret < 0 {
				break
			}
		}
		C.fz_drop_outline_iterator(ctx.ctx, iter)
	}

	// Get fresh iterator and insert our items
	iter = C.safe_pdf_outline_iter(ctx.ctx, doc)
	if iter == nil {
		return fmt.Errorf("go-mupdf: failed to create outline iterator")
	}
	defer C.fz_drop_outline_iterator(ctx.ctx, iter)

	insertOutlineItems(ctx.ctx, iter, items)

	if C.safe_pdf_save(ctx.ctx, doc, cOutput) != 0 {
		return fmt.Errorf("go-mupdf: failed to save PDF: %s", outputPath)
	}

	return nil
}

func insertOutlineItems(ctx *C.fz_context, iter *C.fz_outline_iterator, items []OutlineItem) {
	for i, item := range items {
		uri := fmt.Sprintf("#page=%d", item.Page+1)
		cTitle := C.CString(item.Title)
		cURI := C.CString(uri)
		isOpen := C.int(0)
		if item.IsOpen {
			isOpen = 1
		}

		C.safe_insert_item(ctx, iter, cTitle, cURI, isOpen)

		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cURI))

		if len(item.Children) > 0 {
			ret := C.safe_iter_down(ctx, iter)
			if ret >= 0 {
				insertOutlineItems(ctx, iter, item.Children)
				C.safe_iter_up(ctx, iter)
			}
		}

		if i < len(items)-1 {
			C.safe_iter_next(ctx, iter)
		}
	}
}
