package mupdf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// TestComprehensiveCoverage focuses on increasing test coverage to 90%+
// This test covers edge cases and error conditions not covered by other tests

func TestContextErrorPaths(t *testing.T) {
	// Test NewContext error handling (covers the 80% line in NewContext)
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test double drop (should be safe)
	ctx.Drop()
	ctx.Drop() // This should not crash
}

func TestDocumentErrorPaths(t *testing.T) {
	requireMuPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test opening non-existent document (covers error path in OpenDocument)
	_, err = OpenDocument(ctx, "non-existent-file.pdf")
	if err == nil {
		t.Error("Expected error when opening non-existent file")
	}

	// CountPages after Close must return the documented safe value of 0
	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	if got := doc.CountPages(); got != 1 {
		t.Errorf("Expected 1 page before Close, got %d", got)
	}
	doc.Close()
	if got := doc.CountPages(); got != 0 {
		t.Errorf("Expected CountPages to return 0 after Close, got %d", got)
	}
}

func TestPageBoundsAndExtraction(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createValidTestPDF(t)
	defer os.Remove(pdfPath)

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		// Test Bound method (covers 66.7% line in Bound)
		bounds := page.Bound()
		if bounds.X1 <= bounds.X0 || bounds.Y1 <= bounds.Y0 {
			t.Errorf("Invalid bounds: %+v", bounds)
		}

		// Test ExtractText (covers 77.8% line in ExtractText)
		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text: %v", err)
		}
		defer text.Close()

		// Test String method (covers 71.4% line in String)
		content := text.String()
		t.Logf("Extracted text length: %d", len(content))

		// Don't test text.String() after close as it causes segfault
		// This is expected behavior - accessing closed resources should be avoided
	}
}

func TestPDFDocumentOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createValidTestPDF(t)
	defer os.Remove(pdfPath)

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Test AsPDFDocument (covers 75% line)
	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("Failed to convert to PDF document: %v", err)
	}

	// Test PDF document operations
	pageCount := pdfDoc.CountPages()
	t.Logf("PDF page count: %d", pageCount)

	if pageCount > 0 {
		page, err := pdfDoc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load PDF page: %v", err)
		}
		defer page.Close()

		// Test PDF page bounds (covers 66.7% line in pdf Bound)
		bounds := page.Bound()
		t.Logf("PDF page bounds: %+v", bounds)
	}

	// Test OpenPDFDocument directly (covers 75% line)
	pdfDoc2, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF document directly: %v", err)
	}
	// PDFDocument holds a kept reference, so it must be closed explicitly
	t.Logf("Opened PDF document directly: %+v", pdfDoc2)
	pdfDoc2.Close()
}

func TestPDFWriterComprehensive(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	// Test NewPDFObject with various types (covers 85.7% line)
	testValues := []interface{}{
		nil,
		true,
		false,
		42,
		3.14,
		"test string",
	}

	for _, value := range testValues {
		obj, err := writer.NewPDFObject(value)
		if err != nil {
			t.Errorf("Failed to create PDF object for %v: %v", value, err)
			continue
		}
		obj.Drop()
	}

	// Test unsupported type (error path)
	_, err = writer.NewPDFObject([]int{1, 2, 3})
	if err == nil {
		t.Error("Expected error for unsupported type")
	}

	// Add pages with different sizes
	sizes := []struct{ w, h float64 }{
		{612, 792}, // US Letter
		{595, 842}, // A4
		{420, 595}, // A5
		{297, 420}, // A6
	}

	for i, size := range sizes {
		page, err := writer.AddPage(size.w, size.h)
		if err != nil {
			t.Errorf("Failed to add page %d: %v", i, err)
			continue
		}
		defer page.Close()
		t.Logf("Added page %d: %.0fx%.0f", i+1, size.w, size.h)
	}

	// Test saving to various locations
	dir := testDataDir(t)

	// Test save to valid location
	validPath := filepath.Join(dir, "comprehensive_test.pdf")
	err = writer.Save(validPath)
	if err != nil {
		t.Errorf("Failed to save PDF: %v", err)
	}

	// Test save to invalid location (error path)
	invalidPath := "/invalid/directory/test.pdf"
	err = writer.Save(invalidPath)
	if err == nil {
		t.Error("Expected error when saving to invalid location")
	}
}

// TestCreateCloseEdgeCases exercises creating objects and closing them
// immediately (it does not measure memory usage)
func TestCreateCloseEdgeCases(t *testing.T) {
	requireMuPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test creating and immediately closing objects
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Create page and close immediately
	page, err := writer.AddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}
	if page == nil {
		t.Fatal("AddPage returned nil page")
	}
	page.Close()

	// Create object and close immediately
	obj, err := writer.NewPDFObject("test")
	if err != nil {
		t.Fatalf("Failed to create object: %v", err)
	}
	if obj == nil {
		t.Fatal("NewPDFObject returned nil object")
	}
	obj.Drop()

	writer.Close()

	// Force garbage collection to test finalizers
	runtime.GC()
	runtime.GC()
}

// TestConcurrentSafety verifies the supported concurrency pattern: each
// goroutine creates, uses, and drops its own Context. Sharing a Context
// across goroutines is not supported.
func TestConcurrentSafety(t *testing.T) {
	requireMuPDF(t)

	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errs := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine creates its own Context
			ctx, err := NewContext()
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: failed to create context: %w", id, err)
				return
			}
			defer ctx.Drop()

			// Perform operations with this goroutine's own context
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: failed to create PDF writer: %w", id, err)
				return
			}
			defer writer.Close()

			page, err := writer.AddPage(595, 842)
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: failed to add page: %w", id, err)
				return
			}
			page.Close()
		}(i)
	}

	// Wait for all goroutines, then assert none reported an error
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestErrorMessages(t *testing.T) {
	// Test Error type
	err := Error{message: "test error"}
	expected := "test error"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}

	// Test empty error
	err2 := Error{}
	if err2.Error() != "" {
		t.Errorf("Expected empty string, got %q", err2.Error())
	}
}

// Helper function to create a valid test PDF using our working PDFWriter
func createValidTestPDF(t *testing.T) string {
	t.Helper()

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "valid_test.pdf")

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	_, err = writer.AddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	return pdfPath
}
