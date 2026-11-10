package mupdf

/*
#include <stdlib.h>
#include "ops.h"
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

	doc := C.gomupdf_outline_open_document(ctx.ctx, cPath)
	if doc == nil {
		return nil, fmt.Errorf("go-mupdf: ExtractBookmarks: failed to open %s", path)
	}
	defer C.gomupdf_outline_drop_document(ctx.ctx, doc)

	outline := C.gomupdf_outline_load(ctx.ctx, doc)
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
