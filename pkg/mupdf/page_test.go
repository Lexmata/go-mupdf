package mupdf

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// TestComprehensive is a comprehensive test that exercises multiple aspects of the library
func TestComprehensive(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a temporary directory for test files
	dir := testDataDir(t)

	// Phase 1: Create multiple PDFs with different characteristics
	t.Log("Phase 1: Creating multiple PDFs")
	pdfPaths := createMultiplePDFs(t, ctx, dir)

	// Phase 2: Open and process each PDF
	t.Log("Phase 2: Processing PDFs")
	processPDFs(t, ctx, pdfPaths)

	// Phase 3: Concurrent processing (each worker creates its own context)
	t.Log("Phase 3: Concurrent processing")
	concurrentProcessing(t, pdfPaths)

	// Phase 4: Error handling and recovery
	t.Log("Phase 4: Error handling and recovery")
	errorHandlingAndRecovery(t, ctx, dir)

	// Phase 5: Resource cleanup
	t.Log("Phase 5: Resource cleanup")
	resourceCleanup(t, ctx, pdfPaths[0])

	t.Log("Comprehensive test completed successfully")
}

// Helper function to create multiple PDFs with different characteristics
func createMultiplePDFs(t *testing.T, ctx *Context, dir string) []string {
	pdfPaths := make([]string, 3)

	// Create PDF 1: Single page
	{
		pdfPath := filepath.Join(dir, "comprehensive_1.pdf")
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer 1: %v", err)
		}

		// Add a page
		_, err = writer.AddPage(595, 842) // A4 size
		if err != nil {
			t.Fatalf("Failed to add page to PDF 1: %v", err)
		}

		// Save the PDF
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save PDF 1: %v", err)
		}
		writer.Close()

		pdfPaths[0] = pdfPath
	}

	// Create PDF 2: Multiple pages with different sizes
	{
		pdfPath := filepath.Join(dir, "comprehensive_2.pdf")
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer 2: %v", err)
		}

		// Add pages with different sizes
		pageSizes := []struct {
			width  float64
			height float64
		}{
			{595, 842},  // A4
			{612, 792},  // Letter
			{792, 1224}, // Ledger
		}

		for i, size := range pageSizes {
			_, err = writer.AddPage(size.width, size.height)
			if err != nil {
				t.Fatalf("Failed to add page %d to PDF 2: %v", i, err)
			}
		}

		// Save the PDF
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save PDF 2: %v", err)
		}
		writer.Close()

		pdfPaths[1] = pdfPath
	}

	// Create PDF 3: Empty document
	{
		pdfPath := filepath.Join(dir, "comprehensive_3.pdf")
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer 3: %v", err)
		}

		// Save the PDF without adding any pages
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save PDF 3: %v", err)
		}
		writer.Close()

		pdfPaths[2] = pdfPath
	}

	return pdfPaths
}

// Helper function to process PDFs
func processPDFs(t *testing.T, ctx *Context, pdfPaths []string) {
	for i, pdfPath := range pdfPaths {
		// Open document
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open document %d: %v", i+1, err)
		}

		// Get page count
		pageCount := doc.CountPages()
		t.Logf("PDF %d has %d page(s)", i+1, pageCount)

		// Process each page
		for j := 0; j < pageCount; j++ {
			// Load page
			page, err := doc.LoadPage(j)
			if err != nil {
				t.Fatalf("Failed to load page %d of PDF %d: %v", j, i+1, err)
			}

			// Get page bounds
			bounds := page.Bound()
			t.Logf("PDF %d, Page %d bounds: %+v", i+1, j, bounds)

			// Extract text
			text, err := page.ExtractText()
			if err != nil {
				t.Fatalf("Failed to extract text from page %d of PDF %d: %v", j, i+1, err)
			}

			// Get text content
			content := text.String()
			t.Logf("PDF %d, Page %d text length: %d", i+1, j, len(content))

			// Clean up
			text.Close()
			page.Close()
		}

		// Try to convert to PDF document
		pdfDoc, err := doc.AsPDFDocument()
		if err != nil {
			t.Logf("Could not convert document %d to PDF: %v", i+1, err)
		} else {
			t.Logf("Successfully converted document %d to PDF", i+1)

			// Check page count again
			pdfPageCount := pdfDoc.CountPages()
			if pdfPageCount != pageCount {
				t.Errorf("PDF %d: Page count mismatch: %d vs %d", i+1, pageCount, pdfPageCount)
			}
		}

		// Clean up
		doc.Close()
	}
}

