// Package mupdf - Page Overlay
//
// This file composites the pages of one PDF onto the pages of another.
package mupdf

// DefaultMaxContentBytes is how much decoded content OverlayPages and
// UprightPages will copy into their output unless told otherwise: 1 GiB.
const DefaultMaxContentBytes int64 = 1 << 30

// OverlayOptions controls OverlayPages. The zero value, or a nil pointer,
// gives the defaults.
type OverlayOptions struct {
	// MaxContentBytes caps the decoded page content the operation copies
	// into the output. A small PDF can refer to one compressed stream
	// thousands of times, and each reference has to be decoded and kept
	// until the output is saved; the cap stops such a file exhausting
	// memory. Zero means DefaultMaxContentBytes; a negative value removes
	// the limit.
	MaxContentBytes int64
}

// OverlayPages draws page i of overlayPath on top of page i of inputPath and
// writes the result to outputPath. Both files must have the same page count.
//
// The overlay is placed at the identity transform: a coordinate in the input
// page's stored (MediaBox) space means the same thing in the overlay page.
// The input page's own content, boxes and /Rotate are left untouched, so an
// overlay built for a rotated page must be authored in that page's stored
// space, or the input made upright first with UprightPages.
//
// Each overlay page becomes a form XObject drawn after the input page's
// content. That content stays where it is, wrapped in a q/Q pair sized to
// whatever unmatched q or Q operators it has, so that the graphics state it
// leaves behind cannot displace or recolour the overlay. The finished page
// is then read back to confirm the overlay really is drawn in a clean state.
// If it is not (the content ends inside a text object or marked content, or
// in an unterminated string or inline image that would swallow whatever
// follows), the input page's content is sealed in a form XObject of its own
// instead. That always isolates it, but a tagged PDF's structure tree can
// no longer resolve marked content on that page, which is why it is used
// only when needed.
//
// Fonts and other resources shared by several overlay pages are copied into
// the output once. Overlay pages that draw nothing are skipped. Only page
// content is carried over; the overlay's annotations, bookmarks and
// document-level data are not.
//
// This is the equivalent of `cpdf -combine-pages`, except that it does not
// bake the input page's /Rotate into its content, and does not copy the
// overlay's named destinations into the catalog (the artefact
// RepairCatalogDests exists to undo).
//
// outputPath must not name either input. The output appears there only if
// the whole operation succeeds.
func OverlayPages(ctx *Context, inputPath, overlayPath, outputPath string, opts *OverlayOptions) error {
	if opts == nil {
		opts = &OverlayOptions{}
	}
	return opOverlay(ctx, inputPath, overlayPath, outputPath, resolveLimit(opts.MaxContentBytes, DefaultMaxContentBytes))
}
