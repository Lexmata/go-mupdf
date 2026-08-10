package mupdf

import (
	"runtime"
	"testing"
)

// TestC1RefcountBalance exercises the AsPDFDocument -> Close path that the
// review flagged as a use-after-free, both explicitly and via the finalizer.
func TestC1RefcountBalance(t *testing.T) {
	path := createTestPDF(t)

	// Explicit Close on the PDFDocument, then keep using the parent.
	for i := 0; i < 200; i++ {
		ctx, err := NewContext()
		if err != nil {
			t.Fatalf("NewContext: %v", err)
		}
		doc, err := OpenDocument(ctx, path)
		if err != nil {
			t.Fatalf("OpenDocument: %v", err)
		}
		pdf, err := doc.AsPDFDocument()
		if err != nil {
			t.Fatalf("AsPDFDocument: %v", err)
		}
		pdf.Close()
		// Parent must still be usable after the child dropped its reference.
		if n := doc.CountPages(); n <= 0 {
			t.Fatalf("parent Document unusable after PDFDocument.Close(): %d", n)
		}
		if _, err := doc.LoadPage(0); err != nil {
			t.Fatalf("LoadPage after PDFDocument.Close(): %v", err)
		}
		doc.Close()
		ctx.Drop()
	}

	// PDFDocument outliving its parent Document.
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer ctx.Drop()
	doc, err := OpenDocument(ctx, path)
	if err != nil {
		t.Fatalf("OpenDocument: %v", err)
	}
	pdf, err := doc.AsPDFDocument()
	if err != nil {
		t.Fatalf("AsPDFDocument: %v", err)
	}
	doc.Close() // parent goes away first
	if n := pdf.CountPages(); n <= 0 {
		t.Fatalf("PDFDocument unusable after parent Close(): %d", n)
	}
	pdf.Close()
	pdf.Close() // idempotent

	// Finalizer path: abandon many PDFDocuments without closing them.
	for i := 0; i < 300; i++ {
		func() {
			c, err := NewContext()
			if err != nil {
				t.Fatalf("NewContext: %v", err)
			}
			defer c.Drop()
			d, err := OpenDocument(c, path)
			if err != nil {
				t.Fatalf("OpenDocument: %v", err)
			}
			defer d.Close()
			if _, err := d.AsPDFDocument(); err != nil {
				t.Fatalf("AsPDFDocument: %v", err)
			}
		}()
		runtime.GC()
	}
	runtime.GC()
	runtime.GC()
}
