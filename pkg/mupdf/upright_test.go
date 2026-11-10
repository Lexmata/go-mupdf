package mupdf

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// An asymmetric picture: the corner mark plus a bar along the stored bottom
// edge, so a wrong rotation or a mirror image cannot pass as correct.
const asymmetricContent = cornerMarkBlack + " 0.5 0.5 0.5 rg 100 30 180 15 re f BT /F1 18 Tf 60 200 Td (Upright) Tj ET"

func TestUprightPages_AppearanceUnchanged(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{
		{mediaBox: offsetMediaBox, content: asymmetricContent},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 180"},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 270"},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate -90"},
	})

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 4 {
		t.Errorf("changed = %d, want 4 (every page but the first)", changed)
	}

	geom, err := GetPageGeometry(ctx, out)
	if err != nil {
		t.Fatal(err)
	}
	wantSize := []image.Point{{300, 400}, {400, 300}, {300, 400}, {400, 300}, {400, 300}}
	for i, g := range geom {
		if !g.IsUpright() {
			t.Errorf("page %d still has rotation %d", i+1, g.Rotation)
		}
		if int(g.Width()) != wantSize[i].X || int(g.Height()) != wantSize[i].Y {
			t.Errorf("page %d is %vx%v, want %v", i+1, g.Width(), g.Height(), wantSize[i])
		}
		// Pages that were rotated get a MediaBox at the origin; the
		// untouched first page keeps its offset box.
		wantOrigin := Rect{X0: 0, Y0: 0}
		if i == 0 {
			wantOrigin = Rect{X0: 10, Y0: 20}
		}
		if g.MediaBox.X0 != wantOrigin.X0 || g.MediaBox.Y0 != wantOrigin.Y0 {
			t.Errorf("page %d MediaBox origin is (%v, %v), want (%v, %v)",
				i+1, g.MediaBox.X0, g.MediaBox.Y0, wantOrigin.X0, wantOrigin.Y0)
		}
	}

	for page := 0; page < 5; page++ {
		before := renderTestPage(t, ctx, in, page, 100)
		after := renderTestPage(t, ctx, out, page, 100)
		if d := maxChannelDiff(t, before, after); d > 2 {
			t.Errorf("page %d looks different after UprightPages (max channel diff %d/255)", page+1, d)
		}
	}

	if res, err := CheckPDF(ctx, out); err != nil || !res.Clean() {
		t.Errorf("output is not a clean PDF: %+v, err %v", res, err)
	}
	if err := ValidatePDF(out, nil); err != nil {
		t.Errorf("pdfcpu rejects the output: %v", err)
	}
	if got := pageText(t, ctx, out, 1); !strings.Contains(got, "Upright") {
		t.Errorf("text lost on a rotated page: %q", got)
	}
}

func TestUprightPages_NothingToDo(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{
		{mediaBox: offsetMediaBox, content: asymmetricContent},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 360"},
	})

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 0 {
		t.Errorf("changed = %d, want 0", changed)
	}
	// The output is written regardless.
	if n, err := PageCount(ctx, out); err != nil || n != 2 {
		t.Errorf("output has %d pages (err %v), want 2", n, err)
	}
	for page := 0; page < 2; page++ {
		if d := maxChannelDiff(t, renderTestPage(t, ctx, in, page, 72), renderTestPage(t, ctx, out, page, 72)); d != 0 {
			t.Errorf("page %d changed although nothing needed doing", page+1)
		}
	}
}