// Helper function for concurrent processing. A Context is not thread-safe
// and must not be shared across goroutines, so each worker creates its own
// Context and opens the document itself.
func concurrentProcessing(t *testing.T, pdfPaths []string) {
	// Expected page counts for the PDFs created by createMultiplePDFs
	expectedPages := []int{1, 3, 0}
	if len(pdfPaths) != len(expectedPages) {
		t.Fatalf("Expected %d PDF paths, got %d", len(expectedPages), len(pdfPaths))
	}

	// Number of concurrent workers
	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Process PDFs concurrently, one context per worker
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()

			// Select a PDF to process (cycling through available PDFs)
			pdfIdx := id % len(pdfPaths)
			pdfPath := pdfPaths[pdfIdx]

			// Each worker creates its own Context
			ctx, err := NewContext()
			if err != nil {
				t.Errorf("Worker %d: Failed to create context: %v", id, err)
				return
			}
			defer ctx.Drop()

			// Open document with this worker's own context
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to open document: %v", id, err)
				return
			}
			defer doc.Close()

			// Assert page count matches what createMultiplePDFs produced
			pageCount := doc.CountPages()
			if pageCount != expectedPages[pdfIdx] {
				t.Errorf("Worker %d: Expected %d pages in %s, got %d",
					id, expectedPages[pdfIdx], pdfPath, pageCount)
				return
			}

			// Process each page
			for j := 0; j < pageCount; j++ {
				// Load page
				page, err := doc.LoadPage(j)
				if err != nil {
					t.Errorf("Worker %d: Failed to load page %d: %v", id, j, err)
					return
				}

				// Assert page bounds are valid
				bounds := page.Bound()
				if bounds.X1 <= bounds.X0 || bounds.Y1 <= bounds.Y0 {
					t.Errorf("Worker %d: Page %d has invalid bounds: %+v", id, j, bounds)
				}

				// Extract text
				text, err := page.ExtractText()
				if err != nil {
					t.Errorf("Worker %d: Failed to extract text from page %d: %v", id, j, err)
					page.Close()
					return
				}

				_ = text.String()

				// Clean up
				text.Close()
				page.Close()
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
}

// Helper function for error handling and recovery
func errorHandlingAndRecovery(t *testing.T, ctx *Context, dir string) {
	// Create an invalid PDF file
	invalidPdfPath := filepath.Join(dir, "invalid.pdf")
	f, err := os.Create(invalidPdfPath)
	if err != nil {
		t.Fatalf("Failed to create invalid PDF file: %v", err)
	}

	// Write invalid content
	_, err = f.WriteString("This is not a valid PDF file")
	if err != nil {
		t.Fatalf("Failed to write invalid PDF content: %v", err)
	}
	f.Close()

	// Try to open the invalid file
	_, err = OpenDocument(ctx, invalidPdfPath)
	if err == nil {
		t.Error("Expected error when opening invalid PDF file")
	} else {
		t.Logf("Got expected error when opening invalid file: %v", err)
	}

	// Create a valid PDF after the error
	validPdfPath := filepath.Join(dir, "valid_after_error.pdf")
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer after error: %v", err)
	}

	// Add a page
	_, err = writer.AddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page after error: %v", err)
	}

	// Save the PDF
	err = writer.Save(validPdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF after error: %v", err)
	}
	writer.Close()

	// Open the valid PDF
	doc, err := OpenDocument(ctx, validPdfPath)
	if err != nil {
		t.Fatalf("Failed to open valid PDF after error: %v", err)
	}
	defer doc.Close()

	// Check page count: the writer added exactly one page
	pageCount := doc.CountPages()
	if pageCount != 1 {
		t.Errorf("Expected 1 page in PDF created after error, got %d", pageCount)
	}
}

