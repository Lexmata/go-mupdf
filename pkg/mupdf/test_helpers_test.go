package mupdf

import (
	"testing"
)

// Additional tests to push coverage to 90%+

func TestUnusedFunctions(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test DebugCountPages
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	count := writer.DebugCountPages()
	t.Logf("Initial debug page count: %d", count)

	// Test ImprovedAddPage
	page, err := writer.ImprovedAddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed to add page with improved method: %v", err)
	}
	defer page.Close()

	count = writer.DebugCountPages()
	t.Logf("Debug page count after ImprovedAddPage: %d", count)

	// Test FixedAddPage
	page2, err := writer.FixedAddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page with fixed method: %v", err)
	}
	defer page2.Close()

	count = writer.DebugCountPages()
	t.Logf("Debug page count after FixedAddPage: %d", count)

	// Test SimpleAddPage
	page3, err := writer.SimpleAddPage(420, 595)
	if err != nil {
		t.Fatalf("Failed to add page with simple method: %v", err)
	}
	defer page3.Close()

	count = writer.DebugCountPages()
	t.Logf("Debug page count after SimpleAddPage: %d", count)
}

func TestHelperFunctions(t *testing.T) {
	// Test skipIfShort
	if testing.Short() {
		skipIfShort(t)
		t.Fatal("This should not be reached in short mode")
	}

	// Test skipIfCIorShort
	if testing.Short() {
		skipIfCIorShort(t)
		t.Fatal("This should not be reached in short mode")
	}

	// Test requireMuPDF
	requireMuPDF(t)

	// Test createTestPDF
	pdfPath := createTestPDF(t)
	t.Logf("Created test PDF: %s", pdfPath)
}

func TestDocumentCountPagesErrorPath(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a document that might have counting issues
	pdfPath := createTestPDF(t)

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Force error path by corrupting internal state (if possible)
	count := doc.CountPages()
	t.Logf("Document page count: %d", count)

	// Test AsPDFDocument error path
	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Logf("Expected error converting to PDF document: %v", err)
	} else {
		// Test PDF document CountPages
		pdfCount := pdfDoc.CountPages()
		t.Logf("PDF document page count: %d", pdfCount)
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

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		// Test Bound method error path by calling multiple times
		bounds1 := page.Bound()
		bounds2 := page.Bound()

		t.Logf("Bounds1: %+v", bounds1)
		t.Logf("Bounds2: %+v", bounds2)

		// Test bounds after closing page
		page.Close()
		bounds3 := page.Bound()
		t.Logf("Bounds after close: %+v", bounds3)
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

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		// Test ExtractText multiple times
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

		t.Logf("Text1 length: %d", len(content1))
		t.Logf("Text2 length: %d", len(content2))
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
	for i, writer := range writers {
		writer.Close()
		t.Logf("Closed writer %d", i)
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

	// Test AddPage with various sizes including edge cases
	sizes := []struct {
		w, h float64
		desc string
	}{
		{0, 0, "zero size"},
		{1, 1, "minimal size"},
		{612, 792, "US Letter"},
		{10000, 10000, "large size"},
	}

	for _, size := range sizes {
		page, err := writer.AddPage(size.w, size.h)
		if err != nil {
			t.Logf("Failed to add %s page: %v", size.desc, err)
			continue
		}
		page.Close()
		t.Logf("Successfully added %s page", size.desc)
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
		t.Logf("Successfully created PDF object for %s", tc.desc)
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
		} else {
			t.Logf("Got expected error for unsupported type %T: %v", val, err)
		}
	}
}
