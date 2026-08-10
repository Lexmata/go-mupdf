// Package mupdf - CGO Flags Configuration
//
// This file provides centralized CGO flag configuration documentation.
// All CGO flags are defined inline in each file that needs them.
//
// The project uses MuPDF libraries from third_party/mupdf.
// Libraries are automatically obtained via:
//  1. Pre-built downloads from Bitbucket (preferred)
//  2. Pipeline cache (CI/CD)
//  3. Source build (fallback)
//
// Setup: Run 'make setup' or 'scripts/setup-mupdf.sh' before building
//
// CGO Configuration:
//
//	CFLAGS: -I${SRCDIR}/../../third_party/mupdf/include
//	LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third -lm
//
// The third-party dependencies (harfbuzz, freetype, jpeg, etc.) are bundled
// inside libmupdf-third.a, so no additional system libraries are linked.
package mupdf
