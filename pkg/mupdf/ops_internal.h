// Helpers shared between the ops_*.c files. Not used from Go.
#ifndef GOMUPDF_OPS_INTERNAL_H
#define GOMUPDF_OPS_INTERNAL_H

#include "ops.h"

#include <string.h>

// Copy the caught error's message into err and mark the error as handled.
void gomupdf_set_err(fz_context *ctx, char *err);

// Whether a caught error code is one that must never be downgraded to a
// warning or worked around: out of memory, a resource limit, or a cancel.
int gomupdf_is_fatal(int code);

// Throw FZ_ERROR_ABORT if the cancel flag has been set.
void gomupdf_check_cancel(fz_context *ctx, fz_cookie *cancel);

// The inherited MediaBox of a page object, or US Letter when it is missing
// or degenerate. *present (if not NULL) says which.
fz_rect gomupdf_page_media_box(fz_context *ctx, pdf_obj *page, int *present);

// /Rotate snapped to 0, 90, 180 or 270 the way MuPDF's renderer snaps it.
int gomupdf_page_rotation(fz_context *ctx, pdf_obj *page);

// A new stream object holding text. Returns an owned reference.
pdf_obj *gomupdf_add_stream_from_string(fz_context *ctx, pdf_document *doc, const char *text);

// A budget for decoded bytes copied into an output document. Hostile files
// can reference one small compressed stream thousands of times; without a
// budget every reference becomes a fresh decoded copy held until save.
typedef struct
{
	int64_t limit; // <= 0: unlimited
	int64_t used;
} gomupdf_budget;

// Count n bytes against the budget; throws FZ_ERROR_LIMIT when exhausted.
void gomupdf_budget_charge(fz_context *ctx, gomupdf_budget *budget, size_t n);

// All of a page's content streams, decoded and joined, charged to budget.
fz_buffer *gomupdf_page_contents(fz_context *ctx, pdf_obj *page, gomupdf_budget *budget);

// Whether a content stream draws nothing: empty, or only whitespace.
int gomupdf_is_blank(fz_context *ctx, fz_buffer *buf);

// A growable set of object numbers.
typedef struct
{
	int *nums;
	int len;
	int cap;
} gomupdf_num_set;

// Returns 1 if num was added, 0 if it was already present.
int gomupdf_num_set_add(fz_context *ctx, gomupdf_num_set *set, int num);
void gomupdf_num_set_free(fz_context *ctx, gomupdf_num_set *set);

// Call fn for every key/value pair in a name tree, depth first. Cycles and
// absurd depth are tolerated: a node is visited once and nesting stops at 64
// levels. key and val are borrowed.
typedef void (gomupdf_name_tree_fn)(fz_context *ctx, pdf_obj *key, pdf_obj *val, void *arg);
void gomupdf_walk_name_tree(fz_context *ctx, pdf_obj *node, gomupdf_name_tree_fn *fn, void *arg);

#endif
