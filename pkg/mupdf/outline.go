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

	doc := C.gomupdf_outline_open_pdf(ctx.ctx, cInput)
	if doc == nil {
		return fmt.Errorf("go-mupdf: failed to open PDF: %s", inputPath)
	}
	defer C.gomupdf_outline_drop_pdf(ctx.ctx, doc)

	// Clear existing outlines
	iter := C.gomupdf_outline_iterator(ctx.ctx, doc)
	if iter != nil {
		for {
			ret := C.gomupdf_outline_delete(ctx.ctx, iter)
			if ret < 0 {
				break
			}
		}
		C.fz_drop_outline_iterator(ctx.ctx, iter)
	}

	// Get fresh iterator and insert our items
	iter = C.gomupdf_outline_iterator(ctx.ctx, doc)
	if iter == nil {
		return fmt.Errorf("go-mupdf: failed to create outline iterator")
	}
	defer C.fz_drop_outline_iterator(ctx.ctx, iter)

	// Count how many items (across the whole tree) fail to insert. Saving a
	// document whose outline is silently truncated is worse than failing loudly:
	// callers overwrite the source PDF with the result, so a dropped insert would
	// permanently lose bookmarks with no error. Refuse to save a partial outline.
	inserted, requested := insertOutlineItems(ctx.ctx, iter, items)
	if inserted != requested {
		return fmt.Errorf("go-mupdf: AddBookmarks: only %d of %d outline items inserted; refusing to save partial outline", inserted, requested)
	}

	if C.gomupdf_outline_save_pdf(ctx.ctx, doc, cOutput) != 0 {
		return fmt.Errorf("go-mupdf: failed to save PDF: %s", outputPath)
	}

	return nil
}

// insertOutlineItems writes items at the iterator's current nesting level.
//
// fz_outline_iterator_insert auto-advances past the inserted item (see
// mupdf/fitz/outline.h), so the cursor after an insert is already at the
// slot where the NEXT sibling would go. That means:
//
//   - No explicit _next is needed between sibling inserts.
//   - To descend into the just-inserted item's children, we must step
//     _prev back onto it before calling _down.
//   - _up returns us to the parent item; we then _next once to restore the
//     "next sibling slot" position before inserting the following sibling.
//
// The previous implementation called _next after every insert, which
// skipped every other slot, and _down from an already-advanced position,
// which silently failed to descend. Net effect: only one item survived
// per level and children were dropped. mupdf would then emit "repaired
// broken tree structure in outline" when the resulting PDF was read back.
// insertOutlineItems returns (inserted, requested): how many items across the
// whole subtree were successfully inserted, and how many were requested. The
// caller compares the two to detect a silently-truncated outline.
func insertOutlineItems(ctx *C.fz_context, iter *C.fz_outline_iterator, items []OutlineItem) (inserted, requested int) {
	for _, item := range items {
		requested++
		uri := fmt.Sprintf("#page=%d", item.Page+1)
		cTitle := C.CString(item.Title)
		cURI := C.CString(uri)
		isOpen := C.int(0)
		if item.IsOpen {
			isOpen = 1
		}

		insertRet := C.gomupdf_outline_insert(ctx, iter, cTitle, cURI, isOpen)

		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cURI))

		if insertRet < 0 {
			// Insert failed; count this item's descendants as requested but not
			// inserted, do not attempt children, and leave the cursor where it
			// is for the next sibling attempt.
			requested += countItems(item.Children)
			continue
		}
		inserted++

		if len(item.Children) > 0 {
			// Step back to the item we just inserted, descend into it,
			// recurse to write its children, then ascend and advance past
			// the parent so the next sibling lands in the right slot.
			if C.gomupdf_outline_prev(ctx, iter) >= 0 {
				if C.gomupdf_outline_down(ctx, iter) >= 0 {
					ci, cr := insertOutlineItems(ctx, iter, item.Children)
					inserted += ci
					requested += cr
					C.gomupdf_outline_up(ctx, iter)
				} else {
					// Could not descend: children are requested but unreachable.
					requested += countItems(item.Children)
				}
				C.gomupdf_outline_next(ctx, iter)
			} else {
				// Could not step back onto the inserted item: same as above.
				requested += countItems(item.Children)
			}
		}
	}
	return inserted, requested
}

// countItems returns the total number of items in the subtree rooted at items.
func countItems(items []OutlineItem) int {
	n := 0
	for _, item := range items {
		n++
		n += countItems(item.Children)
	}
	return n
}

// RemoveBookmarks writes a copy of a PDF with its outline (bookmarks)
// removed. A PDF that has no outline is copied through unchanged in content.
// A /PageMode of /UseOutlines is removed with it, so viewers do not open the
// copy showing an empty bookmarks panel. It is the equivalent of
// `cpdf -remove-bookmarks`.
//
// outputPath must not name the input file, and the output appears there
// only if the whole operation succeeds.
func RemoveBookmarks(ctx *Context, inputPath, outputPath string) error {
	return opRemoveBookmarks(ctx, inputPath, outputPath)
}
