package mupdf

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	overlayBox = "0 0 200 200"
	// A black square in the lower-left quadrant (the "scan").
	baseContent = "0 0 0 rg 20 20 60 60 re f"
	// A black square in the upper-right quadrant plus text (the "layer").
	layerContent = "0 0 0 rg 120 120 60 60 re f BT /F1 12 Tf 20 150 Td (LAYERTEXT) Tj ET"
)

// lowerLeft and upperRight are pixel positions inside the two squares when
// an overlayBox page is rendered at 72 DPI.
var (
	lowerLeft  = image.Point{50, 150}
	upperRight = image.Point{150, 50}
)

func TestOverlayPages_CompositesAtIdentity(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: baseContent}})
	pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: layerContent}})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}

	img := renderTestPage(t, ctx, out, 0, 72)
	if !isDark(img.At(lowerLeft.X, lowerLeft.Y)) {
		t.Error("the input page's own content is gone")
	}
	if !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("the overlay was not drawn where its coordinates say")
	}
	if !isWhite(img.At(150, 150)) || !isWhite(img.At(50, 50)) {
		t.Error("the other two quadrants should be untouched")
	}
	if got := pageText(t, ctx, out, 0); !strings.Contains(got, "LAYERTEXT") {
		t.Errorf("overlay text is not extractable: %q", got)
	}
	if res, err := CheckPDF(ctx, out); err != nil || !res.Clean() {
		t.Errorf("output is not a clean PDF: %+v, err %v", res, err)
	}
	if err := ValidatePDF(out, nil); err != nil {
		t.Errorf("pdfcpu rejects the output: %v", err)
	}
	// Well-formed content stays in the page's own stream, where a tagged
	// PDF's structure tree expects to find it.
	if data, _ := os.ReadFile(out); bytes.Contains(data, []byte("GoMuBase")) {
		t.Error("well-formed page content was moved into a form XObject")
	}

	// The input page's geometry is left alone.
	geom, err := GetPageGeometry(ctx, out)
	if err != nil {
		t.Fatal(err)
	}
	if want := (PageGeometry{MediaBox: Rect{0, 0, 200, 200}, HasMediaBox: true}); geom[0] != want {
		t.Errorf("geometry changed to %+v", geom[0])
	}
}

// The input's content can leave any graphics state behind. Here it ends
// with a transform and a white fill colour still in force; neither may reach
// the overlay.
func TestOverlayPages_IsolatesInputGraphicsState(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{
		mediaBox: overlayBox,
		content:  baseContent + " 0.25 0 0 0.25 500 500 cm 1 1 1 rg",
	}})
	pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: "120 120 60 60 re f"}})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("the overlay was displaced or recoloured by the input's leftover graphics state")
	}
}

// Real content streams are not always balanced. Whatever q or Q operators
// the input has too many of, the overlay must still land where its own
// coordinates say and in its own colour.
func TestOverlayPages_UnbalancedInputContent(t *testing.T) {
	const displace = "0.25 0 0 0.25 500 500 cm 1 1 1 rg"
	tests := map[string]string{
		"q left open":                  "q " + displace,
		"transform then q left open":   displace + " q",
		"three q left open":            "q q " + displace + " q",
		"stray Q then state change":    "Q " + displace,
		"two stray Q then q left open": "Q Q " + displace + " q " + displace,
		"balanced inner pair":          "q " + displace + " Q " + displace,
		"q inside a text object":       "BT /F1 12 Tf (x) Tj ET q " + displace,
		"inline image then q":          "q 10 0 0 10 20 20 cm BI /W 1 /H 1 /CS /G /BPC 8 ID \x00 EI Q q " + displace,
	}

	ctx := newTestContext(t)
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "in.pdf")
			layer := filepath.Join(dir, "layer.pdf")
			out := filepath.Join(dir, "out.pdf")
			pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: content}})
			// No colour is set, so the overlay relies on the default black.
			pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: "120 120 60 60 re f"}})

			if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
				t.Fatalf("OverlayPages: %v", err)
			}
			img := renderTestPage(t, ctx, out, 0, 72)
			if !isDark(img.At(upperRight.X, upperRight.Y)) {
				t.Error("the overlay was displaced or recoloured by the input's unbalanced content")
			}
			if !isWhite(img.At(lowerLeft.X, lowerLeft.Y)) {
				t.Error("something was drawn where nothing should be")
			}
		})
	}
}

