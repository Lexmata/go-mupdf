#include "ops_internal.h"

// The operations that open a PDF, change little or nothing, and save it.

int gomupdf_repair(fz_context *ctx, const char *in, const char *out, int use_objstms, const unsigned char *id16, char *err)
{
	pdf_document *doc = NULL;
	pdf_obj *id = NULL;

	fz_var(doc);
	fz_var(id);

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;

		// Opening is what repairs: MuPDF rebuilds a damaged xref on load.
		// Saving in full (never incrementally) then writes a clean file.
		doc = pdf_open_document(ctx, in);
		pdf_count_pages(ctx, doc);

		opts.do_garbage = 1;
		opts.do_use_objstms = use_objstms ? 1 : 0;

		if (id16)
		{
			// Keep an existing permanent identifier and replace only the
			// changing half, as the PDF specification describes.
			pdf_obj *trailer = pdf_trailer(ctx, doc);
			pdf_obj *old = pdf_dict_get(ctx, trailer, PDF_NAME(ID));
			pdf_obj *first = pdf_array_get(ctx, old, 0);

			id = pdf_new_array(ctx, doc, 2);
			if (pdf_is_string(ctx, first))
			{
				pdf_array_push(ctx, id, first);
			}
			else
			{
				pdf_array_push_string(ctx, id, (const char *)id16, 16);
			}
			pdf_array_push_string(ctx, id, (const char *)id16, 16);
			pdf_dict_put(ctx, trailer, PDF_NAME(ID), id);
			opts.dont_regenerate_id = 1;
		}

		pdf_save_document(ctx, doc, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, id);
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	return 0;
}

int gomupdf_remove_bookmarks(fz_context *ctx, const char *in, const char *out, char *err)
{
	pdf_document *doc = NULL;

	fz_var(doc);

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;
		pdf_obj *root;

		doc = pdf_open_document(ctx, in);
		root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
		if (!pdf_is_dict(ctx, root))
		{
			fz_throw(ctx, FZ_ERROR_FORMAT, "document has no catalog");
		}
		pdf_dict_del(ctx, root, PDF_NAME(Outlines));
		// A viewer told to open with the bookmarks panel showing would
		// otherwise open an empty one.
		if (pdf_name_eq(ctx, pdf_dict_get(ctx, root, PDF_NAME(PageMode)), PDF_NAME(UseOutlines)))
		{
			pdf_dict_del(ctx, root, PDF_NAME(PageMode));
		}

		// Garbage collection drops the now-unreachable outline items.
		opts.do_garbage = 1;
		pdf_save_document(ctx, doc, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	return 0;
}

int gomupdf_compress(fz_context *ctx, const char *in, const char *out, char *err)
{
	pdf_document *doc = NULL;

	fz_var(doc);

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;

		doc = pdf_open_document(ctx, in);

		// Lossless only: drop unreachable objects, merge identical objects
		// and streams, deflate whatever is stored uncompressed, and pack
		// small objects into object streams. Content streams are not
		// rewritten and images are not re-encoded.
		opts.do_garbage = 4;
		opts.do_compress = 1;
		opts.do_compress_images = 1;
		opts.do_compress_fonts = 1;
		opts.do_use_objstms = 1;
		pdf_save_document(ctx, doc, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	return 0;
}
