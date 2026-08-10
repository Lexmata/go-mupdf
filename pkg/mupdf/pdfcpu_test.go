package mupdf

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestDefaultPDFCPUConfig tests the DefaultPDFCPUConfig function
func TestDefaultPDFCPUConfig(t *testing.T) {
	config := DefaultPDFCPUConfig()
	if config == nil {
		t.Fatalf("DefaultPDFCPUConfig returned nil")
	}
	if config.Config == nil {
		t.Fatalf("DefaultPDFCPUConfig returned config with nil Config")
	}
}

// TestMergePDFs tests PDF merging functionality with various scenarios
func TestMergePDFs(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)

	// Create multiple test PDFs
	pdf1 := filepath.Join(dir, "test1.pdf")
	pdf2 := filepath.Join(dir, "test2.pdf")
	pdf3 := filepath.Join(dir, "test3.pdf")

	createTestPDFFile(t, pdf1, "PDF 1 Content")
	createTestPDFFile(t, pdf2, "PDF 2 Content")
	createTestPDFFile(t, pdf3, "PDF 3 Content")

	// Test 1: Merge multiple PDFs
	t.Run("MergeMultiplePDFs", func(t *testing.T) {
		outputPath := filepath.Join(dir, "merged.pdf")
		inputPaths := []string{pdf1, pdf2, pdf3}

		err := MergePDFs(inputPaths, outputPath, nil)
		if err != nil {
			t.Fatalf("Failed to merge PDFs: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Merged PDF file was not created")
		}

		// Validate the merged PDF
		err = ValidatePDF(outputPath, nil)
		if err != nil {
			t.Fatalf("Merged PDF is invalid: %v", err)
		}

		// Verify the merge actually happened: three 1-page inputs -> 3 pages
		if pageCount := getPageCount(t, outputPath); pageCount != 3 {
			t.Fatalf("Expected merged PDF to have 3 pages, got %d", pageCount)
		}

		// Verify input ordering: the first page must come from the first input
		doc, err := OpenDocument(ctx, outputPath)
		if err != nil {
			t.Fatalf("Failed to open merged PDF: %v", err)
		}
		defer doc.Close()

		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load first page of merged PDF: %v", err)
		}
		defer page.Close()

		textPage, err := page.ExtractText()
		if err != nil {
			t.Fatalf("Failed to extract text from merged PDF: %v", err)
		}
		defer textPage.Close()

		if text := textPage.String(); !strings.Contains(text, "PDF 1 Content") {
			t.Fatalf("Expected first page of merged PDF to contain %q, got %q", "PDF 1 Content", text)
		}
	})

	// Test 2: Merge single PDF (edge case)
	t.Run("MergeSinglePDF", func(t *testing.T) {
		outputPath := filepath.Join(dir, "merged_single.pdf")
		inputPaths := []string{pdf1}

		err := MergePDFs(inputPaths, outputPath, nil)
		if err != nil {
			t.Fatalf("Failed to merge single PDF: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Merged PDF file was not created")
		}

		if pageCount := getPageCount(t, outputPath); pageCount != 1 {
			t.Fatalf("Expected merged PDF to have 1 page, got %d", pageCount)
		}
	})

	// Test 3: Merge empty list (error case)
	t.Run("MergeEmptyList", func(t *testing.T) {
		outputPath := filepath.Join(dir, "merged_empty.pdf")
		inputPaths := []string{}

		err := MergePDFs(inputPaths, outputPath, nil)
		if err == nil {
			t.Fatalf("Expected error for empty input list, got nil")
		}
	})

	// Test 4: Merge with non-existent file (error case)
	t.Run("MergeNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "merged_error.pdf")
		inputPaths := []string{pdf1, "nonexistent.pdf"}

		err := MergePDFs(inputPaths, outputPath, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 5: Merge with custom config
	t.Run("MergeWithConfig", func(t *testing.T) {
		outputPath := filepath.Join(dir, "merged_config.pdf")
		inputPaths := []string{pdf1, pdf2}

		conf := model.NewDefaultConfiguration()
		conf.ValidationMode = model.ValidationRelaxed
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := MergePDFs(inputPaths, outputPath, config)
		if err != nil {
			t.Fatalf("Failed to merge with config: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Merged PDF file was not created")
		}

		if pageCount := getPageCount(t, outputPath); pageCount != 2 {
			t.Fatalf("Expected merged PDF to have 2 pages, got %d", pageCount)
		}
	})
}

// TestSplitPDF tests PDF splitting functionality
func TestSplitPDF(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)

	// Create a multi-page PDF
	pdfPath := createMultiPagePDF(t, ctx, dir, 5)

	// Test 1: Split into page ranges
	t.Run("SplitPageRanges", func(t *testing.T) {
		outputDir := filepath.Join(dir, "split_output")
		pageRanges := []string{"1-2", "3", "4-5"}

		outputFiles, err := SplitPDF(pdfPath, outputDir, pageRanges, nil)
		if err != nil {
			t.Fatalf("Failed to split PDF: %v", err)
		}

		if len(outputFiles) != len(pageRanges) {
			t.Fatalf("Expected %d output files, got %d", len(pageRanges), len(outputFiles))
		}

		// Verify all output files exist
		for _, file := range outputFiles {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Fatalf("Split file does not exist: %s", file)
			}

			// Validate each split file
			err = ValidatePDF(file, nil)
			if err != nil {
				t.Fatalf("Split PDF is invalid: %s, error: %v", file, err)
			}
		}
	})

	// Test 2: Split single page
	t.Run("SplitSinglePage", func(t *testing.T) {
		outputDir := filepath.Join(dir, "split_single")
		pageRanges := []string{"1"}

		outputFiles, err := SplitPDF(pdfPath, outputDir, pageRanges, nil)
		if err != nil {
			t.Fatalf("Failed to split single page: %v", err)
		}

		if len(outputFiles) != 1 {
			t.Fatalf("Expected 1 output file, got %d", len(outputFiles))
		}
	})

	// Test 3: Split with invalid page range (error case)
	t.Run("SplitInvalidPageRange", func(t *testing.T) {
		outputDir := filepath.Join(dir, "split_invalid")
		pageRanges := []string{"999"}

		// pdfcpu v0.11.1 silently drops out-of-range page numbers, producing
		// an empty selection; no file is extracted, so SplitPDF fails to find
		// an output file for the range and returns an error.
		_, err := SplitPDF(pdfPath, outputDir, pageRanges, nil)
		if err == nil {
			t.Fatalf("Expected error for out-of-range page selection, got nil")
		}
	})

	// Test 4: Split non-existent file (error case)
	t.Run("SplitNonExistentFile", func(t *testing.T) {
		outputDir := filepath.Join(dir, "split_error")
		pageRanges := []string{"1-2"}

		_, err := SplitPDF("nonexistent.pdf", outputDir, pageRanges, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})
}

