package mupdf

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fixture builders shared by the tests of the ops.c-backed operations
// ---------------------------------------------------------------------------

// rawStream formats a stream object body with a correct /Length. dict is the
// extra dictionary content, without the surrounding << >>.
func rawStream(dict, data string) string {
	return fmt.Sprintf("<< %s /Length %d >>\nstream\n%s\nendstream\n", dict, len(data), data)
}

// rawPDFBytes assembles a PDF from object bodies. Object i+1 is objects[i]
// and object 1 must be the catalog. Offsets are computed, so the xref is
// always correct.
func rawPDFBytes(objects []string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")

	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s", i+1, body)
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString("endobj\n")
	}

	startxref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d %05d n \n", off, 0)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, startxref)
	return buf.Bytes()
}

func writeRawPDF(t *testing.T, path string, objects []string) {
	t.Helper()
	if err := os.WriteFile(path, rawPDFBytes(objects), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// testPage describes one page of a pagesPDF fixture.
type testPage struct {
	mediaBox string // e.g. "0 0 300 400"
	extra    string // extra page dictionary entries, e.g. "/Rotate 90"
	content  string // content stream
}

// pagesPDF writes a PDF whose pages each have their own content stream and
// share one Helvetica font as /F1. extraObjects are appended after the page
// objects; the first of them is object number 4+2*len(pages).
func pagesPDF(t *testing.T, path string, pages []testPage, extraObjects ...string) {
	t.Helper()

	kids := make([]string, len(pages))
	for i := range pages {
		kids[i] = fmt.Sprintf("%d 0 R", 4+2*i)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	for i, p := range pages {
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [%s] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R %s >>",
				p.mediaBox, 5+2*i, p.extra),
			rawStream("", p.content),
		)
	}
	objects = append(objects, extraObjects...)
	writeRawPDF(t, path, objects)
}

// The stored-space lower-left 50x50 points of a "10 20 310 420" MediaBox,
// filled black. Where it lands on screen identifies the page's rotation.
const (
	offsetMediaBox  = "10 20 310 420"
	cornerMarkBlack = "0 0 0 rg 10 20 50 50 re f"
)

func renderTestPage(t *testing.T, ctx *Context, path string, page int, dpi float64) *image.RGBA {
	t.Helper()
	img, err := openTestDocument(t, ctx, path).RenderPage(page, dpi, nil)
	if err != nil {
		t.Fatalf("RenderPage(%s, %d): %v", path, page, err)
	}
	return img
}

func isDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r < 0x4000 && g < 0x4000 && b < 0x4000
}

func isWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0xf000 && g > 0xf000 && b > 0xf000
}

// maxChannelDiff returns the largest per-channel difference between two
// images of equal size, on a 0-255 scale.
func maxChannelDiff(t *testing.T, a, b *image.RGBA) int {
	t.Helper()
	if a.Bounds() != b.Bounds() {
		t.Fatalf("image bounds differ: %v vs %v", a.Bounds(), b.Bounds())
	}
	worst := 0
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		if d < 0 {
			d = -d
		}
		if d > worst {
			worst = d
		}
	}
	return worst
}

// ---------------------------------------------------------------------------
// Tests for ops.go
// ---------------------------------------------------------------------------

// runOp takes cgo types, which a test file cannot name, so it is exercised
// through GetPageGeometry, the thinnest wrapper around it.

func TestRunOp_NilContext(t *testing.T) {
	_, err := GetPageGeometry(nil, "any.pdf")
	if err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Fatalf("want nil-context error, got %v", err)
	}
}

func TestRunOp_DroppedContext(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	path := filepath.Join(t.TempDir(), "one.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 100 100"}})
	ctx.Drop()

	_, err = GetPageGeometry(ctx, path)
	if err == nil || !strings.Contains(err.Error(), "dropped") {
		t.Fatalf("want dropped-context error, got %v", err)
	}
}

func TestRunOp_WrapsMuPDFError(t *testing.T) {
	ctx := newTestContext(t)

	_, err := GetPageGeometry(ctx, filepath.Join(t.TempDir(), "missing.pdf"))
	if err == nil {
		t.Fatal("want error for a missing file")
	}
	var mupdfErr Error
	if !errors.As(err, &mupdfErr) {
		t.Fatalf("error %v does not wrap mupdf.Error", err)
	}
	if !strings.HasPrefix(err.Error(), "go-mupdf: GetPageGeometry: ") {
		t.Fatalf("error %q lacks the operation prefix", err)
	}
}

func TestRunOp_Success(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "one.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 100 100"}})

	pages, err := GetPageGeometry(ctx, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
}