// /Rotate and /MediaBox inherited from the page tree, and one content stream
// shared by every page: each page must still be turned on its own.
func TestUprightPages_InheritedAttributesAndSharedContent(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R 5 0 R] /Count 3 /MediaBox [" + offsetMediaBox + "] /Rotate 90 >>",
		"<< /Type /Page /Parent 2 0 R /Contents 6 0 R >>",
		"<< /Type /Page /Parent 2 0 R /Contents 6 0 R /Rotate 0 >>",
		"<< /Type /Page /Parent 2 0 R /Contents [6 0 R 7 0 R] /Rotate 180 >>",
		rawStream("", cornerMarkBlack),
		rawStream("", "0.5 0.5 0.5 rg 100 30 180 15 re f"),
	})

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 2 {
		t.Errorf("changed = %d, want 2", changed)
	}
	for page := 0; page < 3; page++ {
		before := renderTestPage(t, ctx, in, page, 100)
		after := renderTestPage(t, ctx, out, page, 100)
		if d := maxChannelDiff(t, before, after); d > 2 {
			t.Errorf("page %d looks different (max channel diff %d/255)", page+1, d)
		}
	}
	geom, err := GetPageGeometry(ctx, out)
	if err != nil {
		t.Fatal(err)
	}
	for i, g := range geom {
		if !g.IsUpright() {
			t.Errorf("page %d still has rotation %d", i+1, g.Rotation)
		}
	}
}

