package mupdf

import (
	"os"
	"testing"
)

// Targeted tests for helper uniqueness and post-close/error-path behavior.

func TestForceHelperCoverage(t *testing.T) {
	// createTestPDF must return a usable path.
	pdf1 := createTestPDF(t)
	if pdf1 == "" {
		t.Error("createTestPDF returned empty path")
	}

	// Call multiple times to exercise cleanup and different temp directory creation
	pdf2 := createTestPDF(t)
	pdf3 := createTestPDF(t)

	// Verify all are different
	if pdf1 == pdf2 || pdf2 == pdf3 || pdf1 == pdf3 {
		t.Error("createTestPDF should create unique files")
	}
}

func TestTestDataDirCoverage(t *testing.T) {
	// Create multiple test data directories
	dirs := make([]string, 5)
	for i := 0; i < 5; i++ {
		dirs[i] = testDataDir(t)
		if dirs[i] == "" {
			t.Errorf("testDataDir returned empty directory for iteration %d", i)
		}
	}

	// Verify all directories are different and exist
	for i, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("Directory %d doesn't exist: %s, error: %v", i, dir, err)
		}

		// Check uniqueness
		for j, otherDir := range dirs {
			if i != j && dir == otherDir {
				t.Errorf("Directories %d and %d are the same: %s", i, j, dir)
			}
		}
	}
}

func TestBoundErrorPaths(t *testing.T) {
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

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}

		// Repeated Bound calls must agree.
		bounds1 := page.Bound()
		bounds2 := page.Bound()
		bounds3 := page.Bound()

		if bounds1 != bounds2 || bounds2 != bounds3 {
			t.Errorf("Inconsistent bounds: %+v, %+v, %+v", bounds1, bounds2, bounds3)
		}

		page.Close()

		// Post-close contract: Bound on a closed page returns the zero Rect.
		if bounds4 := page.Bound(); bounds4 != (Rect{}) {
			t.Errorf("Bound after Close = %+v, want zero Rect", bounds4)
		}
	}
}

func TestCountPagesErrorPaths(t *testing.T) {
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

	// Repeated CountPages calls must agree.
	count1 := doc.CountPages()
	count2 := doc.CountPages()
	count3 := doc.CountPages()

	if count1 != count2 || count2 != count3 {
		t.Errorf("Inconsistent page counts: %d, %d, %d", count1, count2, count3)
	}

	doc.Close()

	// Post-close contract: CountPages on a closed document returns 0.
	if count4 := doc.CountPages(); count4 != 0 {
		t.Errorf("CountPages after Close = %d, want 0", count4)
	}
}

func TestStringErrorPaths(t *testing.T) {
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

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text: %v", err)
		}

		// Repeated String calls must agree.
		str1 := text.String()
		str2 := text.String()
		str3 := text.String()

		if str1 != str2 || str2 != str3 {
			t.Errorf("Inconsistent text strings: %q, %q, %q", str1, str2, str3)
		}

		text.Close()

		// Post-close contract: String on a closed TextPage returns "".
		if got := text.String(); got != "" {
			t.Errorf("String after Close = %q, want empty string", got)
		}
	}
}

func TestExtractTextErrorPathsTargeted(t *testing.T) {
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

	// Repeated extraction on a valid page must succeed.
	for i := 0; i < 3; i++ {
		text, err := page.ExtractText()
		if err != nil {
			t.Errorf("ExtractText %d on valid page failed: %v", i, err)
			continue
		}
		text.Close()
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

func TestPDFObjectEdgeCases(t *testing.T) {
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

	// All of these values are supported types and must succeed.
	extremeValues := []interface{}{
		nil,
		true,
		false,
		int(-2147483648),                  // min int32
		int(2147483647),                   // max int32
		float64(-1.7976931348623157e+308), // near min float64
		float64(1.7976931348623157e+308),  // near max float64
		float64(4.9406564584124654e-324),  // min positive float64
		"",
		string(make([]byte, 1000)), // very long string
		"Special chars: \x00\x01\x02\xFF\n\r\t",
	}

	for i, val := range extremeValues {
		obj, err := writer.NewPDFObject(val)
		if err != nil {
			t.Errorf("Expected success for value %d (%T), got error: %v", i, val, err)
		} else {
			obj.Drop()
		}
	}
}