// Helper function for resource cleanup
func resourceCleanup(t *testing.T, ctx *Context, pdfPath string) {
	// Open document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document for cleanup test: %v", err)
	}
	defer doc.Close()

	// Load page
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page for cleanup test: %v", err)
	}
	defer page.Close()

	// Extract text
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text for cleanup test: %v", err)
	}
	defer text.Close()

	// Get text content
	_ = text.String()

	// Run garbage collection
	runtime.GC()
}

// TestComprehensiveWithMultipleContexts tests using multiple contexts with comprehensive operations
func TestComprehensiveWithMultipleContexts(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create multiple contexts
	ctx1, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context 1: %v", err)
	}
	defer ctx1.Drop()

	ctx2, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context 2: %v", err)
	}
	defer ctx2.Drop()

	// Create a temporary directory for test files
	dir := testDataDir(t)

	// Create a PDF with ctx1
	pdfPath := filepath.Join(dir, "multi_context.pdf")
	writer, err := NewPDFWriter(ctx1)
	if err != nil {
		t.Fatalf("Failed to create PDF writer with ctx1: %v", err)
	}

	// Add multiple pages
	numPages := 3
	for i := 0; i < numPages; i++ {
		_, err = writer.AddPage(595, 842) // A4 size
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}
	writer.Close()

	// Open the PDF with ctx2
	doc, err := OpenDocument(ctx2, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document with ctx2: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	if pageCount != numPages {
		t.Errorf("Expected %d pages, got %d", numPages, pageCount)
	}

	// Process each page
	for i := 0; i < pageCount; i++ {
		// Load page
		page, err := doc.LoadPage(i)
		if err != nil {
			t.Fatalf("Failed to load page %d: %v", i, err)
		}

		// Get page bounds
		bounds := page.Bound()
		t.Logf("Page %d bounds: %+v", i, bounds)

		// Extract text
		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text from page %d: %v", i, err)
		}

		// Get text content
		content := text.String()
		t.Logf("Page %d text length: %d", i, len(content))

		// Clean up
		text.Close()
		page.Close()
	}

	// Create another PDF with ctx2
	pdfPath2 := filepath.Join(dir, "multi_context_2.pdf")
	writer2, err := NewPDFWriter(ctx2)
	if err != nil {
		t.Fatalf("Failed to create PDF writer with ctx2: %v", err)
	}

	// Add a page
	_, err = writer2.AddPage(612, 792) // Letter size
	if err != nil {
		t.Fatalf("Failed to add page to second PDF: %v", err)
	}

	// Save the PDF
	err = writer2.Save(pdfPath2)
	if err != nil {
		t.Fatalf("Failed to save second PDF: %v", err)
	}
	writer2.Close()

	// Open the second PDF with ctx1
	doc2, err := OpenDocument(ctx1, pdfPath2)
	if err != nil {
		t.Fatalf("Failed to open second document with ctx1: %v", err)
	}
	defer doc2.Close()

	// Check page count
	pageCount2 := doc2.CountPages()
	if pageCount2 != 1 {
		t.Errorf("Expected 1 page in second PDF, got %d", pageCount2)
	}

	// Load the page
	page2, err := doc2.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page from second PDF: %v", err)
	}
	defer page2.Close()

	// Get page bounds
	bounds2 := page2.Bound()
	expectedWidth := 612.0
	expectedHeight := 792.0

	// Allow for some floating-point precision issues
	const epsilon = 0.1
	if bounds2.X0 < -epsilon || bounds2.Y0 < -epsilon ||
		abs(bounds2.X1-expectedWidth) > epsilon ||
		abs(bounds2.Y1-expectedHeight) > epsilon {
		t.Errorf("Page has incorrect bounds: got %+v, want {0, 0, %f, %f}",
			bounds2, expectedWidth, expectedHeight)
	}

	// Run garbage collection
	runtime.GC()
}
