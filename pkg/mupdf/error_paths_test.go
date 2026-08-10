package mupdf

import (
	"strings"
	"testing"
)

// TestUnusedFunctions verifies the alternative AddPage implementations
// (ImprovedAddPage, FixedAddPage, SimpleAddPage) actually insert pages, as
// observed via DebugCountPages.
func TestUnusedFunctions(t *testing.T) {
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

	if count := writer.DebugCountPages(); count != 0 {
		t.Fatalf("Expected 0 pages in fresh writer, got %d", count)
	}

	// Each AddPage variant must insert exactly one page.
	page, err := writer.ImprovedAddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed to add page with improved method: %v", err)
	}
	defer page.Close()

	if count := writer.DebugCountPages(); count != 1 {
		t.Errorf("Expected 1 page after ImprovedAddPage, got %d", count)
	}

	page2, err := writer.FixedAddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page with fixed method: %v", err)
	}
	defer page2.Close()

	if count := writer.DebugCountPages(); count != 2 {
		t.Errorf("Expected 2 pages after FixedAddPage, got %d", count)
	}

	page3, err := writer.SimpleAddPage(420, 595)
	if err != nil {
		t.Fatalf("Failed to add page with simple method: %v", err)
	}
	defer page3.Close()

	if count := writer.DebugCountPages(); count != 3 {
		t.Errorf("Expected 3 pages after SimpleAddPage, got %d", count)
	}
}

func TestHelperFunctions(t *testing.T) {
	// skipIfShort must skip exactly when -short is set.
	var reachedShort bool
	t.Run("skipIfShort", func(t *testing.T) {
		skipIfShort(t)
		reachedShort = true
	})
	if reachedShort == testing.Short() {
		t.Errorf("skipIfShort: reached body = %v, want %v", reachedShort, !testing.Short())
	}

	// skipIfCIorShort must skip when -short is set (CI detection depends on
	// the environment, so only the short-mode contract is asserted here).
	var reachedCI bool
	t.Run("skipIfCIorShort", func(t *testing.T) {
		skipIfCIorShort(t)
		reachedCI = true
	})
	if testing.Short() && reachedCI {
		t.Error("skipIfCIorShort did not skip in short mode")
	}

	// requireMuPDF must not fail the test when MuPDF is available (it skips
	// when unavailable, ending this test here).
	requireMuPDF(t)

	// createTestPDF must produce a readable 1-page PDF.
	pdfPath := createTestPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open PDF from createTestPDF: %v", err)
	}
	defer doc.Close()

	if count := doc.CountPages(); count != 1 {
		t.Errorf("Expected createTestPDF to produce 1 page, got %d", count)
	}
}

func TestDocumentCountPagesErrorPath(t *testing.T) {
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

	if count := doc.CountPages(); count != 1 {
		t.Errorf("Expected 1 page, got %d", count)
	}

	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("Failed to convert valid PDF to PDF document: %v", err)
	}
	if pdfCount := pdfDoc.CountPages(); pdfCount != 1 {
		t.Errorf("Expected 1 page from PDF document, got %d", pdfCount)
	}

	// Error path: CountPages on a closed document returns the documented
	// safe value of 0 instead of crashing.
	doc.Close()
	if count := doc.CountPages(); count != 0 {
		t.Errorf("Expected CountPages on closed document to return 0, got %d", count)
	}
}

func TestPageBoundErrorPath(t *testing.T) {
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

	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	// Bound must be stable across repeated calls.
	bounds1 := page.Bound()
	bounds2 := page.Bound()
	if bounds1 != bounds2 {
		t.Errorf("Bound not stable across calls: first %+v, second %+v", bounds1, bounds2)
	}
	if bounds1.X0 >= bounds1.X1 || bounds1.Y0 >= bounds1.Y1 {
		t.Errorf("Invalid page bounds: %+v", bounds1)
	}

	// Error path: Bound on a closed page returns the documented safe zero
	// Rect instead of crashing.
	page.Close()
	if bounds3 := page.Bound(); bounds3 != (Rect{}) {
		t.Errorf("Expected zero Rect from Bound on closed page, got %+v", bounds3)
	}
}

