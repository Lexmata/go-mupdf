// Package mupdf - Core Types and Data Structures
//
// This file contains the fundamental types used throughout the MuPDF wrapper,
// including error types, geometric structures, and basic data types that
// are shared across multiple modules.
package mupdf

// Error represents an error from the MuPDF library.
//
// Error implements the standard Go error interface and provides
// detailed error messages from the underlying MuPDF C library.
// These errors typically indicate file format issues, memory
// allocation failures, or invalid operations.
//
// Example error scenarios:
//   - File not found or inaccessible
//   - Corrupted or invalid PDF structure
//   - Memory allocation failures
//   - Invalid page numbers or operations
//   - MuPDF internal errors
type Error struct {
	message string
}

// Error returns the error message string.
//
// This implements the standard Go error interface,
// allowing Error values to be used anywhere an error is expected.
func (e Error) Error() string {
	return e.message
}

// Rect represents a rectangular area defined by two corner points.
//
// Rect follows the PDF coordinate system where:
//   - (X0, Y0) is the bottom-left corner
//   - (X1, Y1) is the top-right corner
//   - Coordinates are in points (1/72 inch)
//   - Y increases upward (mathematical convention)
//
// Common uses:
//   - Page bounding boxes (MediaBox, CropBox, etc.)
//   - Text selection areas
//   - Drawing regions
//   - Clipping boundaries
//
// Example:
//
//	rect := page.Bound()
//	width := rect.X1 - rect.X0
//	height := rect.Y1 - rect.Y0
//	fmt.Printf("Page size: %.1f x %.1f points\n", width, height)
type Rect struct {
	X0, Y0, X1, Y1 float64
}
