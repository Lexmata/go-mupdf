// Package mupdf - Page Rendering
//
// This file rasterises document pages to images.
package mupdf

import (
	"context"
	"fmt"
	"image"
	"math"
)

// DefaultMaxPixels is the largest image RenderPage produces unless
// RenderOptions says otherwise: 256 megapixels, a 1 GiB image. A0 at 300 DPI
// is about 139 megapixels.
const DefaultMaxPixels int64 = 256 << 20

// RenderOptions controls RenderPage. The zero value, or a nil pointer, gives
// the defaults.
type RenderOptions struct {
	// MaxPixels caps width x height of the rendered image. A page's size
	// comes from the file, so without a cap a hostile or broken PDF can ask
	// for an image of many gigabytes. Zero means DefaultMaxPixels; a
	// negative value removes the limit.
	MaxPixels int64
}

// RenderPage rasterises a page to an opaque RGBA image on a white background.
//
// pageNum is zero-based. dpi is the resolution in dots per inch; a PDF point
// is 1/72 inch, so a US Letter page at 300 DPI is 2550 x 3300 pixels.
//
// For PDF documents the image covers the page's MediaBox, not its CropBox,
// with /Rotate applied, and content outside the CropBox is drawn. That is
// the frame `pdftoppm` uses by default, and it is the frame OCR coordinates
// are normalised against: a CropBox changes what a viewer shows, not what
// gets recognised. It is not the frame Page.Bound reports, which is the
// CropBox; use GetPageGeometry for the frame this image covers. Pixel
// dimensions can differ from pdftoppm's by one pixel because the two round
// fractional sizes differently.
//
// Memory use is width x height x 4 bytes for the returned image, plus the
// same again inside MuPDF while the call runs. A page that would exceed
// RenderOptions.MaxPixels is refused before any of that is allocated.
//
// RenderPage cannot be interrupted; use RenderPageContext to bound how long
// a page may take.
func (doc *Document) RenderPage(pageNum int, dpi float64, opts *RenderOptions) (*image.RGBA, error) {
	return doc.RenderPageContext(context.Background(), pageNum, dpi, opts)
}

// RenderPageContext is RenderPage with cancellation. When goCtx is done the
// render stops at the next drawing operation and the context's error is
// returned. Content crafted to be slow (deeply nested patterns, enormous
// paths) can otherwise hold the Context's lock for minutes.
func (doc *Document) RenderPageContext(goCtx context.Context, pageNum int, dpi float64, opts *RenderOptions) (*image.RGBA, error) {
	const op = "RenderPage"
	if doc == nil {
		return nil, fmt.Errorf("go-mupdf: %s: document is nil", op)
	}
	if !(dpi > 0) || math.IsInf(dpi, 0) {
		return nil, fmt.Errorf("go-mupdf: %s: dpi must be a positive number, got %v", op, dpi)
	}
	if opts == nil {
		opts = &RenderOptions{}
	}
	return opRenderPage(goCtx, doc, pageNum, dpi, resolveLimit(opts.MaxPixels, DefaultMaxPixels))
}
