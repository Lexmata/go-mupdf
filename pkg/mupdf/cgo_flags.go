// Package mupdf - CGO Flags Configuration
//
// This file provides centralized CGO flag configuration documentation.
// All CGO flags are defined inline in each file that needs them.
//
// The project uses source-built MuPDF from third_party/mupdf:
//
//	CFLAGS: -I${SRCDIR}/../../third_party/mupdf/include
//	LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third -lharfbuzz -lfreetype -ljpeg -lpng -lz -ljbig2dec -lopenjp2 -lm
package mupdf
