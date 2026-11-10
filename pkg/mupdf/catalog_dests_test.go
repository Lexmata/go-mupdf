package mupdf

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// destsPDF writes a one-page PDF whose catalog carries the given extra
// entries. Object 4 is free for the caller's use via extraObjects.
func destsPDF(t *testing.T, path, catalogExtra string, extraObjects ...string) {
	t.Helper()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R " + catalogExtra + " >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>",
	}
	writeRawPDF(t, path, append(objects, extraObjects...))
}

const (
	// A name-tree node: what belongs under /Names /Dests.
	nameTreeLeaf = "<< /Names [(chapter1) [3 0 R /Fit] (chapter2) [3 0 R /Fit]] >>"
	// A flat PDF 1.1 destination dictionary: what belongs in catalog /Dests.
	flatDests = "<< /chapter1 [3 0 R /Fit] /chapter2 << /D [3 0 R /Fit] >> >>"
)

// catalogOf returns the text of the catalog dictionary in an uncompressed
// PDF written by MuPDF.
func catalogOf(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)\d+ 0 obj\n(<<[^\n]*/Type\s*/Catalog.*?)\nendobj`).FindSubmatch(data)
	if m == nil {
		t.Fatalf("no catalog found in %s", path)
	}
	return string(m[1])
}

// withoutNamesDict returns a catalog's text with a direct /Names << ... >>
// sub-dictionary cut out, so what remains is the catalog's own entries.
func withoutNamesDict(catalog string) string {
	start := strings.Index(catalog, "/Names<<")
	if start < 0 {
		return catalog
	}
	depth := 0
	for i := start + len("/Names"); i+1 < len(catalog); i++ {
		switch catalog[i : i+2] {
		case "<<":
			depth++
			i++
		case ">>":
			depth--
			i++
			if depth == 0 {
				return catalog[:start] + catalog[i+1:]
			}
		}
	}
	return catalog
}

func TestWithoutNamesDict(t *testing.T) {
	tests := map[string]string{
		"<</Type/Catalog/Pages 2 0 R>>":                               "<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Catalog/Names 4 0 R/Dests 5 0 R>>":                   "<</Type/Catalog/Names 4 0 R/Dests 5 0 R>>",
		"<</Type/Catalog/Names<</Dests 4 0 R>>/Pages 2 0 R>>":         "<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Catalog/Names<</A<</B[]>>/Dests<</Names[]>>>>/X 1>>": "<</Type/Catalog/X 1>>",
	}
	for in, want := range tests {
		if got := withoutNamesDict(in); got != want {
			t.Errorf("withoutNamesDict(%q) = %q, want %q", in, got, want)
		}
	}
}

// countInLimits counts appearances of (name) inside /Limits arrays, which
// repeat a node's first and last keys.
func countInLimits(data []byte, name string) int {
	n := 0
	for _, m := range regexp.MustCompile(`/Limits\s*\[[^\]]*\]`).FindAll(data, -1) {
		n += bytes.Count(m, []byte("("+name+")"))
	}
	return n
}

// sortedDestNames reports whether the names appear in the output in the
// order given, which the caller supplies sorted.
func sortedDestNames(t *testing.T, data []byte, names []string) bool {
	t.Helper()
	last := -1
	for _, name := range names {
		i := bytes.Index(data, []byte("("+name+")"))
		if i < 0 || i < last {
			return false
		}
		last = i
	}
	return true
}

func TestRepairCatalogDests(t *testing.T) {
	tests := []struct {
		name         string
		catalogExtra string
		extraObjects []string
		want         CatalogDestsRepair
		// wantNames, when set, are the destination names the output must
		// hold, each exactly once.
		wantNames []string
	}{
		{
			name:         "direct name tree, no /Names: moved",
			catalogExtra: "/Dests " + nameTreeLeaf,
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "indirect name tree, no /Names: moved",
			catalogExtra: "/Dests 4 0 R",
			extraObjects: []string{nameTreeLeaf},
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "name tree with /Kids: moved",
			catalogExtra: "/Dests << /Kids [4 0 R] >>",
			extraObjects: []string{"<< /Limits [(chapter1) (chapter2)] /Names [(chapter1) [3 0 R /Fit] (chapter2) [3 0 R /Fit]] >>"},
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "direct /Names without /Dests: moved into it",
			catalogExtra: "/Dests " + nameTreeLeaf + " /Names << /EmbeddedFiles << /Names [] >> >>",
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "indirect /Names without /Dests: moved into it",
			catalogExtra: "/Dests " + nameTreeLeaf + " /Names 4 0 R",
			extraObjects: []string{"<< /EmbeddedFiles << /Names [] >> >>"},
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "/Names is not a dictionary: replaced",
			catalogExtra: "/Dests " + nameTreeLeaf + " /Names 42",
			want:         CatalogDestsMovedToNames,
		},
		{
			name:         "/Names /Dests already present: duplicate removed",
			catalogExtra: "/Dests " + nameTreeLeaf + " /Names << /Dests 4 0 R >>",
			extraObjects: []string{nameTreeLeaf},
			want:         CatalogDestsRemovedDuplicate,
		},
		{
			name:         "indirect /Names with /Dests: duplicate removed",
			catalogExtra: "/Dests " + nameTreeLeaf + " /Names 4 0 R",
			extraObjects: []string{"<< /Dests " + nameTreeLeaf + " >>"},
			want:         CatalogDestsRemovedDuplicate,
		},
		{
			name:         "both trees, catalog has a name of its own: merged",
			catalogExtra: "/Dests << /Names [(chapter1) [3 0 R /Fit] (overlayonly) [3 0 R /FitH 100]] >> /Names << /Dests 4 0 R >>",
			extraObjects: []string{"<< /Names [(chapter1) [3 0 R /Fit] (chapter2) [3 0 R /Fit]] >>"},
			want:         CatalogDestsMerged,
			wantNames:    []string{"chapter1", "chapter2", "overlayonly"},
		},
		{
			name:         "both trees, nothing in common: merged",
			catalogExtra: "/Dests << /Names [(aaa) [3 0 R /Fit] (zzz) [3 0 R /Fit]] >> /Names << /Dests << /Kids [4 0 R] >> >>",
			extraObjects: []string{"<< /Limits [(mmm) (mmm)] /Names [(mmm) [3 0 R /Fit]] >>"},
			want:         CatalogDestsMerged,
			wantNames:    []string{"aaa", "mmm", "zzz"},
		},
		{
			name:         "catalog tree is a subset laid out differently: duplicate removed",
			catalogExtra: "/Dests << /Kids [5 0 R] >> /Names << /Dests 4 0 R >>",
			extraObjects: []string{
				"<< /Names [(chapter1) [3 0 R /Fit] (chapter2) [3 0 R /Fit]] >>",
				"<< /Limits [(chapter2) (chapter2)] /Names [(chapter2) [3 0 R /Fit]] >>",
			},
			want:      CatalogDestsRemovedDuplicate,
			wantNames: []string{"chapter1", "chapter2"},
		},
		{
			name:         "no /Dests at all",
			catalogExtra: "",
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "null /Dests",
			catalogExtra: "/Dests null",
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "genuine flat PDF 1.1 /Dests",
			catalogExtra: "/Dests " + flatDests,
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "flat /Dests with destinations named Kids and Names",
			catalogExtra: "/Dests << /Kids [3 0 R /Fit] /Names [3 0 R /Fit] >>",
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "name-tree keys plus a foreign key",
			catalogExtra: "/Dests << /Names [(a) [3 0 R /Fit]] /chapter1 [3 0 R /Fit] >>",
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "odd-length /Names array",
			catalogExtra: "/Dests << /Names [(a) [3 0 R /Fit] (b)] >>",
			want:         CatalogDestsUnchanged,
		},
		{
			name:         "only /Limits",
			catalogExtra: "/Dests << /Limits [(a) (b)] >>",
			want:         CatalogDestsUnchanged,
		},
	}

	ctx := newTestContext(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "in.pdf")
			out := filepath.Join(dir, "out.pdf")
			destsPDF(t, in, tt.catalogExtra, tt.extraObjects...)

			got, err := RepairCatalogDests(ctx, in, out)
			if err != nil {
				t.Fatalf("RepairCatalogDests: %v", err)
			}
			if got != tt.want {
				t.Fatalf("outcome %v, want %v", got, tt.want)
			}

			// The output is written whatever the outcome, and is a sound
			// PDF with the page it started with.
			if n, err := PageCount(ctx, out); err != nil || n != 1 {
				t.Fatalf("output has %d pages (err %v), want 1", n, err)
			}
			if res, err := CheckPDF(ctx, out); err != nil || !res.Clean() {
				t.Errorf("output is not a clean PDF: %+v, err %v", res, err)
			}
			if tt.want == CatalogDestsUnchanged {
				// Nothing was wrong, so nothing may have moved: a genuine
				// flat /Dests stays in the catalog.
				if strings.Contains(tt.catalogExtra, "/Dests <<") && !strings.Contains(withoutNamesDict(catalogOf(t, out)), "/Dests") {
					t.Error("a catalog /Dests that needed no repair was removed")
				}
				return
			}

			catalog := catalogOf(t, out)
			if strings.Contains(withoutNamesDict(catalog), "/Dests") {
				t.Errorf("catalog still has its own /Dests: %s", catalog)
			}
			data, _ := os.ReadFile(out)
			wantNames := tt.wantNames
			if wantNames == nil {
				wantNames = []string{"chapter1"}
			}
			for _, name := range wantNames {
				if n := bytes.Count(data, []byte("("+name+")")); n < 1 {
					t.Errorf("destination %q was lost", name)
				} else if tt.wantNames != nil && n != countInLimits(data, name)+1 {
					t.Errorf("destination %q appears %d times as a key, want once", name, n-countInLimits(data, name))
				}
			}
			if tt.want == CatalogDestsMerged {
				// A name tree must list its keys in order.
				if !sortedDestNames(t, data, wantNames) {
					t.Errorf("merged names are not in sorted order in the output")
				}
				if err := ValidatePDF(out, nil); err != nil {
					t.Errorf("pdfcpu rejects the merged file: %v", err)
				}
			}
			if !strings.Contains(catalog, "/Names") {
				t.Errorf("catalog has no /Names after the repair: %s", catalog)
			}
			// A second pass finds nothing left to do.
			again, err := RepairCatalogDests(ctx, out, filepath.Join(dir, "again.pdf"))
			if err != nil {
				t.Fatalf("second pass: %v", err)
			}
			if again != CatalogDestsUnchanged {
				t.Errorf("second pass outcome %v, want unchanged", again)
			}
		})
	}
}

// The defect this repairs is what makes pdfcpu reject a file (OCR-92).
func TestRepairCatalogDests_MakesFileAcceptableToPdfcpu(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	destsPDF(t, in, "/Dests << /Kids [4 0 R] >> /Names << /Dests << /Kids [4 0 R] >> >>",
		"<< /Limits [(chapter1) (chapter2)] /Names [(chapter1) [3 0 R /Fit] (chapter2) [3 0 R /Fit]] >>")

	if err := ValidatePDF(in, nil); err == nil {
		t.Fatal("fixture problem: pdfcpu accepts the unrepaired file, so this test proves nothing")
	}
	got, err := RepairCatalogDests(ctx, in, out)
	if err != nil {
		t.Fatalf("RepairCatalogDests: %v", err)
	}
	if got != CatalogDestsRemovedDuplicate {
		t.Fatalf("outcome %v, want removed duplicate", got)
	}
	if err := ValidatePDF(out, nil); err != nil {
		t.Errorf("pdfcpu still rejects the repaired file: %v", err)
	}
}

func TestRepairCatalogDests_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	destsPDF(t, in, "/Dests "+nameTreeLeaf)
	out := filepath.Join(dir, "out.pdf")
	noCatalog := filepath.Join(dir, "nocatalog.pdf")
	writeRawPDF(t, noCatalog, []string{"42", "<< /Type /Pages /Kids [] /Count 0 >>"})

	tests := []struct {
		name    string
		in, out string
		wantErr string
	}{
		{"output equals input", in, in, "must differ"},
		{"empty input", "", out, "must not be empty"},
		{"empty output", in, "", "must not be empty"},
		{"missing input", filepath.Join(dir, "missing.pdf"), out, "cannot open"},
		{"unwritable output", in, filepath.Join(dir, "no-such-dir", "out.pdf"), "RepairCatalogDests"},
		{"catalog is not a dictionary", noCatalog, out, "RepairCatalogDests"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RepairCatalogDests(ctx, tt.in, tt.out)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tt.wantErr, err)
			}
			if got != CatalogDestsUnchanged {
				t.Errorf("outcome %v alongside an error, want unchanged", got)
			}
		})
	}

	// None of the refused calls may have touched the input or left
	// anything behind.
	if n, err := PageCount(ctx, in); err != nil || n != 1 {
		t.Errorf("input damaged: %d pages, err %v", n, err)
	}
	assertNoStrayFiles(t, dir, "in.pdf", "nocatalog.pdf")
}

func TestCatalogDestsRepair_String(t *testing.T) {
	tests := map[CatalogDestsRepair]string{
		CatalogDestsUnchanged:        "unchanged",
		CatalogDestsRemovedDuplicate: "removed duplicate",
		CatalogDestsMovedToNames:     "moved to /Names",
		CatalogDestsMerged:           "merged into /Names",
		CatalogDestsRepair(99):       "unknown",
	}
	for r, want := range tests {
		if got := r.String(); got != want {
			t.Errorf("CatalogDestsRepair(%d).String() = %q, want %q", int(r), got, want)
		}
	}
}