// CropBox and the other boxes move with the page.
func TestUprightPages_TransformsOtherBoxes(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{
		mediaBox: offsetMediaBox,
		extra:    "/Rotate 90 /CropBox [60 70 260 370] /TrimBox [70 80 250 360]",
		content:  asymmetricContent,
	}})

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Rotate 90 maps (x, y) to (y - 20, 310 - x).
	for name, want := range map[string][4]float64{
		"MediaBox": {0, 0, 400, 300},
		"CropBox":  {50, 50, 350, 250},
		"TrimBox":  {60, 60, 340, 240},
	} {
		got, ok := boxInFile(data, name)
		if !ok {
			t.Errorf("%s missing from the output", name)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

// boxInFile finds the first "/Name [a b c d]" in raw PDF bytes.
func boxInFile(data []byte, name string) ([4]float64, bool) {
	re := regexp.MustCompile(`/` + name + `\s*\[\s*([-0-9.]+)\s+([-0-9.]+)\s+([-0-9.]+)\s+([-0-9.]+)\s*\]`)
	m := re.FindSubmatch(data)
	var box [4]float64
	if m == nil {
		return box, false
	}
	for i := range box {
		box[i], _ = strconv.ParseFloat(string(m[i+1]), 64)
	}
	return box, true
}

// An annotation's rectangle, quad points and appearance follow the page.
func TestUprightPages_MovesAnnotations(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")

	// A stamp-like annotation whose appearance is a black block in the
	// left half of its rectangle, and a highlight with quad points. Both
	// annotations share one appearance stream.
	pagesPDF(t, in,
		[]testPage{{
			mediaBox: offsetMediaBox,
			extra:    "/Rotate 90 /Annots [6 0 R 7 0 R]",
			content:  "",
		}},
		"<< /Type /Annot /Subtype /Stamp /Rect [110 220 210 260] /F 4 /AP << /N 8 0 R >> >>",
		"<< /Type /Annot /Subtype /Square /Rect [110 320 210 360] /F 4 /QuadPoints [110 360 210 360 110 320 210 320] /AP << /N 8 0 R >> >>",
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 40]", "0 0 0 rg 0 0 50 40 re f"),
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}

	// Rendering includes annotation appearances, so an identical picture
	// proves rectangle and appearance were both turned, and that the shared
	// appearance stream was turned once, not twice.
	before := renderTestPage(t, ctx, in, 0, 100)
	after := renderTestPage(t, ctx, out, 0, 100)
	dark := 0
	for i := 0; i < len(before.Pix); i += 4 {
		if before.Pix[i] < 0x40 {
			dark++
		}
	}
	if dark == 0 {
		t.Fatal("fixture problem: the annotations did not render at all")
	}
	if d := maxChannelDiff(t, before, after); d > 2 {
		t.Errorf("annotations moved or turned relative to the page (max channel diff %d/255)", d)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Rotate 90 maps (x, y) to (y - 20, 310 - x): [110 220 210 260] becomes
	// [200 100 240 200].
	if !regexp.MustCompile(`/Rect\s*\[\s*200 100 240 200\s*\]`).Match(data) {
		t.Error("the stamp's /Rect was not transformed to [200 100 240 200]")
	}
	// (110,360) -> (340,200), (210,360) -> (340,100), (110,320) -> (300,200),
	// (210,320) -> (300,100).
	if !regexp.MustCompile(`/QuadPoints\s*\[\s*340 200 340 100 300 200 300 100\s*\]`).Match(data) {
		t.Error("/QuadPoints were not transformed point by point")
	}
}

func TestUprightPages_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	pagesPDF(t, in, []testPage{{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 90"}})
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
		{"unwritable output", in, filepath.Join(dir, "no-such-dir", "out.pdf"), "UprightPages"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed, err := UprightPages(ctx, tt.in, tt.out, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tt.wantErr, err)
			}
			if changed != 0 {
				t.Errorf("changed = %d alongside an error, want 0", changed)
			}
		})
	}

	// A refused call must not have touched the input.
	if geom, err := GetPageGeometry(ctx, in); err != nil || len(geom) != 1 || geom[0].Rotation != 90 {
		t.Errorf("input damaged: %+v, err %v", geom, err)
	}
	assertNoStrayFiles(t, dir, "in.pdf")
}

// UprightPages replaces `cpdf -upright`: same page sizes, same picture.
func TestUprightPages_MatchesCpdfUpright(t *testing.T) {
	cpdf := requireTool(t, "cpdf")
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	pagesPDF(t, in, []testPage{
		{mediaBox: offsetMediaBox, content: asymmetricContent},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 180"},
		{mediaBox: offsetMediaBox, content: asymmetricContent, extra: "/Rotate 270"},
	})

	ours := filepath.Join(dir, "ours.pdf")
	theirs := filepath.Join(dir, "theirs.pdf")
	if _, err := UprightPages(ctx, in, ours, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	runTool(t, cpdf, "-upright", in, "-o", theirs)

	got, err := GetPageGeometry(ctx, ours)
	if err != nil {
		t.Fatal(err)
	}
	want, err := GetPageGeometry(ctx, theirs)
	if err != nil {
		t.Fatal(err)
	}
	for i := range want {
		desc := fmt.Sprintf("page %d", i+1)
		if got[i].Rotation != 0 || want[i].Rotation != 0 {
			t.Errorf("%s: rotation ours %d, cpdf %d, want both 0", desc, got[i].Rotation, want[i].Rotation)
		}
		if math.Abs(got[i].Width()-want[i].Width()) > 0.01 || math.Abs(got[i].Height()-want[i].Height()) > 0.01 {
			t.Errorf("%s: size ours %vx%v, cpdf %vx%v", desc,
				got[i].Width(), got[i].Height(), want[i].Width(), want[i].Height())
		}
		// cpdf's own output is not pixel-identical to the original: it
		// writes two transforms where one would do, and glyph edges
		// anti-alias slightly differently. So the bound on how far we may be
		// from cpdf is how far cpdf is from the original page.
		original := renderTestPage(t, ctx, in, i, 100)
		a := renderTestPage(t, ctx, ours, i, 100)
		b := renderTestPage(t, ctx, theirs, i, 100)
		if d := maxChannelDiff(t, original, a); d > 2 {
			t.Errorf("%s: our picture differs from the original by up to %d/255", desc, d)
		}
		limit := maxChannelDiff(t, original, b) + 2
		if d := maxChannelDiff(t, a, b); d > limit {
			t.Errorf("%s: picture differs from cpdf's by up to %d/255, more than cpdf differs from the original (%d)", desc, d, limit)
		}
	}
}

// assertSamePicture renders every page of two files and compares them.
func assertSamePicture(t *testing.T, ctx *Context, before, after string, pages int) {
	t.Helper()
	for page := 0; page < pages; page++ {
		a := renderTestPage(t, ctx, before, page, 100)
		b := renderTestPage(t, ctx, after, page, 100)
		dark := 0
		for i := 0; i < len(a.Pix); i += 4 {
			if a.Pix[i] < 0xc0 {
				dark++
			}
		}
		if dark == 0 {
			t.Fatalf("fixture problem: page %d rendered blank", page+1)
		}
		if d := maxChannelDiff(t, a, b); d > 2 {
			t.Errorf("page %d looks different after UprightPages (max channel diff %d/255)", page+1, d)
		}
	}
}

// Patterns are positioned relative to the content stream's base coordinate
// space, not the current transform, so a `cm` in front of the content turns
// the shapes and leaves the pattern behind.
func TestUprightPages_PatternsTurnWithThePage(t *testing.T) {
	ctx := newTestContext(t)
	for _, rotate := range []int{90, 180, 270} {
		dir := t.TempDir()
		in := filepath.Join(dir, "in.pdf")
		out := filepath.Join(dir, "out.pdf")
		writeRawPDF(t, in, []string{
			"<< /Type /Catalog /Pages 2 0 R >>",
			"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [%s] /Rotate %d /Contents 4 0 R "+
				"/Resources << /Pattern << /Shade 5 0 R /Tile 7 0 R >> >> >>", offsetMediaBox, rotate),
			rawStream("", "/Pattern cs /Shade scn 30 40 120 340 re f /Tile scn 170 40 120 340 re f"),
			// An axial shading running left to right across the page.
			"<< /PatternType 2 /Shading 6 0 R /Matrix [1 0 0 1 10 20] >>",
			"<< /ShadingType 2 /ColorSpace /DeviceGray /Coords [0 0 150 0] /Function << /FunctionType 2 /Domain [0 1] /C0 [0] /C1 [1] /N 1 >> /Extend [true true] >>",
			// A tiling pattern of wide, short black bars.
			rawStream("/PatternType 1 /PaintType 1 /TilingType 1 /BBox [0 0 40 20] /XStep 40 /YStep 20 /Resources << >> /Matrix [1 0 0 1 3 7]",
				"0 0 0 rg 0 0 30 8 re f"),
		})

		changed, err := UprightPages(ctx, in, out, nil)
		if err != nil {
			t.Fatalf("/Rotate %d: %v", rotate, err)
		}
		if changed != 1 {
			t.Errorf("/Rotate %d: changed = %d, want 1", rotate, changed)
		}
		assertSamePicture(t, ctx, in, out, 1)
	}
}

// A NoRotate annotation (flag bit 5) is drawn upright, hanging from the
// upper-left corner of its rectangle; it must not be turned with the page.
func TestUprightPages_NoRotateAnnotationStaysUpright(t *testing.T) {
	ctx := newTestContext(t)
	for _, rotate := range []int{90, 180, 270} {
		dir := t.TempDir()
		in := filepath.Join(dir, "in.pdf")
		out := filepath.Join(dir, "out.pdf")
		pagesPDF(t, in,
			[]testPage{{
				mediaBox: offsetMediaBox,
				extra:    fmt.Sprintf("/Rotate %d /Annots [6 0 R 7 0 R]", rotate),
				content:  cornerMarkBlack,
			}},
			// F 20 = Print | NoRotate; F 4 = Print.
			"<< /Type /Annot /Subtype /Stamp /Rect [110 220 210 260] /F 20 /AP << /N 8 0 R >> >>",
			"<< /Type /Annot /Subtype /Stamp /Rect [110 320 210 360] /F 4 /AP << /N 8 0 R >> >>",
			rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 40]", "0 0 0 rg 0 0 50 40 re f 0.5 g 50 0 50 15 re f"),
		)

		if _, err := UprightPages(ctx, in, out, nil); err != nil {
			t.Fatalf("/Rotate %d: %v", rotate, err)
		}
		assertSamePicture(t, ctx, in, out, 1)
	}
}

