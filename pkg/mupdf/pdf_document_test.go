package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPDFDocument(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Open document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Convert to PDF document
	pdf, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("Failed to convert to PDF document: %v", err)
	}

	if pdf == nil {
		t.Fatal("PDFDocument is nil")
	}

	// Test CountPages (createTestPDF produces a 1-page document)
	pageCount := pdf.CountPages()
	if pageCount != 1 {
		t.Errorf("Expected 1 page from converted PDF document, got %d", pageCount)
	}

	// Test OpenPDFDocument
	pdfDirect, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF document directly: %v", err)
	}

	if pdfDirect == nil {
		t.Fatal("PDFDocument from OpenPDFDocument is nil")
	}

	// Test CountPages on direct PDF
	pageCountDirect := pdfDirect.CountPages()
	if pageCountDirect != 1 {
		t.Errorf("Expected 1 page from directly opened PDF document, got %d", pageCountDirect)
	}
}

func TestPDFPage(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Open PDF document
	pdf, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF document: %v", err)
	}

	// Load page
	page, err := pdf.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	if page == nil {
		t.Fatal("PDFPage is nil")
	}

	// Test invalid page number
	_, err = pdf.LoadPage(-1)
	if err == nil {
		t.Error("Expected error when loading invalid page number")
	}

	_, err = pdf.LoadPage(1) // Only 1 page in test PDF
	if err == nil {
		t.Error("Expected error when loading out-of-bounds page number")
	}

	// Test Bound
	bounds := page.Bound()
	if bounds.X0 >= bounds.X1 || bounds.Y0 >= bounds.Y1 {
		t.Errorf("Invalid page bounds: %+v", bounds)
	}

	// Test Close
	page.Close()

	// Test double Close (should not panic)
	page.Close()
}

func TestPDFObject(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	// Test object creation with different types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"null", nil},
		{"bool_true", true},
		{"bool_false", false},
		{"integer", 42},
		{"float", 3.14},
		{"string", "Hello, World!"},
		// Arrays and dictionaries would require more complex testing
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := writer.NewPDFObject(tc.value)
			if err != nil {
				t.Fatalf("Failed to create PDF object with %v: %v", tc.value, err)
			}

			if obj == nil {
				t.Fatal("PDFObject is nil")
			}

			// Test Drop
			obj.Drop()

			// Test double Drop (should not panic)
			obj.Drop()
		})
	}

	// Test with a PDF document
	pdfPath := createTestPDF(t)
	pdf, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF document: %v", err)
	}

	// Test object creation with PDF document
	for _, tc := range testCases {
		t.Run("doc_"+tc.name, func(t *testing.T) {
			obj, err := pdf.NewPDFObject(tc.value)
			if err != nil {
				t.Fatalf("Failed to create PDF object with %v: %v", tc.value, err)
			}

			if obj == nil {
				t.Fatal("PDFObject is nil")
			}

			// Test Drop
			obj.Drop()
		})
	}
}

func TestPDFWriter(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}

	if writer == nil {
		t.Fatal("PDFWriter is nil")
	}

	// Add a page
	page, err := writer.AddPage(595, 842) // A4 size
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	if page == nil {
		t.Fatal("PDFPage is nil")
	}

	// Test page bounds
	bounds := page.Bound()
	if bounds.X0 != 0 || bounds.Y0 != 0 || bounds.X1 != 595 || bounds.Y1 != 842 {
		t.Errorf("Expected bounds {0, 0, 595, 842}, got %+v", bounds)
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "writer_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Test Close
	writer.Close()

	// Test double Close (should not panic)
	writer.Close()

	// Test opening the saved PDF
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// The added page must survive the save/reopen round trip.
	pageCount := doc.CountPages()
	if pageCount != 1 {
		t.Errorf("Expected 1 page after reopen, got %d", pageCount)
	}
}

func TestPDFMultiPageCreation(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	// Add multiple pages
	for i := 0; i < 3; i++ {
		page, err := writer.AddPage(595, 842) // A4 size
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}

		if page == nil {
			t.Fatalf("PDFPage %d is nil", i)
		}
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "multipage.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Open the saved PDF
	pdf, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved PDF document: %v", err)
	}

	// All three added pages must survive the save/reopen round trip.
	pageCount := pdf.CountPages()
	if pageCount != 3 {
		t.Errorf("Expected 3 pages after reopen, got %d", pageCount)
	}

	// Load each page and check bounds
	for i := 0; i < pageCount; i++ {
		page, err := pdf.LoadPage(i)
		if err != nil {
			t.Fatalf("Failed to load page %d: %v", i, err)
		}

		bounds := page.Bound()
		if bounds.X0 >= bounds.X1 || bounds.Y0 >= bounds.Y1 {
			t.Errorf("Invalid bounds for page %d: %+v", i, bounds)
		}

		page.Close()
	}
}
