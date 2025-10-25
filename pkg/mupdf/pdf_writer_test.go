package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDebugPDFCreation is a focused test to debug PDF creation issues
func TestDebugPDFCreation(t *testing.T) {
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
	t.Logf("Page created: %+v", page)

	// Save the PDF first to check if it can be created
	dir, err := os.MkdirTemp("", "mupdf-debug")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "debug.pdf")

	t.Logf("Saving PDF to: %s", pdfPath)

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	fileInfo, err := os.Stat(pdfPath)
	if err != nil {
		t.Fatalf("Failed to stat PDF file: %v", err)
	}
	t.Logf("PDF file size: %d bytes", fileInfo.Size())

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	docPageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", docPageCount)

	if docPageCount > 0 {
		// Try to load the page
		docPage, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer docPage.Close()

		// Get page bounds
		bounds := docPage.Bound()
		t.Logf("Page bounds: %+v", bounds)
	}
}
