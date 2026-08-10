package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPDFWriterSaveAndReopen exercises the core writer workflow: create a
// writer, add a page, save the document, and reopen it to verify the page
// round-trips.
//
// Per-feature tests (annotations, form fields, outlines, metadata) should be
// added here as each API lands. Encryption is already covered by
// TestEncryptPDF in pdfcpu_test.go.
func TestPDFWriterSaveAndReopen(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "save_and_reopen_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Fatalf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// The added page must survive the save/reopen round trip.
	if pageCount := doc.CountPages(); pageCount != 1 {
		t.Fatalf("Expected 1 page after reopen, got %d", pageCount)
	}
}