func TestExtractTextErrorPath(t *testing.T) {
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

	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	// Extraction must be repeatable and return the fixture's text each time.
	text1, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text first time: %v", err)
	}
	content1 := text1.String()
	text1.Close()

	text2, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text second time: %v", err)
	}
	content2 := text2.String()
	text2.Close()

	if !strings.Contains(content1, "Hello World") {
		t.Errorf("Expected extracted text to contain %q, got %q", "Hello World", content1)
	}
	if content1 != content2 {
		t.Errorf("Extraction not repeatable: first %q, second %q", content1, content2)
	}

	// Error path: ExtractText on a closed page returns an error instead of
	// crashing.
	page.Close()
	if _, err := page.ExtractText(); err == nil {
		t.Error("Expected error from ExtractText on closed page")
	}
}

func TestNewContextErrorPath(t *testing.T) {
	// Test multiple context creation and destruction
	contexts := make([]*Context, 10)

	for i := 0; i < 10; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Failed to create context %d: %v", i, err)
		}
		contexts[i] = ctx
	}

	// Close them in reverse order
	for i := 9; i >= 0; i-- {
		contexts[i].Drop()
	}
}

func TestNewPDFWriterErrorPath(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test multiple writer creation
	writers := make([]*PDFWriter, 5)

	for i := 0; i < 5; i++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create writer %d: %v", i, err)
		}
		writers[i] = writer
	}

	// Close them all
	for _, writer := range writers {
		writer.Close()
	}
}

func TestAddPageErrorPath(t *testing.T) {
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

	// AddPage must reject non-positive dimensions and accept valid ones.
	sizes := []struct {
		w, h    float64
		desc    string
		wantErr bool
	}{
		{0, 0, "zero size", true},
		{-1, 842, "negative width", true},
		{1, 1, "minimal size", false},
		{612, 792, "US Letter", false},
		{10000, 10000, "large size", false},
	}

	for _, size := range sizes {
		page, err := writer.AddPage(size.w, size.h)
		if size.wantErr {
			if err == nil {
				t.Errorf("AddPage(%v, %v) [%s]: expected error, got nil", size.w, size.h, size.desc)
				if page != nil {
					page.Close()
				}
			}
			continue
		}
		if err != nil {
			t.Errorf("AddPage(%v, %v) [%s]: unexpected error: %v", size.w, size.h, size.desc, err)
			continue
		}
		page.Close()
	}
}

func TestPDFObjectErrorPaths(t *testing.T) {
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

	// Test all supported types and edge cases
	testCases := []struct {
		value interface{}
		desc  string
	}{
		{nil, "nil value"},
		{true, "boolean true"},
		{false, "boolean false"},
		{0, "zero integer"},
		{-1, "negative integer"},
		{999999, "large integer"},
		{0.0, "zero float"},
		{-3.14, "negative float"},
		{999.999, "large float"},
		{"", "empty string"},
		{"test", "simple string"},
		{"a very long string with special characters: !@#$%^&*()", "complex string"},
	}

	for _, tc := range testCases {
		obj, err := writer.NewPDFObject(tc.value)
		if err != nil {
			t.Errorf("Failed to create PDF object for %s: %v", tc.desc, err)
			continue
		}
		obj.Drop()
	}

	// Test unsupported types
	unsupported := []interface{}{
		[]int{1, 2, 3},
		map[string]int{"a": 1},
		struct{}{},
		func() {},
	}

	for _, val := range unsupported {
		_, err := writer.NewPDFObject(val)
		if err == nil {
			t.Errorf("Expected error for unsupported type %T", val)
		}
	}
}