// TestEncryptPDF tests PDF encryption functionality
func TestEncryptPDF(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createTestPDF(t)

	// Test 1: Encrypt with user and owner passwords
	t.Run("EncryptWithPasswords", func(t *testing.T) {
		outputPath := filepath.Join(dir, "encrypted.pdf")
		userPassword := "user123"
		ownerPassword := "owner123"
		permissions := model.PermissionsPrint | model.PermissionModify

		err := EncryptPDF(pdfPath, outputPath, userPassword, ownerPassword, permissions, nil)
		if err != nil {
			t.Fatalf("Failed to encrypt PDF: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Encrypted PDF file was not created")
		}

		// Verify encryption by trying to decrypt
		decryptedPath := filepath.Join(dir, "decrypted_test.pdf")
		err = DecryptPDF(outputPath, decryptedPath, userPassword, nil)
		if err != nil {
			t.Fatalf("Failed to decrypt PDF (encryption may have failed): %v", err)
		}
	})

	// Test 2: Encrypt with only user password (pdfcpu requires owner password)
	t.Run("EncryptUserPasswordOnly", func(t *testing.T) {
		outputPath := filepath.Join(dir, "encrypted_user.pdf")
		userPassword := "user456"

		// pdfcpu requires owner password - use user password as both
		err := EncryptPDF(pdfPath, outputPath, userPassword, userPassword, model.PermissionsPrint, nil)
		if err != nil {
			t.Fatalf("Failed to encrypt with user password: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Encrypted PDF file was not created")
		}
	})

	// Test 3: Encrypt with empty passwords (error case)
	t.Run("EncryptEmptyPasswords", func(t *testing.T) {
		outputPath := filepath.Join(dir, "encrypted_empty.pdf")

		// pdfcpu v0.11.1 requires an owner password for encryption
		// ("pdfcpu: please provide owner password and optional user password").
		err := EncryptPDF(pdfPath, outputPath, "", "", model.PermissionsPrint, nil)
		if err == nil {
			t.Fatalf("Expected error for encryption with empty passwords, got nil")
		}
	})

	// Test 4: Encrypt non-existent file (error case)
	t.Run("EncryptNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "encrypted_error.pdf")

		err := EncryptPDF("nonexistent.pdf", outputPath, "pass", "pass", model.PermissionsPrint, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 5: Encrypt with custom config
	t.Run("EncryptWithConfig", func(t *testing.T) {
		outputPath := filepath.Join(dir, "encrypted_config.pdf")
		conf := model.NewAESConfiguration("user", "owner", 128)
		conf.Permissions = model.PermissionsPrint
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := EncryptPDF(pdfPath, outputPath, "user", "owner", model.PermissionsPrint, config)
		if err != nil {
			t.Fatalf("Failed to encrypt with custom config: %v", err)
		}
	})
}

// TestDecryptPDF tests PDF decryption functionality
func TestDecryptPDF(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createTestPDF(t)

	// First encrypt a PDF
	encryptedPath := filepath.Join(dir, "encrypted_for_decrypt.pdf")
	password := "testpassword"

	err = EncryptPDF(pdfPath, encryptedPath, password, password, model.PermissionsPrint, nil)
	if err != nil {
		t.Fatalf("Failed to encrypt PDF for decryption test: %v", err)
	}

	// Test 1: Decrypt with correct password
	t.Run("DecryptCorrectPassword", func(t *testing.T) {
		outputPath := filepath.Join(dir, "decrypted.pdf")

		err := DecryptPDF(encryptedPath, outputPath, password, nil)
		if err != nil {
			t.Fatalf("Failed to decrypt PDF: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Decrypted PDF file was not created")
		}

		// Validate decrypted PDF
		err = ValidatePDF(outputPath, nil)
		if err != nil {
			t.Fatalf("Decrypted PDF is invalid: %v", err)
		}
	})

	// Test 2: Decrypt with wrong password (error case)
	t.Run("DecryptWrongPassword", func(t *testing.T) {
		outputPath := filepath.Join(dir, "decrypted_wrong.pdf")
		wrongPassword := "wrongpassword"

		err := DecryptPDF(encryptedPath, outputPath, wrongPassword, nil)
		if err == nil {
			t.Fatalf("Expected error for wrong password, got nil")
		}
	})

	// Test 3: Decrypt non-encrypted PDF (error case)
	t.Run("DecryptNonEncryptedPDF", func(t *testing.T) {
		outputPath := filepath.Join(dir, "decrypted_nonenc.pdf")

		// pdfcpu v0.11.1 rejects decryption of unencrypted files
		// ("pdfcpu: this file is not encrypted").
		err := DecryptPDF(pdfPath, outputPath, "anypassword", nil)
		if err == nil {
			t.Fatalf("Expected error when decrypting a non-encrypted PDF, got nil")
		}
	})

	// Test 4: Decrypt non-existent file (error case)
	t.Run("DecryptNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "decrypted_error.pdf")

		err := DecryptPDF("nonexistent.pdf", outputPath, "pass", nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})
}

// TestAddWatermark tests PDF watermarking functionality
func TestAddWatermark(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createTestPDF(t)

	// Test 1: Add text watermark
	t.Run("AddTextWatermark", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_text.pdf")
		watermarkText := "CONFIDENTIAL"

		err := AddWatermark(pdfPath, outputPath, watermarkText, "", nil)
		if err != nil {
			t.Fatalf("Failed to add text watermark: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Watermarked PDF file was not created")
		}

		// Validate watermarked PDF
		err = ValidatePDF(outputPath, nil)
		if err != nil {
			t.Fatalf("Watermarked PDF is invalid: %v", err)
		}

		// Watermarking must not change the page count
		inputPages := getPageCount(t, pdfPath)
		if pageCount := getPageCount(t, outputPath); pageCount != inputPages {
			t.Fatalf("Expected watermarked PDF to have %d pages, got %d", inputPages, pageCount)
		}

		// A stamp changes the file content
		inputBytes, err := os.ReadFile(pdfPath)
		if err != nil {
			t.Fatalf("Failed to read input PDF: %v", err)
		}
		outputBytes, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read watermarked PDF: %v", err)
		}
		if bytes.Equal(inputBytes, outputBytes) {
			t.Fatalf("Expected watermarked PDF to differ from input, but bytes are identical")
		}
	})

	// Test 2: Add watermark with custom config
	t.Run("AddWatermarkWithConfig", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_config.pdf")
		conf := model.NewDefaultConfiguration()
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := AddWatermark(pdfPath, outputPath, "CONFIG TEST", "", config)
		if err != nil {
			t.Fatalf("Failed to add watermark with config: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Watermarked PDF file was not created")
		}
	})

	// Test 3: Add watermark with empty text and image (error case)
	t.Run("AddWatermarkEmpty", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_empty.pdf")

		err := AddWatermark(pdfPath, outputPath, "", "", nil)
		if err == nil {
			t.Fatalf("Expected error for empty watermark, got nil")
		}
	})

	// Test 3: Add watermark to non-existent file (error case)
	t.Run("AddWatermarkNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_error.pdf")

		err := AddWatermark("nonexistent.pdf", outputPath, "TEXT", "", nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 4: Add watermark with special characters
	t.Run("AddWatermarkSpecialChars", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_special.pdf")
		watermarkText := "© 2025 Confidential & Proprietary"

		err := AddWatermark(pdfPath, outputPath, watermarkText, "", nil)
		if err != nil {
			t.Fatalf("Failed to add watermark with special characters: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Watermarked PDF file was not created")
		}
	})

	// Test 5: Add watermark with long text
	t.Run("AddWatermarkLongText", func(t *testing.T) {
		outputPath := filepath.Join(dir, "watermarked_long.pdf")
		watermarkText := "This is a very long watermark text that should still work correctly even when it's quite lengthy"

		err := AddWatermark(pdfPath, outputPath, watermarkText, "", nil)
		if err != nil {
			t.Fatalf("Failed to add long watermark: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Watermarked PDF file was not created")
		}
	})
}

