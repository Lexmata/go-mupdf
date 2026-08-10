package mupdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestErrorHandlingExtended tests various error conditions in the library
func TestErrorHandlingExtended(t *testing.T) {
	requireMuPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Test cases for error handling
	t.Run("OpenNonExistentFile", func(t *testing.T) {
		_, err := OpenDocument(ctx, "this_file_does_not_exist.pdf")
		if err == nil {
			t.Error("Expected error when opening non-existent file")
		}
	})

	t.Run("OpenInvalidFile", func(t *testing.T) {
		// Create a temporary file with invalid content
		tempFile, err := os.CreateTemp("", "invalid.pdf")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Write invalid content
		_, err = tempFile.WriteString("This is not a valid PDF file content")
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tempFile.Close()

		// Try to open the invalid file
		_, err = OpenDocument(ctx, tempFile.Name())
		if err == nil {
			t.Error("Expected error when opening invalid PDF file")
		}
	})

	t.Run("LoadInvalidPageNumber", func(t *testing.T) {
		// Create a valid PDF
		pdfPath := createTestPDF(t)

		// Open document
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open document: %v", err)
		}
		defer doc.Close()

		// Try to load invalid page numbers
		_, err = doc.LoadPage(-1)
		if err == nil {
			t.Error("Expected error when loading negative page number")
		}

		_, err = doc.LoadPage(999)
		if err == nil {
			t.Error("Expected error when loading out-of-bounds page number")
		}
	})

	t.Run("SaveToInvalidLocation", func(t *testing.T) {
		// Create a PDF writer
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer: %v", err)
		}
		defer writer.Close()

		// Add a page
		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}

		// Try to save to an invalid location
		err = writer.Save("/non/existent/directory/test.pdf")
		if err == nil {
			t.Error("Expected error when saving to invalid location")
		}
	})

	t.Run("InvalidPDFConversion", func(t *testing.T) {
		// Create a temporary file with invalid content
		tempFile, err := os.CreateTemp("", "invalid.pdf")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Write invalid content
		_, err = tempFile.WriteString("%PDF-1.4\nThis is not a valid PDF file")
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tempFile.Close()

		// Try to open the invalid file
		doc, err := OpenDocument(ctx, tempFile.Name())
		if err == nil {
			// If we can open it (some PDF parsers are very forgiving)
			defer doc.Close()

			// Try to convert to PDF document
			_, err = doc.AsPDFDocument()
			if err == nil {
				t.Error("Expected error when converting invalid document to PDF")
			}
		}
	})
}

// TestErrorRecovery tests recovery from error conditions
func TestErrorRecovery(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	t.Run("RecoverFromInvalidFile", func(t *testing.T) {
		// Try to open an invalid file
		_, err := OpenDocument(ctx, "non_existent.pdf")
		if err == nil {
			t.Error("Expected error when opening non-existent file")
		}

		// Now try to open a valid file with the same context
		pdfPath := createTestPDF(t)
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open valid document after error: %v", err)
		}
		defer doc.Close()

		// Verify the document is valid
		pageCount := doc.CountPages()
		if pageCount != 1 {
			t.Errorf("Expected 1 page, got %d", pageCount)
		}
	})

	t.Run("RecoverFromInvalidPage", func(t *testing.T) {
		// Create a valid PDF
		pdfPath := createTestPDF(t)

		// Open document
		doc, err := OpenDocument(ctx, pdfPath)
		if err != nil {
			t.Fatalf("Failed to open document: %v", err)
		}
		defer doc.Close()

		// Try to load an invalid page
		_, err = doc.LoadPage(999)
		if err == nil {
			t.Error("Expected error when loading out-of-bounds page number")
		}

		// Now try to load a valid page
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load valid page after error: %v", err)
		}
		defer page.Close()

		// Verify the page is valid
		bounds := page.Bound()
		if bounds.X0 >= bounds.X1 || bounds.Y0 >= bounds.Y1 {
			t.Errorf("Invalid page bounds: %+v", bounds)
		}
	})
}

// TestCorruptedPDF tests handling of malformed PDF files. Each input has a
// deterministic expected outcome: mildly corrupt files are repaired by MuPDF,
// while unparseable garbage must fail to open. (Non-PDF garbage bytes are
// also asserted to fail in TestErrorHandlingExtended/OpenInvalidFile.)
func TestCorruptedPDF(t *testing.T) {
	requireMuPDF(t)
	skipIfShort(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)

	// A structurally complete 1-page PDF whose startxref value is garbage.
	// MuPDF's repair pass rebuilds the xref table from the object headers,
	// so this document deterministically opens with exactly 1 page.
	repairableContent := `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 0 >>
stream
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000190 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
CORRUPTED_DATA_HERE
%%EOF`

	testCases := []struct {
		name      string
		filename  string
		content   string
		wantOpen  bool
		wantPages int
	}{
		{
			name:      "RepairableXrefCorruption",
			filename:  "corrupted_xref.pdf",
			content:   repairableContent,
			wantOpen:  true,
			wantPages: 1,
		},
		{
			name:     "GarbageBytesWithPDFHeader",
			filename: "corrupted_garbage.pdf",
			content:  "%PDF-1.4\n\x00\x01\x02garbage bytes with no objects, xref, or trailer",
			wantOpen: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pdfPath := filepath.Join(dir, tc.filename)
			if err := os.WriteFile(pdfPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write malformed PDF fixture: %v", err)
			}

			doc, err := OpenDocument(ctx, pdfPath)
			if !tc.wantOpen {
				if err == nil {
					doc.Close()
					t.Fatal("Expected error when opening unrepairable PDF, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected MuPDF to repair and open document, got error: %v", err)
			}
			defer doc.Close()

			if pageCount := doc.CountPages(); pageCount != tc.wantPages {
				t.Fatalf("Expected %d page(s) in repaired document, got %d", tc.wantPages, pageCount)
			}

			page, err := doc.LoadPage(0)
			if err != nil {
				t.Fatalf("Failed to load page from repaired document: %v", err)
			}
			defer page.Close()

			// Text extraction must complete without error on the repaired
			// document (the fixture's content stream is empty).
			text, err := page.ExtractText()
			if err != nil {
				t.Fatalf("Failed to extract text from repaired document: %v", err)
			}
			text.Close()
		})
	}
}

// TestZeroByteFile tests handling of zero-byte files
func TestZeroByteFile(t *testing.T) {
	requireMuPDF(t)

	// Create context
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	// Create a zero-byte file
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "zero_byte.pdf")

	f, err := os.Create(pdfPath)
	if err != nil {
		t.Fatalf("Failed to create zero-byte file: %v", err)
	}
	f.Close()

	// Try to open the zero-byte file
	_, err = OpenDocument(ctx, pdfPath)
	if err == nil {
		t.Error("Expected error when opening zero-byte file")
	}
}

// TestErrorString tests the Error type's Error() method
func TestErrorString(t *testing.T) {
	testCases := []struct {
		name    string
		message string
	}{
		{"Empty", ""},
		{"Simple", "Test error"},
		{"Complex", "Error with special characters: !@#$%^&*()"},
		{"Multiline", "Error with\nmultiple\nlines"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Error{message: tc.message}
			if err.Error() != tc.message {
				t.Errorf("Expected error message %q, got %q", tc.message, err.Error())
			}
		})
	}
}
