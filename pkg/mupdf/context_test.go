package mupdf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("Expected non-empty version string")
	}
	t.Logf("MuPDF version: %s", version)
}

func TestContext(t *testing.T) {
	requireMuPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	if ctx == nil {
		t.Fatal("Context is nil")
	}

	if ctx.ctx == nil {
		t.Fatal("Internal context pointer is nil")
	}

	// Test Drop
	ctx.Drop()

	// Test double Drop (should not panic)
	ctx.Drop()
}

func TestDocument(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a test PDF
	pdfPath := createTestPDF(t)
	t.Logf("Test PDF path: %s", pdfPath)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()
	t.Log("Created new context for opening document")

	// Test opening a document
	t.Logf("Opening document: %s", pdfPath)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	t.Log("Document opened successfully")

	if doc == nil {
		t.Fatal("Document is nil")
	}

	// Test CountPages
	t.Log("Counting pages in document")
	pageCount := doc.CountPages()
	t.Logf("Document has %d pages", pageCount)
	if pageCount != 1 {
		t.Errorf("Expected 1 page, got %d", pageCount)
	}

	// Test Close
	doc.Close()

	// Test double Close (should not panic)
	doc.Close()

	// Test opening non-existent document
	_, err = OpenDocument(ctx, "non-existent.pdf")
	if err == nil {
		t.Error("Expected error when opening non-existent document")
	}
}

func TestPage(t *testing.T) {
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

	// Test LoadPage
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	if page == nil {
		t.Fatal("Page is nil")
	}

	// Test invalid page number
	_, err = doc.LoadPage(-1)
	if err == nil {
		t.Error("Expected error when loading invalid page number")
	}

	_, err = doc.LoadPage(1) // Only 1 page in test PDF
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

func TestTextExtraction(t *testing.T) {
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

	// Load page
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	// Test ExtractText
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if text == nil {
		t.Fatal("TextPage is nil")
	}

	// Test String method
	content := text.String()
	t.Logf("Extracted text: %q", content)

	// Test Close
	text.Close()

	// Test double Close (should not panic)
	text.Close()
}

func TestErrorHandling(t *testing.T) {
	requireMuPDF(t)

	// Test Error type
	err := Error{message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got %q", err.Error())
	}
}

// TestRepeatedCreateCloseCycles exercises repeated create/use/close cycles of
// every object type (it does not measure memory usage)
func TestRepeatedCreateCloseCycles(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Repeatedly create, use, and destroy the full object graph
	for i := 0; i < 10; i++ {
		func() {
			ctx, err := NewContext()
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}
			defer ctx.Drop()

			// Create a test PDF
			dir, err := os.MkdirTemp("", "mupdf-test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(dir)

			pdfPath := filepath.Join(dir, "test.pdf")

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
			err = writer.Save(pdfPath)
			if err != nil {
				t.Fatalf("Failed to save PDF: %v", err)
			}

			writer.Close()

			// Open document
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Fatalf("Failed to open document: %v", err)
			}

			// Assert the saved page made it into the document
			if pageCount := doc.CountPages(); pageCount != 1 {
				t.Fatalf("Iteration %d: expected 1 page, got %d", i, pageCount)
			}

			// Load page
			page, err := doc.LoadPage(0)
			if err != nil {
				t.Fatalf("Failed to load page: %v", err)
			}

			// Extract text
			text, err := page.ExtractText()
			if err != nil {
				t.Fatalf("Failed to extract text: %v", err)
			}

			// Close everything in reverse order
			text.Close()
			page.Close()
			doc.Close()
		}()

		// Run GC to ensure finalizers are called
		runtime.GC()
	}
}
