package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPDFPageAddition tests adding pages to a PDF document
func TestPDFPageAddition(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

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

	// Add multiple pages with different sizes
	pageSizes := []struct {
		width  float64
		height float64
		name   string
	}{
		{595, 842, "A4"},      // A4
		{612, 792, "Letter"},  // Letter
		{1224, 792, "Ledger"}, // Ledger
		{2384, 3370, "A0"},    // A0
		{297, 420, "A6"},      // A6
	}

	for i, size := range pageSizes {
		page, err := writer.AddPage(size.width, size.height)
		if err != nil {
			t.Fatalf("Failed to add page %d (%s): %v", i, size.name, err)
		}

		if page == nil {
			t.Fatalf("Page %d (%s) is nil", i, size.name)
		}

		// Verify page bounds
		bounds := page.Bound()
		if bounds.X0 != 0 || bounds.Y0 != 0 || bounds.X1 != size.width || bounds.Y1 != size.height {
			t.Errorf("Page %d (%s) has incorrect bounds: got %+v, want {0, 0, %f, %f}",
				i, size.name, bounds, size.width, size.height)
		}
	}

	// Save the PDF
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "multisize_pages.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}

	// Open the PDF to verify
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open saved document: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	if pageCount != len(pageSizes) {
		t.Errorf("Expected %d pages, got %d", len(pageSizes), pageCount)
	}

	// Check each page's bounds
	for i := 0; i < pageCount && i < len(pageSizes); i++ {
		page, err := doc.LoadPage(i)
		if err != nil {
			t.Fatalf("Failed to load page %d: %v", i, err)
		}

		bounds := page.Bound()
		expectedWidth := pageSizes[i].width
		expectedHeight := pageSizes[i].height

		// Allow for some floating-point precision issues
		const epsilon = 0.1
		if bounds.X0 < -epsilon || bounds.Y0 < -epsilon ||
			abs(bounds.X1-expectedWidth) > epsilon ||
			abs(bounds.Y1-expectedHeight) > epsilon {
			t.Errorf("Page %d has incorrect bounds: got %+v, want {0, 0, %f, %f}",
				i, bounds, expectedWidth, expectedHeight)
		}

		page.Close()
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestPDFObjectCreationExtended tests creating various PDF objects
func TestPDFObjectCreationExtended(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

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

	// Test cases for different object types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"null", nil},
		{"bool_true", true},
		{"bool_false", false},
		{"integer_zero", 0},
		{"integer_positive", 42},
		{"integer_negative", -42},
		{"integer_large", 1000000},
		{"float_zero", 0.0},
		{"float_positive", 3.14159},
		{"float_negative", -2.71828},
		{"float_large", 1e6},
		{"float_small", 1e-6},
		{"string_empty", ""},
		{"string_simple", "Hello, World!"},
		{"string_special", "Special characters: !@#$%^&*()_+{}|:<>?"},
		{"string_long", string(make([]byte, 1000))}, // 1000-byte string
	}

	// Create each object type
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := writer.NewPDFObject(tc.value)
			if err != nil {
				t.Fatalf("Failed to create PDF object with %v: %v", tc.value, err)
			}

			if obj == nil {
				t.Fatal("PDFObject is nil")
			}

			obj.Drop()
		})
	}

	// Test invalid object type
	t.Run("invalid_type", func(t *testing.T) {
		_, err := writer.NewPDFObject([]int{1, 2, 3}) // Slices not supported
		if err == nil {
			t.Error("Expected error when creating PDF object with unsupported type")
		}
	})

	// Add a page and save the PDF to ensure objects can be created
	_, err = writer.AddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "objects_extended.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF: %v", err)
	}
}

