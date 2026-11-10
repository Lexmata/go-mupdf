// Package mupdf - cgo bindings for the operations implemented in ops_*.c
//
// Everything that crosses into C for rendering (render.go), page geometry
// (geometry.go), checking and repair (repair.go), overlay (overlay.go),
// upright (upright.go), bookmark removal (outline.go), catalog /Dests repair
// (catalog_dests.go) and compression (compress.go) is in this file. Those
// files hold the exported API and its documentation and are plain Go; this
// one holds the plumbing they share: taking the context lock, turning a C
// error message into a Go error, cancellation, and writing output files
// safely.
package mupdf

/*
#include <stdlib.h>
#include "ops.h"
*/
import "C"

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

// runOp runs fn with the context locked and a zeroed error buffer of
// GOMUPDF_ERRLEN bytes. fn returns false to signal that the buffer holds a
// MuPDF error message, which runOp wraps in an Error so callers can match it
// with errors.As.
func runOp(ctx *Context, op string, fn func(c *C.fz_context, errBuf *C.char) bool) error {
	if ctx == nil {
		return fmt.Errorf("go-mupdf: %s: nil context", op)
	}

	var opErr error
	ctx.withLock(func(c *C.fz_context) {
		if c == nil {
			opErr = fmt.Errorf("go-mupdf: %s: context has been dropped", op)
			return
		}
		errBuf := (*C.char)(C.calloc(1, C.GOMUPDF_ERRLEN))
		if errBuf == nil {
			opErr = fmt.Errorf("go-mupdf: %s: out of memory", op)
			return
		}
		defer C.free(unsafe.Pointer(errBuf))

		if !fn(c, errBuf) {
			opErr = fmt.Errorf("go-mupdf: %s: %w", op, Error{message: C.GoString(errBuf)})
		}
	})
	return opErr
}

// withCancel runs fn with a cancel flag that is raised when goCtx is done.
// A context that can never be done gets a nil flag and costs nothing. If
// the operation failed and goCtx is done, the context's error is returned,
// so callers can test for context.Canceled or context.DeadlineExceeded.
func withCancel(goCtx context.Context, op string, fn func(cancel *C.fz_cookie) error) error {
	if goCtx == nil || goCtx.Done() == nil {
		return fn(nil)
	}
	if err := goCtx.Err(); err != nil {
		return fmt.Errorf("go-mupdf: %s: %w", op, err)
	}

	cancel := C.gomupdf_new_cancel()
	if cancel == nil {
		return fmt.Errorf("go-mupdf: %s: out of memory", op)
	}
	// The watcher writes to the flag, so it must have stopped before the
	// flag is freed: wait for it first.
	stop := make(chan struct{})
	var watcher sync.WaitGroup
	watcher.Add(1)
	go func() {
		defer watcher.Done()
		select {
		case <-goCtx.Done():
			C.gomupdf_cancel(cancel)
		case <-stop:
		}
	}()

	err := fn(cancel)
	close(stop)
	watcher.Wait()
	C.gomupdf_free_cancel(cancel)

	if err != nil && goCtx.Err() != nil {
		return fmt.Errorf("go-mupdf: %s: %w", op, goCtx.Err())
	}
	return err
}

// checkInputOutput rejects empty paths and an output that names the same
// file as an input. MuPDF reads its input lazily while it writes, so saving
// over the file being read corrupts it.
//
// The error names the operation but not the paths: errors from this package
// are often passed on to end users, and server paths are not theirs to see.
func checkInputOutput(op, outputPath string, inputPaths ...string) error {
	if outputPath == "" {
		return fmt.Errorf("go-mupdf: %s: output path must not be empty", op)
	}
	for _, in := range inputPaths {
		if in == "" {
			return fmt.Errorf("go-mupdf: %s: input path must not be empty", op)
		}
		if sameFile(in, outputPath) {
			return fmt.Errorf("go-mupdf: %s: output path must differ from every input path", op)
		}
	}
	return nil
}

// sameFile reports whether two paths name the same file, either textually or
// (when both exist) through links.
func sameFile(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil && absA == absB {
		return true
	}
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}