// TestValidatePDF tests PDF validation functionality
func TestValidatePDF(t *testing.T) {
	requireMuPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)

	// Test 1: Validate valid PDF
	t.Run("ValidateValidPDF", func(t *testing.T) {
		pdfPath := createTestPDF(t)

		err := ValidatePDF(pdfPath, nil)
		if err != nil {
			t.Fatalf("Valid PDF validation failed: %v", err)
		}
	})

	// Test 2: Validate non-existent file (error case)
	t.Run("ValidateNonExistentFile", func(t *testing.T) {
		err := ValidatePDF("nonexistent.pdf", nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 3: Validate empty file (error case)
	t.Run("ValidateEmptyFile", func(t *testing.T) {
		emptyPath := filepath.Join(dir, "empty.pdf")
		f, err := os.Create(emptyPath)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}
		f.Close()

		err = ValidatePDF(emptyPath, nil)
		if err == nil {
			t.Fatalf("Expected error for empty file, got nil")
		}
	})

	// Test 4: Validate corrupted PDF (error case)
	t.Run("ValidateCorruptedPDF", func(t *testing.T) {
		corruptedPath := filepath.Join(dir, "corrupted.pdf")
		f, err := os.Create(corruptedPath)
		if err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}
		if _, err := f.WriteString("This is not a valid PDF file"); err != nil {
			f.Close()
			t.Fatalf("Failed to write corrupted content: %v", err)
		}
		f.Close()

		err = ValidatePDF(corruptedPath, nil)
		if err == nil {
			t.Fatalf("Expected error for corrupted PDF, got nil")
		}
	})

	// Test 5: Validate with custom config (relaxed validation mode)
	t.Run("ValidateWithConfig", func(t *testing.T) {
		pdfPath := createTestPDF(t)

		conf := model.NewDefaultConfiguration()
		conf.ValidationMode = model.ValidationRelaxed
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := ValidatePDF(pdfPath, config)
		if err != nil {
			t.Fatalf("Relaxed validation failed for known-valid PDF: %v", err)
		}
	})
}

