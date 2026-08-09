// Package mupdf - PDFCPU Integration
//
// This module integrates pdfcpu functionality into the MuPDF wrapper,
// providing additional PDF manipulation capabilities that complement
// MuPDF's core features. PDFCPU is a pure Go PDF library that offers
// operations like merging, splitting, encryption, watermarking, and more.
//
// Key Features:
//   - PDF merging: Combine multiple PDF files into one
//   - PDF splitting: Extract pages or split into multiple files
//   - PDF encryption/decryption: Password protection and removal
//   - PDF watermarking: Add text or image watermarks
//   - PDF validation: Verify PDF structure and integrity
//   - PDF optimization: Compress and optimize PDF files
//   - Metadata access: Read PDF metadata
//   - Page operations: Rotate, extract, and manipulate pages
//
// This integration allows users to leverage both MuPDF's rendering
// capabilities and PDFCPU's manipulation features in a unified API.
package mupdf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PDFCPUConfig holds configuration for PDFCPU operations.
//
// This configuration allows fine-grained control over PDF operations
// such as encryption settings, watermark properties, and optimization
// parameters.
type PDFCPUConfig struct {
	// Watermark settings
	WatermarkConfig *model.Watermark

	// Configuration override - if set, this will be used directly
	Config *model.Configuration
}

// DefaultPDFCPUConfig returns a default configuration for PDFCPU operations.
//
// The returned configuration wraps pdfcpu's default configuration
// (model.NewDefaultConfiguration()) and has no watermark settings.
//
// Note: prior to v1.5.0 this returned a config whose Config field was
// nil, which caused every operation to build its own default
// configuration internally. The returned value is now shared by any
// operation it is passed to; operations copy it before use, so pdfcpu
// cannot modify it.
func DefaultPDFCPUConfig() *PDFCPUConfig {
	return &PDFCPUConfig{Config: model.NewDefaultConfiguration()}
}

// resolveConfig returns the pdfcpu configuration to use for an
// operation. When the caller supplied one it is returned as a private
// copy: pdfcpu mutates the configuration it is handed (api.Merge sets
// both Cmd and ValidationMode, api.ExtractPages and api.Validate set
// Cmd), so sharing the caller's pointer would silently alter settings
// such as ValidationMode between calls.
//
// A shallow copy is sufficient: model.Configuration's only pointer
// fields are UserPWNew and OwnerPWNew, which this package never sets.
func resolveConfig(config *PDFCPUConfig) *model.Configuration {
	if config == nil || config.Config == nil {
		return model.NewDefaultConfiguration()
	}

	c := *config.Config

	return &c
}

// sortExtractedPageFiles sorts files produced by pdfcpu page extraction
// (named "<basename>_page_<N>.pdf") in ascending page-number order.
// Paths that do not match the pattern sort after matching ones,
// lexicographically.
func sortExtractedPageFiles(paths []string) {
	pageNr := func(path string) (int, bool) {
		name := strings.TrimSuffix(filepath.Base(path), ".pdf")
		idx := strings.LastIndex(name, "_page_")
		if idx < 0 {
			return 0, false
		}
		n, err := strconv.Atoi(name[idx+len("_page_"):])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	sort.Slice(paths, func(i, j int) bool {
		ni, oki := pageNr(paths[i])
		nj, okj := pageNr(paths[j])
		if oki && okj {
			return ni < nj
		}
		if oki != okj {
			return oki
		}
		return paths[i] < paths[j]
	})
}

// combineExtractedFiles produces outputPath from the files created by a
// pdfcpu page extraction. A single extracted file is moved (or copied if
// the rename fails) to outputPath; multiple extracted files are merged
// into outputPath in ascending page order.
func combineExtractedFiles(matches []string, outputPath string, conf *model.Configuration) error {
	sortExtractedPageFiles(matches)

	if len(matches) == 1 {
		if err := os.Rename(matches[0], outputPath); err != nil {
			// If rename fails, copy the file
			data, readErr := os.ReadFile(matches[0])
			if readErr == nil {
				err = os.WriteFile(outputPath, data, 0644)
			}
			if err != nil {
				return Error{message: fmt.Sprintf("failed to move extracted file: %v", err)}
			}
		}
		return nil
	}

	if err := api.MergeCreateFile(matches, outputPath, false, conf); err != nil {
		return Error{message: fmt.Sprintf("failed to merge extracted pages: %v", err)}
	}

	return nil
}

// MergePDFs combines multiple PDF files into a single PDF document.
//
// This function takes a list of input PDF file paths and merges them
// into a single output PDF file. Pages from all input files are
// concatenated in the order provided.
//
// Parameters:
//   - inputPaths: Slice of file paths to the PDF files to merge
//   - outputPath: Path where the merged PDF will be saved
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if merging fails
//
// Error conditions:
//   - Input files don't exist or are not readable
//   - Input files are not valid PDFs
//   - Output path is not writable
//   - Insufficient disk space
//   - PDF structure corruption in input files
//
// Example:
//
//	inputFiles := []string{"file1.pdf", "file2.pdf", "file3.pdf"}
//	err := MergePDFs(inputFiles, "merged.pdf", nil)
//	if err != nil {
//	    log.Fatalf("Failed to merge PDFs: %v", err)
//	}
func MergePDFs(inputPaths []string, outputPath string, config *PDFCPUConfig) error {
	if len(inputPaths) == 0 {
		return Error{message: "no input files provided for merging"}
	}

	// Validate all input files are accessible
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return Error{message: fmt.Sprintf("input file does not exist: %s", path)}
			}
			return Error{message: fmt.Sprintf("cannot access input file %s: %v", path, err)}
		}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	// Use pdfcpu API to merge
	conf := resolveConfig(config)

	err := api.MergeCreateFile(inputPaths, outputPath, false, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu merge failed: %v", err)}
	}

	return nil
}

