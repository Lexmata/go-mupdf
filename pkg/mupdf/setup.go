//go:build ignore
// +build ignore

// This file provides setup functionality for MuPDF libraries.
// It can be run manually with: go run setup.go (from the pkg/mupdf directory)

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	// MuPDF version to clone
	// must match FZ_VERSION in third_party/mupdf/include/mupdf/fitz/version.h
	mupdfVersion = "1.26.3"
	// MuPDF git repository
	mupdfRepoURL = "https://git.ghostscript.com/mupdf.git"
)

func main() {
	if err := ensureMuPDFLibraries(); err != nil {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "  ERROR: MuPDF Setup Failed\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "MuPDF requires git and a C compiler to build.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Requirements:\n")
		fmt.Fprintf(os.Stderr, "  - git (for cloning submodules)\n")
		fmt.Fprintf(os.Stderr, "  - make, gcc/clang (for building)\n")
		fmt.Fprintf(os.Stderr, "  - zlib, freetype, harfbuzz headers (or built from submodules)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Note: MuPDF uses custom versions of dependencies (like lcms2)\n")
		fmt.Fprintf(os.Stderr, "      that may be incompatible with system libraries.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		os.Exit(1)
	}
}

// ensureMuPDFLibraries checks for libraries and clones/builds if needed
func ensureMuPDFLibraries() error {
	// Get the package directory (two levels up from this file)
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
		fmt.Println("✅ MuPDF libraries already exist")
		return nil
	}

	fmt.Println("🔧 MuPDF libraries not found, setting up...")

	// Check if git is available
	if !commandExists("git") {
		return fmt.Errorf("git is required but not found in PATH")
	}

	// Check if make is available
	if !commandExists("make") {
		return fmt.Errorf("make is required to build MuPDF but not found in PATH")
	}

	// Check if a C compiler is available
	if !commandExists("gcc") && !commandExists("clang") && !commandExists("cc") {
		return fmt.Errorf("a C compiler (gcc, clang, or cc) is required to build MuPDF but none was found in PATH")
	}

	// Check if MuPDF source exists
	if !fileExists(filepath.Join(mupdfDir, "Makefile")) {
		fmt.Printf("📦 Cloning MuPDF %s with submodules (this may take a few minutes)...\n", mupdfVersion)
		if err := cloneMuPDFWithSubmodules(projectRoot, mupdfDir); err != nil {
			return fmt.Errorf("failed to clone MuPDF: %w", err)
		}
	} else {
		// Source exists, make sure submodules are initialized
		fmt.Println("📦 MuPDF source found, ensuring submodules are initialized...")
		if err := initializeSubmodules(mupdfDir); err != nil {
			return fmt.Errorf("failed to initialize submodules: %w", err)
		}
	}

	// Build MuPDF libraries
	fmt.Println("🔨 Building MuPDF libraries with bundled dependencies (5-10 minutes)...")
	fmt.Println("   Using USE_SYSTEM_LIBS=no to avoid incompatible system libraries")
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

// cloneMuPDFWithSubmodules clones the MuPDF repository with all submodules
func cloneMuPDFWithSubmodules(projectRoot, mupdfDir string) error {
	thirdPartyDir := filepath.Join(projectRoot, "third_party")
	if err := os.MkdirAll(thirdPartyDir, 0755); err != nil {
		return fmt.Errorf("failed to create third_party directory: %w", err)
	}

	// Clone with submodules (shallow clone for speed)
	fmt.Println("   Cloning repository...")
	cmd := exec.Command("git", "clone",
		"--branch", mupdfVersion,
		"--depth", "1",
		"--recurse-submodules",
		"--shallow-submodules",
		mupdfRepoURL,
		mupdfDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Verify clone was successful
	if !fileExists(filepath.Join(mupdfDir, "Makefile")) {
		return fmt.Errorf("MuPDF repository not properly cloned")
	}

	fmt.Println("   ✅ Repository cloned successfully")
	return nil
}

// initializeSubmodules ensures submodules are initialized and updated
func initializeSubmodules(mupdfDir string) error {
	// Initialize submodules
	cmd := exec.Command("git", "submodule", "update", "--init", "--recursive")
	cmd.Dir = mupdfDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git submodule update failed: %w", err)
	}

	fmt.Println("   ✅ Submodules initialized")
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

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
