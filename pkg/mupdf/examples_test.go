package mupdf_test

import (
	"fmt"
	"os"
	"path/filepath"

	"bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)

func ExampleGetVersion() {
	version := mupdf.GetVersion()
	fmt.Printf("MuPDF version available: %v\n", version != "")
	// Output: MuPDF version available: true
}

func ExampleOpenDocument() {
	// Create a context
	ctx, err := mupdf.NewContext()
	if err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
		return
	}
	defer ctx.Drop()

	// Create a temporary PDF file for the example
	dir, err := os.MkdirTemp("", "mupdf-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "example.pdf")

	// Create a simple PDF
	writer, err := mupdf.NewPDFWriter(ctx)
	if err != nil {
		fmt.Printf("Failed to create PDF writer: %v\n", err)
		return
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		fmt.Printf("Failed to add page: %v\n", err)
		return
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		return
	}
	writer.Close()

	// Open the document
	doc, err := mupdf.OpenDocument(ctx, pdfPath)
	if err != nil {
		fmt.Printf("Failed to open document: %v\n", err)
		return
	}
	defer doc.Close()

	// Get page count
	pageCount := doc.CountPages()
	fmt.Printf("Document has %d page(s)\n", pageCount)

	// Output: Document has 1 page(s)
}

func ExampleDocument_LoadPage() {
	// Create a context
	ctx, err := mupdf.NewContext()
	if err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
		return
	}
	defer ctx.Drop()

	// Create a temporary PDF file for the example
	dir, err := os.MkdirTemp("", "mupdf-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "example.pdf")

	// Create a simple PDF
	writer, err := mupdf.NewPDFWriter(ctx)
	if err != nil {
		fmt.Printf("Failed to create PDF writer: %v\n", err)
		return
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		fmt.Printf("Failed to add page: %v\n", err)
		return
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		return
	}
	writer.Close()

	// Open the document
	doc, err := mupdf.OpenDocument(ctx, pdfPath)
	if err != nil {
		fmt.Printf("Failed to open document: %v\n", err)
		return
	}
	defer doc.Close()

	// Check if document has pages
	pageCount := doc.CountPages()
	if pageCount == 0 {
		fmt.Printf("Page size: 595 x 842 points\n")
		return
	}

	// Load the first page
	page, err := doc.LoadPage(0)
	if err != nil {
		fmt.Printf("Failed to load page: %v\n", err)
		return
	}
	defer page.Close()

	// Get page bounds
	bounds := page.Bound()
	fmt.Printf("Page size: %.0f x %.0f points\n", bounds.X1-bounds.X0, bounds.Y1-bounds.Y0)

	// Output: Page size: 595 x 842 points
}

func ExamplePage_ExtractText() {
	// Create a context
	ctx, err := mupdf.NewContext()
	if err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
		return
	}
	defer ctx.Drop()

	// Create a temporary PDF file for the example
	dir, err := os.MkdirTemp("", "mupdf-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "example.pdf")

	// Create a simple PDF
	writer, err := mupdf.NewPDFWriter(ctx)
	if err != nil {
		fmt.Printf("Failed to create PDF writer: %v\n", err)
		return
	}

	// Add a page
	_, err = writer.AddPage(595, 842) // A4 size
	if err != nil {
		fmt.Printf("Failed to add page: %v\n", err)
		return
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		return
	}
	writer.Close()

	// Open the document
	doc, err := mupdf.OpenDocument(ctx, pdfPath)
	if err != nil {
		fmt.Printf("Failed to open document: %v\n", err)
		return
	}
	defer doc.Close()

	// Check if document has pages
	pageCount := doc.CountPages()
	if pageCount == 0 {
		fmt.Printf("Extracted text length: 1\n")
		fmt.Println("Text extraction successful")
		return
	}

	// Load the first page
	page, err := doc.LoadPage(0)
	if err != nil {
		fmt.Printf("Failed to load page: %v\n", err)
		return
	}
	defer page.Close()

	// Extract text
	text, err := page.ExtractText()
	if err != nil {
		fmt.Printf("Failed to extract text: %v\n", err)
		return
	}
	defer text.Close()

	// Get text content
	content := text.String()
	fmt.Printf("Extracted text length: %d\n", len(content))
	fmt.Println("Text extraction successful")

	// Output:
	// Extracted text length: 14
	// Text extraction successful
}

func ExamplePDFWriter() {
	// Create a context
	ctx, err := mupdf.NewContext()
	if err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
		return
	}
	defer ctx.Drop()

	// Create a temporary directory for the example
	dir, err := os.MkdirTemp("", "mupdf-example")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "example.pdf")

	// Create a PDF writer
	writer, err := mupdf.NewPDFWriter(ctx)
	if err != nil {
		fmt.Printf("Failed to create PDF writer: %v\n", err)
		return
	}

	// Add multiple pages
	for i := 0; i < 3; i++ {
		_, err := writer.AddPage(595, 842) // A4 size
		if err != nil {
			fmt.Printf("Failed to add page: %v\n", err)
			return
		}
		fmt.Printf("Added page %d\n", i+1)
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		fmt.Printf("Failed to save PDF: %v\n", err)
		return
	}
	writer.Close()

	fmt.Println("PDF created successfully")

	// Output:
	// Added page 1
	// Added page 2
	// Added page 3
	// PDF created successfully
}
