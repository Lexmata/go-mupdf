// Package mupdf - Baking Page Rotation
//
// This file removes /Rotate from pages by rotating their content instead.
package mupdf

// UprightOptions controls UprightPages. The zero value, or a nil pointer,
// gives the defaults.
type UprightOptions struct {
	// MaxContentBytes caps the decoded page content and appearance streams
	// the operation copies into the output; see OverlayOptions. Zero means
	// DefaultMaxContentBytes; a negative value removes the limit.
	MaxContentBytes int64
}

// UprightPages bakes every page's /Rotate into its content, so that all
// pages end up with /Rotate 0. It returns the number of pages changed.
//
// A page that was rotated gets a MediaBox in display orientation whose
// lower-left corner is the origin. A page that was already upright is left
// exactly as it is, including a MediaBox that does not start at the origin;
// use GetPageGeometry on the output for each page's frame.
//
// The rendered appearance is unchanged; only the stored coordinate space
// moves. That lets a caller place marks, or build a text layer, without
// rotating them into each page's stored space. It is the equivalent of
// `cpdf -upright`.
//
// For each rotated page the MediaBox, CropBox, BleedBox, TrimBox and ArtBox
// are transformed, and the page's content is moved into a form XObject that
// carries the rotation, so that patterns and shadings turn with everything
// else. Annotations follow: their rectangles, quad points, line ends,
// vertices, ink strokes and callout lines are transformed, a widget's
// /MK /R is adjusted, and each gets its own turned copy of its appearance
// streams. An annotation flagged NoRotate is not turned; it is moved so that
// it still hangs from the same point on the page.
//
// Destinations that point at a turned page with explicit coordinates are
// rewritten to match: in links, bookmarks, named destinations and the
// document's open action. A quarter turn swaps the axes, so /FitH becomes
// /FitV and the other way round.
//
// One thing is not carried over. Because the content of a rotated page now
// lives in a form XObject, marked-content references from a tagged PDF's
// structure tree to that page no longer resolve.
//
// The output is always written, even when no page needed changing.
// outputPath must not name the input file, and the output appears there
// only if the whole operation succeeds.
func UprightPages(ctx *Context, inputPath, outputPath string, opts *UprightOptions) (int, error) {
	if opts == nil {
		opts = &UprightOptions{}
	}
	return opUpright(ctx, inputPath, outputPath, resolveLimit(opts.MaxContentBytes, DefaultMaxContentBytes))
}
