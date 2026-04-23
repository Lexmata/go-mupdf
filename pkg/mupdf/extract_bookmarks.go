package mupdf

/*
#include <mupdf/fitz.h>
#include <mupdf/pdf.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// ExtractBookmarks reads the bookmark/outline hierarchy from the PDF.
// Returns (nil, nil) if the PDF has no bookmarks.
func ExtractBookmarks(ctx *Context, path string) ([]OutlineItem, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var doc *C.fz_document
	doc = C.fz_open_document(ctx.ctx, cPath)
	if doc == nil {
		return nil, fmt.Errorf("go-mupdf: ExtractBookmarks: failed to open %s", path)
	}
	defer C.fz_drop_document(ctx.ctx, doc)

	outline := C.fz_load_outline(ctx.ctx, doc)
	if outline == nil {
		return nil, nil
	}
	defer C.fz_drop_outline(ctx.ctx, outline)

	return convertOutline(outline), nil
}

func convertOutline(outline *C.fz_outline) []OutlineItem {
	var items []OutlineItem
	for node := outline; node != nil; node = node.next {
		item := OutlineItem{
			Title: C.GoString(node.title),
			Page:  int(node.page.page),
		}
		if node.down != nil {
			item.Children = convertOutline(node.down)
		}
		items = append(items, item)
	}
	return items
}