// One appearance stream used by annotations on pages with different
// rotations, including a page that is not rotated at all. Each page must
// keep its own picture, so the stream has to be copied, not turned in place.
func TestUprightPages_AppearanceSharedAcrossPages(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	// Objects: pages are 4, 6, 8; annotations 10, 11, 12; appearance 13;
	// appearance dictionary 14 (shared too).
	pagesPDF(t, in,
		[]testPage{
			{mediaBox: offsetMediaBox, extra: "/Rotate 90 /Annots [10 0 R]", content: cornerMarkBlack},
			{mediaBox: offsetMediaBox, extra: "/Annots [11 0 R]", content: cornerMarkBlack},
			{mediaBox: offsetMediaBox, extra: "/Rotate 180 /Annots [12 0 R]", content: cornerMarkBlack},
		},
		"<< /Type /Annot /Subtype /Stamp /Rect [110 220 210 260] /F 4 /AP 14 0 R >>",
		"<< /Type /Annot /Subtype /Stamp /Rect [110 220 210 260] /F 4 /AP 14 0 R >>",
		"<< /Type /Annot /Subtype /Stamp /Rect [110 220 210 260] /F 4 /AP << /N 13 0 R >> >>",
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 40]", "0 0 0 rg 0 0 50 40 re f 0.5 g 50 0 50 15 re f"),
		"<< /N 13 0 R >>",
	)

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 2 {
		t.Errorf("changed = %d, want 2", changed)
	}
	assertSamePicture(t, ctx, in, out, 3)
}

