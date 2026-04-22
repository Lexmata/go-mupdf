//go:build !system_mupdf

package mupdf

// #cgo CFLAGS: -I${SRCDIR}/../../third_party/mupdf/include
// #cgo LDFLAGS: -L${SRCDIR}/../../third_party/mupdf/build/release -lmupdf -lmupdf-third -lm
import "C"
