# Go MuPDF Wrapper - API Reference

This document provides a comprehensive API reference for the Go MuPDF wrapper, covering all public types, functions, and methods.

## Table of Contents

- [Core Types](#core-types)
  - [Context](#context)
  - [Document](#document)
  - [Page](#page)
  - [TextPage](#textpage)
- [PDF Types](#pdf-types)
  - [PDFDocument](#pdfdocument)
  - [PDFWriter](#pdfwriter)
  - [PDFPage](#pdfpage)
  - [PDFObject](#pdfobject)
- [Data Types](#data-types)
  - [Error](#error)
  - [Rect](#rect)
- [Functions](#functions)
  - [Library Functions](#library-functions)
  - [Context Functions](#context-functions)
  - [Document Functions](#document-functions)
  - [PDF Functions](#pdf-functions)

---

## Core Types

### Context

The `Context` type manages MuPDF's execution environment and is required for all operations.

```go
type Context struct {
    // Internal fields (unexported)
}
```

**Concurrency:** A `Context` is not safe for concurrent use and must not be shared between goroutines. Create one `Context` per goroutine, and keep all objects created from it (documents, pages, writers) on that goroutine.

#### Functions

##### NewContext

```go
func NewContext() (*Context, error)
```

Creates a new MuPDF execution context.

**Returns:**
- `*Context`: A new context ready for use
- `error`: An error if context creation fails

**Example:**
```go
ctx, err := mupdf.NewContext()
if err != nil {
    log.Fatal(err)
}
defer ctx.Drop()
```

#### Methods

##### Drop

```go
func (ctx *Context) Drop()
```

Releases the MuPDF context and all associated resources. Safe to call multiple times.

**Example:**
```go
ctx, _ := mupdf.NewContext()
defer ctx.Drop() // Always cleanup
```

---

### Document

The `Document` type represents an opened document and provides access to document-level operations.

```go
type Document struct {
    // Internal fields (unexported)
}
```

#### Functions

##### OpenDocument

```go
func OpenDocument(ctx *Context, filename string) (*Document, error)
```

Opens a document from the specified file path.

**Parameters:**
- `ctx`: A valid MuPDF context
- `filename`: Path to the document file

**Returns:**
- `*Document`: A document ready for operations
- `error`: An error if the file cannot be opened

**Supported Formats:**
- PDF (.pdf)
- XPS (.xps)
- EPUB (.epub)
- CBZ/CBR (.cbz, .cbr)

**Example:**
```go
doc, err := mupdf.OpenDocument(ctx, "document.pdf")
if err != nil {
    log.Fatal(err)
}
defer doc.Close()
```

#### Methods

##### Close

```go
func (doc *Document) Close()
```

Closes the document and releases all associated resources. Safe to call multiple times.

##### CountPages

```go
func (doc *Document) CountPages() int
```

Returns the total number of pages in the document.

**Returns:**
- `int`: Number of pages (≥ 0), or 0 if error occurs

**Example:**
```go
count := doc.CountPages()
fmt.Printf("Document has %d pages\n", count)
```

##### LoadPage

```go
func (doc *Document) LoadPage(pageNum int) (*Page, error)
```

Loads a specific page from the document.

**Parameters:**
- `pageNum`: Zero-based page index (0 to CountPages()-1)

**Returns:**
- `*Page`: A page ready for operations
- `error`: An error if the page cannot be loaded

**Example:**
```go
page, err := doc.LoadPage(0)
if err != nil {
    log.Fatal(err)
}
defer page.Close()
```

##### AsPDFDocument

```go
func (doc *Document) AsPDFDocument() (*PDFDocument, error)
```

Converts a Document to a PDFDocument for PDF-specific operations. Returns an error when the underlying document is not a PDF.

**Returns:**
- `*PDFDocument`: A PDF document with extended functionality
- `error`: An error if the document is not a PDF

**Note:** The returned `PDFDocument` keeps its own reference to the underlying PDF document. Always call its `Close` method when done to release that reference.

---

### Page

The `Page` type represents a single page within a document.

```go
type Page struct {
    // Internal fields (unexported)
}
```

#### Methods

##### Close

```go
func (page *Page) Close()
```

Closes the page and releases resources. Safe to call multiple times.

##### Bound

```go
func (page *Page) Bound() Rect
```

Returns the page's bounding rectangle in points.

**Returns:**
- `Rect`: The page's bounding rectangle

**Example:**
```go
bounds := page.Bound()
width := bounds.X1 - bounds.X0
height := bounds.Y1 - bounds.Y0
fmt.Printf("Page size: %.1fx%.1f points\n", width, height)
```

##### ExtractText

```go
func (page *Page) ExtractText() (*TextPage, error)
```

Extracts all text content from the page.

**Returns:**
- `*TextPage`: A text page containing extracted text
- `error`: An error if text extraction fails

**Example:**
```go
text, err := page.ExtractText()
if err != nil {
    log.Fatal(err)
}
defer text.Close()

content := text.String()
fmt.Printf("Extracted %d characters\n", len(content))
```

---

### TextPage

The `TextPage` type represents text content extracted from a document page.

```go
type TextPage struct {
    // Internal fields (unexported)
}
```

#### Methods

##### Close

```go
func (text *TextPage) Close()
```

Closes the text page and releases resources. Safe to call multiple times.

##### String

```go
func (text *TextPage) String() string
```

Returns the extracted text content as a UTF-8 string.

**Returns:**
- `string`: The text content

**Example:**
```go
content := text.String()
lines := strings.Split(content, "\n")
fmt.Printf("Text has %d lines\n", len(lines))
```

---

## PDF Types

### PDFDocument

The `PDFDocument` type provides PDF-specific functionality.

```go
type PDFDocument struct {
    // Internal fields (unexported)
}
```

#### Functions

##### OpenPDFDocument

```go
func OpenPDFDocument(ctx *Context, filename string) (*PDFDocument, error)
```

Opens a file specifically as a PDF document.

**Parameters:**
- `ctx`: A valid MuPDF context
- `filename`: Path to the PDF file

**Returns:**
- `*PDFDocument`: A PDF document ready for operations
- `error`: An error if the file cannot be opened as PDF

#### Methods

##### Close

```go
func (doc *PDFDocument) Close()
```

Releases the PDF document reference (including the reference kept by `AsPDFDocument`). Always call `Close` when done with a `PDFDocument`. Safe to call multiple times.

##### CountPages

```go
func (doc *PDFDocument) CountPages() int
```

Returns the number of pages in the PDF document.

##### LoadPage

```go
func (doc *PDFDocument) LoadPage(pageNum int) (*PDFPage, error)
```

Loads a PDF page with PDF-specific functionality.

---

### PDFWriter

The `PDFWriter` type creates new PDF documents.

```go
type PDFWriter struct {
    // Internal fields (unexported)
}
```

#### Functions

##### NewPDFWriter

```go
func NewPDFWriter(ctx *Context) (*PDFWriter, error)
```

Creates a new PDF writer for document generation.

**Parameters:**
- `ctx`: A valid MuPDF context

**Returns:**
- `*PDFWriter`: A writer ready for PDF creation
- `error`: An error if writer creation fails

**Example:**
```go
writer, err := mupdf.NewPDFWriter(ctx)
if err != nil {
    log.Fatal(err)
}
defer writer.Close()
```

#### Methods

##### Close

```go
func (writer *PDFWriter) Close()
```

Closes the PDF writer and releases resources.

##### AddPage

```go
func (writer *PDFWriter) AddPage(width, height float64) (*PDFPage, error)
```

Adds a new page to the PDF document. Both dimensions must be positive; an error is returned otherwise. New pages are created with an empty content stream (no placeholder content).

**Parameters:**
- `width`: Page width in points (1/72 inch), must be > 0
- `height`: Page height in points (1/72 inch), must be > 0

**Returns:**
- `*PDFPage`: A new page ready for content
- `error`: An error if page creation fails or a dimension is not positive

**Common Page Sizes:**
- US Letter: 612 x 792
- A4: 595 x 842
- A3: 842 x 1191
- Legal: 612 x 1008

**Example:**
```go
page, err := writer.AddPage(612, 792) // US Letter
if err != nil {
    log.Fatal(err)
}
defer page.Close()
```

##### Save

```go
func (writer *PDFWriter) Save(filename string) error
```

Saves the PDF document to a file.

**Parameters:**
- `filename`: Path where the PDF should be saved

**Returns:**
- `error`: An error if saving fails

**Example:**
```go
err := writer.Save("output.pdf")
if err != nil {
    log.Fatal(err)
}
```

##### NewPDFObject

```go
func (writer *PDFWriter) NewPDFObject(value interface{}) (*PDFObject, error)
```

Creates a new PDF object from a Go value.

**Parameters:**
- `value`: The Go value to convert

**Supported Types:**
- `nil` → PDF null object
- `bool` → PDF boolean
- `int` → PDF integer
- `float64` → PDF real number
- `string` → PDF string

**Returns:**
- `*PDFObject`: A PDF object representing the value
- `error`: An error if conversion fails

**Example:**
```go
obj, err := writer.NewPDFObject("Hello World")
if err != nil {
    log.Fatal(err)
}
defer obj.Drop()
```

---

### PDFPage

The `PDFPage` type represents a page within a PDF document with PDF-specific functionality.

```go
type PDFPage struct {
    // Internal fields (unexported)
}
```

#### Methods

##### Close

```go
func (page *PDFPage) Close()
```

Closes the PDF page and releases resources.

##### Bound

```go
func (page *PDFPage) Bound() Rect
```

Returns the PDF page's bounding rectangle.

---

### PDFObject

The `PDFObject` type represents a PDF object within the PDF structure.

```go
type PDFObject struct {
    // Internal fields (unexported)
}
```

#### Methods

##### Drop

```go
func (obj *PDFObject) Drop()
```

Releases the PDF object and associated resources.

---

## Data Types

### Error

The `Error` type represents errors from the MuPDF library.

```go
type Error struct {
    // Internal fields (unexported)
}
```

#### Methods

##### Error

```go
func (e Error) Error() string
```

Returns the error message string. Implements the standard Go error interface.

---

### Rect

The `Rect` type represents a rectangular area.

```go
type Rect struct {
    X0, Y0, X1, Y1 float64
}
```

**Coordinate System:**
- (X0, Y0): Bottom-left corner
- (X1, Y1): Top-right corner
- Units: Points (1/72 inch)
- Y-axis: Increases upward

**Example:**
```go
bounds := page.Bound()
width := bounds.X1 - bounds.X0
height := bounds.Y1 - bounds.Y0

// Convert to inches
widthInches := width / 72.0
heightInches := height / 72.0
```

---

## Functions

### Library Functions

##### GetVersion

```go
func GetVersion() string
```

Returns the version string of the underlying MuPDF library.

**Returns:**
- `string`: Version string (e.g., "1.27.2")

**Example:**
```go
version := mupdf.GetVersion()
fmt.Printf("Using MuPDF version: %s\n", version)
```

---

## Usage Patterns

### Basic Document Processing

```go
// Open and process a document
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop()

doc, err := mupdf.OpenDocument(ctx, "document.pdf")
if err != nil {
    return err
}
defer doc.Close()

for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        continue
    }
    defer page.Close()
    
    // Process page...
}
```

### PDF Creation

```go
// Create a new PDF
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop()

writer, err := mupdf.NewPDFWriter(ctx)
if err != nil {
    return err
}
defer writer.Close()

page, err := writer.AddPage(612, 792)
if err != nil {
    return err
}
defer page.Close()

err = writer.Save("output.pdf")
if err != nil {
    return err
}
```

### Text Extraction

```go
// Extract text from all pages
for i := 0; i < doc.CountPages(); i++ {
    page, err := doc.LoadPage(i)
    if err != nil {
        continue
    }
    defer page.Close()
    
    text, err := page.ExtractText()
    if err != nil {
        continue
    }
    defer text.Close()
    
    content := text.String()
    fmt.Printf("Page %d: %d characters\n", i+1, len(content))
}
```

---

## Error Handling

All functions that can fail return an error following Go conventions. Always check and handle errors appropriately:

```go
// Good error handling
doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return fmt.Errorf("failed to open document: %w", err)
}
defer doc.Close()

// Check for specific error conditions
if doc.CountPages() == 0 {
    return fmt.Errorf("document has no pages")
}
```

---

## Memory Management

The wrapper implements comprehensive memory management:

- **Always use `defer`** for cleanup functions
- **Finalizers provide safety net** but explicit cleanup is recommended
- **Don't access closed resources** - behavior is undefined
- **Resources become invalid** when parent objects are closed

```go
// Best practices
ctx, err := mupdf.NewContext()
if err != nil {
    return err
}
defer ctx.Drop() // Always cleanup

doc, err := mupdf.OpenDocument(ctx, filename)
if err != nil {
    return err
}
defer doc.Close() // Cleanup document

page, err := doc.LoadPage(0)
if err != nil {
    return err
}
defer page.Close() // Cleanup page
```

---

*This API reference covers all public interfaces of the Go MuPDF wrapper. For usage examples and tutorials, see the other documentation files.*