// Checkbox-style appearances: /AP /N is a dictionary of states.
func TestUprightPages_AppearanceStates(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Rotate 270 /Annots [6 0 R]", content: cornerMarkBlack}},
		"<< /Type /Annot /Subtype /Widget /FT /Btn /Rect [110 220 210 260] /F 4 /AS /On /MK << /R 90 /BC [0] >> /AP << /N << /On 7 0 R /Off 8 0 R >> >> >>",
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 40]", "0 0 0 rg 0 0 50 40 re f"),
		rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 40]", "0.5 g 0 0 100 10 re f"),
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	assertSamePicture(t, ctx, in, out, 1)

	// The widget said "turned 90 counterclockwise from the page"; the page
	// has since turned 270 clockwise underneath it: (90 - 270) mod 360.
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`/MK\s*<<[^>]*/R 180\b`).Match(data) {
		t.Error("/MK /R was not adjusted to 180")
	}
	if !regexp.MustCompile(`/MK\s*<<[^>]*/BC`).Match(data) {
		t.Error("the rest of /MK was lost")
	}
}

// The coordinates a viewer uses to regenerate an appearance move too.
func TestUprightPages_TransformsAnnotationGeometry(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Rotate 90 /Annots [6 0 R 7 0 R 8 0 R 9 0 R]", content: cornerMarkBlack}},
		"<< /Type /Annot /Subtype /Line /Rect [110 220 210 260] /L [110 220 210 260] >>",
		"<< /Type /Annot /Subtype /Polygon /Rect [110 220 210 260] /Vertices [110 220 210 220 160 260] >>",
		"<< /Type /Annot /Subtype /Ink /Rect [110 220 210 260] /InkList [[110 220 210 260] [110 260 210 220]] >>",
		"<< /Type /Annot /Subtype /Square /Rect [110 220 210 260] /RD [1 2 3 4] >>",
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Rotate 90 maps (x, y) to (y - 20, 310 - x).
	for name, pattern := range map[string]string{
		"/L":        `/L\s*\[\s*200 200 240 100\s*\]`,
		"/Vertices": `/Vertices\s*\[\s*200 200 200 100 240 150\s*\]`,
		"/InkList":  `/InkList\s*\[\s*\[\s*200 200 240 100\s*\]\s*\[\s*240 200 200 100\s*\]\s*\]`,
		// Insets left 1, bottom 2, right 3, top 4 become: left = old
		// bottom, bottom = old right, right = old top, top = old left.
		"/RD": `/RD\s*\[\s*2 3 4 1\s*\]`,
	} {
		if !regexp.MustCompile(pattern).Match(data) {
			t.Errorf("%s was not transformed as expected (want %s)", name, pattern)
		}
	}
}