// TestOptimizePDF tests PDF optimization functionality
func TestOptimizePDF(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createTestPDF(t)

	// Test 1: Optimize PDF
	t.Run("OptimizePDF", func(t *testing.T) {
		outputPath := filepath.Join(dir, "optimized.pdf")

		err := OptimizePDF(pdfPath, outputPath, nil)
		if err != nil {
			t.Fatalf("Failed to optimize PDF: %v", err)
		}

		outputInfo, err := os.Stat(outputPath)
		if err != nil {
			t.Fatalf("Optimized PDF file was not created: %v", err)
		}

		// Optimization must not grow the file materially. pdfcpu adds a
		// small fixed overhead (producer metadata, rewritten xref), which
		// dominates on tiny fixtures, so allow a modest absolute margin.
		inputInfo, err := os.Stat(pdfPath)
		if err != nil {
			t.Fatalf("Failed to stat input PDF: %v", err)
		}
		if outputInfo.Size() > inputInfo.Size()+1024 {
			t.Fatalf("Expected optimized PDF (%d bytes) to be no larger than input (%d bytes) plus overhead",
				outputInfo.Size(), inputInfo.Size())
		}

		// The optimized document must preserve the page count
		if got := getPageCount(t, outputPath); got != 1 {
			t.Fatalf("Expected optimized PDF to have 1 page, got %d", got)
		}

		// Validate optimized PDF
		err = ValidatePDF(outputPath, nil)
		if err != nil {
			t.Fatalf("Optimized PDF is invalid: %v", err)
		}
	})

	// Test 2: Optimize non-existent file (error case)
	t.Run("OptimizeNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "optimized_error.pdf")

		err := OptimizePDF("nonexistent.pdf", outputPath, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 3: Optimize with custom config
	t.Run("OptimizeWithConfig", func(t *testing.T) {
		outputPath := filepath.Join(dir, "optimized_config.pdf")

		conf := model.NewDefaultConfiguration()
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := OptimizePDF(pdfPath, outputPath, config)
		if err != nil {
			t.Fatalf("Failed to optimize with config: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Optimized PDF file was not created")
		}
	})
}

// TestRotatePages tests PDF page rotation functionality
func TestRotatePages(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createMultiPagePDF(t, ctx, dir, 3)

	// Test 1: Rotate pages 90 degrees
	t.Run("Rotate90Degrees", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_90.pdf")
		pageRanges := []string{"1-2"}

		err := RotatePages(pdfPath, outputPath, pageRanges, 90, nil)
		if err != nil {
			t.Fatalf("Failed to rotate pages 90 degrees: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Rotated PDF file was not created")
		}

		// A 90-degree rotation swaps the page's effective width and height
		inW, inH := pageDims(t, ctx, pdfPath, 0)
		outW, outH := pageDims(t, ctx, outputPath, 0)
		if math.Abs(outW-inH) > 0.5 || math.Abs(outH-inW) > 0.5 {
			t.Fatalf("Expected 90-degree rotation to swap dimensions %.1fx%.1f, got %.1fx%.1f",
				inW, inH, outW, outH)
		}
	})

	// Test 2: Rotate pages 180 degrees
	t.Run("Rotate180Degrees", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_180.pdf")
		pageRanges := []string{"1"}

		err := RotatePages(pdfPath, outputPath, pageRanges, 180, nil)
		if err != nil {
			t.Fatalf("Failed to rotate pages 180 degrees: %v", err)
		}

		// A 180-degree rotation leaves the page dimensions unchanged
		inW, inH := pageDims(t, ctx, pdfPath, 0)
		outW, outH := pageDims(t, ctx, outputPath, 0)
		if math.Abs(outW-inW) > 0.5 || math.Abs(outH-inH) > 0.5 {
			t.Fatalf("Expected 180-degree rotation to keep dimensions %.1fx%.1f, got %.1fx%.1f",
				inW, inH, outW, outH)
		}
	})

	// Test 3: Rotate pages 270 degrees
	t.Run("Rotate270Degrees", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_270.pdf")
		pageRanges := []string{"2"}

		err := RotatePages(pdfPath, outputPath, pageRanges, 270, nil)
		if err != nil {
			t.Fatalf("Failed to rotate pages 270 degrees: %v", err)
		}
	})

	// Test 4: Rotate with invalid angle (error case)
	t.Run("RotateInvalidAngle", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_invalid.pdf")
		pageRanges := []string{"1"}

		err := RotatePages(pdfPath, outputPath, pageRanges, 45, nil)
		if err == nil {
			t.Fatalf("Expected error for invalid rotation angle, got nil")
		}
	})

	// Test 5: Rotate non-existent file (error case)
	t.Run("RotateNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_error.pdf")
		pageRanges := []string{"1"}

		err := RotatePages("nonexistent.pdf", outputPath, pageRanges, 90, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 6: Rotate invalid page range (no-op case)
	t.Run("RotateInvalidPageRange", func(t *testing.T) {
		outputPath := filepath.Join(dir, "rotated_invalid_range.pdf")
		pageRanges := []string{"999"}

		// pdfcpu v0.11.1 silently drops out-of-range page numbers: the
		// rotation becomes a no-op and the output is still written.
		err := RotatePages(pdfPath, outputPath, pageRanges, 90, nil)
		if err != nil {
			t.Fatalf("Expected out-of-range rotation to succeed as a no-op, got: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Rotated PDF file was not created")
		}
	})
}

