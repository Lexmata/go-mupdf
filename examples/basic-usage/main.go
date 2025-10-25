package main

import (
	"fmt"
	"log"

	"bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)

func main() {
	fmt.Println("Go MuPDF Basic Usage Example")
	fmt.Println("============================")

	// Create a new context
	ctx, err := mupdf.NewContext()
	if err != nil {
		log.Fatal("Failed to create context:", err)
	}
	defer ctx.Drop()

	fmt.Println("✅ MuPDF context created successfully")

	// Example: Create a simple PDF
	createSimplePDF(ctx)

	// If you have a PDF file, you can also read it
	// readExistingPDF(ctx, "example.pdf")

	fmt.Println("🎉 Example completed successfully!")
}

func createSimplePDF(ctx *mupdf.Context) {
	fmt.Println("\n📄 Creating a simple PDF...")

	// Create a new PDF document
	doc, err := mupdf.CreatePDF(ctx)
	if err != nil {
		log.Fatal("Failed to create PDF:", err)
	}
	defer doc.Close()

	// Add a page with some text
	page, err := doc.AddPage(mupdf.USLetter)
	if err != nil {
		log.Fatal("Failed to add page:", err)
	}
	defer page.Close()

	fmt.Println("✅ Created PDF with one page")
	fmt.Printf("   Page size: %.1f x %.1f\n", mupdf.USLetter.Width, mupdf.USLetter.Height)

	// Save the document (you would typically save to a file)
	// doc.Save("output.pdf") // This method would need to be implemented
	
	fmt.Println("✅ PDF creation example completed")
}

func readExistingPDF(ctx *mupdf.Context, filename string) {
	fmt.Printf("\n📖 Reading PDF: %s\n", filename)

	// Open the document
	doc, err := mupdf.OpenDocument(ctx, filename)
	if err != nil {
		log.Printf("Failed to open PDF (this is expected if file doesn't exist): %v", err)
		return
	}
	defer doc.Close()

	// Get page count
	pageCount := doc.CountPages()
	fmt.Printf("✅ Document has %d pages\n", pageCount)

	// Read first page if it exists
	if pageCount > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			log.Printf("Failed to load page: %v", err)
			return
		}
		defer page.Close()

		// Extract text
		textPage, err := page.ExtractText()
		if err != nil {
			log.Printf("Failed to extract text: %v", err)
			return
		}
		defer textPage.Close()

		text := textPage.String()
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		
		fmt.Printf("✅ Extracted text preview: %s\n", text)
	}
}