// A /Rotate that is not a right angle is drawn at the nearest one. Baking
// has to use that same angle and leave a clean /Rotate 0 behind.
func TestUprightPages_MalformedRotation(t *testing.T) {
	ctx := newTestContext(t)
	for rotate, wantChanged := range map[int]int{100: 1, 200: 1, 44: 0, 350: 0} {
		t.Run(strconv.Itoa(rotate), func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "in.pdf")
			out := filepath.Join(dir, "out.pdf")
			pagesPDF(t, in, []testPage{{
				mediaBox: offsetMediaBox,
				extra:    "/Rotate " + strconv.Itoa(rotate),
				content:  asymmetricContent,
			}})

			changed, err := UprightPages(ctx, in, out, nil)
			if err != nil {
				t.Fatalf("UprightPages: %v", err)
			}
			if changed != wantChanged {
				t.Errorf("changed = %d, want %d", changed, wantChanged)
			}
			assertSamePicture(t, ctx, in, out, 1)

			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if !regexp.MustCompile(`/Rotate 0\b`).Match(data) || regexp.MustCompile(`/Rotate [1-9]`).Match(data) {
				t.Error("the output does not carry a plain /Rotate 0")
			}
		})
	}
}

// Content that leaves a q open, or that sets state it never restores, is
// contained by the form it now lives in; a later overlay is not displaced.
func TestUprightPages_ThenOverlay(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	upright := filepath.Join(dir, "upright.pdf")
	layer := filepath.Join(dir, "layer.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in, []testPage{{
		mediaBox: "0 0 200 300",
		extra:    "/Rotate 90",
		content:  "q 0.5 0 0 0.5 400 400 cm 1 1 1 rg",
	}})
	// After baking the page is 300 wide and 200 high.
	pagesPDF(t, layer, []testPage{{mediaBox: "0 0 300 200", content: "0 0 0 rg 220 120 60 60 re f"}})

	if _, err := UprightPages(ctx, in, upright, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if err := OverlayPages(ctx, upright, layer, out, nil); err != nil {
		t.Fatalf("OverlayPages: %v", err)
	}
	img := renderTestPage(t, ctx, out, 0, 72)
	if img.Bounds().Size() != (image.Point{300, 200}) {
		t.Fatalf("size %v, want 300x200", img.Bounds().Size())
	}
	// The square's centre, (250, 150) in points, is pixel (250, 50).
	if !isDark(img.At(250, 50)) {
		t.Error("the overlay is not where its coordinates say after an upright")
	}
}

// One small stream listed many times as a page's content, or used as the
// appearance of many annotations, must not be copied without bound.
func TestUprightPages_ContentBudget(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")

	const repeats = 5000
	refs := strings.Repeat("4 0 R ", repeats)
	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 300] /Rotate 90 /Contents [" + refs + "] >>",
		rawStream("", strings.Repeat("0 0 0 rg 20 20 60 60 re f\n", 40)),
	})
	if size := fileSize(t, in); size > 100_000 {
		t.Fatalf("fixture problem: the input is %d bytes, it should be small", size)
	}

	_, err := UprightPages(ctx, in, out, &UprightOptions{MaxContentBytes: 1 << 20})
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want a limit error with a 1 MiB budget, got %v", err)
	}
	assertNoStrayFiles(t, dir, "in.pdf")

	if _, err := UprightPages(ctx, in, out, &UprightOptions{MaxContentBytes: -1}); err != nil {
		t.Errorf("unlimited budget: %v", err)
	}
}

