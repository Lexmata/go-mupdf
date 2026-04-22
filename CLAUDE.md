# CLAUDE.md

Go CGO bindings to [MuPDF](https://mupdf.com/). Profile-level rules (git-flow, commit format with `MPDF-N` prefix, hooks, worktrees, review-bot gate, linear history) live in the global `CLAUDE.md`. This file captures what's specific to this repo.

## MuPDF library

MuPDF is a C library for PDF (and EPUB/XPS/CBZ/SVG) rendering, manipulation, and analysis. This project wraps it for Go via CGO.

- **Vendored copy:** `./third_party/mupdf`. Use this path when the build or tests need to link against MuPDF — do not assume a system-wide install.
- Third-party is checked out to a specific commit — treat it as read-only from Go code. Upgrades happen in dedicated branches that bump the vendored revision deliberately.

## Go conventions

- **Go version:** follow the version pinned in `bitbucket-pipelines.yml` and the Dockerfile. Bump those two and `go.mod` together — never let them drift.
- Use the current toolchain (compiler, stdlib, runtime) that the pipeline image ships with. Don't special-case older Go versions.
- CGO imports are concentrated in `internal/bindings/` (or equivalent) — application code should not talk to C directly. This keeps the unsafe surface area small and testable.

## Testing

Every Go file that defines a function has a corresponding `*_test.go` with unit tests for each function. Missing tests block merge — coverage gaps in this repo turn into segfaults in production.

- `go test ./...` runs the full suite.
- `go test -race ./...` for the race-detected run; also part of CI.
- Prefer table-driven tests for functions with multiple cases.
- For functions that cross the CGO boundary, add at least one test that exercises the happy path end-to-end (not just the Go-side logic).

## Build

- Local: `go build ./...`
- Third-party build (MuPDF): see `third_party/mupdf/README.md` for the toolchain dance; CI handles it via the pipeline image.
- Cross-compile for ARM64: see Lexmata monorepo `CLAUDE.md` for the architecture targets rule.
