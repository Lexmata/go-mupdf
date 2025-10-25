package main

// #cgo CFLAGS: -I${SRCDIR}/third_party/mupdf/include
// #cgo LDFLAGS: -L${SRCDIR}/third_party/mupdf/build/release -lmupdf -lmupdf-third -lm
import "C"
