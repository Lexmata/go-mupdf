package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPDFWriterSaveRoundTrip verifies that a page added via AddPage survives a
// save/reopen round trip with its exact bounds intact. Complements
// TestPDFWriter in pdf_document_test.go, which focuses on the writer lifecycle.
func TestPDFWriterSaveRoundTrip(t *testing.T) {
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

	// Add a page using standard method
	page, err := writer.AddPage(595, 842) // A4 size
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}
	if page == nil {
		t.Fatal("AddPage returned nil page")
	}

	// Save the PDF
	dir, err := os.MkdirTemp("", "mupdf-writer-roundtrip")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "roundtrip.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists and is non-empty
	fileInfo, err := os.Stat(pdfPath)
	if err != nil {
		t.Fatalf("Failed to stat PDF file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Fatal("Saved PDF file is empty")
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// The added page must survive the round trip
	docPageCount := doc.CountPages()
	if docPageCount != 1 {
		t.Fatalf("Expected 1 page after reopen, got %d", docPageCount)
	}

	// Load the page and verify its bounds match exactly what was requested
	docPage, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer docPage.Close()

	bounds := docPage.Bound()
	want := Rect{X0: 0, Y0: 0, X1: 595, Y1: 842}
	if bounds != want {
		t.Errorf("Expected page bounds %+v, got %+v", want, bounds)
	}
}