func TestCheckInputOutput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(in, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.pdf")
	if err := os.Symlink(in, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tests := []struct {
		name    string
		output  string
		inputs  []string
		wantErr string
	}{
		{"distinct paths", filepath.Join(dir, "out.pdf"), []string{in}, ""},
		{"empty output", "", []string{in}, "output path must not be empty"},
		{"empty input", filepath.Join(dir, "out.pdf"), []string{""}, "input path must not be empty"},
		{"same path", in, []string{in}, "must differ"},
		{"same path, unclean", filepath.Join(dir, ".", "in.pdf"), []string{in}, "must differ"},
		{"same file via symlink", link, []string{in}, "must differ"},
		{"second input matches", in, []string{filepath.Join(dir, "other.pdf"), in}, "must differ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkInputOutput("Op", tt.output, tt.inputs...)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSameFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if !sameFile(a, a) {
		t.Error("a file is the same as itself")
	}
	if sameFile(a, b) {
		t.Error("two distinct files reported as the same")
	}
	if !sameFile(filepath.Join(dir, "none.pdf"), filepath.Join(dir, "sub", "..", "none.pdf")) {
		t.Error("equal paths to a missing file should match textually")
	}
}

func TestTempBeside(t *testing.T) {
	out := filepath.Join("some", "dir", "result.pdf")
	a, err := tempBeside(out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := tempBeside(out)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(a) != filepath.Dir(out) {
		t.Errorf("temp %q is not beside %q, so the rename would cross directories", a, out)
	}
	if a == b {
		t.Error("two calls returned the same temporary path")
	}
	if a == out || !strings.HasPrefix(filepath.Base(a), ".") {
		t.Errorf("temp name %q should be a hidden file distinct from the output", filepath.Base(a))
	}
}

func TestResolveLimit(t *testing.T) {
	tests := []struct {
		name            string
		value, def, out int64
	}{
		{"zero means the default", 0, 100, 100},
		{"negative means no limit", -1, 100, 0},
		{"positive is used as given", 7, 100, 7},
	}
	for _, tt := range tests {
		if got := resolveLimit(tt.value, tt.def); got != tt.out {
			t.Errorf("%s: resolveLimit(%d, %d) = %d, want %d", tt.name, tt.value, tt.def, got, tt.out)
		}
	}
}

// A failed operation must leave whatever was at the output path exactly as
// it was, and nothing else behind.
func TestRunFileOp_FailureLeavesExistingOutputAlone(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	one := filepath.Join(dir, "one.pdf")
	two := filepath.Join(dir, "two.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, one, []testPage{{mediaBox: "0 0 100 100", content: "0 0 10 10 re f"}})
	pagesPDF(t, two, []testPage{
		{mediaBox: "0 0 100 100", content: "0 0 10 10 re f"},
		{mediaBox: "0 0 100 100", content: "0 0 10 10 re f"},
	})
	previous := []byte("the result of an earlier, successful run")
	writeFile(t, out, previous)

	// Fails inside MuPDF, after the documents are open: page counts differ.
	if err := OverlayPages(ctx, one, two, out, nil); err == nil {
		t.Fatal("want an error")
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the existing output is gone: %v", err)
	}
	if !bytes.Equal(got, previous) {
		t.Error("the existing output was modified by a failed operation")
	}
	assertNoStrayFiles(t, dir, "one.pdf", "two.pdf", "out.pdf")
}

// A failure that happens while content is being copied (here, a content
// budget running out part way through) is the closest thing to a failed
// save that can be provoked on demand.
func TestRunFileOp_FailureMidwayLeavesNothing(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{
		{mediaBox: "0 0 100 100", extra: "/Rotate 90", content: strings.Repeat("0 0 10 10 re f\n", 50)},
		{mediaBox: "0 0 100 100", extra: "/Rotate 90", content: strings.Repeat("0 0 10 10 re f\n", 50)},
	})

	// Enough for the first page, not for the second.
	if _, err := UprightPages(ctx, in, out, &UprightOptions{MaxContentBytes: 1000}); err == nil {
		t.Fatal("want an error")
	}
	assertNoStrayFiles(t, dir, "in.pdf")
}

func TestRunFileOp_SuccessReplacesOutput(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: "0 0 100 100", content: "0 0 10 10 re f"}})
	writeFile(t, out, []byte("stale"))

	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}
	if n, err := PageCount(ctx, out); err != nil || n != 1 {
		t.Errorf("output has %d pages (err %v), want 1", n, err)
	}
	// The output is readable by others, as a file MuPDF created directly
	// would be, not private like a typical temporary file.
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o044 == 0 {
		t.Errorf("output mode is %v: group and others cannot read it", info.Mode().Perm())
	}
	assertNoStrayFiles(t, dir, "in.pdf", "out.pdf")
}

func TestCheckInputOutput_DoesNotLeakPaths(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "tenant-4711", "upload.pdf")
	err := checkInputOutput("Op", secret, secret)
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), "tenant-4711") {
		t.Errorf("error exposes a server path: %v", err)
	}
}
