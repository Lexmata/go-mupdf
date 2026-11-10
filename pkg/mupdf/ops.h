// C side of the operations whose Go wrappers live in ops.go: rendering
// (render.go), page geometry (geometry.go), checking and repair (repair.go),
// overlay (overlay.go), upright (upright.go), bookmarks (outline.go,
// extract_bookmarks.go), catalog /Dests repair (catalog_dests.go) and
// compression (compress.go).
//
// MuPDF reports errors with setjmp/longjmp, so every call that can throw has
// to sit inside fz_try/fz_catch. cgo cannot expand those macros, so the work
// is done in the ops_*.c files and each function reports failure through its
// return value plus a message copied into a caller-supplied buffer of
// GOMUPDF_ERRLEN bytes.
#ifndef GOMUPDF_OPS_H
#define GOMUPDF_OPS_H

#include "mupdf/fitz.h"
#include "mupdf/pdf.h"

// The error-acknowledgement API these files rely on (fz_convert_error,
// fz_report_error) arrived with MuPDF 1.24. The bundled library is newer;
// this guards builds made with -tags system_mupdf.
#if FZ_VERSION_MAJOR == 1 && FZ_VERSION_MINOR < 24
#error "go-mupdf requires MuPDF 1.24 or later"
#endif

#define GOMUPDF_ERRLEN 512
#define GOMUPDF_WARNLEN 4096

// Outcomes of gomupdf_repair_catalog_dests.
#define GOMUPDF_DESTS_UNCHANGED 0
#define GOMUPDF_DESTS_REMOVED_DUPLICATE 1
#define GOMUPDF_DESTS_MOVED_TO_NAMES 2
#define GOMUPDF_DESTS_MERGED 3

typedef struct
{
	float x0, y0, x1, y1;
	int has_media_box;
	int rotate;
} gomupdf_page_geometry;

typedef struct
{
	int repaired;
	int warning_count;
	size_t used;
	char warnings[GOMUPDF_WARNLEN];
} gomupdf_check_result;

// A cancel flag is an fz_cookie allocated in C memory. Setting its abort
// field from another thread asks a running operation to stop; a stopped
// operation fails with an FZ_ERROR_ABORT error. NULL means "cannot be
// cancelled".
fz_cookie *gomupdf_new_cancel(void);
void gomupdf_cancel(fz_cookie *cancel);
void gomupdf_free_cancel(fz_cookie *cancel);

void gomupdf_free(fz_context *ctx, void *ptr);

// Returns a new RGBA pixmap (alpha is 255 everywhere) or NULL on error.
// max_pixels <= 0 means no limit.
fz_pixmap *gomupdf_render_page(fz_context *ctx, fz_document *doc, int number, float dpi, int64_t max_pixels, fz_cookie *cancel, char *err);

// On success *out is an fz_calloc'd array of *count entries; free it with
// gomupdf_free.
int gomupdf_page_geometry_list(fz_context *ctx, const char *path, gomupdf_page_geometry **out, int *count, char *err);

int gomupdf_check(fz_context *ctx, const char *path, gomupdf_check_result *res, fz_cookie *cancel, char *err);

// id16 is either NULL or 16 bytes to write as the file identifier.
int gomupdf_repair(fz_context *ctx, const char *in, const char *out, int use_objstms, const unsigned char *id16, char *err);

// max_bytes bounds the decoded content the operation may copy into the
// output; <= 0 means no limit.
int gomupdf_overlay_pages(fz_context *ctx, const char *in, const char *overlay, const char *out, int64_t max_bytes, char *err);

// *changed receives the number of pages that were rotated.
int gomupdf_upright_pages(fz_context *ctx, const char *in, const char *out, int64_t max_bytes, int *changed, char *err);

int gomupdf_remove_bookmarks(fz_context *ctx, const char *in, const char *out, char *err);

// *outcome receives one of the GOMUPDF_DESTS_* values. out is always
// written.
int gomupdf_repair_catalog_dests(fz_context *ctx, const char *in, const char *out, int *outcome, char *err);

int gomupdf_compress(fz_context *ctx, const char *in, const char *out, char *err);

// Outline (bookmark) primitives used by AddBookmarks and ExtractBookmarks.
// Each returns -1 or NULL on failure; the MuPDF error is swallowed.
pdf_document *gomupdf_outline_open_pdf(fz_context *ctx, const char *path);
void gomupdf_outline_drop_pdf(fz_context *ctx, pdf_document *doc);
int gomupdf_outline_save_pdf(fz_context *ctx, pdf_document *doc, const char *path);
fz_outline_iterator *gomupdf_outline_iterator(fz_context *ctx, pdf_document *doc);
int gomupdf_outline_insert(fz_context *ctx, fz_outline_iterator *iter, const char *title, const char *uri, int is_open);
int gomupdf_outline_next(fz_context *ctx, fz_outline_iterator *iter);
int gomupdf_outline_prev(fz_context *ctx, fz_outline_iterator *iter);
int gomupdf_outline_down(fz_context *ctx, fz_outline_iterator *iter);
int gomupdf_outline_up(fz_context *ctx, fz_outline_iterator *iter);
int gomupdf_outline_delete(fz_context *ctx, fz_outline_iterator *iter);
fz_document *gomupdf_outline_open_document(fz_context *ctx, const char *path);
void gomupdf_outline_drop_document(fz_context *ctx, fz_document *doc);
fz_outline *gomupdf_outline_load(fz_context *ctx, fz_document *doc);

#endif
