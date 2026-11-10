package mupdf

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// twoPageObjects is a healthy two-page document with text on each page.
func twoPageObjects() []string {
	return []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [4 0 R 6 0 R] /Count 2 >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents 5 0 R >>",
		rawStream("", "BT /F1 24 Tf 72 700 Td (First page) Tj ET"),
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents 7 0 R >>",
		rawStream("", "BT /F1 24 Tf 72 700 Td (Second page) Tj ET"),
		// Nothing refers to this object.
		rawStream("", "unreferenced-object-payload"),
	}
}

// damagedPDF writes the two-page document with its cross-reference table
// broken in the named way.
func damagedPDF(t *testing.T, path, how string) {
	t.Helper()
	data := rawPDFBytes(twoPageObjects())
	xref := bytes.LastIndex(data, []byte("xref\n0 "))
	if xref < 0 {
		t.Fatal("fixture has no xref")
	}
	switch how {
	case "bad startxref":
		i := bytes.LastIndex(data, []byte("startxref\n"))
		data = append(data[:i:i], []byte("startxref\n7\n%%EOF\n")...)
	case "truncated before xref":
		data = data[:xref]
	case "wrong offsets":
		// Shift every object by inserting bytes after the header, leaving
		// the xref pointing at the old positions.
		header := bytes.IndexByte(data, '\n') + 1
		shifted := append([]byte{}, data[:header]...)
		shifted = append(shifted, []byte("% padding that invalidates every offset\n")...)
		data = append(shifted, data[header:]...)
	default:
		t.Fatalf("unknown damage %q", how)
	}
	writeFile(t, path, data)
}

func TestCheckPDF_CleanFile(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "clean.pdf")
	writeRawPDF(t, path, twoPageObjects())

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("CheckPDF: %v", err)
	}
	if !res.Clean() {
		t.Errorf("want a clean result, got %+v", res)
	}
}

// A free cross-reference entry is not damage. Real files have them wherever
// an object was deleted, and linearised files leave gaps.
func TestCheckPDF_FreeXrefEntriesAreClean(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "free.pdf")
	data := rawPDFBytes(twoPageObjects())

	// Mark the last object (which nothing refers to) as free. Each xref
	// entry is exactly 20 bytes, so the table stays well formed.
	trailer := bytes.LastIndex(data, []byte("trailer"))
	entry := trailer - 20
	if !bytes.HasSuffix(data[entry:trailer], []byte(" 00000 n \n")) {
		t.Fatalf("fixture problem: last xref entry is %q", data[entry:trailer])
	}
	copy(data[entry:trailer], "0000000000 00001 f \n")
	writeFile(t, path, data)

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("a free xref entry must not be an error: %v", err)
	}
	if !res.Clean() {
		t.Errorf("want a clean result, got %+v", res)
	}
}

// In an encrypted file /Length counts encrypted bytes, which for AES are
// more than the decrypted bytes MuPDF hands back. That is not truncation.
func TestCheckPDF_EncryptedFileIsClean(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.pdf")
	encrypted := filepath.Join(dir, "encrypted.pdf")
	writeRawPDF(t, plain, twoPageObjects())
	// An empty user password: the file opens without one, as most
	// owner-password-only PDFs do.
	if err := EncryptPDF(plain, encrypted, "", "owner", model.PermissionsPrint, nil); err != nil {
		t.Fatalf("EncryptPDF: %v", err)
	}
	data, err := os.ReadFile(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("/Encrypt")) {
		t.Fatal("fixture problem: the file is not encrypted")
	}

	res, err := CheckPDF(ctx, encrypted)
	if err != nil {
		t.Fatalf("CheckPDF: %v", err)
	}
	if !res.Clean() {
		t.Errorf("a sound encrypted file was reported as %+v", res)
	}
}

