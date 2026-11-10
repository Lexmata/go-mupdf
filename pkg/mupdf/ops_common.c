#include "ops_internal.h"

#include <stdlib.h>

void gomupdf_set_err(fz_context *ctx, char *err)
{
	int code;

	fz_strlcpy(err, fz_convert_error(ctx, &code), GOMUPDF_ERRLEN);
}

int gomupdf_is_fatal(int code)
{
	return code == FZ_ERROR_SYSTEM || code == FZ_ERROR_LIMIT || code == FZ_ERROR_ABORT || code == FZ_ERROR_TRYLATER;
}

void gomupdf_free(fz_context *ctx, void *ptr)
{
	fz_free(ctx, ptr);
}

fz_cookie *gomupdf_new_cancel(void)
{
	return calloc(1, sizeof(fz_cookie));
}

void gomupdf_cancel(fz_cookie *cancel)
{
	if (cancel)
	{
		cancel->abort = 1;
	}
}

void gomupdf_free_cancel(fz_cookie *cancel)
{
	free(cancel);
}

void gomupdf_check_cancel(fz_context *ctx, fz_cookie *cancel)
{
	if (cancel && cancel->abort)
	{
		fz_throw(ctx, FZ_ERROR_ABORT, "cancelled");
	}
}

fz_rect gomupdf_page_media_box(fz_context *ctx, pdf_obj *page, int *present)
{
	pdf_obj *obj = pdf_dict_get_inheritable(ctx, page, PDF_NAME(MediaBox));
	fz_rect r = fz_empty_rect;
	int ok = 0;

	if (pdf_is_array(ctx, obj) && pdf_array_len(ctx, obj) >= 4)
	{
		r = pdf_to_rect(ctx, obj);
		ok = r.x1 > r.x0 && r.y1 > r.y0;
	}
	if (present)
	{
		*present = ok;
	}
	if (!ok)
	{
		r = fz_make_rect(0, 0, 612, 792);
	}
	return r;
}

// Geometry, rendering and UprightPages all go through here, so they always
// agree about which way a malformed page faces. The result matches MuPDF's
// own snapping (nearest right angle); the value is reduced modulo 360 first
// so that an extreme /Rotate cannot overflow.
int gomupdf_page_rotation(fz_context *ctx, pdf_obj *page)
{
	int rot = pdf_dict_get_inheritable_int(ctx, page, PDF_NAME(Rotate));

	rot %= 360;
	if (rot < 0)
	{
		rot += 360;
	}
	rot = 90 * ((rot + 45) / 90);
	if (rot >= 360)
	{
		rot = 0;
	}
	return rot;
}

