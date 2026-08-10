package mupdf

import (
	"runtime"
	"sync"
	"testing"
)

// TestCleanupDoesNotRaceWithContextDrop reproduces the data race that failed
// the v1.8.0 release pipeline.
//
// Cleanup reaches the C library from two goroutines: the one that owns the
// object, and the GC finalizer goroutine that runs the safety-net finalizer.
// Every Close/Drop used to test ctx.ctx without synchronisation while
// Context.Drop wrote it, which the race detector reported as
//
//	Read at ... by goroutine N: (*PDFDocument).Close()
//	Previous write at ... by goroutine M: (*Context).Drop()
//
// and which could hand MuPDF a context that Drop was in the middle of
// freeing.
//
// A finalizer is just "Close called from another goroutine", so each subtest
// below calls Close concurrently with Drop directly. That exercises the same
// read/write pair the finalizer hits, without depending on when the GC
// decides to run finalizers. Only meaningful under -race.
func TestCleanupDoesNotRaceWithContextDrop(t *testing.T) {
	requireMuPDF(t)

	pdfPath := createTestPDF(t)

	// Each case abandons one object type to concurrent cleanup. cleanup is
	// what the finalizer would have called.
	cases := []struct {
		name    string
		acquire func(t *testing.T, ctx *Context) (cleanup func())
	}{
		{
			name: "Document",
			acquire: func(t *testing.T, ctx *Context) func() {
				doc, err := OpenDocument(ctx, pdfPath)
				if err != nil {
					t.Fatalf("OpenDocument: %v", err)
				}
				return doc.Close
			},
		},
		{
			name: "PDFDocument",
			acquire: func(t *testing.T, ctx *Context) func() {
				doc, err := OpenDocument(ctx, pdfPath)
				if err != nil {
					t.Fatalf("OpenDocument: %v", err)
				}
				pdfDoc, err := doc.AsPDFDocument()
				if err != nil {
					t.Fatalf("AsPDFDocument: %v", err)
				}
				return pdfDoc.Close
			},
		},
		{
			name: "Page",
			acquire: func(t *testing.T, ctx *Context) func() {
				doc, err := OpenDocument(ctx, pdfPath)
				if err != nil {
					t.Fatalf("OpenDocument: %v", err)
				}
				page, err := doc.LoadPage(0)
				if err != nil {
					t.Fatalf("LoadPage: %v", err)
				}
				return page.Close
			},
		},
		{
			name: "TextPage",
			acquire: func(t *testing.T, ctx *Context) func() {
				doc, err := OpenDocument(ctx, pdfPath)
				if err != nil {
					t.Fatalf("OpenDocument: %v", err)
				}
				page, err := doc.LoadPage(0)
				if err != nil {
					t.Fatalf("LoadPage: %v", err)
				}
				text, err := page.ExtractText()
				if err != nil {
					t.Fatalf("ExtractText: %v", err)
				}
				return text.Close
			},
		},
		{
			name: "PDFWriter",
			acquire: func(t *testing.T, ctx *Context) func() {
				writer, err := NewPDFWriter(ctx)
				if err != nil {
					t.Fatalf("NewPDFWriter: %v", err)
				}
				return writer.Close
			},
		},
		{
			name: "PDFPage",
			acquire: func(t *testing.T, ctx *Context) func() {
				writer, err := NewPDFWriter(ctx)
				if err != nil {
					t.Fatalf("NewPDFWriter: %v", err)
				}
				page, err := writer.AddPage(595, 842)
				if err != nil {
					t.Fatalf("AddPage: %v", err)
				}
				return page.Close
			},
		},
		{
			name: "PDFObject",
			acquire: func(t *testing.T, ctx *Context) func() {
				writer, err := NewPDFWriter(ctx)
				if err != nil {
					t.Fatalf("NewPDFWriter: %v", err)
				}
				obj, err := writer.NewPDFObject(42)
				if err != nil {
					t.Fatalf("NewPDFObject: %v", err)
				}
				return obj.Drop
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Repeat so the two goroutines get a chance to interleave in
			// both orders; a single pass can miss the window.
			for i := 0; i < 25; i++ {
				ctx, err := NewContext()
				if err != nil {
					t.Fatalf("iteration %d: NewContext: %v", i, err)
				}

				cleanup := tc.acquire(t, ctx)

				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					cleanup()
				}()
				go func() {
					defer wg.Done()
					ctx.Drop()
				}()
				wg.Wait()

				// Both must stay idempotent after racing.
				cleanup()
				ctx.Drop()
			}
		})
	}
}

// TestFinalizerCleanupDoesNotRaceWithContextDrop covers the same hazard
// through the real GC finalizer goroutine rather than a stand-in. It is
// timing-dependent by nature, so it is a complement to the deterministic
// test above, not a replacement.
func TestFinalizerCleanupDoesNotRaceWithContextDrop(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	pdfPath := createTestPDF(t)

	for i := 0; i < 30; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("iteration %d: NewContext: %v", i, err)
		}

		// Create objects of every finalized type and abandon them without
		// closing, leaving cleanup to the finalizer goroutine.
		func() {
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				t.Fatalf("iteration %d: OpenDocument: %v", i, err)
			}
			if pdfDoc, err := doc.AsPDFDocument(); err == nil {
				_ = pdfDoc.CountPages()
			}
			if page, err := doc.LoadPage(0); err == nil {
				if text, err := page.ExtractText(); err == nil {
					_ = text.String()
				}
			}
			if writer, err := NewPDFWriter(ctx); err == nil {
				_, _ = writer.AddPage(595, 842)
				_, _ = writer.NewPDFObject(42)
			}
		}()

		// Keep GC (and therefore the finalizer goroutine) busy while the
		// Context is dropped underneath it.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for j := 0; j < 5; j++ {
				runtime.GC()
			}
		}()
		ctx.Drop()
		<-done

		ctx.Drop()
	}

	// Any finalizers still queued must no-op against the dropped contexts
	// rather than touching freed memory.
	runtime.GC()
	runtime.GC()
}

// TestCloseIsIdempotentAcrossGoroutines checks that concurrent Close calls on
// the same object drop the underlying reference exactly once. Both the owning
// goroutine and the finalizer goroutine can call Close, so a lost update here
// would mean a double free.
func TestCloseIsIdempotentAcrossGoroutines(t *testing.T) {
	requireMuPDF(t)

	pdfPath := createTestPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	pdfDoc, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("AsPDFDocument failed: %v", err)
	}

	const closers = 8
	var wg sync.WaitGroup
	wg.Add(closers)
	for i := 0; i < closers; i++ {
		go func() {
			defer wg.Done()
			pdfDoc.Close()
		}()
	}
	wg.Wait()

	// The parent Document keeps its own reference, so it must still work.
	if got := doc.CountPages(); got != 1 {
		t.Errorf("parent Document unusable after concurrent Close: got %d pages, want 1", got)
	}
}
