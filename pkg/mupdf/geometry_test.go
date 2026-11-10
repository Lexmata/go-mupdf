package mupdf

import (
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGetPageGeometry(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "geom.pdf")
	pagesPDF(t, path, []testPage{
		{mediaBox: "0 0 612 792"},
		{mediaBox: offsetMediaBox, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, extra: "/Rotate 180 /CropBox [110 120 210 320]"},
		{mediaBox: "0 0 200.5 300.25", extra: "/Rotate 270"},
		{mediaBox: "0 0 100 100", extra: "/Rotate -90"},
		{mediaBox: "0 0 100 100", extra: "/Rotate 450"},
		{mediaBox: "0 0 100 100", extra: "/Rotate 45"},
		{mediaBox: "0 0 100 100", extra: "/Rotate 44"},
		{mediaBox: "0 0 100 100", extra: "/Rotate 100"},
		{mediaBox: "300 400 0 0"},
	})

	got, err := GetPageGeometry(ctx, path)
	if err != nil {
		t.Fatalf("GetPageGeometry: %v", err)
	}

	want := []PageGeometry{
		{MediaBox: Rect{0, 0, 612, 792}, HasMediaBox: true, Rotation: 0},
		{MediaBox: Rect{10, 20, 310, 420}, HasMediaBox: true, Rotation: 90},
		// The CropBox must not leak into the reported box.
		{MediaBox: Rect{10, 20, 310, 420}, HasMediaBox: true, Rotation: 180},
		{MediaBox: Rect{0, 0, 200.5, 300.25}, HasMediaBox: true, Rotation: 270},
		{MediaBox: Rect{0, 0, 100, 100}, HasMediaBox: true, Rotation: 270},
		{MediaBox: Rect{0, 0, 100, 100}, HasMediaBox: true, Rotation: 90},
		// Not a right angle: malformed, snapped to the nearest one (45
		// rounds up), as the renderer does.
		{MediaBox: Rect{0, 0, 100, 100}, HasMediaBox: true, Rotation: 90},
		{MediaBox: Rect{0, 0, 100, 100}, HasMediaBox: true, Rotation: 0},
		{MediaBox: Rect{0, 0, 100, 100}, HasMediaBox: true, Rotation: 90},
		// Corners given in reverse order are normalised.
		{MediaBox: Rect{0, 0, 300, 400}, HasMediaBox: true, Rotation: 0},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d pages, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("page %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGetPageGeometry_InheritedAndMissingMediaBox(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()

	inherited := filepath.Join(dir, "inherited.pdf")
	writeRawPDF(t, inherited, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 420 595] /Rotate 90 >>",
		"<< /Type /Page /Parent 2 0 R >>",
	})
	got, err := GetPageGeometry(ctx, inherited)
	if err != nil {
		t.Fatalf("inherited: %v", err)
	}
	want := PageGeometry{MediaBox: Rect{0, 0, 420, 595}, HasMediaBox: true, Rotation: 90}
	if len(got) != 1 || got[0] != want {
		t.Errorf("inherited: got %+v, want [%+v]", got, want)
	}

	for name, box := range map[string]string{
		"missing":    "",
		"degenerate": "/MediaBox [0 0 0 792]",
		"too short":  "/MediaBox [0 0 612]",
		"not array":  "/MediaBox 612",
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".pdf")
		writeRawPDF(t, path, []string{
			"<< /Type /Catalog /Pages 2 0 R >>",
			"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
			"<< /Type /Page /Parent 2 0 R " + box + " >>",
		})
		t.Run(name, func(t *testing.T) {
			got, err := GetPageGeometry(ctx, path)
			if err != nil {
				t.Fatalf("GetPageGeometry: %v", err)
			}
			want := PageGeometry{MediaBox: Rect{0, 0, 612, 792}, HasMediaBox: false, Rotation: 0}
			if len(got) != 1 || got[0] != want {
				t.Errorf("got %+v, want the US Letter fallback", got)
			}
		})
	}
}

func TestGetPageGeometry_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()

	if _, err := GetPageGeometry(ctx, filepath.Join(dir, "missing.pdf")); err == nil {
		t.Error("want error for a missing file")
	}

	garbage := filepath.Join(dir, "garbage.pdf")
	writeFile(t, garbage, []byte("this is not a PDF at all"))
	if _, err := GetPageGeometry(ctx, garbage); err == nil {
		t.Error("want error for a file that is not a PDF")
	}
}

func TestPageGeometry_Methods(t *testing.T) {
	tests := []struct {
		name          string
		g             PageGeometry
		width, height float64
		upright       bool
	}{
		{"letter", PageGeometry{MediaBox: Rect{0, 0, 612, 792}}, 612, 792, true},
		{"offset origin", PageGeometry{MediaBox: Rect{10, 20, 310, 420}, Rotation: 90}, 300, 400, false},
		{"reversed corners", PageGeometry{MediaBox: Rect{310, 420, 10, 20}, Rotation: 180}, 300, 400, false},
		{"zero value", PageGeometry{}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.Width(); got != tt.width {
				t.Errorf("Width() = %v, want %v", got, tt.width)
			}
			if got := tt.g.Height(); got != tt.height {
				t.Errorf("Height() = %v, want %v", got, tt.height)
			}
			if got := tt.g.IsUpright(); got != tt.upright {
				t.Errorf("IsUpright() = %v, want %v", got, tt.upright)
			}
		})
	}
}

