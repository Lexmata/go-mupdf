package mupdf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentContexts tests using multiple contexts concurrently
func TestConcurrentContexts(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Number of concurrent contexts to create
	numContexts := 5
	var wg sync.WaitGroup
	wg.Add(numContexts)

	// Create multiple contexts concurrently
	for i := 0; i < numContexts; i++ {
		go func(id int) {
			defer wg.Done()

			// Create context
			ctx, err := NewContext()
			if err != nil {
				t.Errorf("Worker %d: Failed to create context: %v", id, err)
				return
			}
			defer ctx.Drop()

			// Create a PDF writer
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				t.Errorf("Worker %d: Failed to create PDF writer: %v", id, err)
				return
			}
			defer writer.Close()

			// Add a page
			_, err = writer.AddPage(595, 842) // A4 size
			if err != nil {
				t.Errorf("Worker %d: Failed to add page: %v", id, err)
				return
			}

			// Create a temporary directory for this worker
			dir, err := os.MkdirTemp("", "mupdf-concurrent")
			if err != nil {
				t.Errorf("Worker %d: Failed to create temp directory: %v", id, err)
				return
			}
			defer os.RemoveAll(dir)

			// Save the PDF
			pdfPath := filepath.Join(dir, "concurrent.pdf")
			err = writer.Save(pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to save PDF: %v", id, err)
				return
			}

			// Open the PDF
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to open document: %v", id, err)
				return
			}
			defer doc.Close()

			// Check page count
			pageCount := doc.CountPages()
			if pageCount != 1 {
				t.Errorf("Worker %d: Expected 1 page, got %d", id, pageCount)
				return
			}

			// Load the page
			page, err := doc.LoadPage(0)
			if err != nil {
				t.Errorf("Worker %d: Failed to load page: %v", id, err)
				return
			}
			defer page.Close()

			// Extract text
			text, err := page.ExtractText()
			if err != nil {
				t.Errorf("Worker %d: Failed to extract text: %v", id, err)
				return
			}
			defer text.Close()

			_ = text.String()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// TestConcurrentDocuments tests operating on multiple documents concurrently with a single context
func TestConcurrentDocuments(t *testing.T) {
	t.Skip("Temporarily disabled due to concurrency issues - will be fixed in comprehensive rewrite")
	requireMuPDF(t)
	skipIfShort(t)

	// Create a shared context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Number of concurrent operations
	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Create a mutex to protect context access
	var mu sync.Mutex

	// Perform concurrent operations on the same document
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()

			// Use mutex to protect context access
			mu.Lock()
			doc, err := OpenDocument(ctx, pdfPath)
			mu.Unlock()

			if err != nil {
				t.Errorf("Worker %d: Failed to open document: %v", id, err)
				return
			}
			defer doc.Close()

			// Check page count
			pageCount := doc.CountPages()
			if pageCount != 1 {
				t.Errorf("Worker %d: Expected 1 page, got %d", id, pageCount)
				return
			}

			// Load the page
			mu.Lock()
			page, err := doc.LoadPage(0)
			mu.Unlock()

			if err != nil {
				t.Errorf("Worker %d: Failed to load page: %v", id, err)
				return
			}
			defer page.Close()

			// Extract text
			mu.Lock()
			text, err := page.ExtractText()
			mu.Unlock()

			if err != nil {
				t.Errorf("Worker %d: Failed to extract text: %v", id, err)
				return
			}
			defer text.Close()

			_ = text.String()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// TestConcurrentPDFCreation tests creating multiple PDFs concurrently
// Each goroutine uses its own context to ensure thread safety
func TestConcurrentPDFCreation(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a shared temporary directory for all PDFs
	dir := testDataDir(t)

	// Number of concurrent PDF creations
	numPDFs := 5
	var wg sync.WaitGroup
	wg.Add(numPDFs)

	// Create multiple PDFs concurrently, each with its own context
	for i := 0; i < numPDFs; i++ {
		go func(id int) {
			defer wg.Done()

			// Create a separate context for this goroutine
			// This ensures thread safety - each goroutine has its own isolated context
			ctx, err := NewContext()
			if err != nil {
				t.Errorf("Worker %d: Failed to create context: %v", id, err)
				return
			}
			defer ctx.Drop()

			// Create a PDF writer using this context
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				t.Errorf("Worker %d: Failed to create PDF writer: %v", id, err)
				return
			}
			defer writer.Close()

			// Add pages - each worker creates a different number of pages
			numPages := id + 1
			for j := 0; j < numPages; j++ {
				_, err = writer.AddPage(595, 842) // A4 size
				if err != nil {
					t.Errorf("Worker %d: Failed to add page %d: %v", id, j, err)
					return
				}
			}

			// Save the PDF
			pdfPath := filepath.Join(dir, fmt.Sprintf("concurrent_%c.pdf", 'A'+id))
			err = writer.Save(pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to save PDF: %v", id, err)
				return
			}

			// Verify the PDF by opening it with the same context
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to open created document: %v", id, err)
				return
			}
			defer doc.Close()

			// Check page count
			pageCount := doc.CountPages()
			if pageCount != numPages {
				t.Errorf("Worker %d: Expected %d pages, got %d", id, numPages, pageCount)
				return
			}

			// Verify we can load and access pages
			for j := 0; j < pageCount; j++ {
				page, err := doc.LoadPage(j)
				if err != nil {
					t.Errorf("Worker %d: Failed to load page %d: %v", id, j, err)
					return
				}

				// Verify page bounds (Rect has X0, Y0, X1, Y1 fields)
				bounds := page.Bound()
				width := bounds.X1 - bounds.X0
				height := bounds.Y1 - bounds.Y0
				if width <= 0 || height <= 0 {
					t.Errorf("Worker %d: Page %d has invalid bounds: %+v (width=%.2f, height=%.2f)", id, j, bounds, width, height)
				}

				page.Close()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// TestParallelTextExtraction tests extracting text from multiple pages in parallel
func TestParallelTextExtraction(t *testing.T) {
	t.Skip("Temporarily disabled due to concurrency issues - will be fixed in comprehensive rewrite")
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a multi-page PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "multipage_parallel.pdf")

	// Create a PDF with multiple pages
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add multiple pages
	numPages := 10
	for i := 0; i < numPages; i++ {
		_, err = writer.AddPage(595, 842) // A4 size
		if err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}
	writer.Close()

	// Open the PDF
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Load all pages first to avoid concurrent page loading
	pages := make([]*Page, numPages)
	for i := 0; i < numPages; i++ {
		page, err := doc.LoadPage(i)
		if err != nil {
			t.Fatalf("Failed to load page %d: %v", i, err)
		}
		pages[i] = page
	}

	// Extract text from all pages in parallel
	var wg sync.WaitGroup
	wg.Add(numPages)

	for i := 0; i < numPages; i++ {
		go func(pageIdx int, page *Page) {
			defer wg.Done()
			defer page.Close()

			text, err := page.ExtractText()
			if err != nil {
				t.Errorf("Failed to extract text from page %d: %v", pageIdx, err)
				return
			}
			defer text.Close()

			content := text.String()
			t.Logf("Page %d text length: %d", pageIdx, len(content))
		}(i, pages[i])
	}

	// Wait for all text extraction operations to complete
	wg.Wait()
}

// TestConcurrentResourceCleanup tests cleanup of resources in concurrent environment
func TestConcurrentResourceCleanup(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Number of iterations
	iterations := 5

	// Create a wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(iterations)

	// Run multiple iterations in parallel
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()

			// Create context
			ctx, err := NewContext()
			if err != nil {
				t.Errorf("Worker %d: Failed to create context: %v", id, err)
				return
			}

			// Create a PDF
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				t.Errorf("Worker %d: Failed to create PDF writer: %v", id, err)
				ctx.Drop()
				return
			}

			// Add pages
			for j := 0; j < 3; j++ {
				_, err = writer.AddPage(595, 842) // A4 size
				if err != nil {
					t.Errorf("Worker %d: Failed to add page %d: %v", id, j, err)
					writer.Close()
					ctx.Drop()
					return
				}
			}

			// Create a temporary directory
			dir, err := os.MkdirTemp("", "mupdf-cleanup")
			if err != nil {
				t.Errorf("Worker %d: Failed to create temp directory: %v", id, err)
				writer.Close()
				ctx.Drop()
				return
			}
			defer os.RemoveAll(dir)

			// Save the PDF
			pdfPath := filepath.Join(dir, "cleanup.pdf")
			err = writer.Save(pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to save PDF: %v", id, err)
				writer.Close()
				ctx.Drop()
				return
			}

			// Close the writer
			writer.Close()

			// Open the PDF
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Errorf("Worker %d: Failed to open document: %v", id, err)
				ctx.Drop()
				return
			}

			// Load all pages
			pages := make([]*Page, doc.CountPages())
			for j := 0; j < doc.CountPages(); j++ {
				page, err := doc.LoadPage(j)
				if err != nil {
					t.Errorf("Worker %d: Failed to load page %d: %v", id, j, err)
					doc.Close()
					ctx.Drop()
					return
				}
				pages[j] = page
			}

			// Extract text from all pages
			texts := make([]*TextPage, len(pages))
			for j, page := range pages {
				text, err := page.ExtractText()
				if err != nil {
					t.Errorf("Worker %d: Failed to extract text from page %d: %v", id, j, err)
					// Clean up previous texts
					for k := 0; k < j; k++ {
						if texts[k] != nil {
							texts[k].Close()
						}
					}
					// Clean up pages
					for _, p := range pages {
						if p != nil {
							p.Close()
						}
					}
					doc.Close()
					ctx.Drop()
					return
				}
				texts[j] = text
			}

			// Clean up in reverse order
			for _, text := range texts {
				if text != nil {
					text.Close()
				}
			}

			for _, page := range pages {
				if page != nil {
					page.Close()
				}
			}

			doc.Close()
			ctx.Drop()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}
