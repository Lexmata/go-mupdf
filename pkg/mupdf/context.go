// Package mupdf - Context Management
//
// This file contains the Context type and related functionality for managing
// MuPDF execution contexts, including creation, lifecycle management, and
// cleanup operations.
package mupdf

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/mupdf/include
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third  -lm

#include <stdlib.h>
#include <string.h>
#include "mupdf/fitz.h"

// Get version
const char* go_mupdf_version() {
    return FZ_VERSION;
}

// Create context
fz_context* go_mupdf_new_context(char **out_error) {
    fz_context *ctx = NULL;
    *out_error = NULL;

    ctx = fz_new_context(NULL, NULL, FZ_STORE_UNLIMITED);
    if (!ctx) {
        const char *error_message = "Failed to create context";
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
        return NULL;
    }

    // Register document handlers
    fz_try(ctx) {
        // Register the default document handlers
        fz_register_document_handlers(ctx);
    }
    fz_catch(ctx) {
        const char *error_message = fz_caught_message(ctx);
        *out_error = (char*)malloc(strlen(error_message) + 1);
        strcpy(*out_error, error_message);
        fz_drop_context(ctx);
        return NULL;
    }

    return ctx;
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// GetVersion returns the version string of the underlying MuPDF library.
//
// This can be useful for:
//   - Debugging and diagnostics
//   - Feature compatibility checks
//   - Logging and version tracking
//   - Support and troubleshooting
//
// Returns a version string in the format "X.Y.Z" (e.g., "1.26.3").
//
// Example:
//
//	version := mupdf.GetVersion()
//	fmt.Printf("Using MuPDF version: %s\n", version)
func GetVersion() string {
	return C.GoString(C.go_mupdf_version())
}

// Context represents a MuPDF execution context and manages the library's
// internal state, memory allocation, and error handling.
//
// A Context is required for all MuPDF operations and should be created
// once per thread or goroutine that needs to use MuPDF functionality.
// Contexts are thread-safe and can be used concurrently, but for optimal
// performance, create separate contexts for different goroutines.
//
// The Context manages:
//   - Memory allocation and cleanup
//   - Error handling and exception state
//   - Document type registration
//   - Internal MuPDF state
//
// Memory Management:
//   - Always call Drop() when finished with a Context
//   - A finalizer provides automatic cleanup as a safety net
//   - Contexts should be long-lived for efficiency
//
// Example:
//
//	ctx, err := mupdf.NewContext()
//	if err != nil {
//	    return err
//	}
//	defer ctx.Drop() // Always cleanup
//
//	// Use ctx for document operations...
type Context struct {
	ctx *C.fz_context
}

// NewContext creates a new MuPDF execution context.
//
// This initializes the MuPDF library state and registers document handlers
// for supported file formats (PDF, XPS, CBZ, etc.). The context manages
// memory allocation and error handling for all subsequent operations.
//
// Returns:
//   - *Context: A new context ready for use
//   - error: An error if context creation or initialization fails
//
// The returned Context must be cleaned up with Drop() when no longer needed.
// A finalizer provides automatic cleanup, but explicit cleanup is recommended
// for deterministic resource management.
//
// Error conditions:
//   - Memory allocation failure
//   - Document handler registration failure
//   - MuPDF library initialization failure
//
// Example:
//
//	ctx, err := mupdf.NewContext()
//	if err != nil {
//	    log.Fatalf("Failed to create MuPDF context: %v", err)
//	}
//	defer ctx.Drop()
//
//	// Context is ready for use...
func NewContext() (*Context, error) {
	var cError *C.char
	ctx := C.go_mupdf_new_context(&cError)

	if cError != nil {
		defer C.free(unsafe.Pointer(cError))
		return nil, Error{message: C.GoString(cError)}
	}

	result := &Context{ctx: ctx}
	runtime.SetFinalizer(result, func(c *Context) {
		if c != nil {
			c.Drop()
		}
	})

	return result, nil
}

// Drop releases the MuPDF context and all associated resources.
//
// This method must be called when the Context is no longer needed to
// prevent memory leaks. It's safe to call Drop() multiple times -
// subsequent calls are no-ops.
//
// Drop() will:
//   - Release the underlying MuPDF context
//   - Free all associated memory
//   - Invalidate the Context for further use
//
// After calling Drop(), the Context should not be used for any operations.
// All Documents, Pages, and other objects created from this Context
// become invalid and should also be cleaned up.
//
// Best Practices:
//   - Use defer ctx.Drop() immediately after creating a Context
//   - Ensure Drop() is called even if errors occur
//   - Don't use the Context after calling Drop()
//
// Example:
//
//	ctx, err := mupdf.NewContext()
//	if err != nil {
//	    return err
//	}
//	defer ctx.Drop() // Guaranteed cleanup
//
//	// Use context for operations...
//	// Drop() will be called automatically when function returns
func (ctx *Context) Drop() {
	if ctx.ctx != nil {
		C.fz_drop_context(ctx.ctx)
		ctx.ctx = nil
	}
}
