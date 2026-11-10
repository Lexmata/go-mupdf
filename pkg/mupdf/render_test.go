package mupdf

import (
	"context"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// rotationFixture writes four pages with the same content and MediaBox,
// rotated 0, 90, 180 and 270 degrees.
func rotationFixture(t *testing.T, path string) {
	t.Helper()
	pagesPDF(t, path, []testPage{
		{mediaBox: offsetMediaBox, content: cornerMarkBlack},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 180"},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 270"},
	})
}

// markCorner reports which corner of img holds the dark mark, checking that
// the other three are white.
func markCorner(t *testing.T, img *image.RGBA) string {
	t.Helper()
	b := img.Bounds()
	corners := map[string]image.Point{
		"top-left":     {b.Min.X + 25, b.Min.Y + 25},
		"top-right":    {b.Max.X - 25, b.Min.Y + 25},
		"bottom-left":  {b.Min.X + 25, b.Max.Y - 25},
		"bottom-right": {b.Max.X - 25, b.Max.Y - 25},
	}
	found := ""
	for name, p := range corners {
		c := img.At(p.X, p.Y)
		switch {
		case isDark(c):
			if found != "" {
				t.Fatalf("mark found in both %s and %s", found, name)
			}
			found = name
		case !isWhite(c):
			t.Fatalf("%s corner is neither black nor white: %v", name, c)
		}
	}
	if found == "" {
		t.Fatal("mark not found in any corner")
	}
	return found
}

func TestRenderPage_SizeAndRotation(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "rot.pdf")
	rotationFixture(t, path)
	doc := openTestDocument(t, ctx, path)

	tests := []struct {
		page          int
		width, height int
		corner        string
	}{
		{0, 300, 400, "bottom-left"},
		{1, 400, 300, "top-left"},
		{2, 300, 400, "top-right"},
		{3, 400, 300, "bottom-right"},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.page), func(t *testing.T) {
			img, err := doc.RenderPage(tt.page, 72, nil)
			if err != nil {
				t.Fatalf("RenderPage: %v", err)
			}
			if got := img.Bounds(); got != image.Rect(0, 0, tt.width, tt.height) {
				t.Fatalf("bounds %v, want %dx%d at origin", got, tt.width, tt.height)
			}
			if got := markCorner(t, img); got != tt.corner {
				t.Errorf("mark in %s corner, want %s", got, tt.corner)
			}
		})
	}
}

func TestRenderPage_ScalesWithDPI(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "letter.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 612 792", content: "BT /F1 24 Tf 72 700 Td (Hello) Tj ET"}})
	doc := openTestDocument(t, ctx, path)

	for dpi, want := range map[float64]image.Point{
		72:  {612, 792},
		150: {1275, 1650},
		300: {2550, 3300},
	} {
		t.Run(strconv.FormatFloat(dpi, 'f', -1, 64), func(t *testing.T) {
			img, err := doc.RenderPage(0, dpi, nil)
			if err != nil {
				t.Fatalf("RenderPage: %v", err)
			}
			if got := img.Bounds().Size(); got != want {
				t.Errorf("size %v, want %v", got, want)
			}
		})
	}
}

func TestRenderPage_OpaqueWhiteBackgroundWithInk(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "text.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 200 100", content: "BT /F1 24 Tf 20 40 Td (Hello) Tj ET"}})

	img := renderTestPage(t, ctx, path, 0, 72)
	dark := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] != 0xff {
			t.Fatalf("pixel %d is not opaque (alpha %d)", i/4, img.Pix[i+3])
		}
		if img.Pix[i] < 0x40 {
			dark++
		}
	}
	if !isWhite(img.At(1, 1)) {
		t.Errorf("background pixel is %v, want white", img.At(1, 1))
	}
	if dark == 0 {
		t.Error("no dark pixels: the text was not drawn")
	}
}

// A CropBox smaller than the MediaBox must not shrink the image: the frame
// is the MediaBox, and content outside the CropBox is still drawn.
func TestRenderPage_UsesMediaBoxNotCropBox(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "crop.pdf")
	pagesPDF(t, path, []testPage{{
		mediaBox: offsetMediaBox,
		extra:    "/CropBox [110 120 210 320]",
		content:  cornerMarkBlack,
	}})

	img := renderTestPage(t, ctx, path, 0, 72)
	if got := img.Bounds().Size(); got != (image.Point{300, 400}) {
		t.Fatalf("size %v, want the 300x400 MediaBox", got)
	}
	if got := markCorner(t, img); got != "bottom-left" {
		t.Errorf("mark in %s corner, want bottom-left", got)
	}
}

