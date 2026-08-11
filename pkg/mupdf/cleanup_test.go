package mupdf

import (
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// finalizerSentinel is a small type used to observe that the garbage
// collector actually ran finalizers during a test. The padding is
// load-bearing: a zero-size struct shares runtime.zerobase, and
// runtime.SetFinalizer on such an object never fires. The field is
// deliberately unread, hence the blank name.
type finalizerSentinel struct {
	_ [16]byte
}

// TestResourceCleanupWithFinalizers tests that finalizers properly clean up resources.
//
// The package's C-side finalizers (on Document, Page, TextPage) are not
// directly observable without API changes, so this test asserts what is
// observable: it installs a test-local sentinel finalizer alongside the
// package objects, drops all references, and forces GC until the sentinel
// finalizer fires. That proves the GC + finalizer machinery executed over
// the same collection cycles in which the package finalizers were queued,
// and that running those finalizers did not crash.
func TestResourceCleanupWithFinalizers(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create a context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Open document without explicit Close()
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Load page without explicit Close()
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	// Extract text without explicit Close()
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	// Get text content
	_ = text.String()

	// Install a sentinel finalizer that flips an atomic flag when it runs
	var sentinelFired atomic.Bool
	sentinel := &finalizerSentinel{}
	runtime.SetFinalizer(sentinel, func(*finalizerSentinel) {
		sentinelFired.Store(true)
	})

	// Set all references to nil to make the objects (and their finalizers)
	// eligible for collection
	text = nil
	page = nil
	doc = nil

	// Force GC until the sentinel finalizer has run (or we time out)
	deadline := time.Now().Add(2 * time.Second)
	for !sentinelFired.Load() && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}

	if !sentinelFired.Load() {
		t.Fatal("Sentinel finalizer did not run within 2s; GC finalizer machinery did not execute")
	}
}

// TestNestedResourceCleanup tests cleanup of resources in nested function calls
func TestNestedResourceCleanup(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create a context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Function to test nested resource creation and cleanup
	var processDocument func(depth int) error
	processDocument = func(depth int) error {
		// Base case
		if depth <= 0 {
			return nil
		}

		// Open document
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			return err
		}
		defer doc.Close()

		// Load page
		page, err := doc.LoadPage(0)
		if err != nil {
			return err
		}
		defer page.Close()

		// Extract text
		text, err := page.ExtractText()
		if err != nil {
			return err
		}
		defer text.Close()

		// Get text content
		_ = text.String()

		// Recursive call
		return processDocument(depth - 1)
	}

	// Process document with nested calls
	err = processDocument(5)
	if err != nil {
		t.Fatalf("Failed in nested resource processing: %v", err)
	}

	// Run garbage collection
	runtime.GC()
}

// TestResourceCleanupAfterPanic tests that resources are properly cleaned up after a panic
func TestResourceCleanupAfterPanic(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create a context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Function that will panic
	panickingFunction := func() (err error) {
		// Use defer to recover from panic and ensure cleanup
		defer func() {
			if r := recover(); r != nil {
				err = Error{message: "Recovered from panic"}
			}
		}()

		// Open document with proper cleanup
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			return err
		}
		defer doc.Close()

		// Load page with proper cleanup
		page, err := doc.LoadPage(0)
		if err != nil {
			return err
		}
		defer page.Close()

		// Extract text with proper cleanup
		text, err := page.ExtractText()
		if err != nil {
			return err
		}
		defer text.Close()

		// Cause a panic
		panic("Intentional panic for testing")
	}

	// Call the panicking function
	err = panickingFunction()
	if err == nil {
		t.Fatal("Expected error after panic recovery")
	}

	// Run garbage collection
	runtime.GC()

	// If we reach here without crashes, cleanup worked properly
	t.Log("Resources were properly cleaned up after panic")
}

// TestMultipleContextsCleanup tests creating and cleaning up multiple contexts
func TestMultipleContextsCleanup(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Number of contexts to create
	numContexts := 10

	// Create multiple contexts
	contexts := make([]*Context, numContexts)
	for i := 0; i < numContexts; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Failed to create context %d: %v", i, err)
		}
		contexts[i] = ctx
	}

	// Close contexts in reverse order
	for i := numContexts - 1; i >= 0; i-- {
		contexts[i].Drop()
	}

	// Run garbage collection
	runtime.GC()
}

// TestLargeDocumentCleanup tests cleanup with a larger document
func TestLargeDocumentCleanup(t *testing.T) {
	requireMuPDF(t)
	skipIfCIorShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a larger PDF with multiple pages
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "large_document.pdf")

	// Create a PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add multiple pages
	numPages := 20
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

	// Open the document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Load all pages
	pages := make([]*Page, numPages)
	for i := 0; i < numPages; i++ {
		page, err := doc.LoadPage(i)
		if err != nil {
			t.Fatalf("Failed to load page %d: %v", i, err)
		}
		pages[i] = page
	}

	// Extract text from all pages
	texts := make([]*TextPage, numPages)
	for i, page := range pages {
		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text from page %d: %v", i, err)
		}
		texts[i] = text
	}

	// Close resources in reverse order
	for i := 0; i < numPages; i++ {
		texts[i].Close()
		pages[i].Close()
	}
	doc.Close()

	// Run garbage collection
	runtime.GC()
}