// SplitPDF splits a PDF file into multiple files based on page ranges.
//
// This function can split a PDF in several ways:
//   - Extract specific pages to separate files
//   - Split into multiple files with specified page counts
//   - Extract a single page range to a new file
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputDir: Directory where split files will be saved
//   - pageRanges: Slice of page ranges to extract (e.g., "1-3", "5", "7-10")
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - []string: Paths to the created output files
//   - error: An error if splitting fails
//
// Page range format:
//   - "1" - single page
//   - "1-5" - page range (inclusive)
//   - "1,3,5" - multiple pages/ranges
//
// Each page range produces exactly one output file named split_N.pdf
// (N is the 1-based index of the range); ranges spanning multiple pages
// are combined into a single file in ascending page order.
//
// Example:
//
//	outputFiles, err := SplitPDF("input.pdf", "output/", []string{"1-3", "5", "7-10"}, nil)
//	if err != nil {
//	    log.Fatalf("Failed to split PDF: %v", err)
//	}
func SplitPDF(inputPath string, outputDir string, pageRanges []string, config *PDFCPUConfig) ([]string, error) {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return nil, Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return nil, Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
	}

	conf := resolveConfig(config)

	// Extract each range into its own fresh temp directory so the files
	// produced for that range can be identified deterministically.
	var outputFiles []string

	for i, pageRange := range pageRanges {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("split_%d.pdf", i+1))

		tempExtractDir, err := os.MkdirTemp(outputDir, "split_extract_")
		if err != nil {
			return nil, Error{message: fmt.Sprintf("failed to create temp directory: %v", err)}
		}

		err = func() error {
			defer os.RemoveAll(tempExtractDir)

			// ExtractPagesFile writes one single-page PDF per extracted page
			if err := api.ExtractPagesFile(inputPath, tempExtractDir, []string{pageRange}, conf); err != nil {
				return Error{message: fmt.Sprintf("failed to extract pages %s: %v", pageRange, err)}
			}

			matches, err := filepath.Glob(filepath.Join(tempExtractDir, "*.pdf"))
			if err != nil {
				return Error{message: fmt.Sprintf("failed to list extracted files for page range %s: %v", pageRange, err)}
			}
			if len(matches) == 0 {
				return Error{message: fmt.Sprintf("failed to find created file for page range %s", pageRange)}
			}

			return combineExtractedFiles(matches, outputPath, conf)
		}()
		if err != nil {
			return nil, err
		}

		outputFiles = append(outputFiles, outputPath)
	}

	return outputFiles, nil
}

