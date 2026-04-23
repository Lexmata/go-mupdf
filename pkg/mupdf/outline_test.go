package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddBookmarks_AlphabeticalOrder(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	// Create a test PDF with enough pages
	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("NewPDFWriter: %v", err)
	}
	for i := 0; i < 10; i++ {
		page, err := writer.AddPage(612, 792)
		if err != nil {
			t.Fatalf("AddPage %d: %v", i, err)
		}
		page.Close()
	}

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	if err := writer.Save(inputPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	writer.Close()

	// Bookmarks sorted alphabetically, NOT by page — this is what
	// pdfcpu rejects but mupdf should handle fine.
	items := []OutlineItem{
		{Title: "Organized Records", Page: 0, IsOpen: true, Children: []OutlineItem{
			{Title: "By Provider", Page: 0, IsOpen: true, Children: []OutlineItem{
				{Title: "ABC Hospital", Page: 5, Children: []OutlineItem{
					{Title: "2024-01-01 Progress Note", Page: 5},
					{Title: "2024-02-15 Lab Results", Page: 6},
				}},
				{Title: "St. Anthony's", Page: 1, Children: []OutlineItem{
					{Title: "2024-03-01 Orders", Page: 1},
					{Title: "2024-04-10 Discharge", Page: 2},
				}},
				{Title: "Vivo Healthcare", Page: 8, Children: []OutlineItem{
					{Title: "2024-05-01 Notes", Page: 8},
				}},
			}},
		}},
	}

	outputPath := filepath.Join(tmpDir, "output.pdf")
	if err := AddBookmarks(ctx, inputPath, outputPath, items); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	// Verify the output file exists and has content
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	// Verify we can open the output and it has pages
	doc, err := OpenDocument(ctx, outputPath)
	if err != nil {
		t.Fatalf("OpenDocument on output: %v", err)
	}
	defer doc.Close()

	if doc.CountPages() != 10 {
		t.Errorf("expected 10 pages, got %d", doc.CountPages())
	}
}

func TestAddBookmarks_EmptyItems(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	err = AddBookmarks(ctx, "nonexistent.pdf", "output.pdf", nil)
	if err == nil {
		t.Fatal("expected error for empty items")
	}
}
