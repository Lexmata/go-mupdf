package mupdf

import (
	"math"
	"os"
	"testing"
)

// Tests for PDF object creation and combined API operations.

func TestNewPDFObjectComplete(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	unsupportedPointer := new(int)

	testCases := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"nil", nil, false},
		{"bool_true", true, false},
		{"bool_false", false, false},
		{"int_zero", int(0), false},
		{"int_positive", int(42), false},
		{"int_negative", int(-42), false},
		{"int_max", int(2147483647), false},
		{"int_min", int(-2147483648), false},
		{"float_zero", float64(0.0), false},
		{"float_positive", float64(3.14159), false},
		{"float_negative", float64(-3.14159), false},
		{"float_very_small", float64(1e-10), false},
		{"float_very_large", float64(1e10), false},
		// PDF has no Inf literal, but NewPDFObject currently forwards Inf
		// to pdf_new_real without validation, so creation succeeds.
		// Candidate for future validation.
		{"float_infinity", math.Inf(1), false},
		{"float_neg_infinity", math.Inf(-1), false},
		{"string_empty", "", false},
		{"string_simple", "hello", false},
		{"string_unicode", "Hello 世界 🌍", false},
		{"string_special_chars", "\n\r\t\\\"/", false},
		{"string_null_bytes", "\x00\x01\x02", false},
		{"string_very_long", string(make([]byte, 10000)), false}, // 10KB string
		// Unsupported types must be rejected.
		{"unsupported_slice", []int{1, 2, 3}, true},
		{"unsupported_map", map[string]int{"a": 1}, true},
		{"unsupported_struct", struct{ X int }{42}, true},
		{"unsupported_channel", make(chan int), true},
		{"unsupported_function", func() {}, true},
		{"unsupported_complex", complex(1, 2), true},
		{"unsupported_pointer", unsupportedPointer, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := writer.NewPDFObject(tc.value)
			if tc.wantErr {
				if err == nil {
					obj.Drop()
					t.Errorf("Expected error for unsupported type %s", tc.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPDFObject(%s): %v", tc.name, err)
			}
			obj.Drop()
		})
	}
}

func TestSkipFunctionsWithShortMode(t *testing.T) {
	// This test specifically targets the skip paths that are only hit when testing.Short() is true
	// Since we can't control testing.Short() from within the test, we'll test the behavior
	// when the conditions are met in other ways

	// Test skipIfShort - we can only test the non-skip path here
	// The skip path requires go test -short
	skipIfShort(t)

	// Test skipIfCIorShort with various CI environment setups
	// This might hit different code paths based on environment detection
	skipIfCIorShort(t)
}

func TestCreateTestPDFExtensive(t *testing.T) {
	// Repeated createTestPDF calls must produce existing files.
	for i := 0; i < 3; i++ {
		pdf := createTestPDF(t)
		if pdf == "" {
			t.Errorf("createTestPDF %d returned empty path", i)
		}

		// Immediately verify the file exists and has content
		_, err := os.Stat(pdf)
		if err != nil {
			t.Errorf("PDF %d doesn't exist: %v", i, err)
		}
	}
}

func TestRequireMuPDFExtensive(t *testing.T) {
	// requireMuPDF must be safe to call repeatedly.
	for i := 0; i < 3; i++ {
		requireMuPDF(t)
	}
}

func TestAllCombinationsForCoverage(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Document operations in various combinations.
	pdfPath := createTestPDF(t)

	for i := 0; i < 3; i++ {
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open document %d: %v", i, err)
		}

		count := doc.CountPages()
		if count == 0 {
			t.Fatalf("Document %d: expected at least one page", i)
		}

		pdfDoc, err := doc.AsPDFDocument()
		if err != nil {
			t.Fatalf("Document %d: AsPDFDocument failed: %v", i, err)
		}
		if pdfCount := pdfDoc.CountPages(); pdfCount != count {
			t.Errorf("Document %d: PDF page count = %d, want %d", i, pdfCount, count)
		}

		pdfPage, err := pdfDoc.LoadPage(0)
		if err != nil {
			t.Fatalf("Document %d: PDF LoadPage failed: %v", i, err)
		}
		pdfPage.Bound()
		pdfPage.Close()

		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Document %d: LoadPage failed: %v", i, err)
		}
		page.Bound()

		text, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Document %d: ExtractText failed: %v", i, err)
		}
		if s1, s2 := text.String(), text.String(); s1 != s2 {
			t.Errorf("Document %d: inconsistent text: %d vs %d chars", i, len(s1), len(s2))
		}
		text.Close()
		page.Close()

		doc.Close()
	}

	// PDF writer operations: every page-addition variant must succeed.
	for i := 0; i < 2; i++ {
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create writer %d: %v", i, err)
		}

		adds := []struct {
			name string
			add  func(float64, float64) (*PDFPage, error)
			w, h float64
		}{
			{"AddPage", writer.AddPage, 612, 792},
			{"SimpleAddPage", writer.SimpleAddPage, 595, 842},
			{"ImprovedAddPage", writer.ImprovedAddPage, 420, 595},
			{"FixedAddPage", writer.FixedAddPage, 297, 420},
		}

		for _, a := range adds {
			page, err := a.add(a.w, a.h)
			if err != nil {
				t.Fatalf("Writer %d: %s failed: %v", i, a.name, err)
			}
			page.Close()
		}

		if got := writer.DebugCountPages(); got != len(adds) {
			t.Errorf("Writer %d: page count = %d, want %d", i, got, len(adds))
		}

		for j := 0; j < 3; j++ {
			obj, err := writer.NewPDFObject(j)
			if err != nil {
				t.Errorf("Writer %d: NewPDFObject(%d) failed: %v", i, j, err)
				continue
			}
			obj.Drop()
		}

		writer.Close()
	}
}

func TestErrorPathsComprehensive(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Invalid files must fail to open through both APIs.
	invalidFiles := []string{
		"",
		"non-existent.pdf",
		"/invalid/path/file.pdf",
		"not-a-pdf.txt",
	}

	for _, file := range invalidFiles {
		if _, err := OpenDocument(ctx, file); err == nil {
			t.Errorf("OpenDocument(%q): expected error, got nil", file)
		}

		if _, err := OpenPDFDocument(ctx, file); err == nil {
			t.Errorf("OpenPDFDocument(%q): expected error, got nil", file)
		}
	}
}
