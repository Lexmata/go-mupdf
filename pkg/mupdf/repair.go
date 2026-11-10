// Package mupdf - Structural Checking and Repair
//
// This file checks whether a PDF is structurally sound and rewrites damaged
// PDFs into clean ones.
package mupdf

import (
	"context"
	"crypto/md5" //nolint:gosec // file identifier, not a security control
	"fmt"
	"io"
	"os"
)

// CheckResult describes a PDF that MuPDF could read in full.
type CheckResult struct {
	// Repaired is true when MuPDF had to rebuild the cross-reference table
	// to read the file. The content was recovered, but the file on disk is
	// damaged; RepairPDF writes a clean copy.
	Repaired bool

	// Warnings are the non-fatal problems MuPDF reported while reading,
	// including errors it recovered from on its own.
	// WarningCount can exceed len(Warnings): only the messages that fit in a
	// fixed buffer are kept.
	Warnings     []string
	WarningCount int
}

// Clean reports whether the file was read without repair or warnings.
func (r *CheckResult) Clean() bool {
	return r != nil && !r.Repaired && r.WarningCount == 0
}

// CheckPDF reads a PDF's whole structure and reports what it found.
//
// It opens the file, walks the page tree, loads every object in the
// cross-reference table and reads every stream's raw bytes, comparing the
// amount read with the stream's declared /Length. Streams are not decoded,
// so the cost is roughly that of reading the file once, and a stream whose
// compressed data is corrupt but of the right length is not detected. In an
// encrypted file stream lengths cannot be compared and are not.
//
// The error is non-nil only when the file cannot be opened as a PDF at all.
// A file that MuPDF could recover is returned with Repaired or Warnings set
// and a nil error, and so is a file in which individual objects cannot be
// loaded: each is reported as a warning naming the object. This mirrors
// `qpdf --check`, which exits 3 (warnings) rather than 2 (errors) for a
// damaged but recoverable file. Callers that want only pristine files should
// test CheckResult.Clean.
//
// Messages MuPDF would otherwise print to stderr while reading the file are
// captured in Warnings instead.
//
// CheckPDF cannot be interrupted; use CheckPDFContext for large or
// untrusted files.
func CheckPDF(ctx *Context, path string) (*CheckResult, error) {
	return opCheck(context.Background(), ctx, path)
}

// CheckPDFContext is CheckPDF with cancellation. When goCtx is done the
// check stops at the next object and the context's error is returned.
func CheckPDFContext(goCtx context.Context, ctx *Context, path string) (*CheckResult, error) {
	return opCheck(goCtx, ctx, path)
}

// RepairOptions controls how RepairPDF writes its output.
type RepairOptions struct {
	// ObjectStreams packs small objects into compressed object streams
	// (PDF 1.5). The default, false, writes every object in plain form,
	// which every PDF reader accepts; it is the equivalent of
	// `qpdf --object-streams=disable`.
	ObjectStreams bool

	// DeterministicID makes the file identifier a function of the input
	// file's bytes instead of a random value, so that repairing the same
	// input twice yields byte-identical output. An existing permanent
	// identifier (the first half of /ID) is kept.
	//
	// The identifier is the MD5 of the input file, written in the clear.
	// Anyone holding the output and a candidate original can tell whether
	// the one was made from the other. Leave this off if that matters.
	DeterministicID bool
}

// RepairPDF rewrites a PDF into a structurally clean file.
//
// MuPDF rebuilds a damaged cross-reference table when it opens a file;
// RepairPDF then saves the document in full, producing a fresh xref and
// trailer and dropping objects nothing refers to. Page content, fonts and
// images are copied through unchanged. Running it on a healthy file is safe
// and simply normalises the structure.
//
// A nil opts is the same as the zero RepairOptions. outputPath must not name
// the input file. The output appears at outputPath only if the whole
// operation succeeds; on failure nothing is created there and a file already
// at that path is left as it was.
func RepairPDF(ctx *Context, inputPath, outputPath string, opts *RepairOptions) error {
	const op = "RepairPDF"
	if err := checkInputOutput(op, outputPath, inputPath); err != nil {
		return err
	}
	if opts == nil {
		opts = &RepairOptions{}
	}

	var id []byte
	if opts.DeterministicID {
		sum, err := fileDigest(inputPath)
		if err != nil {
			return fmt.Errorf("go-mupdf: %s: %w", op, err)
		}
		id = sum
	}
	return opRepair(ctx, inputPath, outputPath, opts.ObjectStreams, id)
}

// fileDigest returns the 16-byte MD5 of a file, the size and algorithm PDF
// writers conventionally use for /ID.
func fileDigest(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	h := md5.New() //nolint:gosec // file identifier, not a security control
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
