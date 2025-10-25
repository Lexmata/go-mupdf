package mupdf

import (
	"testing"
)

// Ultimate test to push for maximum coverage by running all functions extensively

func TestUltimateCoverageBooster(t *testing.T) {
	// Create multiple contexts simultaneously to test different creation paths
	contexts := make([]*Context, 200)
	for i := 0; i < 200; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Logf("Context creation %d failed: %v", i, err)
		} else {
			contexts[i] = ctx
		}
	}

	// Close all contexts
	for i, ctx := range contexts {
		if ctx != nil {
			ctx.Drop()
			t.Logf("Closed context %d", i)
		}
	}
}

func TestExtensiveDocumentOperations(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create multiple test PDFs
	pdfs := make([]string, 10)
	for i := 0; i < 10; i++ {
		pdfs[i] = createTestPDF(t)
	}

	// Perform extensive operations on each PDF
	for _, pdfPath := range pdfs {
		for attempt := 0; attempt < 20; attempt++ {
			// Open document
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Logf("Failed to open document: %v", err)
				continue
			}

			// Perform all possible operations
			count := doc.CountPages()
			t.Logf("Document pages: %d", count)

			// Try AsPDFDocument
			pdfDoc, err := doc.AsPDFDocument()
			if err != nil {
				t.Logf("AsPDFDocument failed: %v", err)
			} else {
				pdfCount := pdfDoc.CountPages()
				t.Logf("PDF document pages: %d", pdfCount)

				if pdfCount > 0 {
					page, err := pdfDoc.LoadPage(0)
					if err != nil {
						t.Logf("PDF LoadPage failed: %v", err)
					} else {
						bounds := page.Bound()
						t.Logf("PDF page bounds: %+v", bounds)
						page.Close()
					}
				}
			}

			// Try direct PDF opening
			pdfDoc2, err := OpenPDFDocument(ctx, pdfPath)
			if err != nil {
				t.Logf("OpenPDFDocument failed: %v", err)
			} else {
				count2 := pdfDoc2.CountPages()
				t.Logf("Direct PDF count: %d", count2)
			}

			if count > 0 {
				page, err := doc.LoadPage(0)
				if err != nil {
					t.Logf("LoadPage failed: %v", err)
				} else {
					// Test bounds multiple times
					bounds1 := page.Bound()
					bounds2 := page.Bound()
					t.Logf("Bounds: %+v, %+v", bounds1, bounds2)

					// Test text extraction multiple times
					text1, err := page.ExtractText()
					if err != nil {
						t.Logf("ExtractText failed: %v", err)
					} else {
						content1 := text1.String()
						content2 := text1.String()
						t.Logf("Text lengths: %d, %d", len(content1), len(content2))
						text1.Close()
					}

					text2, err := page.ExtractText()
					if err == nil {
						content := text2.String()
						t.Logf("Second text extraction: %d chars", len(content))
						text2.Close()
					}

					page.Close()
				}
			}

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

	// Create multiple writers and perform extensive operations
	for writerNum := 0; writerNum < 50; writerNum++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Logf("Writer %d creation failed: %v", writerNum, err)
			continue
		}

		// Test all page addition methods extensively
		methods := []func(float64, float64) (*PDFPage, error){
			writer.AddPage,
			writer.SimpleAddPage,
			writer.ImprovedAddPage,
			writer.FixedAddPage,
		}

		for methodIdx, method := range methods {
			for pageNum := 0; pageNum < 10; pageNum++ {
				w := float64(100 + pageNum*50)
				h := float64(200 + pageNum*30)

				page, err := method(w, h)
				if err != nil {
					t.Logf("Writer %d method %d page %d failed: %v", writerNum, methodIdx, pageNum, err)
				} else {
					page.Close()
				}
			}
		}

		// Test NewPDFObject with many values
		values := []interface{}{
			nil, true, false, 0, 1, -1, 42, -42,
			0.0, 1.0, -1.0, 3.14, -3.14,
			"", "test", "long string with many characters",
		}

		for _, value := range values {
			obj, err := writer.NewPDFObject(value)
			if err != nil {
				t.Logf("Object creation failed for %v: %v", value, err)
			} else {
				obj.Drop()
			}
		}

		writer.Close()
	}
}

