package mupdf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// buildTestPDF writes an empty pageCount-page PDF at path.
func buildTestPDF(t *testing.T, ctx *Context, path string, pageCount int) {
	t.Helper()
	w, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("NewPDFWriter: %v", err)
	}
	for i := 0; i < pageCount; i++ {
		p, err := w.AddPage(612, 792)
		if err != nil {
			t.Fatalf("AddPage %d: %v", i, err)
		}
		p.Close()
	}
	if err := w.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	w.Close()
}

// outlineEqual compares two outline trees, ignoring IsOpen (mupdf writes
// all childless nodes as closed, and parents with children as open,
// regardless of what we requested).
func outlineEqual(a, b []OutlineItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Page != b[i].Page {
			return false
		}
		if !outlineEqual(a[i].Children, b[i].Children) {
			return false
		}
	}
	return true
}

func dumpOutline(items []OutlineItem, depth int) string {
	out := ""
	for _, it := range items {
		out += fmt.Sprintf("%s%q (page=%d)\n", indent(depth), it.Title, it.Page)
		out += dumpOutline(it.Children, depth+1)
	}
	return out
}

func indent(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "  "
	}
	return s
}

func TestAddBookmarks_AlphabeticalOrder(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	buildTestPDF(t, ctx, inputPath, 10)

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

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	doc, err := OpenDocument(ctx, outputPath)
	if err != nil {
		t.Fatalf("OpenDocument on output: %v", err)
	}
	defer doc.Close()
	if doc.CountPages() != 10 {
		t.Errorf("expected 10 pages, got %d", doc.CountPages())
	}

	got, err := ExtractBookmarks(ctx, outputPath)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if !outlineEqual(items, got) {
		t.Errorf("outline mismatch\n--- want ---\n%s\n--- got ---\n%s",
			dumpOutline(items, 0), dumpOutline(got, 0))
	}
}

// TestAddBookmarks_NestedChildrenRoundTrip is the regression test for
// MPDF-99: insertOutlineItems used to drop children and subsequent
// siblings because fz_outline_iterator_insert auto-advances and the old
// code didn't account for that. Every node below the root was lost.
func TestAddBookmarks_NestedChildrenRoundTrip(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "in.pdf")
	buildTestPDF(t, ctx, inputPath, 6)

	items := []OutlineItem{
		{Title: "Root A", Page: 0, IsOpen: true, Children: []OutlineItem{
			{Title: "A child 1", Page: 1, IsOpen: true, Children: []OutlineItem{
				{Title: "A grandchild 1", Page: 2},
				{Title: "A grandchild 2", Page: 3},
			}},
			{Title: "A child 2", Page: 4},
		}},
		{Title: "Root B", Page: 5},
	}

	outputPath := filepath.Join(tmpDir, "out.pdf")
	if err := AddBookmarks(ctx, inputPath, outputPath, items); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	got, err := ExtractBookmarks(ctx, outputPath)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}

	if !outlineEqual(items, got) {
		t.Errorf("outline mismatch\n--- want ---\n%s\n--- got ---\n%s",
			dumpOutline(items, 0), dumpOutline(got, 0))
	}
}

// TestAddBookmarks_MultipleTopLevelSiblings covers the sibling-skip bug
// specifically: with the old implementation the explicit _next after an
// _insert skipped every other slot, so only the first top-level sibling
// was persisted.
func TestAddBookmarks_MultipleTopLevelSiblings(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "in.pdf")
	buildTestPDF(t, ctx, inputPath, 5)

	items := []OutlineItem{
		{Title: "One", Page: 0},
		{Title: "Two", Page: 1},
		{Title: "Three", Page: 2},
		{Title: "Four", Page: 3},
		{Title: "Five", Page: 4},
	}

	outputPath := filepath.Join(tmpDir, "out.pdf")
	if err := AddBookmarks(ctx, inputPath, outputPath, items); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	got, err := ExtractBookmarks(ctx, outputPath)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if len(got) != len(items) {
		t.Fatalf("top-level sibling count = %d, want %d\n%s",
			len(got), len(items), dumpOutline(got, 0))
	}
	if !outlineEqual(items, got) {
		t.Errorf("outline mismatch\n--- want ---\n%s\n--- got ---\n%s",
			dumpOutline(items, 0), dumpOutline(got, 0))
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
