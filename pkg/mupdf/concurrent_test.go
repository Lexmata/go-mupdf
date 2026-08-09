package mupdf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Concurrency contract verified by the tests in this file:
//
// A Context is NOT thread-safe and must not be shared across goroutines.
// The supported pattern is one Context per goroutine: each goroutine creates
// its own Context (and opens its own Document) and drops it when done.
// No test in this file shares a Context between goroutines.

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

// TestConcurrentDocuments tests opening and reading the same PDF file from
// multiple goroutines. Each goroutine uses its own Context: sharing a single
// Context across goroutines is not supported.
func TestConcurrentDocuments(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a test PDF that every worker opens independently
	pdfPath := createTestPDF(t)

	// Number of concurrent operations
	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	errs := make(chan error, numWorkers)

	// Open and process the same document concurrently, one context per goroutine
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine creates its own Context
			ctx, err := NewContext()
			if err != nil {
				errs <- fmt.Errorf("worker %d: failed to create context: %w", id, err)
				return
			}
			defer ctx.Drop()

			// Open the shared PDF file with this goroutine's own context
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				errs <- fmt.Errorf("worker %d: failed to open document: %w", id, err)
				return
			}
			defer doc.Close()

			// Check page count
			pageCount := doc.CountPages()
			if pageCount != 1 {
				errs <- fmt.Errorf("worker %d: expected 1 page, got %d", id, pageCount)
				return
			}

			// Load the page
			page, err := doc.LoadPage(0)
			if err != nil {
				errs <- fmt.Errorf("worker %d: failed to load page: %w", id, err)
				return
			}
			defer page.Close()

			// Extract text
			text, err := page.ExtractText()
			if err != nil {
				errs <- fmt.Errorf("worker %d: failed to extract text: %w", id, err)
				return
			}
			defer text.Close()

			if text.String() == "" {
				errs <- fmt.Errorf("worker %d: extracted empty text, expected non-empty text", id)
			}
		}(i)
	}

	// Wait for all goroutines to complete, then report any errors
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
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

// TestParallelTextExtraction tests extracting text from the same PDF file in
// parallel. Each goroutine uses its own Context and opens its own Document
// and Page, then the extracted text is compared against a sequentially
// extracted reference: all goroutines must produce identical non-empty text.
func TestParallelTextExtraction(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create a test PDF containing known text ("Hello World")
	pdfPath := createTestPDF(t)

	// extract opens the PDF with a fresh Context and returns the text of page 0
	extract := func() (string, error) {
		ctx, err := NewContext()
		if err != nil {
			return "", fmt.Errorf("failed to create context: %w", err)
		}
		defer ctx.Drop()

		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			return "", fmt.Errorf("failed to open document: %w", err)
		}
		defer doc.Close()

		page, err := doc.LoadPage(0)
		if err != nil {
			return "", fmt.Errorf("failed to load page: %w", err)
		}
		defer page.Close()

		text, err := page.ExtractText()
		if err != nil {
			return "", fmt.Errorf("failed to extract text: %w", err)
		}
		defer text.Close()

		return text.String(), nil
	}

	// Sequential reference extraction
	reference, err := extract()
	if err != nil {
		t.Fatalf("Sequential reference extraction failed: %v", err)
	}
	if reference == "" {
		t.Fatal("Reference extraction produced empty text, expected non-empty text")
	}

	// Extract the same text from multiple goroutines, one context per goroutine
	numWorkers := 8
	results := make([]string, numWorkers)
	workerErrs := make([]error, numWorkers)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			results[id], workerErrs[id] = extract()
		}(i)
	}
	wg.Wait()

	// Every goroutine must have produced the same non-empty text
	for i := 0; i < numWorkers; i++ {
		if workerErrs[i] != nil {
			t.Errorf("Worker %d: %v", i, workerErrs[i])
			continue
		}
		if results[i] == "" {
			t.Errorf("Worker %d: extracted empty text, expected non-empty text", i)
			continue
		}
		if results[i] != reference {
			t.Errorf("Worker %d: extracted text mismatch: got %q, want %q", i, results[i], reference)
		}
	}
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
