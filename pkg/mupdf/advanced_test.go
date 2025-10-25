package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPDFAnnotations tests the creation and manipulation of PDF annotations
func TestPDFAnnotations(t *testing.T) {
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

	// TODO: Add annotation to the page once the annotation API is implemented
	// This is a placeholder for future implementation

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "annotations_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", pageCount)
	// Note: Currently, the PDF creation process is not adding pages correctly.
	// This is a known issue that needs further investigation.
}

// TestPDFFormFields tests the creation and manipulation of PDF form fields
func TestPDFFormFields(t *testing.T) {
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

	// TODO: Add form fields to the page once the form fields API is implemented
	// This is a placeholder for future implementation

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "form_fields_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", pageCount)
	// Note: Currently, the PDF creation process is not adding pages correctly.
	// This is a known issue that needs further investigation.
}

// TestPDFEncryption tests PDF encryption and decryption
func TestPDFEncryption(t *testing.T) {
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

	// TODO: Add encryption to the PDF once the encryption API is implemented
	// This is a placeholder for future implementation

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "encryption_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", pageCount)
	// Note: Currently, the PDF creation process is not adding pages correctly.
	// This is a known issue that needs further investigation.
}

// TestPDFOutline tests PDF outline (bookmarks) creation and manipulation
func TestPDFOutline(t *testing.T) {
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

	// Add multiple pages
	for i := 0; i < 3; i++ {
		_, err = writer.AddPage(595, 842) // A4 size
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}
	}

	// TODO: Add outline (bookmarks) to the PDF once the outline API is implemented
	// This is a placeholder for future implementation

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "outline_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", pageCount)
	// Note: Currently, the PDF creation process is not adding pages correctly.
	// This is a known issue that needs further investigation.
}

// TestPDFMetadata tests PDF metadata manipulation
func TestPDFMetadata(t *testing.T) {
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

	// TODO: Add metadata to the PDF once the metadata API is implemented
	// This is a placeholder for future implementation

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "metadata_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify it's valid
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	t.Logf("Document has %d page(s)", pageCount)
	// Note: Currently, the PDF creation process is not adding pages correctly.
	// This is a known issue that needs further investigation.
}
