package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for helper functions and post-close behavior of core types.

func TestRequireMuPDFComplete(t *testing.T) {
	// requireMuPDF must leave the library usable when MuPDF is available.
	requireMuPDF(t)

	if GetVersion() == "" {
		t.Error("GetVersion returned empty string after requireMuPDF")
	}
}

func TestCreateTestPDFComplete(t *testing.T) {
	// Test normal case
	pdf1 := createTestPDF(t)
	if pdf1 == "" {
		t.Fatal("createTestPDF returned empty path")
	}

	// Test creating multiple PDFs rapidly to test cleanup paths
	pdfs := make([]string, 10)
	for i := 0; i < 10; i++ {
		pdfs[i] = createTestPDF(t)
		if pdfs[i] == "" {
			t.Fatalf("createTestPDF %d returned empty path", i)
		}
	}

	// Verify all files exist and are different
	for i, pdf := range pdfs {
		if _, err := os.Stat(pdf); err != nil {
			t.Errorf("PDF %d doesn't exist: %s", i, pdf)
		}
		for j, other := range pdfs {
			if i != j && pdf == other {
				t.Errorf("PDFs %d and %d have same path: %s", i, j, pdf)
			}
		}
	}
}

func TestTestDataDirComplete(t *testing.T) {
	// Test normal case
	dir1 := testDataDir(t)
	if dir1 == "" {
		t.Fatal("testDataDir returned empty directory")
	}

	// Create multiple directories
	dirs := make([]string, 3)
	for i := range dirs {
		dirs[i] = testDataDir(t)
		if dirs[i] == "" {
			t.Fatalf("testDataDir %d returned empty directory", i)
		}
	}

	for i, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("Directory %d doesn't exist: %s, error: %v", i, dir, err)
		}

		// Test that we can write to the directory
		testFile := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Errorf("Cannot write to directory %d: %s, error: %v", i, dir, err)
		}
	}
}

func TestCountPagesComplete(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test with valid document
	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Repeated CountPages calls must agree.
	counts := make([]int, 10)
	for i := 0; i < 10; i++ {
		counts[i] = doc.CountPages()
	}

	for i := 1; i < len(counts); i++ {
		if counts[i] != counts[0] {
			t.Errorf("Inconsistent page count at index %d: got %d, expected %d", i, counts[i], counts[0])
		}
	}

	// Test with PDF document
	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("Failed to convert to PDF document: %v", err)
	}

	pdfCounts := make([]int, 10)
	for i := 0; i < 10; i++ {
		pdfCounts[i] = pdfDoc.CountPages()
	}

	for i := 1; i < len(pdfCounts); i++ {
		if pdfCounts[i] != pdfCounts[0] {
			t.Errorf("Inconsistent PDF page count at index %d: got %d, expected %d", i, pdfCounts[i], pdfCounts[0])
		}
	}

	doc.Close()

	// Post-close contract: CountPages on a closed document returns 0.
	if got := doc.CountPages(); got != 0 {
		t.Errorf("CountPages after Close = %d, want 0", got)
	}
}

func TestBoundComplete(t *testing.T) {
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
		// Test regular page bounds
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}

		// Repeated Bound calls must agree.
		bounds := make([]Rect, 10)
		for i := 0; i < 10; i++ {
			bounds[i] = page.Bound()
		}

		for i := 1; i < len(bounds); i++ {
			if bounds[i] != bounds[0] {
				t.Errorf("Inconsistent bounds at index %d: got %+v, expected %+v", i, bounds[i], bounds[0])
			}
		}

		page.Close()

		// Post-close contract: Bound on a closed page returns the zero Rect.
		if got := page.Bound(); got != (Rect{}) {
			t.Errorf("Bound after Close = %+v, want zero Rect", got)
		}

		// Test PDF page bounds
		pdfDoc, err := doc.AsPDFDocument()
		if err != nil {
			t.Fatalf("Failed to convert to PDF document: %v", err)
		}

		if pdfDoc.CountPages() > 0 {
			pdfPage, err := pdfDoc.LoadPage(0)
			if err != nil {
				t.Fatalf("Failed to load PDF page: %v", err)
			}

			pdfBounds := make([]Rect, 10)
			for i := 0; i < 10; i++ {
				pdfBounds[i] = pdfPage.Bound()
			}

			for i := 1; i < len(pdfBounds); i++ {
				if pdfBounds[i] != pdfBounds[0] {
					t.Errorf("Inconsistent PDF bounds at index %d: got %+v, expected %+v", i, pdfBounds[i], pdfBounds[0])
				}
			}

			pdfPage.Close()

			// Post-close contract: Bound on a closed PDF page returns the zero Rect.
			if got := pdfPage.Bound(); got != (Rect{}) {
				t.Errorf("PDF Bound after Close = %+v, want zero Rect", got)
			}
		}
	}
}

func TestStringComplete(t *testing.T) {
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
		strings := make([]string, 10)
		for i := 0; i < 10; i++ {
			strings[i] = text.String()
		}

		for i := 1; i < len(strings); i++ {
			if strings[i] != strings[0] {
				t.Errorf("Inconsistent text at index %d: got %q, expected %q", i, strings[i], strings[0])
			}
		}

		text.Close()

		// Post-close contract: String on a closed TextPage returns "".
		if got := text.String(); got != "" {
			t.Errorf("String after Close = %q, want empty string", got)
		}
	}
}
