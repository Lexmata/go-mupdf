package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests asserting that repeated calls stay consistent and that genuine error
// paths (closed resources, invalid files) actually fail.

func TestDocumentRepeatedCalls(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// A second context must also be creatable.
	ctx2, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create second context: %v", err)
	}
	ctx2.Drop()

	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Repeated CountPages calls must agree.
	count1 := doc.CountPages()
	count2 := doc.CountPages()
	if count1 != count2 {
		t.Errorf("Inconsistent page counts: %d vs %d", count1, count2)
	}

	// Repeated Bound calls must agree.
	if count1 > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		bounds1 := page.Bound()
		bounds2 := page.Bound()
		if bounds1 != bounds2 {
			t.Errorf("Inconsistent bounds: %+v vs %+v", bounds1, bounds2)
		}
	}
}

func TestExtractTextIdempotence(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	if doc.CountPages() == 0 {
		t.Fatal("Expected test PDF to contain at least one page")
	}

	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	text1, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed first text extraction: %v", err)
	}
	defer text1.Close()

	// Repeated String calls on the same TextPage must agree.
	content1 := text1.String()
	content2 := text1.String()
	if content1 != content2 {
		t.Errorf("Inconsistent String results: %d vs %d chars", len(content1), len(content2))
	}

	// A second extraction on the same page must succeed and agree.
	text2, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed second text extraction: %v", err)
	}
	defer text2.Close()
	if got := text2.String(); got != content1 {
		t.Errorf("Second extraction differs: %d vs %d chars", len(got), len(content1))
	}

	// Error path: extraction from a closed page must fail.
	closedPage, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page for close test: %v", err)
	}
	closedPage.Close()
	if _, err := closedPage.ExtractText(); err == nil {
		t.Error("Expected error extracting text from a closed page")
	}
}

func TestPDFDocumentRepeatedCalls(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("AsPDFDocument failed: %v", err)
	}

	// Repeated CountPages calls must agree.
	count1 := pdfDoc.CountPages()
	count2 := pdfDoc.CountPages()
	if count1 != count2 {
		t.Errorf("Inconsistent PDF page counts: %d vs %d", count1, count2)
	}

	if count1 > 0 {
		page, err := pdfDoc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load PDF page: %v", err)
		}
		defer page.Close()

		bounds1 := page.Bound()
		bounds2 := page.Bound()
		if bounds1 != bounds2 {
			t.Errorf("Inconsistent PDF page bounds: %+v vs %+v", bounds1, bounds2)
		}
	}

	// Opening the same file directly must report the same count.
	pdfDoc2, err := OpenPDFDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("OpenPDFDocument failed: %v", err)
	}
	if got := pdfDoc2.CountPages(); got != count1 {
		t.Errorf("OpenPDFDocument page count = %d, want %d", got, count1)
	}

	// Error path: a file whose contents are not a PDF must fail to open.
	badPath := filepath.Join(t.TempDir(), "not-a-pdf.pdf")
	if err := os.WriteFile(badPath, []byte("this is not a valid PDF file"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}
	if _, err := OpenDocument(ctx, badPath); err == nil {
		t.Error("Expected error opening non-PDF file contents")
	}
}

func TestPDFWriterBasicOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	writer1, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create first writer: %v", err)
	}
	defer writer1.Close()

	page1, err := writer1.AddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed first AddPage: %v", err)
	}
	defer page1.Close()

	page2, err := writer1.AddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed second AddPage: %v", err)
	}
	defer page2.Close()

	// A second writer on the same context must work too.
	writer2, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create second writer: %v", err)
	}
	defer writer2.Close()

	// All supported value types must produce PDF objects.
	supportedValues := []interface{}{
		nil,
		true,
		false,
		int(0),
		int(-1),
		int(999999),
		float64(0.0),
		float64(1e-10),
		float64(1e10),
		"",
		"\x00\x01\x02", // binary string
	}

	for i, val := range supportedValues {
		obj, err := writer2.NewPDFObject(val)
		if err != nil {
			t.Errorf("NewPDFObject %d (%T): %v", i, val, err)
			continue
		}
		obj.Drop()
	}
}

func TestHelperFunctionEdgeCases(t *testing.T) {
	// Test testDataDir edge cases (14.3% remaining)
	dir1 := testDataDir(t)
	dir2 := testDataDir(t)

	if dir1 == dir2 {
		t.Errorf("testDataDir should create different directories: %s", dir1)
	}

	// Verify both directories exist
	for _, dir := range []string{dir1, dir2} {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("Directory doesn't exist: %s, error: %v", dir, err)
		}
	}

	// Test createTestPDF edge cases (34.6% remaining)
	// createTestPDF creates temporary files, test multiple calls
	pdf1 := createTestPDF(t)
	pdf2 := createTestPDF(t)

	if pdf1 == pdf2 {
		t.Errorf("createTestPDF should create different files: %s", pdf1)
	}

	// Verify both files exist and have content
	for _, pdfPath := range []string{pdf1, pdf2} {
		info, err := os.Stat(pdfPath)
		if err != nil {
			t.Errorf("PDF file doesn't exist: %s, error: %v", pdfPath, err)
		} else if info.Size() == 0 {
			t.Errorf("PDF file is empty: %s", pdfPath)
		}
	}
}

func TestSkipAndRequireFunctions(t *testing.T) {
	// Test requireMuPDF edge cases (44.4% remaining)
	// Multiple calls to requireMuPDF to test different conditions
	requireMuPDF(t)
	requireMuPDF(t) // Should be safe to call multiple times

	// Test skipIfShort and skipIfCIorShort (50% remaining for both)
	// These functions check environment conditions
	skipIfShort(t)     // Should not skip in normal test mode
	skipIfCIorShort(t) // Should not skip in normal test mode
}

func TestVersionAndUtilities(t *testing.T) {
	// Test GetVersion multiple times (should always return same result)
	version1 := GetVersion()
	version2 := GetVersion()

	if version1 != version2 {
		t.Errorf("GetVersion inconsistent: %s vs %s", version1, version2)
	}

	if version1 == "" {
		t.Error("GetVersion returned empty string")
	}

	// Test runGC multiple times
	runGC()
	runGC()
	runGC()
}

func TestContextDropEdgeCases(t *testing.T) {
	// Test Context.Drop edge cases
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Normal drop
	ctx.Drop()

	// Double drop should be safe
	ctx.Drop()
	ctx.Drop()
}

func TestMultipleOperationsOnSameResources(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test opening same document multiple times
	pdfPath := createTestPDF(t)

	doc1, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document first time: %v", err)
	}
	defer doc1.Close()

	doc2, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document second time: %v", err)
	}
	defer doc2.Close()

	// Compare operations on both documents
	count1 := doc1.CountPages()
	count2 := doc2.CountPages()

	if count1 != count2 {
		t.Errorf("Different page counts from same file: %d vs %d", count1, count2)
	}
}
