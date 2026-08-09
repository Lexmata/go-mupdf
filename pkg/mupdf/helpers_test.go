// Package mupdf - Test Helper Functions
//
// This file contains utility functions specifically designed for testing
// the MuPDF wrapper functionality. These helpers provide common test
// infrastructure, test data creation, and test environment management.
//
// Helper Categories:
//   - Test Data Management: Creating temporary directories and test files
//   - PDF File Generation: Creating valid PDF files for testing
//   - Test Environment: Checking test conditions and requirements
//   - Resource Management: Memory and garbage collection utilities
//
// These functions are designed to:
//   - Simplify test setup and teardown
//   - Provide consistent test data across test files
//   - Handle platform-specific test requirements
//   - Support both unit and integration testing scenarios
//   - Ensure proper cleanup of test resources
package mupdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testDataDir creates and returns a temporary directory for test data.
//
// This function creates a unique temporary directory for each test,
// ensuring test isolation and automatic cleanup. The directory is
// automatically removed when the test completes.
//
// Parameters:
//   - t: The testing.T instance for the current test
//
// Returns:
//   - string: Path to the temporary test data directory
//
// Features:
//   - Creates unique directory per test invocation
//   - Automatic cleanup via t.Cleanup()
//   - Fails the test if directory creation fails
//   - Uses system temporary directory as base
//
// The directory is suitable for:
//   - Temporary file creation during tests
//   - PDF output file storage
//   - Test artifact isolation
//   - Cross-platform temporary storage
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    dir := testDataDir(t)
//
//	    // Create files in the directory
//	    filePath := filepath.Join(dir, "test.pdf")
//	    // ... use filePath
//
//	    // Directory automatically cleaned up when test ends
//	}
func testDataDir(t *testing.T) string {
	t.Helper()

	// Create a temporary directory for test data
	dir, err := os.MkdirTemp("", "mupdf-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// createTestPDF creates a valid PDF file for testing purposes.
//
// This function generates a minimal but complete PDF file with proper
// structure, including a single page with "Hello World" text content.
// The PDF is created in a temporary directory and automatically cleaned
// up when the test completes.
//
// Parameters:
//   - t: The testing.T instance for the current test
//
// Returns:
//   - string: Path to the created PDF file, or empty string on failure
//
// PDF Structure Created:
//   - PDF version 1.4 header
//   - Root catalog object
//   - Pages tree with single page
//   - Page object with A4 dimensions (595x842 points)
//   - Content stream with "Hello World" text
//   - Complete cross-reference table and trailer
//
// The generated PDF:
//   - Is valid and can be opened by any PDF viewer
//   - Contains exactly one page
//   - Has "Hello World" text at position (50, 750)
//   - Uses standard A4 page size
//   - Includes proper PDF structure for testing document operations
//
// The file is assembled programmatically: each object's byte offset is
// recorded as it is written, the content stream /Length is computed from
// the actual stream bytes, and each xref entry is emitted as exactly 20
// bytes ("%010d %05d n \n"), so the cross-reference table and startxref
// are always correct.
//
// Use Cases:
//   - Testing document opening and parsing
//   - Validating text extraction functionality
//   - Testing page operations and bounds calculation
//   - Integration testing with real PDF content
//
// Example:
//
//	func TestDocumentOperations(t *testing.T) {
//	    pdfPath := createTestPDF(t)
//	    if pdfPath == "" {
//	        t.Skip("Cannot create test PDF")
//	    }
//
//	    // Open and test the PDF
//	    ctx, err := NewContext()
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer ctx.Drop()
//
//	    doc, err := OpenDocument(ctx, pdfPath)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer doc.Close()
//
//	    // Test operations...
//	}
func createTestPDF(t *testing.T) string {
	t.Helper()
	t.Log("Creating test PDF")

	// Create a temporary directory
	dir := testDataDir(t)
	pdfPath := filepath.Join(dir, "test.pdf")
	t.Logf("PDF path: %s", pdfPath)

	// Build the PDF programmatically so that all cross-reference offsets,
	// the stream /Length, and the startxref value are computed from the
	// actual bytes written rather than hard-coded.
	var buf bytes.Buffer

	// Header
	buf.WriteString("%PDF-1.4\n")

	// The content stream body; its /Length is computed from the real bytes.
	streamContent := "BT\n/F1 12 Tf\n50 750 Td\n(Hello World) Tj\nET\n"

	// Object bodies, in object-number order (1..5).
	objects := []string{
		// 1: Catalog
		"<<\n/Type /Catalog\n/Pages 2 0 R\n>>\n",
		// 2: Pages
		"<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\n",
		// 3: Page
		"<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 595 842]\n/Contents 4 0 R\n/Resources <<\n/ProcSet [/PDF /Text]\n/Font <<\n/F1 5 0 R\n>>\n>>\n>>\n",
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

	if err := os.WriteFile(pdfPath, buf.Bytes(), 0644); err != nil {
		t.Skipf("Failed to write PDF file: %v", err)
		return ""
	}

	t.Log("Created minimal PDF file")

	// Check if the file exists and has content
	fileInfo, err := os.Stat(pdfPath)
	if err != nil {
		t.Skipf("Failed to stat PDF file: %v", err)
		return ""
	}
	t.Logf("PDF file size: %d bytes", fileInfo.Size())

	return pdfPath
}

// skipIfShort conditionally skips the test when running in short mode.
//
// This function checks if the test suite is running in short mode
// (go test -short) and skips the current test if so. It's useful
// for tests that are time-consuming or resource-intensive.
//
// Parameters:
//   - t: The testing.T instance for the current test
//
// Usage Guidelines:
//   - Use for tests that take more than a few milliseconds
//   - Apply to integration tests or complex operations
//   - Call early in test functions for immediate skipping
//   - Combine with other skip conditions as needed
//
// Short Mode Scenarios:
//   - Quick validation during development
//   - Fast CI/CD pipeline checks
//   - Pre-commit hook testing
//   - IDE test runner quick checks
//
// Example:
//
//	func TestTimeConsumingOperation(t *testing.T) {
//	    skipIfShort(t) // Skip if -short flag is used
//
//	    // Perform time-consuming test operations...
//	    // This code only runs in full test mode
//	}
func skipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// skipIfCIorShort conditionally skips tests in CI environments or short mode.
//
// This function checks both the testing.Short() flag and the CI environment
// variable to determine if the test should be skipped. It's useful for tests
// that may be unreliable in CI environments or are too resource-intensive.
//
// Parameters:
//   - t: The testing.T instance for the current test
//
// Skip Conditions:
//   - testing.Short() returns true (go test -short)
//   - CI environment variable is set (common in CI/CD systems)
//
// CI Environment Detection:
//   - Checks for CI environment variable
//   - Most CI systems set this variable automatically
//   - Includes GitHub Actions, Travis CI, CircleCI, etc.
//
// Use Cases:
//   - Tests that require specific hardware resources
//   - Tests that depend on external services
//   - Platform-specific tests that may fail in containers
//   - Performance tests that need dedicated resources
//
// Example:
//
//	func TestWithExternalDependencies(t *testing.T) {
//	    skipIfCIorShort(t) // Skip in CI or short mode
//
//	    // Test code that requires specific environment
//	    // Only runs in local development with full test mode
//	}
func skipIfCIorShort(t *testing.T) {
	if testing.Short() || os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI or short mode")
	}
}

// requireMuPDF verifies MuPDF availability and skips tests if unavailable.
//
// This function performs a series of checks to ensure the MuPDF library
// is properly installed, linked, and functional. If any check fails,
// the test is skipped with an appropriate message.
//
// Parameters:
//   - t: The testing.T instance for the current test
//
// Verification Steps:
//  1. Check if GetVersion() returns a valid version string
//  2. Attempt to create a MuPDF context
//  3. Verify context creation and destruction works properly
//
// Skip Conditions:
//   - MuPDF version cannot be retrieved (library not linked)
//   - Context creation fails (initialization problems)
//   - Any MuPDF operation throws an exception
//
// Use Cases:
//   - Ensuring MuPDF is available before running tests
//   - Graceful handling of missing dependencies
//   - Cross-platform compatibility testing
//   - Development environment validation
//
// Integration with CI/CD:
//   - Allows tests to run even when MuPDF is not installed
//   - Provides clear skip messages for debugging
//   - Enables conditional test execution based on environment
//
// Example:
//
//	func TestMuPDFOperations(t *testing.T) {
//	    requireMuPDF(t) // Ensure MuPDF is available
//
//	    // MuPDF-dependent test code here
//	    // This code only runs when MuPDF is properly available
//	    ctx, err := NewContext()
//	    // ... rest of test
//	}
func requireMuPDF(t *testing.T) {
	// Try to get the version
	version := GetVersion()
	if version == "" {
		t.Skip("MuPDF not available")
		return
	}

	// Try to create a context
	ctx, err := NewContext()
	if err != nil {
		t.Skipf("MuPDF context creation failed: %v", err)
		return
	}
	ctx.Drop()
}

// runGC forces garbage collection for memory management testing.
//
// This function calls runtime.GC() to force immediate garbage collection.
// It's primarily used in tests to ensure finalizers run and memory is
// properly released, helping to detect memory leaks and validate
// resource cleanup.
//
// Memory Testing Use Cases:
//   - Triggering finalizers to test automatic cleanup
//   - Validating memory leak prevention
//   - Ensuring proper resource deallocation
//   - Testing memory-sensitive operations
//   - Forcing cleanup before memory assertions
//
// When to Use:
//   - After creating and releasing many objects
//   - Before checking memory usage or leak detection
//   - In stress tests that create many resources
//   - When testing finalizer behavior
//   - Before making memory-related assertions
//
// Note: This function forces garbage collection which can impact
// performance. It should only be used in testing scenarios where
// deterministic memory behavior is required.
//
// Example:
//
//	func TestMemoryCleanup(t *testing.T) {
//	    // Create many objects
//	    for i := 0; i < 1000; i++ {
//	        ctx, _ := NewContext()
//	        ctx.Drop()
//	    }
//
//	    // Force cleanup to test finalizers
//	    runGC()
//
//	    // Now check that memory was properly released
//	    // Memory assertions or leak detection here...
//	}
func runGC() {
	runtime.GC()
}