// tempBeside returns a path for a temporary file in the same directory as
// path, so that renaming it over path stays on one filesystem and is atomic.
// The file is not created: MuPDF creates it, with the permissions it would
// have given the real output.
func tempBeside(path string) (string, error) {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	name := "." + filepath.Base(path) + ".gomupdf-" + hex.EncodeToString(suffix[:]) + ".tmp"
	return filepath.Join(filepath.Dir(path), name), nil
}

// runFileOp runs an operation that reads the input files and writes
// outputPath.
//
// MuPDF saves by truncating its output file and writing into it, and a save
// can fail partway through, most of all on the damaged input these
// operations exist for. So fn is given a temporary path beside outputPath
// instead, and that file is renamed into place only once fn has succeeded.
// On failure it is removed, and whatever was at outputPath before is still
// there, untouched.
func runFileOp(ctx *Context, op, outputPath string, inputPaths []string,
	fn func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool) error {
	if err := checkInputOutput(op, outputPath, inputPaths...); err != nil {
		return err
	}
	tmp, err := tempBeside(outputPath)
	if err != nil {
		return fmt.Errorf("go-mupdf: %s: %w", op, err)
	}

	cIn := make([]*C.char, len(inputPaths))
	for i, p := range inputPaths {
		cIn[i] = C.CString(p)
		defer C.free(unsafe.Pointer(cIn[i]))
	}
	cTmp := C.CString(tmp)
	defer C.free(unsafe.Pointer(cTmp))

	err = runOp(ctx, op, func(c *C.fz_context, errBuf *C.char) bool {
		return fn(c, cIn, cTmp, errBuf)
	})
	if err == nil {
		if renameErr := os.Rename(tmp, outputPath); renameErr != nil {
			err = fmt.Errorf("go-mupdf: %s: %w", op, renameErr)
		}
	}
	if err != nil {
		// Best effort: the file may never have been created.
		_ = os.Remove(tmp)
	}
	return err
}

// ---------------------------------------------------------------------------
// Typed bindings, one per C entry point
// ---------------------------------------------------------------------------

func opRenderPage(goCtx context.Context, doc *Document, pageNum int, dpi float64, maxPixels int64) (*image.RGBA, error) {
	const op = "RenderPage"

	var img *image.RGBA
	err := withCancel(goCtx, op, func(cancel *C.fz_cookie) error {
		var closed bool
		opErr := runOp(doc.ctx, op, func(c *C.fz_context, errBuf *C.char) bool {
			// Checked under the context lock, so the document cannot be
			// closed between the check and its use.
			if doc.doc == nil {
				closed = true
				return true
			}
			pix := C.gomupdf_render_page(c, doc.doc, C.int(pageNum), C.float(dpi), C.int64_t(maxPixels), cancel, errBuf)
			if pix == nil {
				return false
			}
			defer C.fz_drop_pixmap(c, pix)

			width := int(C.fz_pixmap_width(c, pix))
			height := int(C.fz_pixmap_height(c, pix))
			stride := int(C.fz_pixmap_stride(c, pix))
			samples := unsafe.Slice((*byte)(unsafe.Pointer(C.fz_pixmap_samples(c, pix))), stride*height)

			// The pixmap is premultiplied RGBA with alpha 255 throughout,
			// which is exactly image.RGBA's layout; only the row stride can
			// differ.
			img = image.NewRGBA(image.Rect(0, 0, width, height))
			for y := 0; y < height; y++ {
				copy(img.Pix[y*img.Stride:y*img.Stride+width*4], samples[y*stride:y*stride+width*4])
			}
			return true
		})
		if opErr == nil && closed {
			return fmt.Errorf("go-mupdf: %s: document is closed", op)
		}
		return opErr
	})
	if err != nil {
		return nil, err
	}
	return img, nil
}

