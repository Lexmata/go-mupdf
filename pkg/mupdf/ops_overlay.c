#include "ops_internal.h"

// Replace a page's /Contents with [pre, <existing streams...>, post].
static void wrap_page_contents(fz_context *ctx, pdf_document *doc, pdf_obj *page, pdf_obj *pre, pdf_obj *post)
{
	pdf_obj *old = pdf_dict_get(ctx, page, PDF_NAME(Contents));
	pdf_obj *arr = pdf_new_array(ctx, doc, 4);
	int i, n;

	fz_try(ctx)
	{
		pdf_array_push(ctx, arr, pre);
		if (pdf_is_array(ctx, old))
		{
			n = pdf_array_len(ctx, old);
			for (i = 0; i < n; i++)
			{
				pdf_array_push(ctx, arr, pdf_array_get(ctx, old, i));
			}
		}
		else if (old && !pdf_is_null(ctx, old))
		{
			pdf_array_push(ctx, arr, old);
		}
		pdf_array_push(ctx, arr, post);
		pdf_dict_put(ctx, page, PDF_NAME(Contents), arr);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, arr);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

// Give a page its own /Resources and /Resources/XObject dictionaries and
// return the latter (borrowed). Resources are routinely inherited from the
// page tree or shared between pages, so adding an entry in place would leak
// it onto every other page that uses the same dictionary.
static pdf_obj *own_xobject_dict(fz_context *ctx, pdf_document *doc, pdf_obj *page)
{
	pdf_obj *inherited = pdf_dict_get_inheritable(ctx, page, PDF_NAME(Resources));
	pdf_obj *res = NULL;
	pdf_obj *xobjs = NULL;
	pdf_obj *existing;

	fz_var(res);
	fz_var(xobjs);

	fz_try(ctx)
	{
		res = pdf_is_dict(ctx, inherited) ? pdf_copy_dict(ctx, inherited) : pdf_new_dict(ctx, doc, 2);
		existing = pdf_dict_get(ctx, res, PDF_NAME(XObject));
		xobjs = pdf_is_dict(ctx, existing) ? pdf_copy_dict(ctx, existing) : pdf_new_dict(ctx, doc, 2);
		pdf_dict_put(ctx, res, PDF_NAME(XObject), xobjs);
		pdf_dict_put(ctx, page, PDF_NAME(Resources), res);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, xobjs);
		pdf_drop_obj(ctx, res);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
	return pdf_dict_get(ctx, pdf_dict_get(ctx, page, PDF_NAME(Resources)), PDF_NAME(XObject));
}

// A content-stream processor that tracks only the state that can leak from
// one content stream into the next: the q/Q stack, an open text object, and
// open marked content. Given a target name it also records the state each
// time that form XObject is drawn.
typedef struct
{
	pdf_processor super;
	int depth;      // q operators still open
	int underflow;  // Q operators with no q to close
	int in_text;    // inside BT ... ET
	int marked;     // BMC/BDC not yet closed by EMC
	int ended;      // end of data seen
	int open;       // depth when the data ended
	int text_open;  // in_text when the data ended
	int marked_open; // marked when the data ended
	const char *target;
	int target_seen;
	int target_clean; // drawn at depth 1, outside text and marked content
} state_tracker;

static void tracker_q(fz_context *ctx, pdf_processor *proc)
{
	state_tracker *t = (state_tracker *)proc;

	(void)ctx;
	if (!t->ended)
	{
		t->depth++;
	}
}

static void tracker_Q(fz_context *ctx, pdf_processor *proc)
{
	state_tracker *t = (state_tracker *)proc;

	(void)ctx;
	if (t->ended)
	{
		return;
	}
	if (t->depth > 0)
	{
		t->depth--;
	}
	else
	{
		t->underflow++;
	}
}

static void tracker_BT(fz_context *ctx, pdf_processor *proc)
{
	(void)ctx;
	((state_tracker *)proc)->in_text = 1;
}

static void tracker_ET(fz_context *ctx, pdf_processor *proc)
{
	(void)ctx;
	((state_tracker *)proc)->in_text = 0;
}

static void tracker_BMC(fz_context *ctx, pdf_processor *proc, const char *tag)
{
	(void)ctx;
	(void)tag;
	((state_tracker *)proc)->marked++;
}

static void tracker_BDC(fz_context *ctx, pdf_processor *proc, const char *tag, pdf_obj *raw, pdf_obj *cooked)
{
	(void)ctx;
	(void)tag;
	(void)raw;
	(void)cooked;
	((state_tracker *)proc)->marked++;
}

static void tracker_EMC(fz_context *ctx, pdf_processor *proc)
{
	state_tracker *t = (state_tracker *)proc;

	(void)ctx;
	if (t->marked > 0)
	{
		t->marked--;
	}
}

static void tracker_Do_form(fz_context *ctx, pdf_processor *proc, const char *name, pdf_obj *form)
{
	state_tracker *t = (state_tracker *)proc;

	(void)ctx;
	(void)form;
	if (t->target && name && !strcmp(name, t->target))
	{
		t->target_seen++;
		if (t->depth == 1 && t->underflow == 0 && !t->in_text && t->marked == 0)
		{
			t->target_clean++;
		}
	}
}

// The interpreter closes any q the content left open by calling op_Q itself
// once the data runs out, so the state has to be read here, at end of data,
// before those synthetic pops arrive.
static void tracker_EOD(fz_context *ctx, pdf_processor *proc)
{
	state_tracker *t = (state_tracker *)proc;

	(void)ctx;
	t->open = t->depth;
	t->text_open = t->in_text;
	t->marked_open = t->marked;
	t->ended = 1;
}

// Run a page's content through a tracker and copy the result to *out.
// Returns 0 if the content could not be interpreted to the end. An error
// that must not be worked around (out of memory, a limit, a cancel) is
// rethrown.
static int track_page(fz_context *ctx, pdf_document *doc, pdf_obj *page, const char *target, state_tracker *out)
{
	pdf_obj *contents = pdf_dict_get(ctx, page, PDF_NAME(Contents));
	pdf_obj *res = pdf_dict_get_inheritable(ctx, page, PDF_NAME(Resources));
	state_tracker *proc = NULL;
	int ok = 0;

	fz_var(proc);
	fz_var(ok);

	memset(out, 0, sizeof(*out));
	if (!contents || pdf_is_null(ctx, contents))
	{
		return 1;
	}

	fz_try(ctx)
	{
		proc = pdf_new_processor(ctx, sizeof(*proc));
		proc->super.op_q = tracker_q;
		proc->super.op_Q = tracker_Q;
		proc->super.op_BT = tracker_BT;
		proc->super.op_ET = tracker_ET;
		proc->super.op_BMC = tracker_BMC;
		proc->super.op_BDC = tracker_BDC;
		proc->super.op_EMC = tracker_EMC;
		proc->super.op_Do_form = tracker_Do_form;
		proc->super.op_EOD = tracker_EOD;
		proc->target = target;
		pdf_process_contents(ctx, &proc->super, doc, res, contents, NULL, NULL);
		pdf_close_processor(ctx, &proc->super);
		ok = proc->ended;
		*out = *proc;
	}
	fz_always(ctx)
	{
		pdf_drop_processor(ctx, (pdf_processor *)proc);
	}
	fz_catch(ctx)
	{
		if (gomupdf_is_fatal(fz_caught(ctx)))
		{
			fz_rethrow(ctx);
		}
		fz_report_error(ctx);
		ok = 0;
	}
	return ok;
}

// Draw the overlay after the page's content, leaving that content where it
// is. Returns 1 if the overlay was then verified to be drawn in a clean
// state, 0 if the page's content makes that impossible.
//
// Leaving the content in place matters: marked-content references from a
// tagged PDF's structure tree point into the page's own content stream, and
// keep working only if it stays there.
static int overlay_in_place(fz_context *ctx, pdf_document *dst, pdf_obj *dpage, pdf_obj *form)
{
	state_tracker before, after;
	fz_buffer *text = NULL;
	pdf_obj *pre = NULL;
	pdf_obj *post = NULL;
	char name[32];
	int clean = 0;

	fz_var(text);
	fz_var(pre);
	fz_var(post);
	fz_var(clean);

	// Content that ends inside a text object or inside marked content
	// cannot be closed off with q/Q, and content that cannot be interpreted
	// to the end cannot be measured at all.
	if (!track_page(ctx, dst, dpage, NULL, &before) || before.text_open || before.marked_open)
	{
		return 0;
	}

	fz_try(ctx)
	{
		pdf_obj *xobjs = own_xobject_dict(ctx, dst, dpage);
		int k;

		for (k = 0;; k++)
		{
			fz_snprintf(name, sizeof(name), "GoMuOv%d", k);
			if (!pdf_dict_gets(ctx, xobjs, name))
			{
				break;
			}
		}
		pdf_dict_puts(ctx, xobjs, name, form);

		// q ... Q around the existing content keeps whatever graphics state
		// it leaves behind from moving or recolouring the overlay. Real
		// content is not always balanced, so the wrapper is sized to it: one
		// extra q in front for every Q the content has too many of, and one
		// extra Q behind for every q it leaves open.
		text = fz_new_buffer(ctx, 64);
		for (k = 0; k <= before.underflow; k++)
		{
			fz_append_string(ctx, text, "q\n");
		}
		pre = pdf_add_stream(ctx, dst, text, NULL, 0);
		fz_drop_buffer(ctx, text);
		text = NULL;

		text = fz_new_buffer(ctx, 64);
		fz_append_string(ctx, text, "\n");
		for (k = 0; k <= before.open; k++)
		{
			fz_append_string(ctx, text, "Q\n");
		}
		fz_append_printf(ctx, text, "q\n/%s Do\nQ\n", name);
		post = pdf_add_stream(ctx, dst, text, NULL, 0);

		wrap_page_contents(ctx, dst, dpage, pre, post);

		// A viewer reads the /Contents array as one continuous stream, so
		// content can still swallow what follows it: an unterminated string
		// or inline image, for instance. Rather than try to predict every
		// such case, read the finished page back and see whether the overlay
		// really is drawn, once, with every wrapper level closed.
		// Depth 1 with no underflow means: all 1 + underflow wrapper levels
		// and everything the content opened have been popped, and only the
		// overlay's own q is open.
		if (track_page(ctx, dst, dpage, name, &after))
		{
			clean = after.target_seen == 1 && after.target_clean == 1;
		}
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, post);
		pdf_drop_obj(ctx, pre);
		fz_drop_buffer(ctx, text);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
	return clean;
}

// Draw the overlay after the page's content with that content sealed inside
// a form XObject of its own. A form is parsed on its own and its graphics
// state is saved and restored around it, so nothing it leaves open (q, BT,
// marked content, an unterminated token) can reach the overlay.
//
// The cost is that the content is no longer in the page's own stream, so a
// tagged PDF's structure tree loses its references into this page. That is
// why this is the fallback, used only for pages whose content is malformed.
static void overlay_isolated(fz_context *ctx, pdf_document *dst, pdf_obj *dpage, pdf_obj *form, gomupdf_budget *budget)
{
	fz_buffer *base_contents = NULL;
	pdf_obj *base_res = NULL;
	pdf_obj *base = NULL;
	pdf_obj *res = NULL;
	pdf_obj *xobjs = NULL;
	pdf_obj *draw = NULL;

	fz_var(base_contents);
	fz_var(base_res);
	fz_var(base);
	fz_var(res);
	fz_var(xobjs);
	fz_var(draw);

	fz_try(ctx)
	{
		pdf_obj *inherited = pdf_dict_get_inheritable(ctx, dpage, PDF_NAME(Resources));

		res = pdf_new_dict(ctx, dst, 1);
		xobjs = pdf_new_dict(ctx, dst, 2);
		pdf_dict_put(ctx, res, PDF_NAME(XObject), xobjs);
		pdf_dict_puts(ctx, xobjs, "GoMuOv0", form);

		base_contents = gomupdf_page_contents(ctx, dpage, budget);
		if (gomupdf_is_blank(ctx, base_contents))
		{
			draw = gomupdf_add_stream_from_string(ctx, dst, "q\n/GoMuOv0 Do\nQ\n");
		}
		else
		{
			base_res = pdf_is_dict(ctx, inherited) ? pdf_keep_obj(ctx, inherited) : pdf_new_dict(ctx, dst, 1);
			base = pdf_new_xobject(ctx, dst, gomupdf_page_media_box(ctx, dpage, NULL), fz_identity, base_res, base_contents);
			pdf_dict_puts(ctx, xobjs, "GoMuBase0", base);
			draw = gomupdf_add_stream_from_string(ctx, dst, "q\n/GoMuBase0 Do\nQ\nq\n/GoMuOv0 Do\nQ\n");
		}
		pdf_dict_put(ctx, dpage, PDF_NAME(Resources), res);
		pdf_dict_put(ctx, dpage, PDF_NAME(Contents), draw);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, draw);
		pdf_drop_obj(ctx, xobjs);
		pdf_drop_obj(ctx, res);
		pdf_drop_obj(ctx, base);
		pdf_drop_obj(ctx, base_res);
		fz_drop_buffer(ctx, base_contents);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

static void overlay_page(fz_context *ctx, pdf_document *dst, pdf_graft_map *map, pdf_obj *dpage, pdf_obj *spage, gomupdf_budget *budget)
{
	fz_buffer *contents = NULL;
	pdf_obj *res = NULL;
	pdf_obj *form = NULL;
	pdf_obj *saved_contents = NULL;
	pdf_obj *saved_res = NULL;

	fz_var(contents);
	fz_var(res);
	fz_var(form);
	fz_var(saved_contents);
	fz_var(saved_res);

	fz_try(ctx)
	{
		pdf_obj *sres;
		fz_rect bbox;

		contents = gomupdf_page_contents(ctx, spage, budget);
		if (!gomupdf_is_blank(ctx, contents))
		{
			// The overlay page becomes a form XObject drawn at the identity
			// transform, so a coordinate means the same in both pages. Its
			// clip is the union of the two MediaBoxes so nothing is cut off
			// when the boxes differ.
			sres = pdf_dict_get_inheritable(ctx, spage, PDF_NAME(Resources));
			res = pdf_is_dict(ctx, sres) ? pdf_graft_mapped_object(ctx, map, sres) : pdf_new_dict(ctx, dst, 1);
			bbox = fz_union_rect(gomupdf_page_media_box(ctx, spage, NULL), gomupdf_page_media_box(ctx, dpage, NULL));
			form = pdf_new_xobject(ctx, dst, bbox, fz_identity, res, contents);

			// The page's own entries (not inherited ones), so that a failed
			// in-place attempt can be undone exactly.
			saved_contents = pdf_keep_obj(ctx, pdf_dict_get(ctx, dpage, PDF_NAME(Contents)));
			saved_res = pdf_keep_obj(ctx, pdf_dict_get(ctx, dpage, PDF_NAME(Resources)));

			if (!overlay_in_place(ctx, dst, dpage, form))
			{
				if (saved_contents)
				{
					pdf_dict_put(ctx, dpage, PDF_NAME(Contents), saved_contents);
				}
				else
				{
					pdf_dict_del(ctx, dpage, PDF_NAME(Contents));
				}
				if (saved_res)
				{
					pdf_dict_put(ctx, dpage, PDF_NAME(Resources), saved_res);
				}
				else
				{
					pdf_dict_del(ctx, dpage, PDF_NAME(Resources));
				}
				overlay_isolated(ctx, dst, dpage, form, budget);
			}
		}
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, saved_res);
		pdf_drop_obj(ctx, saved_contents);
		pdf_drop_obj(ctx, form);
		pdf_drop_obj(ctx, res);
		fz_drop_buffer(ctx, contents);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

int gomupdf_overlay_pages(fz_context *ctx, const char *in, const char *overlay, const char *out, int64_t max_bytes, char *err)
{
	pdf_document *dst = NULL;
	pdf_document *src = NULL;
	pdf_graft_map *map = NULL;
	gomupdf_budget budget = { max_bytes, 0 };

	fz_var(dst);
	fz_var(src);
	fz_var(map);

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;
		int i, n, m;

		dst = pdf_open_document(ctx, in);
		src = pdf_open_document(ctx, overlay);
		n = pdf_count_pages(ctx, dst);
		m = pdf_count_pages(ctx, src);
		if (n != m)
		{
			fz_throw(ctx, FZ_ERROR_ARGUMENT, "page count mismatch: input has %d pages, overlay has %d", n, m);
		}

		map = pdf_new_graft_map(ctx, dst);
		for (i = 0; i < n; i++)
		{
			overlay_page(ctx, dst, map, pdf_lookup_page_obj(ctx, dst, i), pdf_lookup_page_obj(ctx, src, i), &budget);
		}

		opts.do_compress = 1;
		pdf_save_document(ctx, dst, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_graft_map(ctx, map);
		pdf_drop_document(ctx, src);
		pdf_drop_document(ctx, dst);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	return 0;
}