// TestPDFMultipleWriteSave tests creating and saving a PDF multiple times
func TestPDFMultipleWriteSave(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

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

	// Create temporary directory
	dir := testDataDir(t)

	// Add a page
	_, err = writer.AddPage(595, 842)
	if err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// Save the PDF for the first time
	pdfPath1 := filepath.Join(dir, "multiple_save_1.pdf")
	err = writer.Save(pdfPath1)
	if err != nil {
		t.Fatalf("Failed to save PDF first time: %v", err)
	}

	// Add another page
	_, err = writer.AddPage(612, 792)
	if err != nil {
		t.Fatalf("Failed to add second page: %v", err)
	}

	// Save the PDF for the second time
	pdfPath2 := filepath.Join(dir, "multiple_save_2.pdf")
	err = writer.Save(pdfPath2)
	if err != nil {
		t.Fatalf("Failed to save PDF second time: %v", err)
	}

	// Verify the first PDF
	doc1, err := OpenDocument(ctx, pdfPath1)
	if err != nil {
		t.Fatalf("Failed to open first saved document: %v", err)
	}
	defer doc1.Close()

	// Check page count of first PDF
	pageCount1 := doc1.CountPages()
	if pageCount1 != 1 {
		t.Errorf("Expected 1 page in first PDF, got %d", pageCount1)
	}

	// Verify the second PDF
	doc2, err := OpenDocument(ctx, pdfPath2)
	if err != nil {
		t.Fatalf("Failed to open second saved document: %v", err)
	}
	defer doc2.Close()

	// Check page count of second PDF
	pageCount2 := doc2.CountPages()
	if pageCount2 != 2 {
		t.Errorf("Expected 2 pages in second PDF, got %d", pageCount2)
	}
}

// TestPDFOverwrite tests overwriting an existing PDF file
func TestPDFOverwrite(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create temporary directory
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "overwrite.pdf")

	// Create first PDF with 1 page
	{
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create first PDF writer: %v", err)
		}

		// Add a page
		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add page to first PDF: %v", err)
		}

		// Save the first PDF
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save first PDF: %v", err)
		}
		writer.Close()
	}

	// Verify the first PDF
	doc1, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open first PDF: %v", err)
	}
	pageCount1 := doc1.CountPages()
	doc1.Close()

	if pageCount1 != 1 {
		t.Errorf("Expected 1 page in first PDF, got %d", pageCount1)
	}

	// Create second PDF with 2 pages, overwriting the first
	{
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create second PDF writer: %v", err)
		}

		// Add two pages
		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add first page to second PDF: %v", err)
		}

		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add second page to second PDF: %v", err)
		}

		// Save the second PDF, overwriting the first
		err = writer.Save(pdfPath)
		if err != nil {
			t.Fatalf("Failed to save second PDF: %v", err)
		}
		writer.Close()
	}

	// Verify the second PDF
	doc2, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open second PDF: %v", err)
	}
	defer doc2.Close()

	pageCount2 := doc2.CountPages()
	if pageCount2 != 2 {
		t.Errorf("Expected 2 pages in second PDF, got %d", pageCount2)
	}
}

// TestPDFSaveToNewlyCreatedDirectory tests saving a PDF to a directory created
// just before the save. The genuine missing-directory error case is covered by
// memory_test.go and types_test.go (SaveToInvalidLocation).
func TestPDFSaveToNewlyCreatedDirectory(t *testing.T) {
	requireMuPDF(t)

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

	// Save into a directory created immediately before the save
	dir := testDataDir(t)
	newDir := filepath.Join(dir, "newly_created_directory")
	pdfPath := filepath.Join(newDir, "test.pdf")

	// Create the directory
	err = os.MkdirAll(newDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Save the PDF
	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save PDF to newly created directory: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}
}

// TestPDFEmptyDocument tests creating an empty PDF document
func TestPDFEmptyDocument(t *testing.T) {
	requireMuPDF(t)

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

	// Save the PDF without adding any pages
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "empty.pdf")

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save empty PDF: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file was not created at %s", pdfPath)
	}

	// Open the PDF to verify
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open empty PDF: %v", err)
	}
	defer doc.Close()

	// Check page count
	pageCount := doc.CountPages()
	if pageCount != 0 {
		t.Errorf("Expected 0 pages in empty PDF, got %d", pageCount)
	}
}
