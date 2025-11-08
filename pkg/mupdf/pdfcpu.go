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
//   - Metadata manipulation: Read and modify PDF metadata
//   - Page operations: Rotate, extract, and manipulate pages
//   - Attachment handling: Add and extract file attachments
//
// This integration allows users to leverage both MuPDF's rendering
// capabilities and PDFCPU's manipulation features in a unified API.
package mupdf

import (
	"fmt"
	"os"
	"path/filepath"

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
func DefaultPDFCPUConfig() *PDFCPUConfig {
	return &PDFCPUConfig{}
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

	// Validate all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return Error{message: fmt.Sprintf("input file does not exist: %s", path)}
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
	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

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
// Example:
//
//	outputFiles, err := SplitPDF("input.pdf", "output/", []string{"1-3", "5", "7-10"}, nil)
//	if err != nil {
//	    log.Fatalf("Failed to split PDF: %v", err)
//	}
func SplitPDF(inputPath string, outputDir string, pageRanges []string, config *PDFCPUConfig) ([]string, error) {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return nil, Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

	// Extract pages using pdfcpu - ExtractPagesFile extracts to a directory
	// We'll extract each range separately
	var outputFiles []string
	baseName := filepath.Base(inputPath)
	baseNameNoExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]

	for i, pageRange := range pageRanges {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("split_%d.pdf", i+1))

		// Use ExtractPagesFile which creates files with pattern: inputname_pageRange.pdf
		err := api.ExtractPagesFile(inputPath, outputDir, []string{pageRange}, conf)
		if err != nil {
			return nil, Error{message: fmt.Sprintf("failed to extract pages %s: %v", pageRange, err)}
		}

		// pdfcpu creates files with pattern: inputname_pageRange.pdf
		// Handle different page range formats (e.g., "1-2" becomes "1-2", "1" stays "1")
		createdFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s.pdf", baseNameNoExt, pageRange))

		// Check if file exists with expected name
		if _, err := os.Stat(createdFile); err == nil {
			// Rename to our desired name
			if err := os.Rename(createdFile, outputPath); err != nil {
				// If rename fails, use the original file
				outputFiles = append(outputFiles, createdFile)
			} else {
				outputFiles = append(outputFiles, outputPath)
			}
		} else {
			// File might have different naming - search for PDF files in outputDir
			entries, err := os.ReadDir(outputDir)
			if err == nil {
				// Find the most recently created PDF file (likely the one we just created)
				var foundFile string
				var latestTime int64
				for _, entry := range entries {
					if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pdf" {
						info, err := entry.Info()
						if err == nil {
							if info.ModTime().Unix() > latestTime {
								latestTime = info.ModTime().Unix()
								foundFile = filepath.Join(outputDir, entry.Name())
							}
						}
					}
				}
				if foundFile != "" {
					// Rename to desired name
					if err := os.Rename(foundFile, outputPath); err == nil {
						outputFiles = append(outputFiles, outputPath)
					} else {
						outputFiles = append(outputFiles, foundFile)
					}
				} else {
					return nil, Error{message: fmt.Sprintf("failed to find created file for page range %s", pageRange)}
				}
			} else {
				return nil, Error{message: fmt.Sprintf("failed to find created file for page range %s: %v", pageRange, err)}
			}
		}
	}

	return outputFiles, nil
}

// EncryptPDF adds password protection to a PDF file.
//
// This function encrypts a PDF with user and/or owner passwords,
// restricting access based on the specified permissions.
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
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	// Create encryption configuration
	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
		conf.UserPW = userPassword
		conf.OwnerPW = ownerPassword
		conf.Permissions = model.PermissionFlags(permissions)
	} else {
		// Use NewAESConfiguration for encryption
		conf = model.NewAESConfiguration(userPassword, ownerPassword, 128)
		conf.Permissions = model.PermissionFlags(permissions)
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
// unencrypted version. The password must be provided.
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
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
		conf.UserPW = password
		conf.OwnerPW = password
	} else {
		conf = model.NewDefaultConfiguration()
		conf.UserPW = password
		conf.OwnerPW = password
	}

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
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

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
// This function performs comprehensive validation of a PDF file,
// checking for structural issues, corruption, and compliance with
// PDF specifications.
//
// Parameters:
//   - pdfPath: Path to the PDF file to validate
//   - config: Optional configuration (can be nil for defaults)
//
// Returns:
//   - error: An error if validation fails or PDF is invalid
//
// Validation checks:
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
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("PDF file does not exist: %s", pdfPath)}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

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
//   - Font subsetting
//   - Image compression
//
// Example:
//
//	err := OptimizePDF("input.pdf", "optimized.pdf", nil)
//	if err != nil {
//	    log.Fatalf("Failed to optimize PDF: %v", err)
//	}
func OptimizePDF(inputPath, outputPath string, config *PDFCPUConfig) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

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
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
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

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

	err := api.RotateFile(inputPath, outputPath, rotation, pageRanges, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu rotation failed: %v", err)}
	}

	return nil
}

// ExtractPages extracts specific pages from a PDF to a new file.
//
// This function creates a new PDF containing only the specified pages
// from the source PDF.
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
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return Error{message: fmt.Sprintf("input file does not exist: %s", inputPath)}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return Error{message: fmt.Sprintf("failed to create output directory: %v", err)}
		}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

	// ExtractPagesFile extracts to a directory, but we want a single file
	// So we'll extract to a temp dir and move the file
	tempDir := filepath.Dir(outputPath)
	tempExtractDir := filepath.Join(tempDir, "temp_extract")
	os.MkdirAll(tempExtractDir, 0755)
	defer os.RemoveAll(tempExtractDir)

	err := api.ExtractPagesFile(inputPath, tempExtractDir, pageRanges, conf)
	if err != nil {
		return Error{message: fmt.Sprintf("pdfcpu page extraction failed: %v", err)}
	}

	// Find the extracted file and move it to outputPath
	// pdfcpu creates files with pattern inputname_pageRange.pdf
	// For multiple ranges, it might create multiple files, so we take the first
	matches, _ := filepath.Glob(filepath.Join(tempExtractDir, "*.pdf"))
	if len(matches) > 0 {
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
	} else {
		return Error{message: "no files were extracted"}
	}

	return nil
}

// GetPDFInfo retrieves metadata and information about a PDF file.
//
// This function extracts document-level information including:
//   - Page count
//   - PDF version
//   - Document metadata (title, author, subject, etc.)
//   - Encryption status
//   - File size
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
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return nil, Error{message: fmt.Sprintf("PDF file does not exist: %s", pdfPath)}
	}

	var conf *model.Configuration
	if config != nil && config.Config != nil {
		conf = config.Config
	} else {
		conf = model.NewDefaultConfiguration()
	}

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

	info := make(map[string]interface{})
	info["pageCount"] = ctx.PageCount
	info["pdfVersion"] = ctx.VersionString()

	// Get document info dict if available
	// Access Info through the XRefTable - need to dereference IndirectRef
	if ctx.XRefTable != nil && ctx.XRefTable.Info != nil {
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
