#include "ops_internal.h"

int gomupdf_page_geometry_list(fz_context *ctx, const char *path, gomupdf_page_geometry **out, int *count, char *err)
{
	pdf_document *doc = NULL;
	gomupdf_page_geometry *list = NULL;
	int n = 0;
	int i;

	fz_var(doc);
	fz_var(list);
	fz_var(n);

	*out = NULL;
	*count = 0;

	fz_try(ctx)
	{
		doc = pdf_open_document(ctx, path);
		n = pdf_count_pages(ctx, doc);
		// The page count can come straight from a number the file declares.
		// Every page is an object of its own, so a count above the number of
		// objects is a lie, and must not size an allocation.
		if (n < 0 || n > pdf_xref_len(ctx, doc))
		{
			fz_throw(ctx, FZ_ERROR_FORMAT, "document declares %d pages but has only %d objects", n, pdf_xref_len(ctx, doc));
		}
		list = fz_calloc(ctx, n > 0 ? n : 1, sizeof(*list));
		for (i = 0; i < n; i++)
		{
			pdf_obj *page = pdf_lookup_page_obj(ctx, doc, i);
			fz_rect r = gomupdf_page_media_box(ctx, page, &list[i].has_media_box);

			list[i].x0 = r.x0;
			list[i].y0 = r.y0;
			list[i].x1 = r.x1;
			list[i].y1 = r.y1;
			list[i].rotate = gomupdf_page_rotation(ctx, page);
		}
	}
	fz_always(ctx)
	{
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		fz_free(ctx, list);
		gomupdf_set_err(ctx, err);
		return -1;
	}
	*out = list;
	*count = n;
	return 0;
}
