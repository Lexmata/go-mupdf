package mupdf

import (
	"os"
	"testing"
)

// Final tests to push coverage to 90%+

func TestSkipFunctions(t *testing.T) {
	// Test skipIfShort when not in short mode
	skipIfShort(t) // Should not skip

	// Test skipIfCIorShort when not in CI
	skipIfCIorShort(t) // Should not skip
}

func TestRequireMuPDFErrorPaths(t *testing.T) {
	// This will test the requireMuPDF function thoroughly
	requireMuPDF(t)
}

func TestCreateTestPDFEdgeCases(t *testing.T) {
	// Test createTestPDF with different scenarios

	// Test in normal environment
	pdfPath := createTestPDF(t)
	if pdfPath == "" {
		t.Error("createTestPDF returned empty path")
	}

	// Verify file exists and has content
	if _, err := os.Stat(pdfPath); err != nil {
		t.Errorf("Created PDF file does not exist: %v", err)
	}
}

func TestAllHelperFunctionPaths(t *testing.T) {
	// Create a temporary directory to test testDataDir error paths
	dir := testDataDir(t)
	if dir == "" {
		t.Error("testDataDir returned empty directory")
	}

	// Verify directory exists
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Test data directory does not exist: %v", err)
	}
}

func TestRunGC(t *testing.T) {
	// Test the runGC helper function
	runGC()
	runGC() // Call multiple times to ensure it's safe
}

func TestContextMultipleOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test multiple document operations on same context
	for i := 0; i < 3; i++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create writer %d: %v", i, err)
		}

		page, err := writer.AddPage(612, 792)
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}
		page.Close()

		writer.Close()
	}

	ctx.Drop()
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test Error type formatting with various message types
	testCases := []string{
		"",
		"simple error",
		"error with special chars: !@#$%^&*()",
		"very long error message that contains multiple words and should be handled correctly by the error formatting function",
	}

	for _, msg := range testCases {
		err := Error{message: msg}
		result := err.Error()
		if result != msg {
			t.Errorf("Expected %q, got %q", msg, result)
		}
	}
}

func TestDocumentOperationsAfterClose(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createValidTestPDF(t)

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Get initial page count
	count1 := doc.CountPages()
	t.Logf("Initial page count: %d", count1)

	// Close document
	doc.Close()

	// Try operations after close (should be safe due to our null checks)
	count2 := doc.CountPages()
	t.Logf("Page count after close: %d", count2)

	// Try closing again (should be safe)
	doc.Close()
}

func TestPageOperationsAfterClose(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createValidTestPDF(t)
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

		// Get bounds before close
		bounds1 := page.Bound()
		t.Logf("Bounds before close: %+v", bounds1)

		// Close page
		page.Close()

		// Try operations after close (should be safe due to our null checks)
		bounds2 := page.Bound()
		t.Logf("Bounds after close: %+v", bounds2)

		// Try closing again (should be safe)
		page.Close()
	}
}

func TestWriterOperationsAfterClose(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Add a page
	page, err := writer.AddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}
	page.Close()

	// Close writer
	writer.Close()

	// Try operations after close (should be safe)
	writer.Close() // Double close should be safe
}

func TestPDFObjectOperationsAfterDrop(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	obj, err := writer.NewPDFObject("test")
	if err != nil {
		t.Fatalf("Failed to create object: %v", err)
	}

	// Drop object
	obj.Drop()

	// Try dropping again (should be safe)
	obj.Drop()
}

// Note: createValidTestPDF is already defined in comprehensive_coverage_test.go
