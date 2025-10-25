//go:build !nobuild
// +build !nobuild

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// This file is used to build the MuPDF library when using go get.
// It will be executed before the Go package is built.

func main() {
	fmt.Println("Building MuPDF library...")

	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Path to MuPDF source
	mupdfDir := filepath.Join(dir, "third_party", "mupdf")

	// Check if MuPDF source exists
	if _, err := os.Stat(mupdfDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "MuPDF source not found at %s\n", mupdfDir)
		os.Exit(1)
	}

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
