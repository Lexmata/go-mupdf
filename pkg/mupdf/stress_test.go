package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// Smoke tests that exercise the full API surface with real assertions.

func TestUltimateCoverageBooster(t *testing.T) {
	// Several contexts alive simultaneously must all be creatable.
	contexts := make([]*Context, 5)
	for i := range contexts {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Context creation %d failed: %v", i, err)
		}
		contexts[i] = ctx
	}

	for _, ctx := range contexts {
		ctx.Drop()
	}
}

func TestExtensiveDocumentOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfs := make([]string, 3)
	for i := range pdfs {
		pdfs[i] = createTestPDF(t)
	}

	for _, pdfPath := range pdfs {
		for attempt := 0; attempt < 2; attempt++ {
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Fatalf("Failed to open document %s: %v", pdfPath, err)
			}

			count := doc.CountPages()
			if count == 0 {
				t.Fatalf("Expected at least one page in %s", pdfPath)
			}

			pdfDoc, err := doc.AsPDFDocument()
			if err != nil {
				t.Fatalf("AsPDFDocument failed: %v", err)
			}
			if pdfCount := pdfDoc.CountPages(); pdfCount != count {
				t.Errorf("AsPDFDocument page count = %d, want %d", pdfCount, count)
			}

			pdfPage, err := pdfDoc.LoadPage(0)
			if err != nil {
				t.Fatalf("PDF LoadPage failed: %v", err)
			}
			pdfBounds1 := pdfPage.Bound()
			pdfBounds2 := pdfPage.Bound()
			if pdfBounds1 != pdfBounds2 {
				t.Errorf("Inconsistent PDF page bounds: %+v vs %+v", pdfBounds1, pdfBounds2)
			}
			pdfPage.Close()

			pdfDoc2, err := OpenPDFDocument(ctx, pdfPath)
			if err != nil {
				t.Fatalf("OpenPDFDocument failed: %v", err)
			}
			if count2 := pdfDoc2.CountPages(); count2 != count {
				t.Errorf("OpenPDFDocument page count = %d, want %d", count2, count)
			}

			page, err := doc.LoadPage(0)
			if err != nil {
				t.Fatalf("LoadPage failed: %v", err)
			}

			bounds1 := page.Bound()
			bounds2 := page.Bound()
			if bounds1 != bounds2 {
				t.Errorf("Inconsistent bounds: %+v vs %+v", bounds1, bounds2)
			}

			text1, err := page.ExtractText()
			if err != nil {
				t.Fatalf("ExtractText failed: %v", err)
			}
			content1 := text1.String()
			content2 := text1.String()
			if content1 != content2 {
				t.Errorf("Inconsistent String results: %d vs %d chars", len(content1), len(content2))
			}
			text1.Close()

			text2, err := page.ExtractText()
			if err != nil {
				t.Fatalf("Second ExtractText failed: %v", err)
			}
			if got := text2.String(); got != content1 {
				t.Errorf("Second extraction differs: %d vs %d chars", len(got), len(content1))
			}
			text2.Close()

			page.Close()
			doc.Close()
		}
	}
}

func TestExtensivePDFWriterOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	const pagesPerMethod = 2

	for writerNum := 0; writerNum < 2; writerNum++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Writer %d creation failed: %v", writerNum, err)
		}

		// Every page-addition variant must insert its pages.
		methods := []struct {
			name string
			add  func(float64, float64) (*PDFPage, error)
		}{
			{"AddPage", writer.AddPage},
			{"SimpleAddPage", writer.SimpleAddPage},
			{"ImprovedAddPage", writer.ImprovedAddPage},
			{"FixedAddPage", writer.FixedAddPage},
		}

		added := 0
		for _, method := range methods {
			for pageNum := 0; pageNum < pagesPerMethod; pageNum++ {
				w := float64(100 + pageNum*50)
				h := float64(200 + pageNum*30)

				page, err := method.add(w, h)
				if err != nil {
					t.Fatalf("Writer %d %s page %d failed: %v", writerNum, method.name, pageNum, err)
				}
				page.Close()
				added++
			}
		}

		if got := writer.DebugCountPages(); got != added {
			t.Errorf("Writer %d page count = %d, want %d", writerNum, got, added)
		}

		// All supported value types must produce PDF objects.
		values := []interface{}{
			nil, true, false, 0, 1, -1, 42, -42,
			0.0, 1.0, -1.0, 3.14, -3.14,
			"", "test", "long string with many characters",
		}

		for _, value := range values {
			obj, err := writer.NewPDFObject(value)
			if err != nil {
				t.Errorf("Object creation failed for %v (%T): %v", value, value, err)
				continue
			}
			obj.Drop()
		}

		writer.Close()
	}
}

