package mupdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// TestCountItems guards the accounting AddBookmarks relies on to detect a
// silently-truncated outline: countItems must total every node in the tree,
// including nested children, so a failed insert can be reported as inserted !=
// requested rather than saved as a partial outline.
func TestCountItems(t *testing.T) {
	cases := []struct {
		name  string
		items []OutlineItem
		want  int
	}{
		{"nil", nil, 0},
		{"flat", []OutlineItem{{Title: "a"}, {Title: "b"}}, 2},
		{
			"nested",
			[]OutlineItem{
				{Title: "root", Children: []OutlineItem{
					{Title: "c1"},
					{Title: "c2", Children: []OutlineItem{{Title: "g1"}}},
				}},
				{Title: "sibling"},
			},
			5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := countItems(tc.items); got != tc.want {
				t.Errorf("countItems = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRemoveBookmarks(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.pdf")
	marked := filepath.Join(dir, "marked.pdf")
	out := filepath.Join(dir, "out.pdf")
	buildTestPDF(t, ctx, plain, 4)
	items := []OutlineItem{
		{Title: "Part One", Page: 0, Children: []OutlineItem{
			{Title: "Chapter 1", Page: 1},
			{Title: "Chapter 2", Page: 2},
		}},
		{Title: "Part Two", Page: 3},
	}
	if err := AddBookmarks(ctx, plain, marked, items); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	if err := RemoveBookmarks(ctx, marked, out); err != nil {
		t.Fatalf("RemoveBookmarks: %v", err)
	}

	got, err := ExtractBookmarks(ctx, out)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("bookmarks survived:\n%s", dumpOutline(got, 0))
	}
	if n, err := PageCount(ctx, out); err != nil || n != 4 {
		t.Errorf("output has %d pages (err %v), want 4", n, err)
	}

	// The outline items themselves are gone from the file, not just
	// unlinked from the catalog.
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Part One", "Chapter 1", "Part Two"} {
		if bytes.Contains(data, []byte(title)) {
			t.Errorf("outline item %q is still stored in the output", title)
		}
	}
	if bytes.Contains(data, []byte("/Outlines")) {
		t.Error("the catalog still refers to /Outlines")
	}

	// New bookmarks can be added to the stripped file: the remove-then-add
	// sequence callers use to replace an outline.
	replaced := filepath.Join(dir, "replaced.pdf")
	fresh := []OutlineItem{{Title: "Only", Page: 2}}
	if err := AddBookmarks(ctx, out, replaced, fresh); err != nil {
		t.Fatalf("AddBookmarks after RemoveBookmarks: %v", err)
	}
	got, err = ExtractBookmarks(ctx, replaced)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if !outlineEqual(got, fresh) {
		t.Errorf("replacement outline is wrong:\n%s", dumpOutline(got, 0))
	}
}

func TestRemoveBookmarks_NoOutline(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	buildTestPDF(t, ctx, in, 2)

	if err := RemoveBookmarks(ctx, in, out); err != nil {
		t.Fatalf("RemoveBookmarks on a PDF without bookmarks: %v", err)
	}
	if n, err := PageCount(ctx, out); err != nil || n != 2 {
		t.Errorf("output has %d pages (err %v), want 2", n, err)
	}
}

func TestRemoveBookmarks_Errors(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	defer ctx.Drop()

	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	buildTestPDF(t, ctx, in, 1)
	out := filepath.Join(dir, "out.pdf")

	tests := []struct {
		name    string
		in, out string
		wantErr string
	}{
		{"output equals input", in, in, "must differ"},
		{"empty input", "", out, "must not be empty"},
		{"empty output", in, "", "must not be empty"},
		{"missing input", filepath.Join(dir, "missing.pdf"), out, "cannot open"},
		{"unwritable output", in, filepath.Join(dir, "no-such-dir", "out.pdf"), "RemoveBookmarks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveBookmarks(ctx, tt.in, tt.out)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tt.wantErr, err)
			}
		})
	}

	if n, err := PageCount(ctx, in); err != nil || n != 1 {
		t.Errorf("input damaged: %d pages, err %v", n, err)
	}
	assertNoStrayFiles(t, dir, "in.pdf")
}

// A document set to open with the bookmarks panel showing must not open
// showing an empty one.
func TestRemoveBookmarks_ClearsUseOutlinesPageMode(t *testing.T) {
	ctx := newTestContext(t)
	for mode, wantKept := range map[string]bool{"UseOutlines": false, "UseThumbs": true} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "in.pdf")
			out := filepath.Join(dir, "out.pdf")
			writeRawPDF(t, in, []string{
				"<< /Type /Catalog /Pages 2 0 R /Outlines 4 0 R /PageMode /" + mode + " >>",
				"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
				"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>",
				"<< /Type /Outlines /First 5 0 R /Last 5 0 R /Count 1 >>",
				"<< /Title (Only) /Parent 4 0 R /Dest [3 0 R /Fit] >>",
			})

			if err := RemoveBookmarks(ctx, in, out); err != nil {
				t.Fatalf("RemoveBookmarks: %v", err)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(data, []byte("/Outlines")) && !bytes.Contains(data, []byte("/UseOutlines")) {
				t.Error("the catalog still refers to /Outlines")
			}
			if kept := bytes.Contains(data, []byte("/PageMode")); kept != wantKept {
				t.Errorf("/PageMode kept = %v, want %v", kept, wantKept)
			}
			if bytes.Contains(data, []byte("/UseOutlines")) {
				t.Error("/PageMode /UseOutlines survived the removal of the outline")
			}
		})
	}
}