func opPageGeometry(ctx *Context, path string) ([]PageGeometry, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var pages []PageGeometry
	err := runOp(ctx, "GetPageGeometry", func(c *C.fz_context, errBuf *C.char) bool {
		var list *C.gomupdf_page_geometry
		var count C.int
		if C.gomupdf_page_geometry_list(c, cPath, &list, &count, errBuf) != 0 {
			return false
		}
		defer C.gomupdf_free(c, unsafe.Pointer(list))

		pages = make([]PageGeometry, int(count))
		for i, g := range unsafe.Slice(list, int(count)) {
			pages[i] = PageGeometry{
				MediaBox: Rect{
					X0: float64(g.x0),
					Y0: float64(g.y0),
					X1: float64(g.x1),
					Y1: float64(g.y1),
				},
				HasMediaBox: g.has_media_box != 0,
				Rotation:    int(g.rotate),
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func opCheck(goCtx context.Context, ctx *Context, path string) (*CheckResult, error) {
	const op = "CheckPDF"
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var result *CheckResult
	err := withCancel(goCtx, op, func(cancel *C.fz_cookie) error {
		return runOp(ctx, op, func(c *C.fz_context, errBuf *C.char) bool {
			// The result holds no Go pointers, so it can live in Go memory.
			// MuPDF's warning and error callbacks point at it during the
			// call; gomupdf_check puts the previous callbacks back before
			// it returns, on every path.
			var res C.gomupdf_check_result
			if C.gomupdf_check(c, cPath, &res, cancel, errBuf) != 0 {
				return false
			}
			result = &CheckResult{
				Repaired:     res.repaired != 0,
				WarningCount: int(res.warning_count),
			}
			if text := strings.TrimSpace(C.GoString(&res.warnings[0])); text != "" {
				result.Warnings = strings.Split(text, "\n")
			}
			return true
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func opRepair(ctx *Context, inputPath, outputPath string, objectStreams bool, id []byte) error {
	var cID *C.uchar
	if id != nil {
		cID = (*C.uchar)(C.CBytes(id))
		defer C.free(unsafe.Pointer(cID))
	}
	useObjStms := C.int(0)
	if objectStreams {
		useObjStms = 1
	}
	return runFileOp(ctx, "RepairPDF", outputPath, []string{inputPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_repair(c, in[0], out, useObjStms, cID, errBuf) == 0
		})
}

func opOverlay(ctx *Context, inputPath, overlayPath, outputPath string, maxBytes int64) error {
	return runFileOp(ctx, "OverlayPages", outputPath, []string{inputPath, overlayPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_overlay_pages(c, in[0], in[1], out, C.int64_t(maxBytes), errBuf) == 0
		})
}

func opUpright(ctx *Context, inputPath, outputPath string, maxBytes int64) (int, error) {
	var changed C.int
	err := runFileOp(ctx, "UprightPages", outputPath, []string{inputPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_upright_pages(c, in[0], out, C.int64_t(maxBytes), &changed, errBuf) == 0
		})
	if err != nil {
		return 0, err
	}
	return int(changed), nil
}

func opRemoveBookmarks(ctx *Context, inputPath, outputPath string) error {
	return runFileOp(ctx, "RemoveBookmarks", outputPath, []string{inputPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_remove_bookmarks(c, in[0], out, errBuf) == 0
		})
}

func opRepairCatalogDests(ctx *Context, inputPath, outputPath string) (CatalogDestsRepair, error) {
	var outcome C.int
	err := runFileOp(ctx, "RepairCatalogDests", outputPath, []string{inputPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_repair_catalog_dests(c, in[0], out, &outcome, errBuf) == 0
		})
	if err != nil {
		return CatalogDestsUnchanged, err
	}
	switch outcome {
	case C.GOMUPDF_DESTS_REMOVED_DUPLICATE:
		return CatalogDestsRemovedDuplicate, nil
	case C.GOMUPDF_DESTS_MOVED_TO_NAMES:
		return CatalogDestsMovedToNames, nil
	case C.GOMUPDF_DESTS_MERGED:
		return CatalogDestsMerged, nil
	default:
		return CatalogDestsUnchanged, nil
	}
}

func opCompress(ctx *Context, inputPath, outputPath string) error {
	return runFileOp(ctx, "CompressPDF", outputPath, []string{inputPath},
		func(c *C.fz_context, in []*C.char, out *C.char, errBuf *C.char) bool {
			return C.gomupdf_compress(c, in[0], out, errBuf) == 0
		})
}

// resolveLimit turns an option value into what the C side expects: zero
// means the default, a negative value means no limit (0 in C).
func resolveLimit(value, def int64) int64 {
	switch {
	case value == 0:
		return def
	case value < 0:
		return 0
	default:
		return value
	}
}