// Content can leave more than the graphics state open. A viewer reads the
// /Contents array as one stream, so an open text object, open marked
// content, or an unterminated string or inline image would take the overlay
// down with it. Such a page's content is sealed in a form XObject instead.
func TestOverlayPages_IsolatesMalformedInputContent(t *testing.T) {
	tests := map[string]string{
		"text object left open":        baseContent + " BT /F1 12 Tf 10 10 Td (x) Tj",
		"marked content left open":     baseContent + " /Span << /MCID 0 >> BDC",
		"two levels of marked content": baseContent + " /P BMC /Span BMC",
		"unterminated string":          baseContent + " BT /F1 12 Tf (never closed",
		"unterminated hex string":      baseContent + " BT /F1 12 Tf <48656c6c6f",
		"inline image with no end":     baseContent + " q 10 0 0 10 0 0 cm BI /W 1 /H 1 /CS /G /BPC 8 ID \x00\x00\x00",
		"unterminated dictionary":      baseContent + " /Span << /MCID 0 /Lang (en",
	}

	ctx := newTestContext(t)
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "in.pdf")
			layer := filepath.Join(dir, "layer.pdf")
			out := filepath.Join(dir, "out.pdf")
			pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: content}})
			pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: layerContent}})

			if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
				t.Fatalf("OverlayPages: %v", err)
			}
			img := renderTestPage(t, ctx, out, 0, 72)
			if !isDark(img.At(upperRight.X, upperRight.Y)) {
				t.Error("the overlay was swallowed by the input page's malformed content")
			}
			if !isDark(img.At(lowerLeft.X, lowerLeft.Y)) {
				t.Error("the input page's own content is gone")
			}
			if got := pageText(t, ctx, out, 0); !strings.Contains(got, "LAYERTEXT") {
				t.Errorf("overlay text is not extractable: %q", got)
			}
			if res, err := CheckPDF(ctx, out); err != nil {
				t.Errorf("output cannot be read: %+v, err %v", res, err)
			}
			// Each of these must have been caught and sealed away; if one
			// were left in place the checks above would be passing by luck.
			if data, _ := os.ReadFile(out); !bytes.Contains(data, []byte("/GoMuBase0")) {
				t.Error("malformed content was left in the page's own stream")
			}
		})
	}
}

// The overlay's own content is decoded and copied into the output. A small
// file can list one stream thousands of times; the budget stops that
// turning into gigabytes.
func TestOverlayPages_ContentBudget(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: baseContent}})

	// One 1 kB stream, listed 5000 times: 5 MB once decoded.
	const repeats = 5000
	refs := strings.Repeat("4 0 R ", repeats)
	writeRawPDF(t, layer, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + overlayBox + "] /Contents [" + refs + "] >>",
		rawStream("", strings.Repeat("0 0 0 rg 120 120 60 60 re f\n", 36)),
	})
	if size := fileSize(t, layer); size > 100_000 {
		t.Fatalf("fixture problem: the overlay file is %d bytes, it should be small", size)
	}

	err := OverlayPages(ctx, in, layer, out, &OverlayOptions{MaxContentBytes: 1 << 20})
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want a limit error with a 1 MiB budget, got %v", err)
	}
	var mupdfErr Error
	if !errors.As(err, &mupdfErr) {
		t.Errorf("error %v does not wrap mupdf.Error", err)
	}
	assertNoStrayFiles(t, dir, "in.pdf", "layer.pdf")

	// The same file is fine under the default budget, and with none.
	for name, opts := range map[string]*OverlayOptions{"default": nil, "unlimited": {MaxContentBytes: -1}} {
		if err := OverlayPages(ctx, in, layer, out, opts); err != nil {
			t.Errorf("%s budget: %v", name, err)
		}
	}
}

// An input page with no /Contents at all still gets its overlay.
func TestOverlayPages_InputPageWithoutContents(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + overlayBox + "] >>",
	})
	pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: layerContent}})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("the overlay is missing")
	}
	if got := pageText(t, ctx, out, 0); !strings.Contains(got, "LAYERTEXT") {
		t.Errorf("overlay text is not extractable: %q", got)
	}
}

// Two pages share one inherited /Resources dictionary. Overlaying page 1
// must not add its XObject to page 2.
func TestOverlayPages_DoesNotLeakIntoSharedResources(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")

	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [4 0 R 5 0 R] /Count 2 /MediaBox [" + overlayBox + "] /Resources 3 0 R >>",
		"<< /XObject << /Im0 6 0 R >> >>",
		"<< /Type /Page /Parent 2 0 R /Contents 7 0 R >>",
		"<< /Type /Page /Parent 2 0 R /Contents 7 0 R >>",
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 200 200]", "0 0 0 rg 20 20 60 60 re f"),
		rawStream("", "/Im0 Do"),
	})
	// Only the first overlay page has content.
	pagesPDF(t, layer, []testPage{
		{mediaBox: overlayBox, content: layerContent},
		{mediaBox: overlayBox, content: " \n\t "},
	})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}

	first := renderTestPage(t, ctx, out, 0, 72)
	second := renderTestPage(t, ctx, out, 1, 72)
	for i, img := range []*image.RGBA{first, second} {
		if !isDark(img.At(lowerLeft.X, lowerLeft.Y)) {
			t.Errorf("page %d lost its existing XObject", i+1)
		}
	}
	if !isDark(first.At(upperRight.X, upperRight.Y)) {
		t.Error("page 1 did not get its overlay")
	}
	if !isWhite(second.At(upperRight.X, upperRight.Y)) {
		t.Error("page 2 shows page 1's overlay")
	}
	if got := pageText(t, ctx, out, 1); strings.Contains(got, "LAYERTEXT") {
		t.Error("page 2 carries page 1's overlay text")
	}

	// Exactly one page refers to the overlay XObject by name.
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(data, []byte("/GoMuOv0")); n != 1 {
		t.Errorf("/GoMuOv0 appears in %d resource dictionaries, want 1: a blank overlay page must be skipped", n)
	}
}

