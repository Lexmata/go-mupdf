//go:build system_mupdf

package mupdf

// #cgo pkg-config: mupdf
// #cgo LDFLAGS: -lm
import "C"