// TestExtractPages tests PDF page extraction functionality
func TestExtractPages(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createMultiPagePDF(t, ctx, dir, 5)

	// Test 1: Extract single page
	t.Run("ExtractSinglePage", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_single.pdf")
		pageRanges := []string{"1"}

		err := ExtractPages(pdfPath, outputPath, pageRanges, nil)
		if err != nil {
			t.Fatalf("Failed to extract single page: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Extracted PDF file was not created")
		}

		// Validate extracted PDF
		err = ValidatePDF(outputPath, nil)
		if err != nil {
			t.Fatalf("Extracted PDF is invalid: %v", err)
		}

		if pageCount := getPageCount(t, outputPath); pageCount != 1 {
			t.Fatalf("Expected extracted PDF to have 1 page, got %d", pageCount)
		}
	})

	// Test 2: Extract pages with custom config
	t.Run("ExtractPagesWithConfig", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_config.pdf")
		pageRanges := []string{"1"}
		conf := model.NewDefaultConfiguration()
		config := &PDFCPUConfig{
			Config: conf,
		}

		err := ExtractPages(pdfPath, outputPath, pageRanges, config)
		if err != nil {
			t.Fatalf("Failed to extract pages with config: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Extracted PDF file was not created")
		}

		if pageCount := getPageCount(t, outputPath); pageCount != 1 {
			t.Fatalf("Expected extracted PDF to have 1 page, got %d", pageCount)
		}
	})

	// Test 3: Extract page range
	t.Run("ExtractPageRange", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_range.pdf")
		pageRanges := []string{"2-4"}

		err := ExtractPages(pdfPath, outputPath, pageRanges, nil)
		if err != nil {
			t.Fatalf("Failed to extract page range: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Extracted PDF file was not created")
		}

		// Pages 2-4 of a 5-page document -> 3 pages
		if pageCount := getPageCount(t, outputPath); pageCount != 3 {
			t.Fatalf("Expected extracted PDF to have 3 pages, got %d", pageCount)
		}
	})

	// Test 4: Extract multiple ranges
	t.Run("ExtractMultipleRanges", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_multi.pdf")
		pageRanges := []string{"1", "3", "5"}

		err := ExtractPages(pdfPath, outputPath, pageRanges, nil)
		if err != nil {
			t.Fatalf("Failed to extract multiple ranges: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Extracted PDF file was not created")
		}

		// All requested ranges are merged into the single output:
		// pages 1, 3 and 5 -> 3 pages
		if pageCount := getPageCount(t, outputPath); pageCount != 3 {
			t.Fatalf("Expected extracted PDF to have 3 pages, got %d", pageCount)
		}
	})

	// Test 5: Extract non-existent file (error case)
	t.Run("ExtractNonExistentFile", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_error.pdf")
		pageRanges := []string{"1"}

		err := ExtractPages("nonexistent.pdf", outputPath, pageRanges, nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 6: Extract invalid page range (error case)
	t.Run("ExtractInvalidPageRange", func(t *testing.T) {
		outputPath := filepath.Join(dir, "extracted_invalid.pdf")
		pageRanges := []string{"999"}

		// pdfcpu v0.11.1 silently drops out-of-range page numbers, producing
		// an empty selection and extracting no files; ExtractPages then fails
		// because no files were extracted.
		err := ExtractPages(pdfPath, outputPath, pageRanges, nil)
		if err == nil {
			t.Fatalf("Expected error for out-of-range page selection, got nil")
		}
	})
}

// TestGetPDFInfo tests PDF information retrieval
func TestGetPDFInfo(t *testing.T) {
	requireMuPDF(t)

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)
	pdfPath := createTestPDF(t)

	// Test 1: Get info from valid PDF
	t.Run("GetInfoValidPDF", func(t *testing.T) {
		info, err := GetPDFInfo(pdfPath, nil)
		if err != nil {
			t.Fatalf("Failed to get PDF info: %v", err)
		}

		// Check that info contains expected keys
		if _, ok := info["pageCount"]; !ok {
			t.Fatalf("PDF info missing pageCount")
		}

		if _, ok := info["pdfVersion"]; !ok {
			t.Fatalf("PDF info missing pdfVersion")
		}

		if _, ok := info["encrypted"]; !ok {
			t.Fatalf("PDF info missing encrypted status")
		}

		if _, ok := info["fileSize"]; !ok {
			t.Fatalf("PDF info missing fileSize")
		}

		t.Logf("PDF Info: %+v", info)
	})

	// Test 2: Get info from non-existent file (error case)
	t.Run("GetInfoNonExistentFile", func(t *testing.T) {
		_, err := GetPDFInfo("nonexistent.pdf", nil)
		if err == nil {
			t.Fatalf("Expected error for non-existent file, got nil")
		}
	})

	// Test 3: Get info with custom config (relaxed validation mode)
	t.Run("GetInfoWithRelaxedValidation", func(t *testing.T) {
		conf := model.NewDefaultConfiguration()
		conf.ValidationMode = model.ValidationRelaxed
		config := &PDFCPUConfig{
			Config: conf,
		}

		info, err := GetPDFInfo(pdfPath, config)
		if err != nil {
			t.Fatalf("Failed to get PDF info with config: %v", err)
		}

		if _, ok := info["pageCount"]; !ok {
			t.Fatalf("PDF info missing pageCount")
		}
	})

	// Test 4: Get info from encrypted PDF (requires password in config)
	t.Run("GetInfoEncryptedPDF", func(t *testing.T) {
		encryptedPath := filepath.Join(dir, "encrypted_info.pdf")
		password := "pass"
		err := EncryptPDF(pdfPath, encryptedPath, password, password, model.PermissionsPrint, nil)
		if err != nil {
			t.Fatalf("Failed to encrypt PDF for info test: %v", err)
		}

		// GetPDFInfo requires password for encrypted PDFs - provide it via config
		conf := model.NewDefaultConfiguration()
		conf.UserPW = password
		conf.OwnerPW = password
		config := &PDFCPUConfig{
			Config: conf,
		}

		info, err := GetPDFInfo(encryptedPath, config)
		if err != nil {
			t.Fatalf("Failed to get info from encrypted PDF: %v", err)
		}

		if encrypted, ok := info["encrypted"].(bool); !ok || !encrypted {
			t.Fatalf("Expected encrypted status to be true, got %v", info["encrypted"])
		}
	})

	// Test 5: Get info from PDF with metadata (test metadata extraction paths)
	t.Run("GetInfoWithMetadata", func(t *testing.T) {
		// Create a PDF with metadata using MuPDF
		writer, err := NewPDFWriter(ctx)
		if err != nil {
			t.Fatalf("Failed to create PDF writer: %v", err)
		}
		defer writer.Close()

		_, err = writer.AddPage(595, 842)
		if err != nil {
			t.Fatalf("Failed to add page: %v", err)
		}

		metadataPath := filepath.Join(dir, "metadata_test.pdf")
		err = writer.Save(metadataPath)
		if err != nil {
			t.Fatalf("Failed to save PDF: %v", err)
		}

		// Get info - this should test the metadata extraction code paths
		info, err := GetPDFInfo(metadataPath, nil)
		if err != nil {
			t.Fatalf("Failed to get PDF info: %v", err)
		}

		// Verify basic info exists
		if _, ok := info["pageCount"]; !ok {
			t.Fatalf("PDF info missing pageCount")
		}
		if _, ok := info["pdfVersion"]; !ok {
			t.Fatalf("PDF info missing pdfVersion")
		}
		t.Logf("PDF Info with metadata: %+v", info)
	})

	// Test 6: Test GetPDFInfo with PDF that has Info dict (test metadata extraction)
	t.Run("GetInfoWithInfoDict", func(t *testing.T) {
		// Create a multi-page PDF to test different code paths
		multiPagePath := createMultiPagePDF(t, ctx, dir, 3)

		// Get info multiple times to test different code paths
		info1, err := GetPDFInfo(multiPagePath, nil)
		if err != nil {
			t.Fatalf("Failed to get PDF info: %v", err)
		}

		info2, err := GetPDFInfo(multiPagePath, nil)
		if err != nil {
			t.Fatalf("Failed to get PDF info second time: %v", err)
		}

		// Verify consistency
		if info1["pageCount"] != info2["pageCount"] {
			t.Errorf("Inconsistent page counts: %v vs %v", info1["pageCount"], info2["pageCount"])
		}

		// Test with config that has validation mode
		conf := model.NewDefaultConfiguration()
		conf.ValidationMode = model.ValidationRelaxed
		config := &PDFCPUConfig{Config: conf}

		info3, err := GetPDFInfo(multiPagePath, config)
		if err != nil {
			t.Fatalf("Failed to get PDF info with config: %v", err)
		}

		if info3["pageCount"] != info1["pageCount"] {
			t.Errorf("Page count differs with config: %v vs %v", info3["pageCount"], info1["pageCount"])
		}
	})

	// Test 7: Test error path - file open failure (test error handling)
	t.Run("GetInfoFileOpenError", func(t *testing.T) {
		// A directory instead of a file: os.Stat succeeds but reading the
		// PDF context fails, so GetPDFInfo must return an error.
		dirPath := dir
		_, err := GetPDFInfo(dirPath, nil)
		if err == nil {
			t.Fatalf("Expected error for directory path, got nil")
		}
	})
}

