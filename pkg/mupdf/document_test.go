package mupdf

import (
	"path/filepath"
	"testing"
)

// Tests exercising the core document and writer APIs end to end.

func TestExtractTextComplete(t *testing.T) {
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

	// Repeated extraction must succeed and return identical content.
	var first string
	for i := 0; i < 3; i++ {
		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text %d: %v", i, err)
		}
		content := text.String()
		text.Close()

		if i == 0 {
			first = content
		} else if content != first {
			t.Errorf("Extraction %d returned different content: %d chars vs %d chars", i, len(content), len(first))
		}
	}
}

func TestAsPDFDocumentComplete(t *testing.T) {
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

	want := doc.CountPages()

	// Repeated conversion must succeed and report the same page count.
	for i := 0; i < 3; i++ {
		pdfDoc, err := doc.AsPDFDocument()
		if err != nil {
			t.Fatalf("AsPDFDocument %d failed: %v", i, err)
		}
		if got := pdfDoc.CountPages(); got != want {
			t.Errorf("AsPDFDocument %d: page count = %d, want %d", i, got, want)
		}
	}
}

func TestOpenPDFDocumentComplete(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)

	for i := 0; i < 3; i++ {
		pdfDoc, err := OpenPDFDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("OpenPDFDocument %d failed: %v", i, err)
		}
		if got := pdfDoc.CountPages(); got == 0 {
			t.Errorf("OpenPDFDocument %d: expected at least one page, got 0", i)
		}
	}

	if _, err := OpenPDFDocument(ctx, "non-existent-file.pdf"); err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestAddPageComplete(t *testing.T) {
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

	// AddPage must reject non-positive dimensions and accept positive ones.
	sizes := []struct {
		name    string
		w, h    float64
		wantErr bool
	}{
		{"zero size", 0, 0, true},
		{"negative width", -100, 200, true},
		{"negative height", 200, -100, true},
		{"minimal size", 1, 1, false},
		{"US Letter", 612, 792, false},
		{"A4", 595, 842, false},
		{"tall and narrow", 100, 1000, false},
		{"wide and short", 1000, 100, false},
		{"large square", 5000, 5000, false},
	}

	added := 0
	for _, tc := range sizes {
		page, err := writer.AddPage(tc.w, tc.h)
		if tc.wantErr {
			if err == nil {
				page.Close()
				t.Errorf("AddPage(%s %.0fx%.0f): expected error for invalid dimensions, got nil", tc.name, tc.w, tc.h)
			}
			continue
		}
		if err != nil {
			t.Fatalf("AddPage(%s %.0fx%.0f): %v", tc.name, tc.w, tc.h, err)
		}
		page.Close()
		added++
		if got := writer.DebugCountPages(); got != added {
			t.Errorf("page count after %d successful adds = %d, want %d", added, got, added)
		}
	}

	// All added pages must survive a save/reopen round trip.
	outPath := filepath.Join(t.TempDir(), "addpage.pdf")
	if err := writer.Save(outPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc, err := OpenDocument(ctx, outPath)
	if err != nil {
		t.Fatalf("Failed to reopen saved PDF: %v", err)
	}
	defer doc.Close()
	if got := doc.CountPages(); got != added {
		t.Errorf("save/reopen page count = %d, want %d", got, added)
	}
}

// testAddPageVariantSavesPages verifies that an AddPage variant actually
// inserts pages: the in-memory count grows per add and the pages survive a
// save/reopen round trip.
func testAddPageVariantSavesPages(t *testing.T, addPage func(w *PDFWriter, width, height float64) (*PDFPage, error)) {
	t.Helper()

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

	sizes := []struct{ w, h float64 }{
		{200, 300},
		{612, 792},
		{595, 842},
	}

	for i, size := range sizes {
		page, err := addPage(writer, size.w, size.h)
		if err != nil {
			t.Fatalf("add page %d (%.0fx%.0f): %v", i, size.w, size.h, err)
		}
		page.Close()
		if got := writer.DebugCountPages(); got != i+1 {
			t.Errorf("page count after %d adds = %d, want %d", i+1, got, i+1)
		}
	}

	outPath := filepath.Join(t.TempDir(), "out.pdf")
	if err := writer.Save(outPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	doc, err := OpenDocument(ctx, outPath)
	if err != nil {
		t.Fatalf("Failed to reopen saved PDF: %v", err)
	}
	defer doc.Close()
	if got := doc.CountPages(); got != len(sizes) {
		t.Errorf("save/reopen page count = %d, want %d", got, len(sizes))
	}
}

func TestSimpleAddPageComplete(t *testing.T) {
	testAddPageVariantSavesPages(t, (*PDFWriter).SimpleAddPage)
}

func TestImprovedAddPageComplete(t *testing.T) {
	testAddPageVariantSavesPages(t, (*PDFWriter).ImprovedAddPage)
}

func TestFixedAddPageComplete(t *testing.T) {
	testAddPageVariantSavesPages(t, (*PDFWriter).FixedAddPage)
}

func TestNewContextComplete(t *testing.T) {
	// Several contexts alive at once.
	contexts := make([]*Context, 5)
	for i := range contexts {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Failed to create context %d: %v", i, err)
		}
		contexts[i] = ctx
	}
	for _, ctx := range contexts {
		ctx.Drop()
	}

	// Rapid creation and destruction.
	for i := 0; i < 5; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Failed to create rapid context %d: %v", i, err)
		}
		ctx.Drop()
	}
}

func TestNewPDFWriterComplete(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Several writers alive at once.
	writers := make([]*PDFWriter, 5)
	for i := range writers {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create writer %d: %v", i, err)
		}
		writers[i] = writer
	}
	for _, writer := range writers {
		writer.Close()
	}

	// Rapid creation and destruction.
	for i := 0; i < 5; i++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create rapid writer %d: %v", i, err)
		}
		writer.Close()
	}
}