// The warning buffer is a fixed size inside a struct in Go memory. When the
// messages outgrow it, whole messages are dropped and the count keeps
// going; nothing is written past the end.
func TestCheckPDF_TruncatesWarnings(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "noisy.pdf")
	objects := twoPageObjects()
	const broken = 300
	for i := 0; i < broken; i++ {
		objects = append(objects, "<< /Length 99999999 >>\nstream\nBT ET\nendstream\n")
	}
	writeRawPDF(t, path, objects)

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("CheckPDF: %v", err)
	}
	if res.WarningCount < broken {
		t.Errorf("WarningCount = %d, want at least %d", res.WarningCount, broken)
	}
	if len(res.Warnings) >= res.WarningCount {
		t.Errorf("kept %d of %d warnings: the fixture did not overflow the buffer", len(res.Warnings), res.WarningCount)
	}
	if total := len(strings.Join(res.Warnings, "\n")); total >= 4096 {
		t.Errorf("kept %d bytes of warnings, more than the 4096-byte buffer holds", total)
	}
	for i, w := range res.Warnings {
		if !strings.Contains(w, "/Length of 99999999") {
			t.Errorf("warning %d was cut short: %q", i, w)
		}
	}
}

// CheckPDF points MuPDF's warning and error callbacks at a struct that goes
// out of scope when it returns. If they were not put back, the next warning
// from any operation on the same context would be written into freed
// memory. MuPDF's default callbacks print to stderr, so the check is made in
// a child process: after CheckPDF, on its success path and on its failure
// path, a repair of a damaged file must print its warnings there again.
func TestCheckPDF_RestoresCallbacks(t *testing.T) {
	if mode := os.Getenv("GOMUPDF_CALLBACK_CHILD"); mode != "" {
		runCallbackChild(t, mode)
		return
	}
	for _, mode := range []string{"after-clean-check", "after-failed-check"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestCheckPDF_RestoresCallbacks$")
			cmd.Env = append(os.Environ(), "GOMUPDF_CALLBACK_CHILD="+mode)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("child failed: %v\n%s", err, stderr.String())
			}
			if !strings.Contains(stderr.String(), "repair") {
				t.Errorf("MuPDF's own warning callback did not run after CheckPDF; child stderr:\n%s", stderr.String())
			}
		})
	}
}

func runCallbackChild(t *testing.T, mode string) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	damaged := filepath.Join(dir, "damaged.pdf")
	damagedPDF(t, damaged, "bad startxref")

	switch mode {
	case "after-clean-check":
		clean := filepath.Join(dir, "clean.pdf")
		writeRawPDF(t, clean, twoPageObjects())
		if _, err := CheckPDF(ctx, clean); err != nil {
			t.Fatalf("CheckPDF: %v", err)
		}
	case "after-failed-check":
		garbage := filepath.Join(dir, "garbage.pdf")
		writeFile(t, garbage, []byte("this is not a PDF at all"))
		if _, err := CheckPDF(ctx, garbage); err == nil {
			t.Fatal("want an error from CheckPDF")
		}
	default:
		t.Fatalf("unknown mode %q", mode)
	}

	// This makes MuPDF warn. With the callbacks restored the warning goes
	// to stderr; with a stale callback it would go nowhere, or crash.
	if err := RepairPDF(ctx, damaged, filepath.Join(dir, "out.pdf"), nil); err != nil {
		t.Fatalf("RepairPDF: %v", err)
	}
}

// One object that cannot be parsed, among sound ones, makes a file damaged,
// not unreadable: it is reported, not returned as an error.
func TestCheckPDF_BrokenObjectIsNotFatal(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "broken-object.pdf")
	objects := twoPageObjects()
	objects[7] = "<< /Unclosed [ (\n"
	writeRawPDF(t, path, objects)

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("one bad object must not make the whole file an error: %v", err)
	}
	if res.Clean() {
		t.Errorf("a file with an unparseable object was reported clean")
	}
	if n, err := PageCount(ctx, path); err != nil || n != 2 {
		t.Errorf("fixture problem: the file should still open with 2 pages, got %d (%v)", n, err)
	}
}

func TestCheckPDFContext_Cancelled(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "clean.pdf")
	writeRawPDF(t, path, twoPageObjects())

	goCtx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := CheckPDFContext(goCtx, ctx, path)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v (result %+v)", err, res)
	}

	// A context that is never cancelled changes nothing.
	res, err = CheckPDFContext(context.Background(), ctx, path)
	if err != nil || !res.Clean() {
		t.Fatalf("uncancelled check: %+v, %v", res, err)
	}
	live, stop := context.WithCancel(context.Background())
	defer stop()
	res, err = CheckPDFContext(live, ctx, path)
	if err != nil || !res.Clean() {
		t.Fatalf("check with a live cancellable context: %+v, %v", res, err)
	}
}

