// Package mupdf - Catalog /Dests Repair
//
// This file fixes named destinations stored in the wrong catalog slot.
package mupdf

// CatalogDestsRepair is what RepairCatalogDests did to a PDF.
type CatalogDestsRepair int

const (
	// CatalogDestsUnchanged means there was nothing to repair. The output
	// is a plain copy of the input.
	CatalogDestsUnchanged CatalogDestsRepair = iota
	// CatalogDestsRemovedDuplicate means every name in the catalog /Dests
	// was already in /Names /Dests, so the catalog copy was removed and
	// /Names /Dests left as it was.
	CatalogDestsRemovedDuplicate
	// CatalogDestsMovedToNames means the catalog /Dests held the only copy
	// of the tree and was moved to /Names /Dests.
	CatalogDestsMovedToNames
	// CatalogDestsMerged means both trees existed and the catalog /Dests
	// held names that /Names /Dests did not. They were added to it and the
	// catalog copy removed. Where a name was in both, the destination from
	// /Names /Dests was kept.
	CatalogDestsMerged
)

// String returns a short description of the outcome.
func (r CatalogDestsRepair) String() string {
	switch r {
	case CatalogDestsUnchanged:
		return "unchanged"
	case CatalogDestsRemovedDuplicate:
		return "removed duplicate"
	case CatalogDestsMovedToNames:
		return "moved to /Names"
	case CatalogDestsMerged:
		return "merged into /Names"
	default:
		return "unknown"
	}
}

// RepairCatalogDests removes or relocates a named-destination name tree that
// has been written into the catalog's PDF 1.1 /Dests slot.
//
// Since PDF 1.2 named destinations live in a name tree under
// /Root /Names /Dests; the catalog's own /Dests is the older form and must
// be a flat dictionary mapping each name to a destination. Some tools
// (notably `cpdf -combine-pages`) write a name tree into the catalog /Dests.
// Lenient readers accept that, but strict ones such as pdfcpu read /Kids
// there as a destination and reject the whole file.
//
// A catalog /Dests shaped like a name-tree node (only /Kids, /Names and
// /Limits, with values of the right shape) is moved to /Names /Dests,
// creating /Names if needed. If /Names /Dests already exists the two are
// not assumed to be copies: any name only the catalog tree has is added to
// /Names /Dests before the catalog tree is removed, so no destination is
// lost. A genuine flat PDF 1.1 /Dests is left alone.
//
// outputPath is always written, whatever the result, so a caller can carry
// on with it unconditionally. It must not name the input file, and it
// appears only if the whole operation succeeds.
func RepairCatalogDests(ctx *Context, inputPath, outputPath string) (CatalogDestsRepair, error) {
	return opRepairCatalogDests(ctx, inputPath, outputPath)
}
