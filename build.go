//go:build !nobuild
// +build !nobuild

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// This file is used to build the MuPDF library when using go get.
// It will be executed before the Go package is built.
//
// This build hook automatically:
// 1. Downloads git submodules if they don't exist (works even with go get)
// 2. Builds MuPDF from source
// 3. Ensures the library is ready for use

// parseGitModules parses .gitmodules file and returns submodule info
func parseGitModules(gitmodulesPath string) (map[string]map[string]string, error) {
	file, err := os.Open(gitmodulesPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	submodules := make(map[string]map[string]string)
	var currentSubmodule string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for submodule section header: [submodule "path"]
		if strings.HasPrefix(line, "[submodule") && strings.HasSuffix(line, "]") {
			// Extract submodule name from [submodule "third_party/mupdf"]
			start := strings.Index(line, `"`)
			end := strings.LastIndex(line, `"`)
			if start != -1 && end != -1 && start < end {
				currentSubmodule = line[start+1 : end]
				submodules[currentSubmodule] = make(map[string]string)
			}
			continue
		}

		// Parse key = value pairs
		if currentSubmodule != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				submodules[currentSubmodule][key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return submodules, nil
}

// downloadSubmodule downloads a submodule directly using git clone
func downloadSubmodule(dir, submodulePath, url, branch string) error {
	fmt.Printf("Downloading MuPDF submodule from %s (branch: %s)...\n", url, branch)

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(submodulePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %v", err)
	}

	// Clone the repository
	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, url, submodulePath)
	cloneCmd.Dir = dir
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr

	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone submodule: %v", err)
	}

	fmt.Println("✅ MuPDF submodule downloaded successfully")
	return nil
}

func main() {
	fmt.Println("Setting up MuPDF library...")

	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Path to MuPDF source
	mupdfDir := filepath.Join(dir, "third_party", "mupdf")

	// Check if MuPDF source exists, if not, download it
	if _, err := os.Stat(mupdfDir); os.IsNotExist(err) {
		fmt.Println("MuPDF submodule not found. Attempting to download...")

		gitmodulesPath := filepath.Join(dir, ".gitmodules")

		// Check if .gitmodules exists
		if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "  ERROR: MuPDF submodule not found\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "This library requires MuPDF to be built from source.\n")
			fmt.Fprintf(os.Stderr, ".gitmodules file not found. Cannot determine submodule location.\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Please clone the repository with:\n")
			fmt.Fprintf(os.Stderr, "  git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}

		// Parse .gitmodules to get submodule info
		submodules, err := parseGitModules(gitmodulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse .gitmodules: %v\n", err)
			os.Exit(1)
		}

		// Find MuPDF submodule (look for "third_party/mupdf")
		mupdfSubmodule, found := submodules["third_party/mupdf"]
		if !found {
			fmt.Fprintf(os.Stderr, "Error: MuPDF submodule not found in .gitmodules\n")
			os.Exit(1)
		}

		submodulePath := mupdfSubmodule["path"]
		submoduleURL := mupdfSubmodule["url"]
		submoduleBranch := mupdfSubmodule["branch"]

		if submodulePath == "" || submoduleURL == "" {
			fmt.Fprintf(os.Stderr, "Error: Invalid submodule configuration in .gitmodules\n")
			fmt.Fprintf(os.Stderr, "  path: %s\n", submodulePath)
			fmt.Fprintf(os.Stderr, "  url: %s\n", submoduleURL)
			os.Exit(1)
		}

		// If branch is not specified, use default branch
		if submoduleBranch == "" {
			submoduleBranch = "master"
		}

		// Check if we're in a git repository
		gitDir := filepath.Join(dir, ".git")
		isGitRepo := false
		if info, err := os.Stat(gitDir); err == nil {
			isGitRepo = info.IsDir() || (info.Mode()&os.ModeSymlink != 0)
		}

		if isGitRepo {
			// We're in a git repo, try standard git submodule init first
			fmt.Println("Initializing git submodules (standard method)...")
			cmd := exec.Command("git", "submodule", "update", "--init", "--recursive")
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Printf("Standard git submodule init failed, trying direct clone...\n")
				// Fall through to direct clone method
			} else {
				// Verify submodule was initialized
				if _, err := os.Stat(mupdfDir); err == nil {
					fmt.Println("✅ MuPDF submodule initialized successfully (via git submodule)")
					goto build
				}
			}
		}

		// Direct clone method (works even without .git directory, e.g., from go get)
		fullSubmodulePath := filepath.Join(dir, submodulePath)
		if err := downloadSubmodule(dir, fullSubmodulePath, submoduleURL, submoduleBranch); err != nil {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "  ERROR: Failed to download MuPDF submodule\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "This may happen if:\n")
			fmt.Fprintf(os.Stderr, "  - Git is not installed\n")
			fmt.Fprintf(os.Stderr, "  - Network connection is unavailable\n")
			fmt.Fprintf(os.Stderr, "  - The submodule repository is inaccessible\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Please ensure git is installed and try again, or clone manually:\n")
			fmt.Fprintf(os.Stderr, "  git clone --recurse-submodules https://bitbucket.org/lexmata/go-mupdf.git\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}

		// Verify submodule was downloaded
		if _, err := os.Stat(mupdfDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: MuPDF submodule still not found after download\n")
			os.Exit(1)
		}
	}

build:

	// Build MuPDF
	cmd := exec.Command("make")
	cmd.Dir = mupdfDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set number of CPU cores for parallel build
	numCPU := runtime.NumCPU()
	cmd.Args = append(cmd.Args, fmt.Sprintf("-j%d", numCPU))

	fmt.Printf("Running build command: %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build MuPDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("MuPDF built successfully!")
}