func TestCheckPDF_DamagedButRecoverable(t *testing.T) {
	ctx := newTestContext(t)
	for _, how := range []string{"bad startxref", "truncated before xref", "wrong offsets"} {
		t.Run(how, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "damaged.pdf")
			damagedPDF(t, path, how)

			res, err := CheckPDF(ctx, path)
			if err != nil {
				t.Fatalf("a recoverable file must not be an error: %v", err)
			}
			if !res.Repaired {
				t.Errorf("want Repaired, got %+v", res)
			}
			if res.Clean() {
				t.Error("a repaired file must not be reported clean")
			}
			if res.WarningCount < len(res.Warnings) {
				t.Errorf("WarningCount %d is below len(Warnings) %d", res.WarningCount, len(res.Warnings))
			}
		})
	}
}

func TestCheckPDF_Unreadable(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()

	garbage := filepath.Join(dir, "garbage.pdf")
	writeFile(t, garbage, []byte("this is not a PDF at all"))
	empty := filepath.Join(dir, "empty.pdf")
	writeFile(t, empty, nil)

	for name, path := range map[string]string{
		"missing": filepath.Join(dir, "missing.pdf"),
		"garbage": garbage,
		"empty":   empty,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := CheckPDF(ctx, path)
			if err == nil {
				t.Fatalf("want an error, got result %+v", res)
			}
			if res != nil {
				t.Error("want a nil result alongside the error")
			}
			var mupdfErr Error
			if !errors.As(err, &mupdfErr) {
				t.Errorf("error %v does not wrap mupdf.Error", err)
			}
		})
	}
}

// A stream whose /Length runs past the end of the file is what truncation
// looks like. MuPDF reads it without complaint, so CheckPDF measures it.
func TestCheckPDF_ReportsShortStream(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "stream.pdf")
	objects := twoPageObjects()
	objects[4] = "<< /Length 99999999 >>\nstream\nBT ET\nendstream\n"
	writeRawPDF(t, path, objects)

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("CheckPDF: %v", err)
	}
	if res.Clean() {
		t.Fatalf("a stream with an impossible /Length went unnoticed: %+v", res)
	}
	if joined := strings.Join(res.Warnings, "\n"); !strings.Contains(joined, "/Length") {
		t.Errorf("warnings do not name the short stream: %q", joined)
	}
}

// What MuPDF recovered from is reported, not printed and lost.
func TestCheckPDF_CapturesRecoveredErrors(t *testing.T) {
	ctx := newTestContext(t)
	path := filepath.Join(t.TempDir(), "damaged.pdf")
	damagedPDF(t, path, "bad startxref")

	res, err := CheckPDF(ctx, path)
	if err != nil {
		t.Fatalf("CheckPDF: %v", err)
	}
	if len(res.Warnings) == 0 || res.WarningCount == 0 {
		t.Fatalf("no messages captured for a damaged file: %+v", res)
	}
	if joined := strings.Join(res.Warnings, "\n"); !strings.Contains(joined, "xref") {
		t.Errorf("warnings do not mention the xref: %q", joined)
	}
}

// The warning callback is swapped in for the duration of a check; warnings
// from one call must not show up in the next.
func TestCheckPDF_DoesNotLeakStateBetweenCalls(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	damaged := filepath.Join(dir, "damaged.pdf")
	damagedPDF(t, damaged, "bad startxref")
	clean := filepath.Join(dir, "clean.pdf")
	writeRawPDF(t, clean, twoPageObjects())

	if res, err := CheckPDF(ctx, damaged); err != nil || res.Clean() {
		t.Fatalf("damaged: res %+v, err %v", res, err)
	}
	res, err := CheckPDF(ctx, clean)
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if !res.Clean() {
		t.Errorf("clean file reported as %+v after checking a damaged one", res)
	}
}