// TestResourceLeakCheck tests for resource leaks by creating and destroying
// many objects, asserting that heap usage stays within loose bounds relative
// to a post-warm-up baseline. The bounds are deliberately very generous:
// they catch runaway per-iteration leaks without flaking on normal allocator
// variance.
func TestResourceLeakCheck(t *testing.T) {
	requireMuPDF(t)

	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "leak_test.pdf")

	// One full create/use/destroy cycle of every object type
	iteration := func() {
		// Create context
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("Failed to create context: %v", err)
		}
		defer ctx.Drop()

		// Create a PDF writer
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer: %v", err)
		}
		defer writer.Close()

		// Add a page
		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}

		// Create some PDF objects
		for j := 0; j < 10; j++ {
			obj, err := writer.NewPDFObject(j)
			if err != nil {
				t.Fatalf("Failed to create object: %v", err)
			}
			obj.Drop()
		}

		// Save to a temporary file
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save PDF: %v", err)
		}

		// Open the document
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open document: %v", err)
		}
		defer doc.Close()

		// Load the page
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		// Extract text
		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text: %v", err)
		}
		defer text.Close()

		_ = text.String()

		// Delete the file to avoid filling the disk
		os.Remove(pdfPath)
	}

	// Warm-up iteration: let one-time allocations (lazy initialization,
	// caches) settle before taking the baseline measurement
	iteration()

	runtime.GC()
	runtime.GC()
	var warmupStats runtime.MemStats
	runtime.ReadMemStats(&warmupStats)

	// Create and destroy many objects
	iterations := 50
	for i := 0; i < iterations; i++ {
		iteration()

		// Run garbage collection every 10 iterations
		if i%10 == 0 {
			runGC()
		}
	}

	// Final garbage collection before measuring
	runtime.GC()
	runtime.GC()
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)

	t.Logf("Warm-up heap objects: %d, final heap objects: %d",
		warmupStats.HeapObjects, finalStats.HeapObjects)
	t.Logf("Warm-up heap alloc: %d bytes, final heap alloc: %d bytes",
		warmupStats.HeapAlloc, finalStats.HeapAlloc)

	// Assert loose-but-real bounds against the warm-up baseline
	maxHeapObjects := warmupStats.HeapObjects*2 + 10000
	if finalStats.HeapObjects >= maxHeapObjects {
		t.Errorf("Possible resource leak: final heap objects %d exceeds bound %d (warm-up baseline %d)",
			finalStats.HeapObjects, maxHeapObjects, warmupStats.HeapObjects)
	}

	const allocSlack = 32 << 20 // 32 MB
	if finalStats.HeapAlloc > warmupStats.HeapAlloc+allocSlack {
		t.Errorf("Possible resource leak: heap alloc grew from %d to %d bytes (more than %d bytes of slack)",
			warmupStats.HeapAlloc, finalStats.HeapAlloc, int64(allocSlack))
	}
}

// TestDoubleCloseAllTypes tests double-closing all resource types
func TestDoubleCloseAllTypes(t *testing.T) {
	requireMuPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test double-closing context
	ctx.Drop()
	ctx.Drop() // Should not panic

	// Create a new context for further tests
	ctx, err = NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Test double-closing document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	doc.Close()
	doc.Close() // Should not panic

	// Open document again for page test
	doc, err = OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Test double-closing page
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	page.Close()
	page.Close() // Should not panic

	// Load page again for text test
	page, err = doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	// Test double-closing text
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}
	text.Close()
	text.Close() // Should not panic

	// Test PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	writer.Close()
	writer.Close() // Should not panic

	// Create writer again for PDF object test
	writer, err = NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	// Test double-closing PDF object
	obj, err := writer.NewPDFObject("test")
	if err != nil {
		t.Fatalf("Failed to create PDF object: %v", err)
	}
	obj.Drop()
	obj.Drop() // Should not panic
}

// TestCleanupOrder tests that resources are cleaned up in the correct order
func TestCleanupOrder(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Create a test PDF
	pdfPath := createTestPDF(t)

	// Open document
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Load page
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	// Extract text
	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	// Close in correct order: text, page, document, context
	text.Close()
	page.Close()
	doc.Close()
	ctx.Drop()

	// Run garbage collection
	runtime.GC()

	// Now test incorrect order (should still work due to finalizers and reference counting in MuPDF)
	ctx, err = NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Open document
	doc, err = OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}

	// Load page
	page, err = doc.LoadPage(0)
	if err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	// Extract text
	text, err = page.ExtractText()
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	// Close in reverse order: context, document, page, text
	// This relies on MuPDF's reference counting to handle cleanup properly
	ctx.Drop()
	doc.Close()
	page.Close()
	text.Close()

	// Run garbage collection
	runtime.GC()
}