// Many annotations on a page sharing one appearance get one turned copy
// between them, and an annotation listed twice is turned once.
func TestUprightPages_SharedAndRepeatedAnnotations(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")

	// Objects: page 4, content 5, then 40 annotations (6..45), the shared
	// appearance (46). The first annotation is listed three times.
	const annots = 40
	var list strings.Builder
	list.WriteString("6 0 R 6 0 R ")
	extra := make([]string, 0, annots+1)
	for i := 0; i < annots; i++ {
		fmt.Fprintf(&list, "%d 0 R ", 6+i)
		y := 30 + i*9
		extra = append(extra, fmt.Sprintf("<< /Type /Annot /Subtype /Stamp /Rect [110 %d 210 %d] /F 4 /AP << /N %d 0 R >> >>", y, y+6, 6+annots))
	}
	extra = append(extra, rawStream("/Type /XObject /Subtype /Form /BBox [0 0 100 6]", "0 0 0 rg 0 0 50 6 re f"))
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Rotate 90 /Annots [" + list.String() + "]", content: cornerMarkBlack}},
		extra...,
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	// Identical pictures prove the thrice-listed annotation was turned
	// once: turned again, its rectangle would be somewhere else.
	assertSamePicture(t, ctx, in, out, 1)

	// Two form matrices in the whole file: the page's content, and one
	// copy of the appearance shared by all forty annotations.
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(regexp.MustCompile(`/Matrix\s*\[`).FindAll(data, -1)); n != 2 {
		t.Errorf("found %d form matrices, want 2: the shared appearance was copied once per annotation", n)
	}
}

// A widget with no /MK gets one, so a viewer that regenerates its
// appearance turns it the right way.
func TestUprightPages_WidgetWithoutMK(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Rotate 90 /Annots [6 0 R]", content: cornerMarkBlack}},
		"<< /Type /Annot /Subtype /Widget /FT /Tx /Rect [110 220 210 260] /F 4 >>",
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// (0 - 90) mod 360.
	if !regexp.MustCompile(`/MK\s*<<\s*/R 270\s*>>`).Match(data) {
		t.Error("the widget was not given /MK << /R 270 >>")
	}
}

// An extreme /MK /R in the file must not overflow.
func TestUprightPages_ExtremeWidgetRotation(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Rotate 90 /Annots [6 0 R]", content: cornerMarkBlack}},
		"<< /Type /Annot /Subtype /Widget /FT /Tx /Rect [110 220 210 260] /F 4 /MK << /R -2147483648 >> >>",
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`/MK\s*<<\s*/R (-?\d+)`).FindSubmatch(data)
	if m == nil {
		t.Fatal("no /MK /R in the output")
	}
	if r, _ := strconv.Atoi(string(m[1])); r < 0 || r >= 360 {
		t.Errorf("/MK /R = %d, want a value in [0, 360)", r)
	}
}

// The same page object listed twice in the page tree is turned once.
func TestUprightPages_PageListedTwice(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeRawPDF(t, in, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 3 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + offsetMediaBox + "] /Rotate 90 /Contents 4 0 R >>",
		rawStream("", cornerMarkBlack),
	})

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 1 {
		t.Errorf("changed = %d, want 1: there is one page object", changed)
	}
	assertSamePicture(t, ctx, in, out, 2)
}

