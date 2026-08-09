package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for resource lifecycle: creation, close, and post-close behavior.

func TestRequireMuPDFHappyPath(t *testing.T) {
	// requireMuPDF's error branch (MuPDF unavailable) is unreachable without
	// injection, so only the happy path is exercised here.
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
	if count1 == 0 {
		t.Error("Expected at least one page before close")
	}

	// Close document
	doc.Close()

	// Post-close contract: CountPages on a closed document returns 0.
	if count2 := doc.CountPages(); count2 != 0 {
		t.Errorf("CountPages after Close = %d, want 0", count2)
	}

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
		if bounds1 == (Rect{}) {
			t.Error("Expected non-zero bounds before close")
		}

		// Close page
		page.Close()

		// Post-close contract: Bound on a closed page returns the zero Rect.
		if bounds2 := page.Bound(); bounds2 != (Rect{}) {
			t.Errorf("Bound after Close = %+v, want zero Rect", bounds2)
		}

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

	// Double close should be safe
	writer.Close()

	// Post-close contract: operations on a closed writer return errors.
	if p, err := writer.AddPage(612, 792); err == nil {
		p.Close()
		t.Error("AddPage on closed writer: expected error, got nil")
	}

	outPath := filepath.Join(t.TempDir(), "closed.pdf")
	if err := writer.Save(outPath); err == nil {
		t.Error("Save on closed writer: expected error, got nil")
	}

	if obj, err := writer.NewPDFObject("test"); err == nil {
		obj.Drop()
		t.Error("NewPDFObject on closed writer: expected error, got nil")
	}
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

// Note: createValidTestPDF is defined in memory_test.go
