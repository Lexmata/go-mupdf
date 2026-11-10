package mupdf

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// bloatedPDF writes a three-page document with plenty to squeeze: long
// uncompressed content streams, the same stream stored three times, and an
// object nothing refers to.
func bloatedPDF(t *testing.T, path string) {
	t.Helper()
	content := "BT /F1 12 Tf 72 700 Td (Compress me) Tj ET\n" +
		strings.Repeat("0.9 0.9 0.9 rg 100 100 200 200 re f\n", 400)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [4 0 R 6 0 R 8 0 R] /Count 3 >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	for i := 0; i < 3; i++ {
		objects = append(objects,
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents "+
				strconv.Itoa(5+2*i)+" 0 R >>",
			rawStream("", content),
		)
	}
	objects = append(objects, rawStream("", strings.Repeat("unreferenced padding ", 500)))
	writeRawPDF(t, path, objects)
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

func TestCompressPDF_ShrinksWithoutChangingAppearance(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	bloatedPDF(t, in)

	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}

	before, after := fileSize(t, in), fileSize(t, out)
	if after*10 > before {
		t.Errorf("output is %d bytes against %d: expected at least a tenfold saving on this fixture", after, before)
	}
	if n, err := PageCount(ctx, out); err != nil || n != 3 {
		t.Fatalf("output has %d pages (err %v), want 3", n, err)
	}
	for page := 0; page < 3; page++ {
		if d := maxChannelDiff(t, renderTestPage(t, ctx, in, page, 72), renderTestPage(t, ctx, out, page, 72)); d != 0 {
			t.Errorf("page %d renders differently after compression (max channel diff %d/255)", page+1, d)
		}
		if got := pageText(t, ctx, out, page); !strings.Contains(got, "Compress me") {
			t.Errorf("page %d lost its text: %q", page+1, got)
		}
	}
	if res, err := CheckPDF(ctx, out); err != nil || !res.Clean() {
		t.Errorf("output is not a clean PDF: %+v, err %v", res, err)
	}
}

// Lossless means images are carried through untouched. A JPEG's bytes must
// come out exactly as they went in, and an image stored uncompressed must
// decode to the same pixels.
func TestCompressPDF_DoesNotReencodeImages(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")

	// A small JPEG with real detail, so that re-encoding would change it.
	src := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			src.Set(x, y, color.RGBA{uint8(x * 8), uint8(y * 8), uint8((x ^ y) * 8), 255})
		}
	}
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, src, &jpeg.Options{Quality: 60}); err != nil {
		t.Fatal(err)
	}
	// An uncompressed 4x4 gray image.
	raw := strings.Repeat("\x00\x55\xaa\xff", 4)

	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources << /XObject << /J 5 0 R /R 6 0 R >> >> >>",
		rawStream("", "q 100 0 0 100 0 100 cm /J Do Q q 100 0 0 100 100 0 cm /R Do Q"),
		rawStream("/Type /XObject /Subtype /Image /Width 32 /Height 32 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode", jpg.String()),
		rawStream("/Type /XObject /Subtype /Image /Width 4 /Height 4 /ColorSpace /DeviceGray /BitsPerComponent 8", raw),
	})

	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, jpg.Bytes()) {
		t.Error("the JPEG's bytes are not in the output unchanged: the image was re-encoded")
	}
	if d := maxChannelDiff(t, renderTestPage(t, ctx, in, 0, 144), renderTestPage(t, ctx, out, 0, 144)); d != 0 {
		t.Errorf("the page renders differently after compression (max channel diff %d/255)", d)
	}
}

// The three pages of the fixture carry identical content streams; merging
// them is where most of the saving on a scanned, templated document comes
// from.
func TestCompressPDF_MergesIdenticalStreams(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	bloatedPDF(t, in)

	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}
	before, _ := os.ReadFile(in)
	after, _ := os.ReadFile(out)
	// In: three content streams and one unreferenced stream. Out: one
	// content stream, plus the object stream and cross-reference stream the
	// packed format itself needs.
	if n := bytes.Count(before, []byte("endstream")); n != 4 {
		t.Fatalf("fixture problem: input has %d streams, want 4", n)
	}
	if n := bytes.Count(after, []byte("endstream")); n > 3 {
		t.Errorf("output has %d streams: the three identical content streams were not merged into one", n)
	}
}

func TestCompressPDF_PreservesBookmarks(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.pdf")
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	bloatedPDF(t, plain)
	want := []OutlineItem{
		{Title: "One", Page: 0, Children: []OutlineItem{{Title: "One-A", Page: 1}}},
		{Title: "Two", Page: 2},
	}
	if err := AddBookmarks(ctx, plain, in, want); err != nil {
		t.Fatalf("AddBookmarks: %v", err)
	}

	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}
	got, err := ExtractBookmarks(ctx, out)
	if err != nil {
		t.Fatalf("ExtractBookmarks: %v", err)
	}
	if !outlineEqual(got, want) {
		t.Errorf("bookmarks changed:\ngot:\n%swant:\n%s", dumpOutline(got, 0), dumpOutline(want, 0))
	}
}

// The compressed file must be readable by other tools, object streams and
// all.
func TestCompressPDF_OutputAcceptedByOtherTools(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	bloatedPDF(t, in)
	if err := CompressPDF(ctx, in, out); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}

	if err := ValidatePDF(out, nil); err != nil {
		t.Errorf("pdfcpu rejects the compressed file: %v", err)
	}
	t.Run("qpdf", func(t *testing.T) {
		runTool(t, requireTool(t, "qpdf"), "--check", out)
	})
}

// CompressPDF replaces `cpdf -squeeze`; it should save about as much.
func TestCompressPDF_ComparableToCpdfSqueeze(t *testing.T) {
	cpdf := requireTool(t, "cpdf")
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	ours := filepath.Join(dir, "ours.pdf")
	theirs := filepath.Join(dir, "theirs.pdf")
	bloatedPDF(t, in)

	if err := CompressPDF(ctx, in, ours); err != nil {
		t.Fatalf("CompressPDF: %v", err)
	}
	runTool(t, cpdf, "-squeeze", in, "-o", theirs)

	a, b := fileSize(t, ours), fileSize(t, theirs)
	t.Logf("input %d bytes, ours %d, cpdf -squeeze %d", fileSize(t, in), a, b)
	if a > b*2 {
		t.Errorf("our output (%d bytes) is more than twice cpdf's (%d bytes)", a, b)
	}
}

func TestCompressPDF_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	bloatedPDF(t, in)
	out := filepath.Join(dir, "out.pdf")
	garbage := filepath.Join(dir, "garbage.pdf")
	writeFile(t, garbage, []byte("this is not a PDF at all"))

	tests := []struct {
		name    string
		in, out string
		wantErr string
	}{
		{"output equals input", in, in, "must differ"},
		{"empty input", "", out, "must not be empty"},
		{"empty output", in, "", "must not be empty"},
		{"missing input", filepath.Join(dir, "missing.pdf"), out, "cannot open"},
		{"unreadable input", garbage, out, "CompressPDF"},
		{"unwritable output", in, filepath.Join(dir, "no-such-dir", "out.pdf"), "CompressPDF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompressPDF(ctx, tt.in, tt.out)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
	assertNoStrayFiles(t, dir, "in.pdf", "garbage.pdf")

	// A refused in-place compression must leave the input alone.
	if n, err := PageCount(ctx, in); err != nil || n != 3 {
		t.Errorf("input damaged: %d pages, err %v", n, err)
	}
}