// EncryptPDF adds password protection to a PDF file.
//
// This function encrypts a PDF with user and/or owner passwords,
// restricting access based on the specified permissions. When a
// configuration is supplied, the caller's configuration is not modified.
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputPath: Path where the encrypted PDF will be saved
//   - userPassword: User password (can be empty)
//   - ownerPassword: Owner password (can be empty)
//   - permissions: PDF permissions (printing, copying, etc.)
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if encryption fails
//
// Permissions can be set using pdfcpu permission constants:
//   - model.PermPrint
//   - model.PermModify
//   - model.PermExtract
//   - model.PermAnnot
//
// Example:
//
//	err := EncryptPDF("input.pdf", "encrypted.pdf", "user123", "owner123", model.PermPrint, nil)
//	if err != nil {
//	    log.Fatalf("Failed to encrypt PDF: %v", err)
//	}
func EncryptPDF(inputPath, outputPath, userPassword, ownerPassword string, permissions model.PermissionFlags, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	// Create encryption configuration
	conf := resolveConfig(config)
	conf.UserPW = userPassword
	conf.OwnerPW = ownerPassword
	conf.Permissions = permissions

	// A caller-supplied configuration built from a zero-valued
	// model.Configuration carries EncryptKeyLength == 0 and
	// EncryptUsingAES == false, which would silently encrypt with RC4
	// instead of AES. Normalize to pdfcpu's own defaults (AES-256) so
	// supplying a configuration never weakens encryption.
	if conf.EncryptKeyLength == 0 {
		conf.EncryptUsingAES = true
		conf.EncryptKeyLength = 256
	}

	err := api.EncryptFile(inputPath, outputPath, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu encryption failed: %v", err)}
	}

	return nil
}

// DecryptPDF removes password protection from a PDF file.
//
// This function decrypts a password-protected PDF, creating an
// unencrypted version. The password must be provided. When a
// configuration is supplied, the caller's configuration is not modified.
//
// Parameters:
//   - inputPath: Path to the encrypted PDF file
//   - outputPath: Path where the decrypted PDF will be saved
//   - password: Password for the encrypted PDF
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if decryption fails
//
// Example:
//
//	err := DecryptPDF("encrypted.pdf", "decrypted.pdf", "password123", nil)
//	if err != nil {
//	    log.Fatalf("Failed to decrypt PDF: %v", err)
//	}
func DecryptPDF(inputPath, outputPath, password string, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	conf := resolveConfig(config)
	conf.UserPW = password
	conf.OwnerPW = password

	err := api.DecryptFile(inputPath, outputPath, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu decryption failed: %v", err)}
	}

	return nil
}

// AddWatermark adds a text or image watermark to a PDF file.
//
// This function adds a watermark to all pages of a PDF. The watermark
// can be text-based or image-based, with configurable position, opacity,
// and rotation.
//
// config.WatermarkConfig, when set, takes precedence; watermarkText and
// imagePath are ignored in that case.
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputPath: Path where the watermarked PDF will be saved
//   - watermarkText: Text to use as watermark (if imagePath is empty)
//   - imagePath: Path to image file for watermark (if text is empty)
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if watermarking fails
//
// Example:
//
//	err := AddWatermark("input.pdf", "watermarked.pdf", "CONFIDENTIAL", "", nil)
//	if err != nil {
//	    log.Fatalf("Failed to add watermark: %v", err)
//	}
func AddWatermark(inputPath, outputPath, watermarkText, imagePath string, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	conf := resolveConfig(config)

	// Create watermark configuration
	var wm *model.Watermark
	var err error
	if config != nil && config.WatermarkConfig != nil {
		wm = config.WatermarkConfig
	} else {
		// Default watermark configuration
		if watermarkText != "" {
			wm, err = api.TextWatermark(watermarkText, "", false, false, types.POINTS)
			if err != nil {
				return Error{message: fmt.Sprintf("failed to create text watermark: %v", err)}
			}
		} else if imagePath != "" {
			wm, err = api.ImageWatermark(imagePath, "", false, false, types.POINTS)
			if err != nil {
				return Error{message: fmt.Sprintf("failed to create image watermark: %v", err)}
			}
		} else {
			return Error{message: "either watermarkText or imagePath must be provided"}
		}
	}

	err = api.AddWatermarksFile(inputPath, outputPath, nil, wm, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu watermarking failed: %v", err)}
	}

	return nil
}