// cpdfGeometry reads MediaBox size and rotation per page from
// `cpdf -page-info`, the tool GetPageGeometry replaces.
func cpdfGeometry(t *testing.T, cpdf, path string) []PageGeometry {
	t.Helper()
	out := runTool(t, cpdf, "-page-info", path)

	numbers := regexp.MustCompile(`-?[0-9]+(\.[0-9]+)?`)
	var pages []PageGeometry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Page ") && strings.HasSuffix(line, ":"):
			pages = append(pages, PageGeometry{})
		case len(pages) == 0:
		case strings.HasPrefix(line, "MediaBox:"):
			v := numbers.FindAllString(strings.TrimPrefix(line, "MediaBox:"), -1)
			if len(v) != 4 {
				t.Fatalf("unparseable cpdf MediaBox line %q", line)
			}
			f := make([]float64, 4)
			for i := range v {
				f[i], _ = strconv.ParseFloat(v[i], 64)
			}
			pages[len(pages)-1].MediaBox = Rect{f[0], f[1], f[2], f[3]}
		case strings.HasPrefix(line, "Rotation:"):
			r, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Rotation:")))
			if err != nil {
				t.Fatalf("unparseable cpdf Rotation line %q", line)
			}
			pages[len(pages)-1].Rotation = r
		}
	}
	return pages
}

func TestGetPageGeometry_MatchesCpdf(t *testing.T) {
	cpdf := requireTool(t, "cpdf")
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "geom.pdf")
	pagesPDF(t, path, []testPage{
		{mediaBox: "0 0 612 792"},
		{mediaBox: offsetMediaBox, extra: "/Rotate 90"},
		{mediaBox: offsetMediaBox, extra: "/Rotate 180 /CropBox [110 120 210 320]"},
		{mediaBox: "0 0 200.5 300.25", extra: "/Rotate 270"},
	})

	got, err := GetPageGeometry(ctx, path)
	if err != nil {
		t.Fatalf("GetPageGeometry: %v", err)
	}
	want := cpdfGeometry(t, cpdf, path)
	if len(got) != len(want) {
		t.Fatalf("got %d pages, cpdf reports %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i].Width()-want[i].Width()) > 0.01 ||
			math.Abs(got[i].Height()-want[i].Height()) > 0.01 ||
			got[i].Rotation != want[i].Rotation {
			t.Errorf("page %d: got %.2fx%.2f rot %d, cpdf reports %.2fx%.2f rot %d", i,
				got[i].Width(), got[i].Height(), got[i].Rotation,
				want[i].Width(), want[i].Height(), want[i].Rotation)
		}
	}
}

// Whatever a malformed /Rotate is snapped to, the renderer must agree: the
// reported rotation is what decides whether the image is portrait or
// landscape.
func TestGetPageGeometry_AgreesWithRenderPage(t *testing.T) {
	ctx := newTestContext(t)
	for _, rotate := range []int{0, 44, 45, 90, 100, 135, 180, 200, 270, 315, 350, 360, 450, -90, -100} {
		t.Run(strconv.Itoa(rotate), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.pdf")
			pagesPDF(t, path, []testPage{{mediaBox: "0 0 200 300", extra: "/Rotate " + strconv.Itoa(rotate)}})

			geom, err := GetPageGeometry(ctx, path)
			if err != nil {
				t.Fatalf("GetPageGeometry: %v", err)
			}
			img := renderTestPage(t, ctx, path, 0, 72)
			landscape := img.Bounds().Dx() > img.Bounds().Dy()
			if want := geom[0].Rotation == 90 || geom[0].Rotation == 270; landscape != want {
				t.Errorf("geometry says %d but the render is %v", geom[0].Rotation, img.Bounds().Size())
			}
		})
	}
}

// The page count can be a number the file simply declares. It must not be
// trusted to size an allocation.
func TestGetPageGeometry_ImplausiblePageCount(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "liar.pdf")
	writeRawPDF(t, path, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 2000000000 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
	})

	pages, err := GetPageGeometry(ctx, path)
	if err == nil && len(pages) > 3 {
		t.Fatalf("a 3-object file was reported as having %d pages", len(pages))
	}
}

// An extreme /Rotate must not overflow while being normalised.
func TestGetPageGeometry_ExtremeRotation(t *testing.T) {
	ctx := newTestContext(t)
	for _, rotate := range []string{"-2147483648", "2147483647", "-2147483647"} {
		t.Run(rotate, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.pdf")
			pagesPDF(t, path, []testPage{{mediaBox: "0 0 200 300", extra: "/Rotate " + rotate}})

			geom, err := GetPageGeometry(ctx, path)
			if err != nil {
				t.Fatalf("GetPageGeometry: %v", err)
			}
			switch geom[0].Rotation {
			case 0, 90, 180, 270:
			default:
				t.Errorf("Rotation = %d, want a right angle", geom[0].Rotation)
			}
		})
	}
}