pdf_obj *gomupdf_add_stream_from_string(fz_context *ctx, pdf_document *doc, const char *text)
{
	fz_buffer *buf = fz_new_buffer_from_copied_data(ctx, (const unsigned char *)text, strlen(text));
	pdf_obj *ref = NULL;

	fz_try(ctx)
	{
		ref = pdf_add_stream(ctx, doc, buf, NULL, 0);
	}
	fz_always(ctx)
	{
		fz_drop_buffer(ctx, buf);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
	return ref;
}

void gomupdf_budget_charge(fz_context *ctx, gomupdf_budget *budget, size_t n)
{
	if (!budget)
	{
		return;
	}
	budget->used += (int64_t)n;
	if (budget->limit > 0 && budget->used > budget->limit)
	{
		// MuPDF's formatter reads %ld as int64_t.
		fz_throw(ctx, FZ_ERROR_LIMIT, "decoded content exceeds the limit of %ld bytes", budget->limit);
	}
}

static void append_stream(fz_context *ctx, fz_buffer *dst, pdf_obj *stream, gomupdf_budget *budget)
{
	fz_buffer *part = NULL;

	if (!pdf_is_stream(ctx, stream))
	{
		return;
	}
	part = pdf_load_stream(ctx, stream);
	fz_try(ctx)
	{
		gomupdf_budget_charge(ctx, budget, fz_buffer_storage(ctx, part, NULL));
		fz_append_buffer(ctx, dst, part);
		fz_append_byte(ctx, dst, '\n');
	}
	fz_always(ctx)
	{
		fz_drop_buffer(ctx, part);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

fz_buffer *gomupdf_page_contents(fz_context *ctx, pdf_obj *page, gomupdf_budget *budget)
{
	pdf_obj *contents = pdf_dict_get(ctx, page, PDF_NAME(Contents));
	fz_buffer *buf = fz_new_buffer(ctx, 1024);
	int i, n;

	fz_try(ctx)
	{
		if (pdf_is_array(ctx, contents))
		{
			n = pdf_array_len(ctx, contents);
			for (i = 0; i < n; i++)
			{
				append_stream(ctx, buf, pdf_array_get(ctx, contents, i), budget);
			}
		}
		else
		{
			append_stream(ctx, buf, contents, budget);
		}
	}
	fz_catch(ctx)
	{
		fz_drop_buffer(ctx, buf);
		fz_rethrow(ctx);
	}
	return buf;
}

int gomupdf_is_blank(fz_context *ctx, fz_buffer *buf)
{
	unsigned char *data = NULL;
	size_t len = fz_buffer_storage(ctx, buf, &data);
	size_t i;

	for (i = 0; i < len; i++)
	{
		// The six PDF whitespace characters; strchr matches the terminating
		// NUL of its string too, which covers the sixth.
		if (!strchr(" \t\n\r\f", data[i]))
		{
			return 0;
		}
	}
	return 1;
}

int gomupdf_num_set_add(fz_context *ctx, gomupdf_num_set *set, int num)
{
	int i;

	for (i = 0; i < set->len; i++)
	{
		if (set->nums[i] == num)
		{
			return 0;
		}
	}
	if (set->len == set->cap)
	{
		int cap = set->cap ? set->cap * 2 : 16;

		set->nums = fz_realloc_array(ctx, set->nums, cap, int);
		set->cap = cap;
	}
	set->nums[set->len++] = num;
	return 1;
}

void gomupdf_num_set_free(fz_context *ctx, gomupdf_num_set *set)
{
	fz_free(ctx, set->nums);
	set->nums = NULL;
	set->len = 0;
	set->cap = 0;
}

static void walk_name_tree(fz_context *ctx, pdf_obj *node, gomupdf_name_tree_fn *fn, void *arg, int depth, gomupdf_num_set *seen)
{
	pdf_obj *names, *kids;
	int i, n;

	if (depth > 64 || !pdf_is_dict(ctx, node))
	{
		return;
	}
	if (pdf_is_indirect(ctx, node) && !gomupdf_num_set_add(ctx, seen, pdf_to_num(ctx, node)))
	{
		return;
	}

	names = pdf_dict_get(ctx, node, PDF_NAME(Names));
	n = pdf_array_len(ctx, names);
	for (i = 0; i + 1 < n; i += 2)
	{
		fn(ctx, pdf_array_get(ctx, names, i), pdf_array_get(ctx, names, i + 1), arg);
	}

	kids = pdf_dict_get(ctx, node, PDF_NAME(Kids));
	n = pdf_array_len(ctx, kids);
	for (i = 0; i < n; i++)
	{
		walk_name_tree(ctx, pdf_array_get(ctx, kids, i), fn, arg, depth + 1, seen);
	}
}

void gomupdf_walk_name_tree(fz_context *ctx, pdf_obj *node, gomupdf_name_tree_fn *fn, void *arg)
{
	gomupdf_num_set seen = { NULL, 0, 0 };

	fz_try(ctx)
	{
		walk_name_tree(ctx, node, fn, arg, 0, &seen);
	}
	fz_always(ctx)
	{
		gomupdf_num_set_free(ctx, &seen);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}