// Destinations that point at a turned page with explicit coordinates move
// with it. Page 1 is rotated 90 degrees, which maps (x, y) to
// (y - 20, 310 - x) and swaps the axes; page 2 is upright and its
// destinations must not change.
func TestUprightPages_RewritesDestinations(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeRawPDF(t, in, []string{
		// 1: catalog
		"<< /Type /Catalog /Pages 2 0 R /Outlines 7 0 R /Names << /Dests 10 0 R >> /OpenAction [3 0 R /FitV 110] /Dests << /old [3 0 R /XYZ 110 220 0] /olddict << /D [3 0 R /FitBH 220] >> >> >>",
		// 2: pages
		"<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>",
		// 3, 4: page 1, rotated
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + offsetMediaBox + "] /Rotate 90 /Contents 4 0 R >>",
		rawStream("", cornerMarkBlack),
		// 5, 6: page 2, upright, carrying the links
		"<< /Type /Page /Parent 2 0 R /MediaBox [" + offsetMediaBox + "] /Contents 6 0 R /Annots [11 0 R 12 0 R 13 0 R 14 0 R] >>",
		rawStream("", cornerMarkBlack),
		// 7, 8, 9: outline with a nested item
		"<< /Type /Outlines /First 8 0 R /Last 8 0 R /Count 2 >>",
		"<< /Title (Top) /Parent 7 0 R /First 9 0 R /Last 9 0 R /Count 1 /Dest [3 0 R /FitR 110 220 210 260] >>",
		"<< /Title (Child) /Parent 8 0 R /A << /S /GoTo /D [3 0 R /FitH 220] >> >>",
		// 10: named destinations
		"<< /Names [(nulltop) [3 0 R /XYZ 110 null null] (nullleft) [3 0 R /XYZ null 220 null] (onupright) [5 0 R /XYZ 5 6 0] (shared) 15 0 R] >>",
		// 11..14: links on page 2
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /Dest [3 0 R /XYZ 110 220 0] >>",
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /A << /S /GoTo /D [3 0 R /FitH 220] >> >>",
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /Dest [5 0 R /XYZ 5 6 0] >>",
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /Dest 15 0 R >>",
		// 15: one destination array used from two places
		"[3 0 R /XYZ 110 220 0]",
	})

	changed, err := UprightPages(ctx, in, out, nil)
	if err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	count := func(pattern string) int {
		return len(regexp.MustCompile(pattern).FindAll(data, -1))
	}
	tests := []struct {
		what    string
		pattern string
		want    int
	}{
		// (110, 220) -> (200, 200). Found in: the link's /Dest, the PDF 1.1
		// /Dests entry, and the shared array (once: it is one object, and a
		// second pass over it would move it again).
		{"/XYZ with both coordinates", `\d+ 0 R\s*/XYZ 200 200 0\s*\]`, 3},
		// A top of 220 becomes a left of 200: the link's GoTo action and
		// the nested outline item.
		{"/FitH turned into /FitV", `/FitV 200\s*\]`, 2},
		{"/FitBH turned into /FitBV", `/FitBV 200\s*\]`, 1},
		// The open action's left of 110 becomes a top of 200.
		{"/FitV turned into /FitH", `/FitH 200\s*\]`, 1},
		{"/FitR", `/FitR 200 100 240 200\s*\]`, 1},
		// Only a left was given; after a quarter turn it is only a top.
		{"/XYZ with only a left", `/XYZ null 200 null\s*\]`, 1},
		{"/XYZ with only a top", `/XYZ 200 null null\s*\]`, 1},
		// Destinations on the upright page: the link and the named one.
		{"destinations on the upright page", `/XYZ 5 6 0\s*\]`, 2},
		{"anything left in the old coordinates", `/XYZ 110 220|/FitH 220|/FitBH 220|/FitV 110|/FitR 110`, 0},
	}
	for _, tt := range tests {
		if got := count(tt.pattern); got != tt.want {
			t.Errorf("%s: found %d, want %d", tt.what, got, tt.want)
		}
	}

	if err := ValidatePDF(out, nil); err != nil {
		t.Errorf("pdfcpu rejects the output: %v", err)
	}
}

// A document with no rotated page has its destinations left alone.
func TestUprightPages_LeavesDestinationsOfUprightDocumentAlone(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	pagesPDF(t, in,
		[]testPage{{mediaBox: offsetMediaBox, extra: "/Annots [6 0 R]", content: cornerMarkBlack}},
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /Dest [4 0 R /XYZ 110 220 0] >>",
	)

	if _, err := UprightPages(ctx, in, out, nil); err != nil {
		t.Fatalf("UprightPages: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`/XYZ 110 220 0\s*\]`).Match(data) {
		t.Error("a destination on an upright page was changed")
	}
}
