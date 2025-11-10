//go:build !nobuild
// +build !nobuild

package mupdf

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// MuPDF version to download
	mupdfVersion = "1.26.11"
	// GitHub release tarball URL
	mupdfTarballURL = "https://github.com/ArtifexSoftware/mupdf/archive/refs/tags/" + mupdfVersion + ".tar.gz"
)

var (
	setupOnce sync.Once
	setupErr  error
)

// init runs before CGO compilation and ensures MuPDF libraries are available
func init() {
	setupOnce.Do(func() {
		setupErr = ensureMuPDFLibraries()
	})

	if setupErr != nil {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "  WARNING: MuPDF Setup Issue\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n", setupErr)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "MuPDF libraries are required for CGO compilation.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "To manually set up:\n")
		fmt.Fprintf(os.Stderr, "  cd $(go list -m -f '{{.Dir}}' bitbucket.org/lexmata/go-mupdf)\n")
		fmt.Fprintf(os.Stderr, "  ./scripts/setup-mupdf.sh\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		// Don't exit - let CGO linker provide the final error
	}
}

// ensureMuPDFLibraries checks for libraries and downloads/builds if needed
func ensureMuPDFLibraries() error {
	// Get the package directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}

	pkgDir := filepath.Dir(filename)
	projectRoot := filepath.Join(pkgDir, "..", "..")
	mupdfDir := filepath.Join(projectRoot, "third_party", "mupdf")
	libsDir := filepath.Join(mupdfDir, "build", "release")

	// Check if libraries already exist
	libMupdf := filepath.Join(libsDir, "libmupdf.a")
	libMupdfThird := filepath.Join(libsDir, "libmupdf-third.a")

	if fileExists(libMupdf) && fileExists(libMupdfThird) {
		// Libraries exist, we're good
		return nil
	}

	fmt.Println("🔧 MuPDF libraries not found, setting up...")

	// Check if MuPDF source exists
	if !fileExists(filepath.Join(mupdfDir, "Makefile")) {
		fmt.Printf("📦 Downloading MuPDF %s from GitHub...\n", mupdfVersion)
		if err := downloadAndExtractMuPDF(projectRoot, mupdfDir); err != nil {
			return fmt.Errorf("failed to download MuPDF: %w", err)
		}
	}

	// Build MuPDF libraries
	fmt.Println("🔨 Building MuPDF libraries (this may take 5-10 minutes)...")
	if err := buildMuPDFLibraries(mupdfDir); err != nil {
		return fmt.Errorf("failed to build MuPDF: %w", err)
	}

	// Verify libraries were created
	if !fileExists(libMupdf) {
		return fmt.Errorf("libmupdf.a was not created")
	}
	if !fileExists(libMupdfThird) {
		return fmt.Errorf("libmupdf-third.a was not created")
	}

	fmt.Println("✅ MuPDF libraries are ready!")
	return nil
}

// downloadAndExtractMuPDF downloads the tarball and extracts it
func downloadAndExtractMuPDF(projectRoot, mupdfDir string) error {
	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "mupdf-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download tarball
	fmt.Printf("Downloading from: %s\n", mupdfTarballURL)
	resp, err := http.Get(mupdfTarballURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save tarball: %w", err)
	}

	// Rewind temp file
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek temp file: %w", err)
	}

	// Extract tarball
	fmt.Println("📂 Extracting MuPDF source...")
	thirdPartyDir := filepath.Join(projectRoot, "third_party")
	if err := os.MkdirAll(thirdPartyDir, 0755); err != nil {
		return fmt.Errorf("failed to create third_party directory: %w", err)
	}

	if err := extractTarGz(tmpFile, thirdPartyDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	// GitHub creates a directory like "mupdf-1.26.11", rename it to "mupdf"
	extractedDir := filepath.Join(thirdPartyDir, "mupdf-"+mupdfVersion)
	if fileExists(extractedDir) {
		// Remove old mupdf directory if it exists
		if fileExists(mupdfDir) {
			if err := os.RemoveAll(mupdfDir); err != nil {
				return fmt.Errorf("failed to remove old mupdf directory: %w", err)
			}
		}
		// Rename extracted directory
		if err := os.Rename(extractedDir, mupdfDir); err != nil {
			return fmt.Errorf("failed to rename directory: %w", err)
		}
	}

	if !fileExists(filepath.Join(mupdfDir, "Makefile")) {
		return fmt.Errorf("MuPDF source not properly extracted")
	}

	return nil
}

// extractTarGz extracts a .tar.gz file
func extractTarGz(gzipStream io.Reader, destDir string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		// Security check: prevent path traversal
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tarball: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}

			outFile.Close()

			// Set file permissions
			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions: %w", err)
			}

		case tar.TypeSymlink:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create symlink
			if err := os.Symlink(header.Linkname, target); err != nil {
				// Ignore symlink errors on Windows
				if runtime.GOOS != "windows" {
					return fmt.Errorf("failed to create symlink: %w", err)
				}
			}
		}
	}

	return nil
}

// buildMuPDFLibraries builds the static libraries
func buildMuPDFLibraries(mupdfDir string) error {
	numCPU := runtime.NumCPU()

	cmd := exec.Command("make",
		fmt.Sprintf("-j%d", numCPU),
		"USE_SYSTEM_LIBS=no",
		"HAVE_X11=no",
		"HAVE_GLUT=no",
		"build=release",
		"libs",
	)
	cmd.Dir = mupdfDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running: make -j%d USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release libs\n", numCPU)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
