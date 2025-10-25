package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkPDFCreation(b *testing.B) {
	// Create context
	ctx, err := NewContext()
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create temporary directory
	dir, err := os.MkdirTemp("", "mupdf-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() {
			// Create PDF writer
			writer, err := NewPDFWriter(ctx)
			if err != nil {
				b.Fatalf("Failed to create PDF writer: %v", err)
			}

			// Add a page
			_, err = writer.AddPage(595, 842) // A4 size
			if err != nil {
				b.Fatalf("Failed to add page: %v", err)
			}

			// Save the PDF
			pdfPath := filepath.Join(dir, "benchmark.pdf")
			err = writer.Save(pdfPath)
			if err != nil {
				b.Fatalf("Failed to save PDF: %v", err)
			}

			writer.Close()
		}()
	}
}

func BenchmarkPDFOpening(b *testing.B) {
	// Create context
	ctx, err := NewContext()
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	dir, err := os.MkdirTemp("", "mupdf-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "benchmark.pdf")

	// Create a PDF to benchmark opening
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		b.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		b.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		b.Fatalf("Failed to save PDF: %v", err)
	}

	writer.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() {
			// Open the PDF
			doc, err := OpenDocument(ctx, pdfPath)
			if err != nil {
				b.Fatalf("Failed to open document: %v", err)
			}

			doc.Close()
		}()
	}
}

func BenchmarkPDFPageLoading(b *testing.B) {
	// Create context
	ctx, err := NewContext()
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF with multiple pages
	dir, err := os.MkdirTemp("", "mupdf-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "benchmark.pdf")

	// Create a PDF with multiple pages
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		b.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add multiple pages
	numPages := 10
	for i := 0; i < numPages; i++ {
		_, err = writer.AddPage(595, 842) // A4 size
		if err != nil {
			b.Fatalf("Failed to add page: %v", err)
		}
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		b.Fatalf("Failed to save PDF: %v", err)
	}

	writer.Close()

	// Open the PDF
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		b.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Load a page (cycling through all pages)
		pageNum := i % numPages
		page, err := doc.LoadPage(pageNum)
		if err != nil {
			b.Fatalf("Failed to load page: %v", err)
		}

		page.Close()
	}
}

func BenchmarkTextExtraction(b *testing.B) {
	// Create context
	ctx, err := NewContext()
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF
	dir, err := os.MkdirTemp("", "mupdf-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "benchmark.pdf")

	// Create a PDF to benchmark text extraction
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		b.Fatalf("Failed to create PDF writer: %v", err)
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		b.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		b.Fatalf("Failed to save PDF: %v", err)
	}

	writer.Close()

	// Open the PDF
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		b.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Load the page
	page, err := doc.LoadPage(0)
	if err != nil {
		b.Fatalf("Failed to load page: %v", err)
	}
	defer page.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() {
			// Extract text
			text, err := page.ExtractText()
			if err != nil {
				b.Fatalf("Failed to extract text: %v", err)
			}

			_ = text.String()
			text.Close()
		}()
	}
}

func BenchmarkPDFObjectCreation(b *testing.B) {
	// Create context
	ctx, err := NewContext()
	if err != nil {
		b.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create PDF writer
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		b.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create objects of different types
		switch i % 5 {
		case 0:
			obj, err := writer.NewPDFObject(nil)
			if err != nil {
				b.Fatalf("Failed to create null object: %v", err)
			}
			obj.Drop()
		case 1:
			obj, err := writer.NewPDFObject(true)
			if err != nil {
				b.Fatalf("Failed to create boolean object: %v", err)
			}
			obj.Drop()
		case 2:
			obj, err := writer.NewPDFObject(i)
			if err != nil {
				b.Fatalf("Failed to create integer object: %v", err)
			}
			obj.Drop()
		case 3:
			obj, err := writer.NewPDFObject(float64(i) / 10.0)
			if err != nil {
				b.Fatalf("Failed to create float object: %v", err)
			}
			obj.Drop()
		case 4:
			obj, err := writer.NewPDFObject("Benchmark string")
			if err != nil {
				b.Fatalf("Failed to create string object: %v", err)
			}
			obj.Drop()
		}
	}
}
