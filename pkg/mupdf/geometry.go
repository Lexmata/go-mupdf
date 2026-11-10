// Package mupdf - Page Geometry
//
// This file reads the stored size and rotation of PDF pages.
package mupdf

import "math"

// PageGeometry is the stored MediaBox and display rotation of one PDF page.
//
// Only the MediaBox is reported, deliberately. RenderPage rasterises the
// MediaBox, so it is the frame OCR bounding boxes are normalised against; a
// CropBox or TrimBox changes what a viewer shows but not what was recognised.
type PageGeometry struct {
	// MediaBox in the page's stored (unrotated) coordinate space, in points.
	// Inherited from the page tree when the page has none of its own. A
	// missing or degenerate box is reported as US Letter (0 0 612 792), the
	// default MuPDF renders with.
	MediaBox Rect

	// HasMediaBox is false when MediaBox is the US Letter fallback.
	HasMediaBox bool

	// Rotation is /Rotate snapped to 0, 90, 180 or 270 degrees clockwise.
	// A value that is not a multiple of 90 is malformed; it is snapped to
	// the nearest right angle, which is what RenderPage draws and what
	// UprightPages bakes in, so the three always agree.
	Rotation int
}

// Width returns the MediaBox width in points, before rotation.
func (g PageGeometry) Width() float64 {
	return math.Abs(g.MediaBox.X1 - g.MediaBox.X0)
}

// Height returns the MediaBox height in points, before rotation.
func (g PageGeometry) Height() float64 {
	return math.Abs(g.MediaBox.Y1 - g.MediaBox.Y0)
}

// IsUpright reports whether the page's stored coordinate space is the one a
// viewer shows, i.e. /Rotate is 0. See UprightPages.
func (g PageGeometry) IsUpright() bool {
	return g.Rotation == 0
}

// GetPageGeometry returns the geometry of every page of a PDF, in page order.
func GetPageGeometry(ctx *Context, path string) ([]PageGeometry, error) {
	return opPageGeometry(ctx, path)
}