// ValidatePDF validates a PDF file for structure and integrity.
//
// This function validates a PDF file for structural issues and
// corruption using pdfcpu's default (relaxed) validation mode.
// Strict mode can be requested by setting
// config.Config.ValidationMode = model.ValidationStrict.
//
// Parameters:
//   - pdfPath: Path to the PDF file to validate
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if validation fails or PDF is invalid
//
// pdfcpu's validator checks the following, with the depth of each check
// depending on the configured validation mode:
//   - PDF header and structure
//   - Cross-reference table integrity
//   - Object references and streams
//   - Page tree structure
//   - Font and resource validity
//
// Example:
//
//	err := ValidatePDF("document.pdf", nil)
//	if err != nil {
//	    log.Fatalf("PDF validation failed: %v", err)
//	}
func ValidatePDF(pdfPath string, config *PDFCPUConfig) error {
	if _, err := os.Stat(pdfPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("PDF file does not exist: %s", pdfPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", pdfPath, err)}
	}

	conf := resolveConfig(config)

	err := api.ValidateFile(pdfPath, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu validation failed: %v", err)}
	}

	return nil
}

// OptimizePDF compresses and optimizes a PDF file.
//
// This function reduces PDF file size by removing redundant data,
// compressing streams, and optimizing the document structure.
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputPath: Path where the optimized PDF will be saved
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if optimization fails
//
// Optimization features:
//   - Stream compression
//   - Duplicate object removal
//   - Unused object cleanup
//
// Example:
//
//	err := OptimizePDF("input.pdf", "optimized.pdf", nil)
//	if err != nil {
//	    log.Fatalf("Failed to optimize PDF: %v", err)
//	}
func OptimizePDF(inputPath, outputPath string, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	conf := resolveConfig(config)

	err := api.OptimizeFile(inputPath, outputPath, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu optimization failed: %v", err)}
	}

	return nil
}

// RotatePages rotates pages in a PDF file.
//
// This function rotates specified pages by 90, 180, or 270 degrees.
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputPath: Path where the rotated PDF will be saved
//   - pageRanges: Page ranges to rotate (e.g., "1-3", "5", "7-10")
//   - rotation: Rotation angle in degrees (90, 180, or 270)
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if rotation fails
//
// Example:
//
//	err := RotatePages("input.pdf", "rotated.pdf", []string{"1-3"}, 90, nil)
//	if err != nil {
//	    log.Fatalf("Failed to rotate pages: %v", err)
//	}
func RotatePages(inputPath, outputPath string, pageRanges []string, rotation int, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Validate rotation angle
	if rotation != 90 && rotation != 180 && rotation != 270 {
		return Error{message: "rotation must be 90, 180, or 270 degrees"}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	conf := resolveConfig(config)

	err := api.RotateFile(inputPath, outputPath, rotation, pageRanges, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu rotation failed: %v", err)}
	}

	return nil
}

// ExtractPages extracts specific pages from a PDF to a new file.
//
// This function creates a new PDF containing only the specified pages
// from the source PDF. When the page ranges select multiple pages, the
// extracted pages are combined into a single output PDF in ascending
// page order.
//
// Parameters:
//   - inputPath: Path to the input PDF file
//   - outputPath: Path where the extracted pages PDF will be saved
//   - pageRanges: Page ranges to extract (e.g., "1-3", "5", "7-10")
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if extraction fails
//
// Example:
//
//	err := ExtractPages("input.pdf", "extracted.pdf", []string{"1-3", "5"}, nil)
//	if err != nil {
//	    log.Fatalf("Failed to extract pages: %v", err)
//	}
func ExtractPages(inputPath, outputPath string, pageRanges []string, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
		}
		return Error{message: fmt.Sprintf("cannot access input file %s: %v", inputPath, err)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	conf := resolveConfig(config)

	// ExtractPagesFile extracts to a directory, but we want a single
	// file, so extract into a fresh private temp directory and combine
	// the results. A unique directory (rather than a fixed name) keeps
	// concurrent calls that share an output directory from globbing and
	// merging each other's extracted pages.
	tempExtractDir, err := os.MkdirTemp(filepath.Dir(outputPath), "extract_")
	if err != nil {
		return Error{message: fmt.Sprintf("failed to create temp directory: %v", err)}
	}
	defer os.RemoveAll(tempExtractDir)

	if err := api.ExtractPagesFile(inputPath, tempExtractDir, pageRanges, conf); err != nil {
		return Error{message: fmt.Sprintf("pdfcpu page extraction failed: %v", err)}
	}

	// Collect the extracted files and combine them into outputPath.
	// pdfcpu creates one single-page PDF per extracted page, named
	// inputname_page_N.pdf; multiple pages are merged in ascending
	// page order.
	matches, err := filepath.Glob(filepath.Join(tempExtractDir, "*.pdf"))
	if err != nil {
		return Error{message: fmt.Sprintf("failed to list extracted files: %v", err)}
	}
	if len(matches) == 0 {
		return Error{message: "no files were extracted"}
	}

	return combineExtractedFiles(matches, outputPath, conf)
}

// GetPDFInfo retrieves metadata and information about a PDF file.
//
// This function extracts document-level information. The returned map
// contains the following keys:
//   - "pageCount": number of pages
//   - "pdfVersion": PDF version string
//   - "title", "author", "subject", "creator", "producer", "keywords",
//     "creationDate", "modDate": document metadata (each present only if
//     set in the document; unreadable metadata fields are omitted)
//   - "encrypted": whether the PDF is encrypted
//   - "fileSize": file size in bytes
//
// Keys may be added in future minor releases; callers should not assume
// the set is closed.
//
// Parameters:
//   - pdfPath: Path to the PDF file
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - map[string]interface{}: PDF information as key-value pairs
//   - error: An error if information retrieval fails
//
// Example:
//
//	info, err := GetPDFInfo("document.pdf", nil)
//	if err != nil {
//	    log.Fatalf("Failed to get PDF info: %v", err)
//	}
//	fmt.Printf("Page count: %v\n", info["pageCount"])
func GetPDFInfo(pdfPath string, config *PDFCPUConfig) (map[string]interface{}, error) {
	if _, err := os.Stat(pdfPath); err != nil {
		if os.IsNotExist(err) {
			return nil, Error{message: fmt.Sprintf("PDF file does not exist: %s", pdfPath)}
		}
		return nil, Error{message: fmt.Sprintf("cannot access input file %s: %v", pdfPath, err)}
	}

	conf := resolveConfig(config)

	// Use pdfcpu's InfoFile to get PDF information
	// We'll read the context to get basic info
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, Error{message: fmt.Sprintf("failed to open PDF file: %v", err)}
	}
	defer f.Close()

	ctx, err := api.ReadContext(f, conf)
	if err != nil {
		return nil, Error{message: fmt.Sprintf("failed to read PDF context: %v", err)}
	}

	// ReadContext does not populate PageCount; walk the page tree explicitly.
	// PageCount is reached through the embedded *XRefTable, so a nil table
	// means there is no count to report.
	if ctx.XRefTable == nil {
		return nil, Error{message: "PDF context has no cross-reference table"}
	}
	if err := ctx.XRefTable.EnsurePageCount(); err != nil {
		return nil, Error{message: fmt.Sprintf("failed to determine page count: %v", err)}
	}

	info := make(map[string]interface{})
	info["pageCount"] = ctx.PageCount
	info["pdfVersion"] = ctx.VersionString()

	// Get document info dict if available
	// Access Info through the XRefTable - need to dereference IndirectRef
	if ctx.XRefTable.Info != nil {
		infoDict, err := ctx.XRefTable.DereferenceDict(ctx.XRefTable.Info)
		if err == nil && infoDict != nil {
			// Extract common metadata fields
			// Use V10 as default version for string dereferencing
			version := ctx.XRefTable.Version()
			if titleObj, found := infoDict.Find("Title"); found {
				if title, err := ctx.XRefTable.DereferenceStringOrHexLiteral(titleObj, version, nil); err == nil {
					info["title"] = title
				}
			}
			if authorObj, found := infoDict.Find("Author"); found {
				if author, err := ctx.XRefTable.DereferenceStringOrHexLiteral(authorObj, version, nil); err == nil {
					info["author"] = author
				}
			}
			if subjectObj, found := infoDict.Find("Subject"); found {
				if subject, err := ctx.XRefTable.DereferenceStringOrHexLiteral(subjectObj, version, nil); err == nil {
					info["subject"] = subject
				}
			}
			if creatorObj, found := infoDict.Find("Creator"); found {
				if creator, err := ctx.XRefTable.DereferenceStringOrHexLiteral(creatorObj, version, nil); err == nil {
					info["creator"] = creator
				}
			}
			if producerObj, found := infoDict.Find("Producer"); found {
				if producer, err := ctx.XRefTable.DereferenceStringOrHexLiteral(producerObj, version, nil); err == nil {
					info["producer"] = producer
				}
			}
			if keywordsObj, found := infoDict.Find("Keywords"); found {
				if keywords, err := ctx.XRefTable.DereferenceStringOrHexLiteral(keywordsObj, version, nil); err == nil {
					info["keywords"] = keywords
				}
			}
			if creationDateObj, found := infoDict.Find("CreationDate"); found {
				if creationDate, err := ctx.XRefTable.DereferenceStringOrHexLiteral(creationDateObj, version, nil); err == nil {
					info["creationDate"] = creationDate
				}
			}
			if modDateObj, found := infoDict.Find("ModDate"); found {
				if modDate, err := ctx.XRefTable.DereferenceStringOrHexLiteral(modDateObj, version, nil); err == nil {
					info["modDate"] = modDate
				}
			}
		}
	}

	// Check encryption
	if ctx.Encrypt != nil {
		info["encrypted"] = true
	} else {
		info["encrypted"] = false
	}

	// File size
	if fileInfo, err := os.Stat(pdfPath); err == nil {
		info["fileSize"] = fileInfo.Size()
	}

	return info, nil
}
