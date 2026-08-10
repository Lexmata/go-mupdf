// Package mupdf - Text Extraction
//
// This file contains the TextPage type and related functionality for extracting
// and processing text content from document pages.
package mupdf

/*

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"

// Extract text from page
fz_stext_page* go_mupdf_extract_text(fz_context *ctx, fz_page *page, char **out_error) {
    fz_stext_page *text = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        fz_stext_options opts = { 0 };
        text = fz_new_stext_page_from_page(ctx, page, &opts);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
    }

    return text;
}

// Convert text page to a buffer. On success returns the buffer and sets
// out_data/out_len to its storage; the caller must drop the buffer with
// fz_drop_buffer after copying the data out.
fz_buffer* go_mupdf_stext_page_to_buffer(fz_context *ctx, fz_stext_page *text,
                                         unsigned char **out_data, size_t *out_len,
                                         char **out_error) {
    fz_buffer *buf = NULL;
    *out_data = NULL;
    *out_len = 0;
    *out_error = NULL;

    fz_var(buf);

    fz_try(ctx) {
        buf = fz_new_buffer_from_stext_page(ctx, text);
        *out_len = fz_buffer_storage(ctx, buf, out_data);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
        if (buf) {
            fz_drop_buffer(ctx, buf);
        }
        return NULL;
    }

    return buf;
}
*/
import "C"
import (
	"math"
	"runtime"
	"unsafe"
)

// TextPage represents text content extracted from a document page.
//
// TextPage contains structured text information including:
//   - Character-level text data
//   - Text positioning and layout information
//   - Font and styling metadata (when available)
//   - Text flow and reading order
//
// TextPage objects are created by calling ExtractText() on a Page.
// They provide methods to access the extracted text in various formats.
//
// Memory Management:
//   - Always call Close() when finished with a TextPage
//   - A finalizer provides automatic cleanup as a safety net
//   - TextPages become invalid when their parent Page or Document is closed
//
// Text Extraction Process:
//   - MuPDF analyzes the page content stream
//   - Identifies text objects and their positions
//   - Reconstructs logical text flow and reading order
//   - Provides access to the extracted text as strings
//
// Example:
//
//	text, err := page.ExtractText()
//	if err != nil {
//	    return err
//	}
//	defer text.Close()
//
//	content := text.String()
//	fmt.Printf("Extracted %d characters\n", len(content))
type TextPage struct {
	ctx  *Context
	text *C.fz_stext_page

	// cached holds the result of the first String() call so repeated
	// calls avoid re-serializing the text page. Not consulted after
	// Close(), which reports "" regardless.
	cached    string
	hasCached bool
}

// ExtractText extracts all text content from the page.
//
// This method analyzes the page's content stream and extracts text
// objects, reconstructing the logical reading order and text flow.
// The extraction process handles various text encodings, fonts,
// and layout structures commonly found in documents.
//
// Returns:
//   - *TextPage: A text page containing the extracted text
//   - error: An error if text extraction fails
//
// The text extraction process:
//   - Parses the page's content stream
//   - Identifies text objects and their positions
//   - Reconstructs text flow and reading order
//   - Handles different text encodings and fonts
//   - Preserves layout information where possible
//
// Error conditions:
//   - Page is closed or invalid
//   - Memory allocation failure during extraction
//   - Corrupted page content stream
//   - MuPDF internal processing errors
//
// Memory Management:
//   - The returned TextPage must be closed with Close()
//   - Use defer text.Close() for automatic cleanup
//
// Example:
//
//	text, err := page.ExtractText()
//	if err != nil {
//	    log.Printf("Text extraction failed: %v", err)
//	    return
//	}
//	defer text.Close()
//
//	content := text.String()
//	if len(content) > 0 {
//	    fmt.Printf("Page contains %d characters of text\n", len(content))
//	    // Process the extracted text...
//	} else {
//	    fmt.Println("No text found on this page")
//	}
func (page *Page) ExtractText() (*TextPage, error) {
	if page.page == nil || page.ctx == nil || page.ctx.ctx == nil {
		return nil, Error{message: "page is closed or invalid"}
	}

	var cError *C.char
	text := C.go_mupdf_extract_text(page.ctx.ctx, page.page, &cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &TextPage{ctx: page.ctx, text: text}
	runtime.SetFinalizer(result, func(t *TextPage) {
		t.Close()
	})

	return result, nil
}

// Close closes the text page and releases resources
func (text *TextPage) Close() {
	text.ctx.withLock(func(c *C.fz_context) {
		if text.text == nil {
			return
		}
		if c != nil {
			C.fz_drop_stext_page(c, text.text)
		}
		text.text = nil
	})
}

// String returns the extracted text content as a UTF-8 string.
//
// This method converts the structured text information into a
// plain text string, preserving the logical reading order and
// including appropriate whitespace and line breaks to maintain
// text flow and paragraph structure.
//
// Returns:
//   - string: The text content as a UTF-8 string
//
// The returned string:
//   - Preserves the logical reading order of text
//   - Includes whitespace and line breaks for readability
//   - Uses UTF-8 encoding for proper character representation
//   - Handles various text encodings from the source document
//
// The result is computed once and cached on the TextPage, so repeated
// calls are cheap and do not re-serialize the text page.
//
// Returns an empty string if the TextPage is closed or invalid, or if
// an error occurs during string conversion. The closed check takes
// precedence over the cache: String returns "" after Close() even if a
// value was cached beforehand.
//
// Text Processing:
//   - Reconstructs text flow across text objects
//   - Adds appropriate spacing between words and lines
//   - Handles right-to-left and complex text layouts
//   - Converts to UTF-8 regardless of source encoding
//
// Example:
//
//	text, err := page.ExtractText()
//	if err != nil {
//	    return
//	}
//	defer text.Close()
//
//	content := text.String()
//
//	// Basic text processing
//	lines := strings.Split(content, "\n")
//	fmt.Printf("Text contains %d lines\n", len(lines))
//
//	// Search for specific text
//	if strings.Contains(content, "important") {
//	    fmt.Println("Found important content")
//	}
//
//	// Word count
//	words := strings.Fields(content)
//	fmt.Printf("Word count: %d\n", len(words))
func (text *TextPage) String() string {
	// The closed/invalid check deliberately precedes the cache lookup:
	// String reports "" for a closed TextPage regardless of whether a
	// value was cached earlier.
	if text.text == nil || text.ctx == nil || text.ctx.ctx == nil {
		return ""
	}

	if text.hasCached {
		return text.cached
	}

	var cError *C.char
	var data *C.uchar
	var length C.size_t
	buf := C.go_mupdf_stext_page_to_buffer(text.ctx.ctx, text.text, &data, &length, &cError)

	if cError != nil {
		C.free(unsafe.Pointer(cError))
		return ""
	}

	// GoStringN takes a C.int, so clamp rather than silently truncating
	// a length that exceeds its range.
	if uint64(length) > uint64(math.MaxInt32) {
		length = C.size_t(math.MaxInt32)
	}

	result := C.GoStringN((*C.char)(unsafe.Pointer(data)), C.int(length))
	C.fz_drop_buffer(text.ctx.ctx, buf)

	text.cached = result
	text.hasCached = true
	return result
}
