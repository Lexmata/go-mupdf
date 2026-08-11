package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReorderPages_Reverse(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		p, _ := writer.AddPage(612, 792)
		p.Close()
	}
	tmp := t.TempDir()
	input := filepath.Join(tmp, "in.pdf")
	if err := writer.Save(input); err != nil {
		t.Fatalf("Save: %v", err)
	}
	writer.Close()

	output := filepath.Join(tmp, "out.pdf")
	err = ReorderPages(ctx, input, output, []int{4, 3, 2, 1, 0})
	if err != nil {
		t.Fatalf("ReorderPages: %v", err)
	}

	count, err := PageCount(ctx, output)
	if err != nil {
		t.Fatalf("PageCount: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 pages, got %d", count)
	}
}

func TestExtractBookmarks_AfterAdd(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		p, _ := writer.AddPage(612, 792)
		p.Close()
	}
	tmp := t.TempDir()
	input := filepath.Join(tmp, "in.pdf")
	if err := writer.Save(input); err != nil {
		t.Fatalf("Save: %v", err)
	}
	writer.Close()

	withBM := filepath.Join(tmp, "bm.pdf")
	items := []OutlineItem{
		{Title: "Chapter 1", Page: 0, Children: []OutlineItem{
			{Title: "Section 1.1", Page: 1},
		}},
		{Title: "Chapter 2", Page: 3},
	}
	if err := AddBookmarks(ctx, input, withBM, items); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	extracted, err := ExtractBookmarks(ctx, withBM)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if len(extracted) != 2 {
		t.Fatalf("expected 2 top-level bookmarks, got %d", len(extracted))
	}
	if extracted[0].Title != "Chapter 1" {
		t.Errorf("expected 'Chapter 1', got %q", extracted[0].Title)
	}
	info, _ := os.Stat(withBM)
	if info.Size() == 0 {
		t.Fatal("output empty")
	}
}
