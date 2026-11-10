#include "ops_internal.h"

// Outline primitives for AddBookmarks and ExtractBookmarks. The Go side
// drives the outline iterator one step at a time and treats any failure as
// "could not do that step", so these swallow the MuPDF error (acknowledging
// it, so MuPDF does not report it as unhandled) and return -1 or NULL.

static void swallow(fz_context *ctx)
{
	int code;

	fz_convert_error(ctx, &code);
}

pdf_document *gomupdf_outline_open_pdf(fz_context *ctx, const char *path)
{
	pdf_document *doc = NULL;

	fz_var(doc);

	fz_try(ctx)
	{
		doc = pdf_open_document(ctx, path);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		doc = NULL;
	}
	return doc;
}

void gomupdf_outline_drop_pdf(fz_context *ctx, pdf_document *doc)
{
	fz_try(ctx)
	{
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
	}
}

int gomupdf_outline_save_pdf(fz_context *ctx, pdf_document *doc, const char *path)
{
	int ret = 0;
	pdf_write_options opts = { 0 };

	fz_var(ret);

	fz_try(ctx)
	{
		pdf_save_document(ctx, doc, path, &opts);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		ret = -1;
	}
	return ret;
}

fz_outline_iterator *gomupdf_outline_iterator(fz_context *ctx, pdf_document *doc)
{
	fz_outline_iterator *iter = NULL;

	fz_var(iter);

	fz_try(ctx)
	{
		iter = pdf_new_outline_iterator(ctx, doc);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		iter = NULL;
	}
	return iter;
}

int gomupdf_outline_insert(fz_context *ctx, fz_outline_iterator *iter, const char *title, const char *uri, int is_open)
{
	int ret = -1;
	fz_outline_item item = { 0 };

	fz_var(ret);

	item.title = (char *)title;
	item.uri = (char *)uri;
	item.is_open = is_open;

	fz_try(ctx)
	{
		ret = fz_outline_iterator_insert(ctx, iter, &item);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		ret = -1;
	}
	return ret;
}

// The four cursor moves and delete share one shape.
#define OUTLINE_STEP(name, call) \
	int name(fz_context *ctx, fz_outline_iterator *iter) \
	{ \
		int ret = -1; \
		fz_var(ret); \
		fz_try(ctx) \
		{ \
			ret = call(ctx, iter); \
		} \
		fz_catch(ctx) \
		{ \
			swallow(ctx); \
			ret = -1; \
		} \
		return ret; \
	}

OUTLINE_STEP(gomupdf_outline_next, fz_outline_iterator_next)
OUTLINE_STEP(gomupdf_outline_prev, fz_outline_iterator_prev)
OUTLINE_STEP(gomupdf_outline_down, fz_outline_iterator_down)
OUTLINE_STEP(gomupdf_outline_up, fz_outline_iterator_up)
OUTLINE_STEP(gomupdf_outline_delete, fz_outline_iterator_delete)

fz_document *gomupdf_outline_open_document(fz_context *ctx, const char *path)
{
	fz_document *doc = NULL;

	fz_var(doc);

	fz_try(ctx)
	{
		doc = fz_open_document(ctx, path);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		doc = NULL;
	}
	return doc;
}

void gomupdf_outline_drop_document(fz_context *ctx, fz_document *doc)
{
	fz_try(ctx)
	{
		fz_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
	}
}

fz_outline *gomupdf_outline_load(fz_context *ctx, fz_document *doc)
{
	fz_outline *outline = NULL;

	fz_var(outline);

	fz_try(ctx)
	{
		outline = fz_load_outline(ctx, doc);
	}
	fz_catch(ctx)
	{
		swallow(ctx);
		outline = NULL;
	}
	return outline;
}
