// Package mupdf - Lossless Compression
//
// This file shrinks PDFs without changing how they render.
package mupdf

// CompressPDF writes a smaller, visually identical copy of a PDF.
//
// It is lossless: unreachable objects are dropped, identical objects and
// streams are merged, streams stored uncompressed are deflated, and small
// objects are packed into object streams (PDF 1.5). Content streams are not
// rewritten and images are not re-encoded or downsampled. It is the
// MuPDF-native equivalent of `cpdf -squeeze`; OptimizePDF does a similar job
// through pdfcpu, which rejects some files MuPDF accepts.
//
// The output can be larger than the input for a file that is already tightly
// packed, so callers that only want a saving should compare sizes and keep
// the smaller file. outputPath must not name the input file, and the output
// appears there only if the whole operation succeeds.
func CompressPDF(ctx *Context, inputPath, outputPath string) error {
	return opCompress(ctx, inputPath, outputPath)
}