// TestPDFCPUCountPagesErrorPaths tests error paths in CountPages functions for PDFCPU coverage
func TestPDFCPUCountPagesErrorPaths(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	// Test Document.CountPages error path (cError != nil case)
	// This is hard to trigger artificially, but we can test normal flow
	count1 := doc.CountPages()
	count2 := doc.CountPages()
	if count1 != count2 {
		t.Errorf("Inconsistent page counts: %d vs %d", count1, count2)
	}

	// Test PDFDocument.CountPages error path
	pdfDoc, err := doc.AsPDFDocument()
	if err == nil {
		pdfCount1 := pdfDoc.CountPages()
		pdfCount2 := pdfDoc.CountPages()
		if pdfCount1 != pdfCount2 {
			t.Errorf("Inconsistent PDF page counts: %d vs %d", pdfCount1, pdfCount2)
		}
	}
}

// TestPDFCUBoundErrorPaths tests error paths in Bound functions for PDFCPU coverage
func TestPDFCUBoundErrorPaths(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	pdfPath := createTestPDF(t)
	doc, err := OpenDocument(ctx, pdfPath)
	if err != nil {
		t.Fatalf("Failed to open document: %v", err)
	}
	defer doc.Close()

	if doc.CountPages() > 0 {
		page, err := doc.LoadPage(0)
		if err != nil {
			t.Fatalf("Failed to load page: %v", err)
		}
		defer page.Close()

		// Test Page.Bound error path (cError != nil case)
		bounds1 := page.Bound()
		bounds2 := page.Bound()
		if bounds1 != bounds2 {
			t.Errorf("Inconsistent bounds: %+v vs %+v", bounds1, bounds2)
		}

		// Test PDFPage.Bound error path
		pdfDoc, err := doc.AsPDFDocument()
		if err == nil && pdfDoc.CountPages() > 0 {
			pdfPage, err := pdfDoc.LoadPage(0)
			if err == nil {
				defer pdfPage.Close()
				pdfBounds1 := pdfPage.Bound()
				pdfBounds2 := pdfPage.Bound()
				if pdfBounds1 != pdfBounds2 {
					t.Errorf("Inconsistent PDF bounds: %+v vs %+v", pdfBounds1, pdfBounds2)
				}
			}
		}
	}
}

