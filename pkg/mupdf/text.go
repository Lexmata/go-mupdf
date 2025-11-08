// Package mupdf - Text Extraction
//
// This file contains the TextPage type and related functionality for extracting
// and processing text content from document pages.
package mupdf

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/mupdf/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third  -lm

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

// Convert text page to string
char* go_mupdf_stext_page_to_string(fz_context *ctx, fz_stext_page *text, char **out_error) {
    char *result = NULL;
    *out_error = NULL;

    fz_try(ctx) {
        fz_buffer *buf = fz_new_buffer_from_stext_page(ctx, text);
        size_t len = fz_buffer_storage(ctx, buf, (unsigned char**)&result);

        // Allocate and copy the string
        char *copy = (char*)malloc(len + 1);
        memcpy(copy, result, len);
        copy[len] = '\0';

        fz_drop_buffer(ctx, buf);
        result = copy;
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
        return NULL;
    }

    return result;
}
*/
import "C"
import (
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
	if text.text != nil && text.ctx != nil && text.ctx.ctx != nil {
		C.fz_drop_stext_page(text.ctx.ctx, text.text)
		text.text = nil
	}
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
// If an error occurs during string conversion (e.g., the TextPage
// is closed or invalid), returns an empty string.
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
	var cError *C.char
	cStr := C.go_mupdf_stext_page_to_string(text.ctx.ctx, text.text, &cError)

	if cError != nil {
		C.free(unsafe.Pointer(cError))
		return ""
	}

	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr)
}