func TestRenderPage_Errors(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "one.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 100 100"}})
	doc := openTestDocument(t, ctx, path)

	tests := []struct {
		name string
		page int
		dpi  float64
		want string
	}{
		{"negative page", -1, 72, "out of range"},
		{"page past end", 1, 72, "out of range"},
		{"zero dpi", 0, 0, "dpi must be a positive number"},
		{"negative dpi", 0, -72, "dpi must be a positive number"},
		{"NaN dpi", 0, math.NaN(), "dpi must be a positive number"},
		{"infinite dpi", 0, math.Inf(1), "dpi must be a positive number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := doc.RenderPage(tt.page, tt.dpi, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error containing %q, got %v", tt.want, err)
			}
		})
	}

	t.Run("closed document", func(t *testing.T) {
		closed, err := OpenDocument(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		closed.Close()
		if _, err := closed.RenderPage(0, 72, nil); err == nil || !strings.Contains(err.Error(), "closed") {
			t.Fatalf("want closed-document error, got %v", err)
		}
	})

	t.Run("nil document", func(t *testing.T) {
		var nilDoc *Document
		if _, err := nilDoc.RenderPage(0, 72, nil); err == nil {
			t.Fatal("want error for a nil document")
		}
	})
}

// A dropped context must be reported as that, not as an empty document.
func TestRenderPage_DroppedContext(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("mupdf unavailable: %v", err)
	}
	path := filepath.Join(t.TempDir(), "one.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 100 100"}})
	doc, err := OpenDocument(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	doc.Close()
	ctx.Drop()

	if _, err := doc.RenderPage(0, 72, nil); err == nil || !strings.Contains(err.Error(), "dropped") {
		t.Fatalf("want a dropped-context error, got %v", err)
	}
}

// Rendering a page with a CropBox must not change what a later call sees:
// the temporary CropBox override is undone.
func TestRenderPage_RestoresCropBox(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "crop.pdf")
	pagesPDF(t, path, []testPage{{
		mediaBox: offsetMediaBox,
		extra:    "/CropBox [110 120 210 320]",
		content:  cornerMarkBlack,
	}})
	doc := openTestDocument(t, ctx, path)

	first, err := doc.RenderPage(0, 72, nil)
	if err != nil {
		t.Fatal(err)
	}
	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	bound := page.Bound()
	page.Close()
	if w, h := bound.X1-bound.X0, bound.Y1-bound.Y0; w != 100 || h != 200 {
		t.Errorf("after RenderPage the page is bounded %vx%v, want the 100x200 CropBox", w, h)
	}
	second, err := doc.RenderPage(0, 72, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := maxChannelDiff(t, first, second); d != 0 {
		t.Errorf("two renders of the same page differ by %d/255", d)
	}
}

// The CropBox can be inherited from the page tree rather than set on the
// page. The temporary override then has to be deleted afterwards, not put
// back, or the page would keep a CropBox of its own.
func TestRenderPage_InheritedCropBox(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "inherited.pdf")
	writeRawPDF(t, path, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [" + offsetMediaBox + "] /CropBox [110 120 210 320] >>",
		"<< /Type /Page /Parent 2 0 R /Contents 4 0 R >>",
		rawStream("", cornerMarkBlack),
	})
	doc := openTestDocument(t, ctx, path)

	first, err := doc.RenderPage(0, 72, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := first.Bounds().Size(); got != (image.Point{300, 400}) {
		t.Fatalf("size %v, want the 300x400 MediaBox", got)
	}
	if got := markCorner(t, first); got != "bottom-left" {
		t.Errorf("mark in %s corner, want bottom-left", got)
	}

	page, err := doc.LoadPage(0)
	if err != nil {
		t.Fatal(err)
	}
	bound := page.Bound()
	page.Close()
	if w, h := bound.X1-bound.X0, bound.Y1-bound.Y0; w != 100 || h != 200 {
		t.Errorf("after RenderPage the page is bounded %vx%v, want the inherited 100x200 CropBox", w, h)
	}
	second, err := doc.RenderPage(0, 72, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := maxChannelDiff(t, first, second); d != 0 {
		t.Errorf("two renders of the same page differ by %d/255", d)
	}
}

// The page size comes from the file. A page that would need more pixels
// than the limit must be refused before anything is allocated for it.
func TestRenderPage_MaxPixels(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "square.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 1000 1000", content: cornerMarkBlack}})
	doc := openTestDocument(t, ctx, path)

	tests := []struct {
		name    string
		opts    *RenderOptions
		wantErr bool
	}{
		{"one pixel under the page", &RenderOptions{MaxPixels: 999_999}, true},
		{"exactly the page", &RenderOptions{MaxPixels: 1_000_000}, false},
		{"default limit", nil, false},
		{"zero value means the default", &RenderOptions{}, false},
		{"no limit", &RenderOptions{MaxPixels: -1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := doc.RenderPage(0, 72, tt.opts)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "limit") {
					t.Fatalf("want a limit error, got %v", err)
				}
				var mupdfErr Error
				if !errors.As(err, &mupdfErr) {
					t.Errorf("error %v does not wrap mupdf.Error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderPage: %v", err)
			}
			if img.Bounds().Size() != (image.Point{1000, 1000}) {
				t.Errorf("size %v, want 1000x1000", img.Bounds().Size())
			}
		})
	}

	// The largest page the PDF specification allows, at the resolution the
	// README's example uses, would be 14.4 GB. The default must stop it.
	huge := filepath.Join(dir, "huge.pdf")
	pagesPDF(t, huge, []testPage{{mediaBox: "0 0 14400 14400", content: cornerMarkBlack}})
	if _, err := openTestDocument(t, ctx, huge).RenderPage(0, 300, nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("a 3.6 gigapixel render was not refused by the default limit: %v", err)
	}
}