// A name the overlay would pick is already taken on the input page.
func TestOverlayPages_AvoidsXObjectNameCollision(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")

	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + overlayBox + "] /Resources << /XObject << /GoMuOv0 4 0 R >> >> /Contents 5 0 R >>",
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 200 200]", baseContent),
		rawStream("", "/GoMuOv0 Do"),
	})
	pagesPDF(t, layer, []testPage{{mediaBox: overlayBox, content: layerContent}})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if !isDark(img.At(lowerLeft.X, lowerLeft.Y)) {
		t.Error("the input's own /GoMuOv0 was overwritten")
	}
	if !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("the overlay is missing")
	}
}

// Running the overlay twice (two layers) keeps both.
func TestOverlayPages_CanBeAppliedRepeatedly(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layerA := filepath.Join(dir, "a.pdf")
	layerB := filepath.Join(dir, "b.pdf")
	mid := filepath.Join(dir, "mid.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: ""}})
	pagesPDF(t, layerA, []testPage{{mediaBox: overlayBox, content: baseContent}})
	pagesPDF(t, layerB, []testPage{{mediaBox: overlayBox, content: layerContent}})

	if err := OverlayPages(ctx, in, layerA, mid, nil); err != nil {
		t.Fatalf("first overlay: %v", err)
	}
	if err := OverlayPages(ctx, mid, layerB, out, nil); err != nil {
		t.Fatalf("second overlay: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if !isDark(img.At(lowerLeft.X, lowerLeft.Y)) || !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("one of the two layers is missing")
	}
}

// An overlay page larger than the input page must not be clipped to the
// overlay's own box where the input page is bigger, or vice versa.
func TestOverlayPages_DifferentPageSizes(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: overlayBox, content: ""}})
	// The overlay's box covers only the lower-left corner, but it draws in
	// the upper-right of the input page.
	pagesPDF(t, layer, []testPage{{mediaBox: "0 0 50 50", content: "0 0 0 rg 120 120 60 60 re f"}})

	if err := OverlayPages(ctx, in, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if img.Bounds().Size() != (image.Point{200, 200}) {
		t.Fatalf("output page size changed to %v", img.Bounds().Size())
	}
	if !isDark(img.At(upperRight.X, upperRight.Y)) {
		t.Error("overlay content outside the overlay's own MediaBox was clipped away")
	}
}

func TestOverlayPages_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	one := filepath.Join(dir, "one.pdf")
	two := filepath.Join(dir, "two.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, one, []testPage{{mediaBox: overlayBox, content: baseContent}})
	pagesPDF(t, two, []testPage{
		{mediaBox: overlayBox, content: layerContent},
		{mediaBox: overlayBox, content: layerContent},
	})
	missing := filepath.Join(dir, "missing.pdf")

	tests := []struct {
		name               string
		in, layer, out     string
		wantErrorSubstring string
	}{
		{"page count mismatch", one, two, out, "page count mismatch"},
		{"output equals input", one, one, one, "must differ"},
		{"output equals overlay", one, two, two, "must differ"},
		{"empty overlay path", one, "", out, "must not be empty"},
		{"missing input", missing, one, out, "cannot open"},
		{"missing overlay", one, missing, out, "cannot open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OverlayPages(ctx, tt.in, tt.layer, tt.out, nil)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErrorSubstring) {
				t.Fatalf("error %q does not contain %q", err, tt.wantErrorSubstring)
			}
		})
	}

	if _, err := os.Stat(out); err == nil {
		t.Error("an output file was left behind by a failed overlay")
	}
}

// OverlayPages replaces `cpdf -combine-pages`; on upright pages the two
// must draw the same picture.
func TestOverlayPages_MatchesCpdfCombinePages(t *testing.T) {
	cpdf := requireTool(t, "cpdf")
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	pagesPDF(t, in, []testPage{
		{mediaBox: overlayBox, content: baseContent},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack},
	})
	pagesPDF(t, layer, []testPage{
		{mediaBox: overlayBox, content: layerContent},
		{mediaBox: offsetMediaBox, content: "0 0 0 rg 200 300 80 80 re f"},
	})

	ours := filepath.Join(dir, "ours.pdf")
	theirs := filepath.Join(dir, "theirs.pdf")
	if err := OverlayPages(ctx, in, layer, ours, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	runTool(t, cpdf, "-combine-pages", layer, in, "-o", theirs)

	for page := 0; page < 2; page++ {
		got := renderTestPage(t, ctx, ours, page, 72)
		want := renderTestPage(t, ctx, theirs, page, 72)
		if d := maxChannelDiff(t, got, want); d > 2 {
			t.Errorf("page %d differs from cpdf's output by up to %d/255", page+1, d)
		}
	}
}