func TestHelperFunctionsExtensive(t *testing.T) {
	// Test createTestPDF repeatedly.
	for i := 0; i < 3; i++ {
		pdf := createTestPDF(t)
		if pdf == "" {
			t.Errorf("createTestPDF %d failed", i)
		}
	}

	// Test testDataDir repeatedly.
	for i := 0; i < 3; i++ {
		dir := testDataDir(t)
		if dir == "" {
			t.Errorf("testDataDir %d failed", i)
		}
	}

	// Test requireMuPDF repeatedly.
	for i := 0; i < 3; i++ {
		requireMuPDF(t)
	}

	// Test skip functions.
	for i := 0; i < 2; i++ {
		skipIfShort(t)
		skipIfCIorShort(t)
	}
}

func TestErrorConditionsExtensive(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// A real file whose contents are not a PDF.
	notPDF := filepath.Join(t.TempDir(), "not-a-pdf.pdf")
	if err := os.WriteFile(notPDF, []byte("this is not a valid PDF file"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	invalidFiles := []string{
		"",
		"nonexistent.pdf",
		"/invalid/path.pdf",
		notPDF, // exists but is not a PDF
	}

	for _, file := range invalidFiles {
		if _, err := OpenDocument(ctx, file); err == nil {
			t.Errorf("OpenDocument(%q): expected error, got nil", file)
		}

		if _, err := OpenPDFDocument(ctx, file); err == nil {
			t.Errorf("OpenPDFDocument(%q): expected error, got nil", file)
		}
	}
}

func TestAllCombinationsMaximum(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)

	for iteration := 0; iteration < 2; iteration++ {
		// Document operations.
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Iteration %d: OpenDocument failed: %v", iteration, err)
		}

		count1 := doc.CountPages()
		count2 := doc.CountPages()
		if count1 != count2 {
			t.Errorf("Iteration %d: inconsistent counts: %d vs %d", iteration, count1, count2)
		}
		if count1 == 0 {
			t.Fatalf("Iteration %d: expected at least one page", iteration)
		}

		pdfDoc, err := doc.AsPDFDocument()
		if err != nil {
			t.Fatalf("Iteration %d: AsPDFDocument failed: %v", iteration, err)
		}
		if pdfCount := pdfDoc.CountPages(); pdfCount != count1 {
			t.Errorf("Iteration %d: PDF count = %d, want %d", iteration, pdfCount, count1)
		}

		pdfPage, err := pdfDoc.LoadPage(0)
		if err != nil {
			t.Fatalf("Iteration %d: PDF LoadPage failed: %v", iteration, err)
		}
		if b1, b2 := pdfPage.Bound(), pdfPage.Bound(); b1 != b2 {
			t.Errorf("Iteration %d: inconsistent PDF bounds: %+v vs %+v", iteration, b1, b2)
		}
		pdfPage.Close()

		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Iteration %d: LoadPage failed: %v", iteration, err)
		}
		if b1, b2 := page.Bound(), page.Bound(); b1 != b2 {
			t.Errorf("Iteration %d: inconsistent bounds: %+v vs %+v", iteration, b1, b2)
		}

		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Iteration %d: ExtractText failed: %v", iteration, err)
		}
		if s1, s2 := text.String(), text.String(); s1 != s2 {
			t.Errorf("Iteration %d: inconsistent text: %d vs %d chars", iteration, len(s1), len(s2))
		}
		text.Close()
		page.Close()
		doc.Close()

		// Writer operations.
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: NewPDFWriter failed: %v", iteration, err)
		}

		sizes := [][]float64{
			{100, 200}, {612, 792}, {595, 842},
		}

		for _, size := range sizes {
			for _, add := range []func(float64, float64) (*PDFPage, error){
				writer.AddPage, writer.SimpleAddPage, writer.ImprovedAddPage, writer.FixedAddPage,
			} {
				page, err := add(size[0], size[1])
				if err != nil {
					t.Fatalf("Iteration %d: adding %vx%v page failed: %v", iteration, size[0], size[1], err)
				}
				page.Close()
			}
		}

		objects := []interface{}{nil, true, false, iteration, float64(iteration), "test"}
		for _, obj := range objects {
			pdfObj, err := writer.NewPDFObject(obj)
			if err != nil {
				t.Errorf("Iteration %d: NewPDFObject(%v) failed: %v", iteration, obj, err)
				continue
			}
			pdfObj.Drop()
		}

		writer.Close()
	}
}