func TestCheckResult_Clean(t *testing.T) {
	var nilResult *CheckResult
	tests := []struct {
		name string
		res  *CheckResult
		want bool
	}{
		{"nil", nilResult, false},
		{"zero value", &CheckResult{}, true},
		{"repaired", &CheckResult{Repaired: true}, false},
		{"warnings", &CheckResult{WarningCount: 1, Warnings: []string{"w"}}, false},
	}
	for _, tt := range tests {
		if got := tt.res.Clean(); got != tt.want {
			t.Errorf("%s: Clean() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRepairPDF_FixesDamagedFiles(t *testing.T) {
	ctx := newTestContext(t)
	for _, how := range []string{"bad startxref", "truncated before xref", "wrong offsets"} {
		t.Run(how, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "damaged.pdf")
			out := filepath.Join(dir, "repaired.pdf")
			damagedPDF(t, in, how)

			if err := RepairPDF(ctx, in, out, nil); err != nil {
				t.Fatalf("RepairPDF: %v", err)
			}

			res, err := CheckPDF(ctx, out)
			if err != nil {
				t.Fatalf("CheckPDF on repaired file: %v", err)
			}
			if !res.Clean() {
				t.Errorf("repaired file is not clean: %+v", res)
			}
			if n, err := PageCount(ctx, out); err != nil || n != 2 {
				t.Errorf("repaired file has %d pages (err %v), want 2", n, err)
			}
			if got := pageText(t, ctx, out, 1); !strings.Contains(got, "Second page") {
				t.Errorf("page 2 text %q lost its content", got)
			}
			// An opinion from outside MuPDF, available wherever the tests
			// run.
			if err := ValidatePDF(out, nil); err != nil {
				t.Errorf("pdfcpu rejects the repaired file: %v", err)
			}
		})
	}
}

func TestRepairPDF_DropsUnreferencedObjects(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeRawPDF(t, in, twoPageObjects())

	if err := RepairPDF(ctx, in, out, nil); err != nil {
		t.Fatalf("RepairPDF: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("unreferenced-object-payload")) {
		t.Error("an object nothing refers to survived the rewrite")
	}
	if !bytes.Contains(data, []byte("First page")) {
		t.Error("page content was not copied through unchanged")
	}
}

func TestRepairPDF_ObjectStreams(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	writeRawPDF(t, in, twoPageObjects())

	plain := filepath.Join(dir, "plain.pdf")
	if err := RepairPDF(ctx, in, plain, &RepairOptions{}); err != nil {
		t.Fatalf("plain: %v", err)
	}
	packed := filepath.Join(dir, "packed.pdf")
	if err := RepairPDF(ctx, in, packed, &RepairOptions{ObjectStreams: true}); err != nil {
		t.Fatalf("packed: %v", err)
	}

	plainData, _ := os.ReadFile(plain)
	packedData, _ := os.ReadFile(packed)
	if bytes.Contains(plainData, []byte("/ObjStm")) {
		t.Error("default output contains an object stream")
	}
	if !bytes.Contains(plainData, []byte("/Type/Catalog")) && !bytes.Contains(plainData, []byte("/Type /Catalog")) {
		t.Error("default output does not hold the catalog as a plain object")
	}
	if !bytes.Contains(packedData, []byte("/ObjStm")) {
		t.Error("ObjectStreams output contains no object stream")
	}

	// Flattening: a file that uses object streams comes out without them.
	flat := filepath.Join(dir, "flat.pdf")
	if err := RepairPDF(ctx, packed, flat, nil); err != nil {
		t.Fatalf("flatten: %v", err)
	}
	flatData, _ := os.ReadFile(flat)
	if bytes.Contains(flatData, []byte("/ObjStm")) {
		t.Error("object streams survived a default rewrite")
	}
	if got := pageText(t, ctx, flat, 0); !strings.Contains(got, "First page") {
		t.Errorf("flattened file lost its text: %q", got)
	}
}

func TestRepairPDF_DeterministicID(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	writeRawPDF(t, in, twoPageObjects())
	opts := &RepairOptions{DeterministicID: true}

	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	if err := RepairPDF(ctx, in, a, opts); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := RepairPDF(ctx, in, b, opts); err != nil {
		t.Fatalf("second: %v", err)
	}
	aData, _ := os.ReadFile(a)
	bData, _ := os.ReadFile(b)
	if !bytes.Equal(aData, bData) {
		t.Error("two repairs of the same input differ byte for byte")
	}
	if !bytes.Contains(aData, []byte("/ID")) {
		t.Error("output has no /ID")
	}

	// A different input must get a different identifier.
	other := filepath.Join(dir, "other.pdf")
	objects := twoPageObjects()
	objects[4] = rawStream("", "BT /F1 24 Tf 72 700 Td (Changed) Tj ET")
	writeRawPDF(t, other, objects)
	c := filepath.Join(dir, "c.pdf")
	if err := RepairPDF(ctx, other, c, opts); err != nil {
		t.Fatalf("other: %v", err)
	}
	if idOf(t, a) == idOf(t, c) {
		t.Error("different inputs produced the same /ID")
	}
}

// An existing permanent identifier (the first half of /ID) is kept.
func TestRepairPDF_DeterministicIDKeepsPermanentHalf(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	data := rawPDFBytes(twoPageObjects())
	data = bytes.Replace(data, []byte("/Root 1 0 R >>"),
		[]byte("/Root 1 0 R /ID [<00112233445566778899AABBCCDDEEFF> <FFEEDDCCBBAA99887766554433221100>] >>"), 1)
	writeFile(t, in, data)

	out := filepath.Join(dir, "out.pdf")
	if err := RepairPDF(ctx, in, out, &RepairOptions{DeterministicID: true}); err != nil {
		t.Fatalf("RepairPDF: %v", err)
	}
	id := strings.ToUpper(idOf(t, out))
	if !strings.Contains(id, "00112233445566778899AABBCCDDEEFF") {
		t.Errorf("permanent identifier was not kept: %s", id)
	}
	if strings.Contains(id, "FFEEDDCCBBAA99887766554433221100") {
		t.Errorf("changing identifier was not replaced: %s", id)
	}
}

// idOf returns the raw text of the trailer /ID array.
func idOf(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	i := bytes.LastIndex(data, []byte("/ID"))
	if i < 0 {
		t.Fatalf("%s has no /ID", path)
	}
	end := bytes.IndexByte(data[i:], ']')
	if end < 0 {
		t.Fatalf("%s has an unterminated /ID", path)
	}
	return string(data[i : i+end+1])
}

func TestRepairPDF_Errors(t *testing.T) {
	ctx := newTestContext(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.pdf")
	writeRawPDF(t, in, twoPageObjects())
	garbage := filepath.Join(dir, "garbage.pdf")
	writeFile(t, garbage, []byte("this is not a PDF at all"))
	out := filepath.Join(dir, "out.pdf")

	tests := []struct {
		name    string
		in, out string
		opts    *RepairOptions
		wantErr string
	}{
		{"output equals input", in, in, nil, "must differ"},
		{"empty input", "", out, nil, "must not be empty"},
		{"empty output", in, "", nil, "must not be empty"},
		{"missing input", filepath.Join(dir, "missing.pdf"), out, nil, "cannot open"},
		{"missing input, deterministic", filepath.Join(dir, "missing.pdf"), out, &RepairOptions{DeterministicID: true}, "no such file"},
		{"unrecoverable input", garbage, out, nil, "RepairPDF"},
		{"unwritable output", in, filepath.Join(dir, "no-such-dir", "out.pdf"), nil, "RepairPDF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RepairPDF(ctx, tt.in, tt.out, tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
	assertNoStrayFiles(t, dir, "in.pdf", "garbage.pdf")

	// The input must be intact after a refused in-place repair.
	if n, err := PageCount(ctx, in); err != nil || n != 2 {
		t.Errorf("input damaged by refused repair: %d pages, err %v", n, err)
	}
}

// RepairPDF replaces a plain `qpdf in out` rewrite; qpdf must accept the
// result without warnings.
func TestRepairPDF_OutputPassesQpdfCheck(t *testing.T) {
	qpdf := requireTool(t, "qpdf")
	ctx := newTestContext(t)
	for _, how := range []string{"bad startxref", "truncated before xref", "wrong offsets"} {
		t.Run(how, func(t *testing.T) {
			dir := t.TempDir()
			in := filepath.Join(dir, "damaged.pdf")
			out := filepath.Join(dir, "repaired.pdf")
			damagedPDF(t, in, how)
			if err := RepairPDF(ctx, in, out, nil); err != nil {
				t.Fatalf("RepairPDF: %v", err)
			}
			// runTool fails the test on any non-zero exit, including qpdf's
			// 3 for warnings.
			runTool(t, qpdf, "--check", out)
		})
	}
}

func TestFileDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	content := []byte("some bytes to hash")
	writeFile(t, path, content)

	got, err := fileDigest(path)
	if err != nil {
		t.Fatalf("fileDigest: %v", err)
	}
	want := md5.Sum(content)
	if !bytes.Equal(got, want[:]) {
		t.Errorf("digest %x, want %x", got, want)
	}
	if len(got) != 16 {
		t.Errorf("digest is %d bytes, want 16", len(got))
	}

	if _, err := fileDigest(filepath.Join(dir, "missing.bin")); err == nil {
		t.Error("want an error for a missing file")
	}
}