func TestRenderPageContext_AlreadyCancelled(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "one.pdf")
	pagesPDF(t, path, []testPage{{mediaBox: "0 0 100 100"}})
	doc := openTestDocument(t, ctx, path)

	goCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := doc.RenderPageContext(goCtx, 0, 72, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	// The document is still usable afterwards.
	if _, err := doc.RenderPage(0, 72, nil); err != nil {
		t.Fatalf("render after a cancelled one: %v", err)
	}
}

// A page crafted to be slow must stop when its deadline passes, and must
// not hand back the half-drawn picture.
func TestRenderPageContext_StopsSlowPage(t *testing.T) {
	skipIfShort(t)
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "slow.pdf")
	// Hundreds of thousands of wide, semi-transparent strokes.
	var content strings.Builder
	content.WriteString("/G gs 20 w\n")
	for i := 0; i < 300_000; i++ {
		content.WriteString("0 0 m 600 780 l S 600 0 m 0 780 l S\n")
	}
	writeRawPDF(t, path, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /ExtGState << /G << /CA 0.5 >> >> >> >>",
		rawStream("", content.String()),
	})
	doc := openTestDocument(t, ctx, path)

	goCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	img, err := doc.RenderPageContext(goCtx, 0, 300, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Skipf("the page rendered in %v, before the deadline: this machine is too fast for the fixture", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
	if img != nil {
		t.Error("a partial image was returned alongside the error")
	}
	if elapsed > 20*time.Second {
		t.Errorf("the render took %v to notice a 50ms deadline", elapsed)
	}
}

// RenderPage replaces `pdftoppm -png -r <dpi>`. The two rasterisers
// anti-alias differently, so the comparison is on size (within a pixel) and
// on where the ink is, not on exact pixel values.
func TestRenderPage_MatchesPdftoppm(t *testing.T) {
	pdftoppm := requireTool(t, "pdftoppm")
	ctx := newTestContext(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "rot.pdf")
	pagesPDF(t, path, []testPage{
		{mediaBox: offsetMediaBox, content: cornerMarkBlack},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 180"},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/Rotate 270"},
		{mediaBox: offsetMediaBox, content: cornerMarkBlack, extra: "/CropBox [110 120 210 320]"},
		{mediaBox: "0 0 612.5 792.3", content: "0 0 0 rg 100 100 300 500 re f"},
	})
	doc := openTestDocument(t, ctx, path)

	const dpi = 100
	for page := 0; page < doc.CountPages(); page++ {
		prefix := filepath.Join(dir, "ppm")
		runTool(t, pdftoppm, "-png", "-singlefile", "-r", "100",
			"-f", strconv.Itoa(page+1), "-l", strconv.Itoa(page+1), path, prefix)
		want := decodePNG(t, prefix+".png")

		got, err := doc.RenderPage(page, dpi, nil)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}

		gs, ws := got.Bounds().Size(), want.Bounds().Size()
		if absInt(gs.X-ws.X) > 1 || absInt(gs.Y-ws.Y) > 1 {
			t.Errorf("page %d: size %v, pdftoppm gives %v", page, gs, ws)
			continue
		}

		// Compare on the common area, ignoring a 3-pixel band around every
		// black/white edge where anti-aliasing legitimately differs.
		w, h := min(gs.X, ws.X), min(gs.Y, ws.Y)
		mismatched := 0
		for y := 3; y < h-3; y += 2 {
			for x := 3; x < w-3; x += 2 {
				if nearEdge(want, x, y) {
					continue
				}
				if isDark(got.At(x, y)) != isDark(want.At(x, y)) {
					mismatched++
				}
			}
		}
		if mismatched > 0 {
			t.Errorf("page %d: %d sampled pixels disagree with pdftoppm away from edges", page, mismatched)
		}
	}
}

func nearEdge(img *image.RGBA, x, y int) bool {
	ref := isDark(img.At(x, y))
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if isDark(img.At(x+dx, y+dy)) != ref {
				return true
			}
		}
	}
	return false
}

func decodePNG(t *testing.T, path string) *image.RGBA {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	src, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	dst := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
