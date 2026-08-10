package mupdf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIntegrationBasicWorkflow(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a PDF
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "integration_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}
	writer.Close()

	// Open the PDF as a generic document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Check page count: the writer added exactly one page
	pageCount := doc.CountPages()
	if pageCount != 1 {
		t.Fatalf("Expected 1 page in saved document, got %d", pageCount)
	}

	// Load the page
	docPage, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	// Get page bounds
	bounds := docPage.Bound()
	if bounds.X0 >= bounds.X1 || bounds.Y0 >= bounds.Y1 {
		t.Errorf("Invalid page bounds: %+v", bounds)
	}

	// Extract text (even though it's empty)
	text, err := docPage.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	// Get text content
	content := text.String()
	t.Logf("Extracted text: %q", content)

	// Clean up
	text.Close()
	docPage.Close()
	doc.Close()

	// Open as PDF document
	pdf, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF document: %v", err)
	}

	// Check page count: the same single-page document opened as a PDF
	pdfPageCount := pdf.CountPages()
	if pdfPageCount != 1 {
		t.Fatalf("Expected 1 page in PDF document, got %d", pdfPageCount)
	}

	// Load the page
	pdfPage, err := pdf.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load PDF page: %v", err)
	}

	// Get page bounds
	pdfBounds := pdfPage.Bound()
	if pdfBounds.X0 != 0 || pdfBounds.Y0 != 0 || pdfBounds.X1 != 595 || pdfBounds.Y1 != 842 {
		t.Errorf("Expected bounds {0, 0, 595, 842}, got %+v", pdfBounds)
	}

	// Clean up
	pdfPage.Close()
}

func TestIntegrationPDFObjectManipulation(t *testing.T) {
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

	// Create various PDF objects
	objTypes := []struct {
		name  string
		value interface{}
	}{
		{"null", nil},
		{"bool_true", true},
		{"bool_false", false},
		{"integer", 42},
		{"float", 3.14},
		{"string", "Hello, World!"},
	}

	for _, tc := range objTypes {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := writer.NewPDFObject(tc.value)
			if err != nil {
				t.Fatalf("Failed to create PDF object with %v: %v", tc.value, err)
			}

			if obj == nil {
				t.Fatal("PDFObject is nil")
			}

			obj.Drop()
		})
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "objects_test.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	requireMuPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test opening non-existent file
	_, err = OpenDocument(ctx, "non_existent.pdf")
	if err == nil {
		t.Error("Expected error when opening non-existent file")
	}

	// Test opening invalid file
	tempFile, err := os.CreateTemp("", "invalid.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write invalid content
	_, err = tempFile.WriteString("This is not a PDF file")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Try to open the invalid file
	_, err = OpenDocument(ctx, tempFile.Name())
	if err == nil {
		t.Error("Expected error when opening invalid PDF file")
	}

	// Test loading invalid page
	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Try to load out-of-bounds page
	_, err = doc.LoadPage(999)
	if err == nil {
		t.Error("Expected error when loading out-of-bounds page")
	}
}

// TestIntegrationRepeatedObjectLifecycle exercises repeated create/use/close
// cycles across the full object graph (it does not measure memory usage)
func TestIntegrationRepeatedObjectLifecycle(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Number of pages the writer adds per iteration
	const numPages = 5

	// Repeatedly create, use, and destroy the full object graph
	for i := 0; i < 5; i++ {
		func() {
			ctx, err := NewContext()
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}
			defer ctx.Drop()

			// Create a PDF
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				t.Fatalf("Failed to create PDF writer: %v", err)
			}

			// Add multiple pages
			for j := 0; j < numPages; j++ {
				_, err = writer.AddPage(595, 842) // A4 size
				if err != nil {
					t.Fatalf("Failed to add page: %v", err)
				}

				// Create some objects, dropping each one to avoid leaks
				nullObj, err := writer.NewPDFObject(nil)
				if err != nil {
					t.Fatalf("Failed to create null object: %v", err)
				}
				nullObj.Drop()

				intObj, err := writer.NewPDFObject(42)
				if err != nil {
					t.Fatalf("Failed to create integer object: %v", err)
				}
				intObj.Drop()

				strObj, err := writer.NewPDFObject("Test string")
				if err != nil {
					t.Fatalf("Failed to create string object: %v", err)
				}
				strObj.Drop()
			}

			// Save the PDF
			dir := testDataDir(t)
			pdfPath := filepath.Join(dir, "memory_test.pdf")

			err = writer.Save(pdfPath)
			if err != nil {
				t.Fatalf("Failed to save PDF: %v", err)
			}
			writer.Close()

			// Open the PDF
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Fatalf("Failed to open document: %v", err)
			}

			// Assert all added pages made it into the saved document
			pageCount := doc.CountPages()
			if pageCount != numPages {
				t.Fatalf("Iteration %d: expected %d pages, got %d", i, numPages, pageCount)
			}

			// Load all pages
			for j := 0; j < pageCount; j++ {
				page, err := doc.LoadPage(j)
				if err != nil {
					t.Fatalf("Failed to load page %d: %v", j, err)
				}

				text, err := page.ExtractText()
				if err != nil {
					t.Fatalf("Failed to extract text from page %d: %v", j, err)
				}

				_ = text.String()
				text.Close()
				page.Close()
			}

			doc.Close()
		}()

		// Run GC to ensure finalizers are called
		runtime.GC()
	}
}