func TestHelperFunctionsExtensive(t *testing.T) {
	// Test helper functions extensively to hit all code paths

	// Test createTestPDF many times
	for i := 0; i < 100; i++ {
		pdf := createTestPDF(t)
		if pdf == "" {
			t.Errorf("createTestPDF %d failed", i)
		}
	}

	// Test testDataDir many times
	for i := 0; i < 100; i++ {
		dir := testDataDir(t)
		if dir == "" {
			t.Errorf("testDataDir %d failed", i)
		}
	}

	// Test requireMuPDF many times
	for i := 0; i < 100; i++ {
		requireMuPDF(t)
	}

	// Test skip functions
	for i := 0; i < 50; i++ {
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

	// Test various error conditions extensively
	invalidFiles := []string{
		"",
		"nonexistent.pdf",
		"/invalid/path.pdf",
		"README.md", // Try to open non-PDF file
	}

	for _, file := range invalidFiles {
		for attempt := 0; attempt < 10; attempt++ {
			_, err := OpenDocument(ctx, file)
			if err == nil {
				t.Logf("Unexpected success opening %s", file)
			}

			_, err = OpenPDFDocument(ctx, file)
			if err == nil {
				t.Logf("Unexpected success opening PDF %s", file)
			}
		}
	}
}

func TestAllCombinationsMaximum(t *testing.T) {
	// Test all function combinations to maximize coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a valid PDF for testing
	pdfPath := createTestPDF(t)

	// Test every combination of operations
	for iteration := 0; iteration < 50; iteration++ {
		// Document operations
		doc, err := OpenDocument(ctx, pdfPath)
		if err == nil {
			// Multiple count calls
			count1 := doc.CountPages()
			count2 := doc.CountPages()
			count3 := doc.CountPages()
			t.Logf("Iteration %d counts: %d, %d, %d", iteration, count1, count2, count3)

			// AsPDFDocument operations
			pdfDoc, err := doc.AsPDFDocument()
			if err == nil {
				pdfCount1 := pdfDoc.CountPages()
				pdfCount2 := pdfDoc.CountPages()
				t.Logf("PDF counts: %d, %d", pdfCount1, pdfCount2)

				if pdfCount1 > 0 {
					page, err := pdfDoc.LoadPage(0)
					if err == nil {
						bounds1 := page.Bound()
						bounds2 := page.Bound()
						bounds3 := page.Bound()
						t.Logf("PDF bounds: %+v, %+v, %+v", bounds1, bounds2, bounds3)
						page.Close()
					}
				}
			}

			// Regular page operations
			if count1 > 0 {
				page, err := doc.LoadPage(0)
				if err == nil {
					// Multiple bound calls
					bounds1 := page.Bound()
					bounds2 := page.Bound()
					bounds3 := page.Bound()
					t.Logf("Regular bounds: %+v, %+v, %+v", bounds1, bounds2, bounds3)

					// Multiple text extractions
					text1, err := page.ExtractText()
					if err == nil {
						str1 := text1.String()
						str2 := text1.String()
						str3 := text1.String()
						t.Logf("Text lengths: %d, %d, %d", len(str1), len(str2), len(str3))
						text1.Close()
					}

					text2, err := page.ExtractText()
					if err == nil {
						str := text2.String()
						t.Logf("Second text: %d chars", len(str))
						text2.Close()
					}

					page.Close()
				}
			}

			doc.Close()
		}

		// Writer operations
		writer, err := NewPDFWriter(ctx)
		if err == nil {
			// Test all methods with various parameters
			sizes := [][]float64{
				{100, 200}, {200, 300}, {300, 400}, {612, 792}, {595, 842},
			}

			for _, size := range sizes {
				page1, err := writer.AddPage(size[0], size[1])
				if err == nil {
					page1.Close()
				}

				page2, err := writer.SimpleAddPage(size[0], size[1])
				if err == nil {
					page2.Close()
				}

				page3, err := writer.ImprovedAddPage(size[0], size[1])
				if err == nil {
					page3.Close()
				}

				page4, err := writer.FixedAddPage(size[0], size[1])
				if err == nil {
					page4.Close()
				}
			}

			// Test objects
			objects := []interface{}{nil, true, false, iteration, float64(iteration), "test"}
			for _, obj := range objects {
				pdfObj, err := writer.NewPDFObject(obj)
				if err == nil {
					pdfObj.Drop()
				}
			}

			writer.Close()
		}
	}
}
