# Go MuPDF Wrapper - Examples

This document provides practical examples for common use cases with the Go MuPDF wrapper.

## Table of Contents

- [Basic Document Operations](#basic-document-operations)
- [Text Processing](#text-processing)
- [PDF Creation](#pdf-creation)
- [Advanced Features](#advanced-features)
- [Error Handling](#error-handling)
- [Performance Optimization](#performance-optimization)

## Basic Document Operations

### Opening and Inspecting Documents

```go
package main

import (
    "fmt"
    "log"

    "bitbucket.org/lexmata/go-mupdf/pkg/mupdf"
)

func inspectDocument(filename string) error {
    ctx, err := mupdf.NewContext()
    if err != nil {
        return err
    }
    defer ctx.Drop()

    doc, err := mupdf.OpenDocument(ctx, filename)
    if err != nil {
        return err
    }
    defer doc.Close()

    fmt.Printf("Document: %s\n", filename)
    fmt.Printf("MuPDF Version: %s\n", mupdf.GetVersion())
    
    pageCount := doc.CountPages()
    fmt.Printf("Total Pages: %d\n", pageCount)

    // Inspect each page
    for i := 0; i < pageCount; i++ {
        page, err := doc.LoadPage(i)
        if err != nil {
            log.Printf("Failed to load page %d: %v", i, err)
            continue
        }

        bounds := page.Bound()
        width := bounds.X1 - bounds.X0
        height := bounds.Y1 - bounds.Y0

        fmt.Printf("Page %d: %.1f x %.1f points (%.2f x %.2f inches)\n", 
            i+1, width, height, width/72, height/72)

        page.Close()
    }

    return nil
}

func main() {
    if err := inspectDocument("document.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

### Converting Points to Different Units

```go
func convertPageDimensions(page *mupdf.Page) {
    bounds := page.Bound()
    width := bounds.X1 - bounds.X0
    height := bounds.Y1 - bounds.Y0

    // Points (default)
    fmt.Printf("Dimensions in points: %.1f x %.1f\n", width, height)
    
    // Inches (1 inch = 72 points)
    fmt.Printf("Dimensions in inches: %.2f x %.2f\n", width/72, height/72)
    
    // Millimeters (1 inch = 25.4 mm)
    fmt.Printf("Dimensions in mm: %.1f x %.1f\n", 
        width/72*25.4, height/72*25.4)
    
    // Determine orientation
    if width > height {
        fmt.Println("Orientation: Landscape")
    } else {
        fmt.Println("Orientation: Portrait")
    }
}