// Helper function to create a test PDF file with custom content.
//
// The PDF is built programmatically (mirroring createTestPDF in
// helpers_test.go) so that all cross-reference offsets, the stream
// /Length, and the startxref value are computed from the actual bytes
// written. The content string is embedded in the single page's content
// stream, so files created with different content are distinguishable
// via text extraction. The content must not contain PDF string
// delimiters (parentheses or backslashes).
func createTestPDFFile(t *testing.T, path, content string) {
	t.Helper()

	var buf bytes.Buffer

	// Header
	buf.WriteString("%PDF-1.4\n")

	// The content stream body; its /Length is computed from the real bytes.
	streamContent := fmt.Sprintf("BT\n/F1 12 Tf\n50 750 Td\n(%s) Tj\nET\n", content)

	// Object bodies, in object-number order (1..5).
	objects := []string{
		// 1: Catalog
		"<<\n/Type /Catalog\n/Pages 2 0 R\n>>\n",
		// 2: Pages
		"<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\n",
		// 3: Page (US Letter)
		"<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n/Contents 4 0 R\n/Resources <<\n/ProcSet [/PDF /Text]\n/Font <<\n/F1 5 0 R\n>>\n>>\n>>\n",
		// 4: Content stream
		fmt.Sprintf("<<\n/Length %d\n>>\nstream\n%sendstream\n", len(streamContent), streamContent),
		// 5: Font
		"<<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\n",
	}

	// Write each object, recording its actual byte offset.
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%sendobj\n", i+1, body)
	}

	// Cross-reference table. Each entry must be exactly 20 bytes:
	// 10-digit offset, space, 5-digit generation, space, keyword, space, \n.
	startxref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d %05d n \n", off, 0)
	}

	// Trailer with the real startxref offset.
	fmt.Fprintf(&buf, "trailer\n<<\n/Size %d\n/Root 1 0 R\n>>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, startxref)

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write PDF file: %v", err)
	}
}

