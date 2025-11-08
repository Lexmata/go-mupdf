// Package mupdf - CGO Flags Configuration
//
// This file provides centralized CGO flag configuration that works with both
// source-built MuPDF (default) and system-installed MuPDF (via build tag).
//
// Build tags:
//   - Default: Uses source-built MuPDF from third_party/mupdf
//   - system_mupdf: Uses system-installed MuPDF libraries
package mupdf

// CGO flags are defined inline in each file that needs them.
// This file serves as documentation for the CGO configuration approach.
//
// For source-built MuPDF (default):
//   CFLAGS: -I${SRCDIR}/../third_party/mupdf/include
//   LDFLAGS: -L${SRCDIR}/../third_party/mupdf/build/release -lmupdf -lmupdf-third -lharfbuzz -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lgumbo -lmujs -lm
//
// For system-installed MuPDF (build tag: system_mupdf):
//   CFLAGS: -I/usr/include
//   LDFLAGS: -lmupdf -lmupdf-third -lharfbuzz -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lgumbo -lmujs -lm
