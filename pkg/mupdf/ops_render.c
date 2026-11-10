#include "ops_internal.h"

// Rasterise a page over its MediaBox.
//
// OCR coordinates are normalised against the MediaBox, so the image has to
// cover exactly that, and `pdftoppm` (which this replaces) draws everything
// inside it. MuPDF bounds a page by its CropBox and clips drawing to it.
// pdf_new_pixmap_from_page_with_usage does take an fz_box_type, but that
// only sizes the pixmap: with FZ_MEDIA_BOX the image is MediaBox-sized and
// the content outside the CropBox is still clipped away (measured against
// MuPDF 1.27.2; the tests TestRenderPage_UsesMediaBoxNotCropBox and
// TestRenderPage_MatchesPdftoppm fail with it). So for the duration of the
// call a page that has a CropBox gets one equal to its MediaBox, set on the
// page object itself so it wins over an inherited one, and the original
// entry is put back afterwards. The document is not saved, so nothing
// reaches disk.
fz_pixmap *gomupdf_render_page(fz_context *ctx, fz_document *doc, int number, float dpi, int64_t max_pixels, fz_cookie *cancel, char *err)
{
	fz_page *page = NULL;
	fz_pixmap *pix = NULL;
	fz_device *dev = NULL;
	pdf_obj *page_obj = NULL;
	pdf_obj *saved_crop = NULL;
	int crop_overridden = 0;

	fz_var(page);
	fz_var(pix);
	fz_var(dev);
	fz_var(page_obj);
	fz_var(saved_crop);
	fz_var(crop_overridden);

	fz_try(ctx)
	{
		fz_matrix ctm = fz_scale(dpi / 72.0f, dpi / 72.0f);
		fz_irect bbox;
		pdf_page *ppage;
		int64_t w, h;
		int count = fz_count_pages(ctx, doc);

		if (number < 0 || number >= count)
		{
			fz_throw(ctx, FZ_ERROR_ARGUMENT, "page %d out of range (document has %d pages)", number, count);
		}

		page = fz_load_page(ctx, doc, number);
		ppage = pdf_page_from_fz_page(ctx, page);
		if (ppage && pdf_dict_get_inheritable(ctx, ppage->obj, PDF_NAME(CropBox)))
		{
			page_obj = pdf_keep_obj(ctx, ppage->obj);
			saved_crop = pdf_keep_obj(ctx, pdf_dict_get(ctx, page_obj, PDF_NAME(CropBox)));
			pdf_dict_put_rect(ctx, page_obj, PDF_NAME(CropBox), gomupdf_page_media_box(ctx, page_obj, NULL));
			crop_overridden = 1;
		}

		bbox = fz_round_rect(fz_transform_rect(fz_bound_page(ctx, page), ctm));
		if (fz_is_empty_irect(bbox))
		{
			fz_throw(ctx, FZ_ERROR_ARGUMENT, "page %d has an empty MediaBox", number);
		}
		// The page size comes from the file, so it is checked before any
		// memory is committed to it.
		w = (int64_t)bbox.x1 - bbox.x0;
		h = (int64_t)bbox.y1 - bbox.y0;
		if (max_pixels > 0 && w * h > max_pixels)
		{
			// MuPDF's formatter reads %ld as int64_t.
			fz_throw(ctx, FZ_ERROR_LIMIT, "page %d would render to %ld x %ld pixels, over the limit of %ld", number, w, h, max_pixels);
		}

		pix = fz_new_pixmap_with_bbox(ctx, fz_device_rgb(ctx), bbox, NULL, 1);
		fz_clear_pixmap_with_value(ctx, pix, 0xff);
		dev = fz_new_draw_device(ctx, fz_identity, pix);
		fz_run_page(ctx, page, dev, ctm, cancel);
		fz_close_device(ctx, dev);
		// A cancelled run stops early without throwing, leaving a partial
		// picture that must not be mistaken for the page.
		gomupdf_check_cancel(ctx, cancel);
	}
	fz_always(ctx)
	{
		fz_drop_device(ctx, dev);
		fz_drop_page(ctx, page);
	}
	fz_catch(ctx)
	{
		fz_drop_pixmap(ctx, pix);
		pix = NULL;
		gomupdf_set_err(ctx, err);
	}

	if (crop_overridden)
	{
		// Restoring can itself throw (it allocates), so it gets its own try.
		// If it does, the open document is left with the wider CropBox and
		// every later bound, render or text extraction of this page would
		// silently use it; that is reported as a failure of this call even
		// though the picture was drawn.
		fz_try(ctx)
		{
			if (saved_crop)
			{
				pdf_dict_put(ctx, page_obj, PDF_NAME(CropBox), saved_crop);
			}
			else
			{
				pdf_dict_del(ctx, page_obj, PDF_NAME(CropBox));
			}
		}
		fz_catch(ctx)
		{
			fz_drop_pixmap(ctx, pix);
			pix = NULL;
			gomupdf_set_err(ctx, err);
		}
	}
	pdf_drop_obj(ctx, saved_crop);
	pdf_drop_obj(ctx, page_obj);
	return pix;
}