// getPageCount returns the page count of a PDF as reported by GetPDFInfo.
func getPageCount(t *testing.T, path string) int {
	t.Helper()

	info, err := GetPDFInfo(path, nil)
	if err != nil {
		t.Fatalf("Failed to get PDF info for %s: %v", path, err)
	}

	pageCount, ok := info["pageCount"].(int)
	if !ok {
		t.Fatalf("PDF info missing pageCount for %s", path)
	}

	return pageCount
}

// pageDims opens a PDF with MuPDF and returns the width and height of the
// given zero-based page.
func pageDims(t *testing.T, ctx *Context, path string, pageNum int) (float64, float64) {
	t.Helper()

	doc, err := OpenDocument(ctx, path)
	if err != nil {
		t.Fatalf("Failed to open document %s: %v", path, err)
	}
	defer doc.Close()

	page, err := doc.LoadPage(pageNum)
	if err != nil {
		t.Fatalf("Failed to load page %d of %s: %v", pageNum, path, err)
	}
	defer page.Close()

	bound := page.Bound()
	return bound.X1 - bound.X0, bound.Y1 - bound.Y0
}

// Helper function to create a multi-page PDF
func createMultiPagePDF(t *testing.T, ctx *Context, dir string, pageCount int) string {
	t.Helper()

	pdfPath := filepath.Join(dir, fmt.Sprintf("multipage_%d.pdf", pageCount))

	writer, err := NewPDFWriter(ctx)
	if err != nil {
		t.Fatalf("Failed to create PDF writer: %v", err)
	}
	defer writer.Close()

	for i := 0; i < pageCount; i++ {
		_, err = writer.AddPage(612, 792) // US Letter
		if err != nil {
			t.Fatalf("Failed to add page %d: %v", i, err)
		}
	}

	err = writer.Save(pdfPath)
	if err != nil {
		t.Fatalf("Failed to save multi-page PDF: %v", err)
	}

	return pdfPath
}

// TestPDFCPUIntegration tests integration of multiple pdfcpu operations
func TestPDFCPUIntegration(t *testing.T) {
	requireMuPDF(t)
	// Don't skip in short mode - these tests are needed for coverage

	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}
	defer ctx.Drop()

	dir := testDataDir(t)

	// Create multiple test PDFs
	pdf1 := filepath.Join(dir, "integration1.pdf")
	pdf2 := filepath.Join(dir, "integration2.pdf")

	createTestPDFFile(t, pdf1, "Content 1")
	createTestPDFFile(t, pdf2, "Content 2")

	// Step 1: Merge PDFs
	mergedPath := filepath.Join(dir, "integration_merged.pdf")
	err = MergePDFs([]string{pdf1, pdf2}, mergedPath, nil)
	if err != nil {
		t.Fatalf("Failed to merge PDFs: %v", err)
	}

	// Step 2: Validate merged PDF
	err = ValidatePDF(mergedPath, nil)
	if err != nil {
		t.Fatalf("Merged PDF validation failed: %v", err)
	}

	// Step 3: Add watermark
	watermarkedPath := filepath.Join(dir, "integration_watermarked.pdf")
	err = AddWatermark(mergedPath, watermarkedPath, "INTEGRATION TEST", "", nil)
	if err != nil {
		t.Fatalf("Failed to add watermark: %v", err)
	}

	// Step 4: Encrypt
	encryptedPath := filepath.Join(dir, "integration_encrypted.pdf")
	err = EncryptPDF(watermarkedPath, encryptedPath, "user", "owner", model.PermissionsPrint, nil)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Step 5: Decrypt
	decryptedPath := filepath.Join(dir, "integration_decrypted.pdf")
	err = DecryptPDF(encryptedPath, decryptedPath, "user", nil)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Step 6: Optimize
	optimizedPath := filepath.Join(dir, "integration_optimized.pdf")
	err = OptimizePDF(decryptedPath, optimizedPath, nil)
	if err != nil {
		t.Fatalf("Failed to optimize: %v", err)
	}

	// Step 7: Get info
	info, err := GetPDFInfo(optimizedPath, nil)
	if err != nil {
		t.Fatalf("Failed to get info: %v", err)
	}

	t.Logf("Integration test completed. Final PDF info: %+v", info)
}
