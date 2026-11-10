#include "ops_internal.h"

// The transform taking a page's stored coordinates to the coordinates it is
// displayed in once /Rotate (clockwise) is applied, with the displayed
// MediaBox's lower-left corner at the origin.
static fz_matrix upright_matrix(fz_rect mb, int rot)
{
	switch (rot)
	{
	case 90:
		return fz_make_matrix(0, -1, 1, 0, -mb.y0, mb.x1);
	case 180:
		return fz_make_matrix(-1, 0, 0, -1, mb.x1, mb.y1);
	case 270:
		return fz_make_matrix(0, 1, -1, 0, mb.y1, -mb.x0);
	default:
		return fz_identity;
	}
}

// The pages that were turned, so that destinations pointing at them can be
// turned too.
typedef struct
{
	int num; // object number of the page
	int rot;
	fz_matrix m;
} turned_page;

typedef struct
{
	turned_page *v;
	int len;
	int cap;
} turned_pages;

static void turned_pages_add(fz_context *ctx, turned_pages *pages, int num, int rot, fz_matrix m)
{
	if (pages->len == pages->cap)
	{
		int cap = pages->cap ? pages->cap * 2 : 16;

		pages->v = fz_realloc_array(ctx, pages->v, cap, turned_page);
		pages->cap = cap;
	}
	pages->v[pages->len].num = num;
	pages->v[pages->len].rot = rot;
	pages->v[pages->len].m = m;
	pages->len++;
}

static const turned_page *turned_pages_find(const turned_pages *pages, int num)
{
	int i;

	for (i = 0; i < pages->len; i++)
	{
		if (pages->v[i].num == num)
		{
			return &pages->v[i];
		}
	}
	return NULL;
}

// Appearance streams already copied for the page being turned, so that a
// stream shared by many annotations on the page is copied once.
typedef struct
{
	int *src;
	pdf_obj **dst;
	int len;
	int cap;
} appearance_map;

static pdf_obj *appearance_map_get(const appearance_map *map, int num)
{
	int i;

	for (i = 0; i < map->len; i++)
	{
		if (map->src[i] == num)
		{
			return map->dst[i];
		}
	}
	return NULL;
}

static void appearance_map_put(fz_context *ctx, appearance_map *map, int num, pdf_obj *copy)
{
	if (map->len == map->cap)
	{
		int cap = map->cap ? map->cap * 2 : 16;

		map->src = fz_realloc_array(ctx, map->src, cap, int);
		map->dst = fz_realloc_array(ctx, map->dst, cap, pdf_obj *);
		map->cap = cap;
	}
	map->src[map->len] = num;
	map->dst[map->len] = pdf_keep_obj(ctx, copy);
	map->len++;
}

static void appearance_map_free(fz_context *ctx, appearance_map *map)
{
	int i;

	for (i = 0; i < map->len; i++)
	{
		pdf_drop_obj(ctx, map->dst[i]);
	}
	fz_free(ctx, map->src);
	fz_free(ctx, map->dst);
}

// Everything upright_page needs while turning one page.
typedef struct
{
	pdf_document *doc;
	fz_matrix m;
	int rot;
	gomupdf_budget *budget;
	appearance_map copies;
	gomupdf_num_set annots;
} page_turn;

// A stream object with the same data and dictionary as an appearance
// stream, and its /Matrix followed by the page's turn. Returns an owned
// reference.
//
// The original is copied, never changed: appearance streams are shared
// between annotations, including annotations on pages that are rotated
// differently or not at all. Within one page each stream is copied once.
static pdf_obj *turned_appearance(fz_context *ctx, page_turn *turn, pdf_obj *stream)
{
	pdf_obj *dict = NULL;
	pdf_obj *ref = NULL;
	fz_buffer *buf = NULL;
	int num = pdf_to_num(ctx, stream);
	pdf_obj *known = appearance_map_get(&turn->copies, num);

	if (known)
	{
		return pdf_keep_obj(ctx, known);
	}

	fz_var(dict);
	fz_var(ref);
	fz_var(buf);

	fz_try(ctx)
	{
		fz_matrix rotation = turn->m;

		rotation.e = 0;
		rotation.f = 0;

		dict = pdf_copy_dict(ctx, stream);
		buf = pdf_load_raw_stream(ctx, stream);
		gomupdf_budget_charge(ctx, turn->budget, fz_buffer_storage(ctx, buf, NULL));
		ref = pdf_add_object(ctx, turn->doc, dict);
		pdf_update_stream(ctx, turn->doc, ref, buf, 1);
		// An appearance is fitted to the annotation's /Rect by scaling and
		// translating only, so the rotation has to go into the form's own
		// matrix. A missing /Matrix reads as the identity.
		pdf_dict_put_matrix(ctx, ref, PDF_NAME(Matrix), fz_concat(pdf_dict_get_matrix(ctx, stream, PDF_NAME(Matrix)), rotation));
		appearance_map_put(ctx, &turn->copies, num, ref);
	}
	fz_always(ctx)
	{
		fz_drop_buffer(ctx, buf);
		pdf_drop_obj(ctx, dict);
	}
	fz_catch(ctx)
	{
		pdf_drop_obj(ctx, ref);
		fz_rethrow(ctx);
	}
	return ref;
}

// Give an annotation its own /AP whose streams are turned copies.
static void turn_appearances(fz_context *ctx, page_turn *turn, pdf_obj *annot)
{
	pdf_obj *ap = pdf_dict_get(ctx, annot, PDF_NAME(AP));
	pdf_obj *new_ap = NULL;
	pdf_obj *states = NULL;
	pdf_obj *keys[3];
	int k, i, n;

	fz_var(new_ap);
	fz_var(states);

	if (!pdf_is_dict(ctx, ap))
	{
		return;
	}
	keys[0] = PDF_NAME(N);
	keys[1] = PDF_NAME(R);
	keys[2] = PDF_NAME(D);

	fz_try(ctx)
	{
		new_ap = pdf_copy_dict(ctx, ap);
		for (k = 0; k < 3; k++)
		{
			pdf_obj *entry = pdf_dict_get(ctx, new_ap, keys[k]);

			if (pdf_is_stream(ctx, entry))
			{
				pdf_dict_put_drop(ctx, new_ap, keys[k], turned_appearance(ctx, turn, entry));
			}
			else if (pdf_is_dict(ctx, entry))
			{
				// A dictionary of appearance states (checkboxes, radio
				// buttons).
				states = pdf_copy_dict(ctx, entry);
				n = pdf_dict_len(ctx, states);
				for (i = 0; i < n; i++)
				{
					pdf_obj *val = pdf_dict_get_val(ctx, states, i);

					if (pdf_is_stream(ctx, val))
					{
						pdf_dict_put_drop(ctx, states, pdf_dict_get_key(ctx, states, i), turned_appearance(ctx, turn, val));
					}
				}
				pdf_dict_put(ctx, new_ap, keys[k], states);
				pdf_drop_obj(ctx, states);
				states = NULL;
			}
		}
		pdf_dict_put(ctx, annot, PDF_NAME(AP), new_ap);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, states);
		pdf_drop_obj(ctx, new_ap);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

// Transform an array of alternating x and y numbers in place.
static void transform_point_array(fz_context *ctx, pdf_obj *arr, fz_matrix m)
{
	int n = pdf_array_len(ctx, arr);
	int j;

	for (j = 0; j + 1 < n; j += 2)
	{
		fz_point p = fz_make_point(pdf_array_get_real(ctx, arr, j), pdf_array_get_real(ctx, arr, j + 1));

		p = fz_transform_point(p, m);
		pdf_array_put_real(ctx, arr, j, p.x);
		pdf_array_put_real(ctx, arr, j + 1, p.y);
	}
}

static void upright_annotation(fz_context *ctx, page_turn *turn, pdf_obj *annot)
{
	static const char *const point_keys[] = { "QuadPoints", "L", "Vertices", "CL" };
	pdf_obj *rect_obj = pdf_dict_get(ctx, annot, PDF_NAME(Rect));
	pdf_obj *rd = pdf_dict_get(ctx, annot, PDF_NAME(RD));
	pdf_obj *ink = pdf_dict_get(ctx, annot, PDF_NAME(InkList));
	pdf_obj *mk = pdf_dict_get(ctx, annot, PDF_NAME(MK));
	pdf_obj *new_mk = NULL;
	int has_rect = pdf_is_array(ctx, rect_obj) && pdf_array_len(ctx, rect_obj) >= 4;
	fz_rect rect = has_rect ? pdf_to_rect(ctx, rect_obj) : fz_empty_rect;
	fz_matrix m = turn->m;
	size_t k;
	int i, n;

	fz_var(new_mk);

	if (pdf_dict_get_int(ctx, annot, PDF_NAME(F)) & PDF_ANNOT_IS_NO_ROTATE)
	{
		// A NoRotate annotation is not turned with the page: it is drawn
		// upright, hanging from wherever the upper-left corner of its /Rect
		// lands. So only that corner moves, and its appearance is left as
		// it is.
		if (has_rect)
		{
			fz_point anchor = fz_transform_point(fz_make_point(rect.x0, rect.y1), m);
			float w = rect.x1 - rect.x0;
			float h = rect.y1 - rect.y0;

			pdf_dict_put_rect(ctx, annot, PDF_NAME(Rect), fz_make_rect(anchor.x, anchor.y - h, anchor.x + w, anchor.y));
		}
		return;
	}

	if (has_rect)
	{
		fz_rect turned = fz_transform_rect(rect, m);

		// /RD holds the four insets from /Rect to the drawn shape. Turning
		// the inset rectangle along with /Rect and measuring again moves
		// each inset to the side it now belongs to.
		if (pdf_is_array(ctx, rd) && pdf_array_len(ctx, rd) >= 4)
		{
			fz_rect inner = fz_make_rect(
				rect.x0 + pdf_array_get_real(ctx, rd, 0), rect.y0 + pdf_array_get_real(ctx, rd, 1),
				rect.x1 - pdf_array_get_real(ctx, rd, 2), rect.y1 - pdf_array_get_real(ctx, rd, 3));

			inner = fz_transform_rect(inner, m);
			pdf_dict_put_rect(ctx, annot, PDF_NAME(RD), fz_make_rect(
				inner.x0 - turned.x0, inner.y0 - turned.y0, turned.x1 - inner.x1, turned.y1 - inner.y1));
		}
		pdf_dict_put_rect(ctx, annot, PDF_NAME(Rect), turned);
	}

	for (k = 0; k < nelem(point_keys); k++)
	{
		transform_point_array(ctx, pdf_dict_gets(ctx, annot, point_keys[k]), m);
	}
	n = pdf_array_len(ctx, ink);
	for (i = 0; i < n; i++)
	{
		transform_point_array(ctx, pdf_array_get(ctx, ink, i), m);
	}

	// A widget's /MK /R says how its appearance is turned (counterclockwise)
	// relative to the page. The page has just been turned rot degrees
	// clockwise underneath it, so a viewer that regenerates the appearance
	// needs the difference. /MK is copied because it can be shared. Both
	// numbers are reduced modulo 360 before subtracting so that an extreme
	// /R in the file cannot overflow.
	if (pdf_name_eq(ctx, pdf_dict_get(ctx, annot, PDF_NAME(Subtype)), PDF_NAME(Widget)))
	{
		fz_try(ctx)
		{
			int r = pdf_dict_get_int(ctx, mk, PDF_NAME(R)) % 360;

			new_mk = pdf_is_dict(ctx, mk) ? pdf_copy_dict(ctx, mk) : pdf_new_dict(ctx, turn->doc, 1);
			pdf_dict_put_int(ctx, new_mk, PDF_NAME(R), ((r - turn->rot) % 360 + 360) % 360);
			pdf_dict_put(ctx, annot, PDF_NAME(MK), new_mk);
		}
		fz_always(ctx)
		{
			pdf_drop_obj(ctx, new_mk);
		}
		fz_catch(ctx)
		{
			fz_rethrow(ctx);
		}
	}

	turn_appearances(ctx, turn, annot);
}

static void upright_page(fz_context *ctx, pdf_document *doc, pdf_obj *page, int rot, gomupdf_budget *budget, turned_pages *turned)
{
	pdf_obj *boxes[] = {
		PDF_NAME(MediaBox), PDF_NAME(CropBox), PDF_NAME(BleedBox), PDF_NAME(TrimBox), PDF_NAME(ArtBox),
	};
	fz_rect mb = gomupdf_page_media_box(ctx, page, NULL);
	pdf_obj *annots = pdf_dict_get(ctx, page, PDF_NAME(Annots));
	page_turn turn = { 0 };
	fz_buffer *contents = NULL;
	pdf_obj *res = NULL;
	pdf_obj *form = NULL;
	pdf_obj *new_res = NULL;
	pdf_obj *xobjs = NULL;
	pdf_obj *draw = NULL;
	size_t b;
	int i, n;

	fz_var(contents);
	fz_var(res);
	fz_var(form);
	fz_var(new_res);
	fz_var(xobjs);
	fz_var(draw);

	turn.doc = doc;
	turn.m = upright_matrix(mb, rot);
	turn.rot = rot;
	turn.budget = budget;

	fz_try(ctx)
	{
		pdf_obj *inherited;

		// The page's content moves, whole, into a form XObject whose matrix
		// is the rotation. Prefixing a `cm` to the content would turn the
		// geometry but not tiling or shading patterns, which are positioned
		// relative to the enclosing content stream's base coordinate space;
		// a form's matrix is part of that base, so patterns turn with it.
		// A form also contains anything the content leaves open.
		contents = gomupdf_page_contents(ctx, page, budget);
		if (!gomupdf_is_blank(ctx, contents))
		{
			inherited = pdf_dict_get_inheritable(ctx, page, PDF_NAME(Resources));
			res = pdf_is_dict(ctx, inherited) ? pdf_keep_obj(ctx, inherited) : pdf_new_dict(ctx, doc, 1);
			form = pdf_new_xobject(ctx, doc, mb, turn.m, res, contents);

			new_res = pdf_new_dict(ctx, doc, 1);
			xobjs = pdf_new_dict(ctx, doc, 1);
			pdf_dict_puts(ctx, xobjs, "GoMuUp0", form);
			pdf_dict_put(ctx, new_res, PDF_NAME(XObject), xobjs);
			pdf_dict_put(ctx, page, PDF_NAME(Resources), new_res);

			draw = gomupdf_add_stream_from_string(ctx, doc, "/GoMuUp0 Do\n");
			pdf_dict_put(ctx, page, PDF_NAME(Contents), draw);
		}

		// Every box is written onto the page itself, so one that was
		// inherited stops being shared with pages that have a different
		// rotation.
		pdf_dict_put_rect(ctx, page, PDF_NAME(MediaBox), fz_transform_rect(mb, turn.m));
		for (b = 1; b < nelem(boxes); b++)
		{
			pdf_obj *box = pdf_dict_get_inheritable(ctx, page, boxes[b]);

			if (pdf_is_array(ctx, box) && pdf_array_len(ctx, box) >= 4)
			{
				pdf_dict_put_rect(ctx, page, boxes[b], fz_transform_rect(pdf_to_rect(ctx, box), turn.m));
			}
		}
		pdf_dict_put_int(ctx, page, PDF_NAME(Rotate), 0);

		n = pdf_array_len(ctx, annots);
		for (i = 0; i < n; i++)
		{
			pdf_obj *annot = pdf_array_get(ctx, annots, i);

			// The same annotation object listed twice must be turned once.
			if (pdf_is_dict(ctx, annot) && (!pdf_is_indirect(ctx, annot) || gomupdf_num_set_add(ctx, &turn.annots, pdf_to_num(ctx, annot))))
			{
				upright_annotation(ctx, &turn, annot);
			}
		}

		turned_pages_add(ctx, turned, pdf_to_num(ctx, page), rot, turn.m);
	}
	fz_always(ctx)
	{
		gomupdf_num_set_free(ctx, &turn.annots);
		appearance_map_free(ctx, &turn.copies);
		pdf_drop_obj(ctx, draw);
		pdf_drop_obj(ctx, xobjs);
		pdf_drop_obj(ctx, new_res);
		pdf_drop_obj(ctx, form);
		pdf_drop_obj(ctx, res);
		fz_drop_buffer(ctx, contents);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

// ---------------------------------------------------------------------------
// Destinations
// ---------------------------------------------------------------------------

// Everything needed while rewriting destinations that point at turned pages.
typedef struct
{
	const turned_pages *pages;
	gomupdf_num_set arrays;  // indirect destination arrays already rewritten
	gomupdf_num_set visited; // outline items already walked
} dest_fix;

static void put_coord(fz_context *ctx, pdf_obj *arr, int i, int present, float value)
{
	if (present)
	{
		pdf_array_put_real(ctx, arr, i, value);
	}
	else
	{
		pdf_array_put(ctx, arr, i, PDF_NULL);
	}
}

// Rewrite one explicit destination, [page /Kind coords...], if its page was
// turned. Coordinates are in the page's stored space, which has just moved.
//
// A quarter turn swaps the axes: what was a "top" coordinate now says where
// to put the left edge, and the other way round. So /FitH (a top) becomes
// /FitV (a left) and vice versa, and in /XYZ a missing coordinate ("keep
// the current position") ends up on the other axis.
static void fix_dest_array(fz_context *ctx, dest_fix *fix, pdf_obj *dest)
{
	const turned_page *page;
	pdf_obj *target, *kind;
	int quarter;
	int n;

	if (!pdf_is_array(ctx, dest))
	{
		return;
	}
	n = pdf_array_len(ctx, dest);
	target = pdf_array_get(ctx, dest, 0);
	if (n < 2 || !pdf_is_indirect(ctx, target))
	{
		// A page number rather than a page reference means a destination
		// in another file.
		return;
	}
	page = turned_pages_find(fix->pages, pdf_to_num(ctx, target));
	if (!page)
	{
		return;
	}
	if (pdf_is_indirect(ctx, dest) && !gomupdf_num_set_add(ctx, &fix->arrays, pdf_to_num(ctx, dest)))
	{
		return;
	}

	kind = pdf_array_get(ctx, dest, 1);
	quarter = page->rot == 90 || page->rot == 270;

	if (pdf_name_eq(ctx, kind, PDF_NAME(XYZ)) && n >= 4)
	{
		pdf_obj *left = pdf_array_get(ctx, dest, 2);
		pdf_obj *top = pdf_array_get(ctx, dest, 3);
		int has_left = pdf_is_number(ctx, left);
		int has_top = pdf_is_number(ctx, top);
		fz_point p = fz_transform_point(fz_make_point(has_left ? pdf_to_real(ctx, left) : 0, has_top ? pdf_to_real(ctx, top) : 0), page->m);

		put_coord(ctx, dest, 2, quarter ? has_top : has_left, p.x);
		put_coord(ctx, dest, 3, quarter ? has_left : has_top, p.y);
	}
	else if (pdf_name_eq(ctx, kind, PDF_NAME(FitR)) && n >= 6)
	{
		fz_rect r = fz_make_rect(pdf_array_get_real(ctx, dest, 2), pdf_array_get_real(ctx, dest, 3),
			pdf_array_get_real(ctx, dest, 4), pdf_array_get_real(ctx, dest, 5));

		r = fz_transform_rect(r, page->m);
		pdf_array_put_real(ctx, dest, 2, r.x0);
		pdf_array_put_real(ctx, dest, 3, r.y0);
		pdf_array_put_real(ctx, dest, 4, r.x1);
		pdf_array_put_real(ctx, dest, 5, r.y1);
	}
	else if (n >= 3 && pdf_is_number(ctx, pdf_array_get(ctx, dest, 2)))
	{
		int horizontal = pdf_name_eq(ctx, kind, PDF_NAME(FitH)) || pdf_name_eq(ctx, kind, PDF_NAME(FitBH));
		int vertical = pdf_name_eq(ctx, kind, PDF_NAME(FitV)) || pdf_name_eq(ctx, kind, PDF_NAME(FitBV));
		int bounding = pdf_name_eq(ctx, kind, PDF_NAME(FitBH)) || pdf_name_eq(ctx, kind, PDF_NAME(FitBV));
		float v = pdf_array_get_real(ctx, dest, 2);
		fz_point p;

		if (!horizontal && !vertical)
		{
			return;
		}
		// FitH carries a top (a y), FitV a left (an x).
		p = fz_transform_point(horizontal ? fz_make_point(0, v) : fz_make_point(v, 0), page->m);
		if (quarter)
		{
			// The coordinate changed axis, and the kind changes with it.
			if (horizontal)
			{
				pdf_array_put(ctx, dest, 1, bounding ? PDF_NAME(FitBV) : PDF_NAME(FitV));
				pdf_array_put_real(ctx, dest, 2, p.x);
			}
			else
			{
				pdf_array_put(ctx, dest, 1, bounding ? PDF_NAME(FitBH) : PDF_NAME(FitH));
				pdf_array_put_real(ctx, dest, 2, p.y);
			}
		}
		else
		{
			pdf_array_put_real(ctx, dest, 2, horizontal ? p.y : p.x);
		}
	}
}

// A destination as it appears in a name tree or a PDF 1.1 /Dests
// dictionary: the array itself, or a dictionary holding it under /D.
static void fix_dest_value(fz_context *ctx, dest_fix *fix, pdf_obj *value)
{
	if (pdf_is_array(ctx, value))
	{
		fix_dest_array(ctx, fix, value);
	}
	else if (pdf_is_dict(ctx, value))
	{
		fix_dest_array(ctx, fix, pdf_dict_get(ctx, value, PDF_NAME(D)));
	}
}

// The destination of a link annotation, outline item or the catalog's
// /OpenAction: a /Dest entry, or a GoTo action's /D. Named destinations
// (a name or string here) are reached through the name tree instead.
static void fix_dest_holder(fz_context *ctx, dest_fix *fix, pdf_obj *holder)
{
	pdf_obj *action = pdf_dict_get(ctx, holder, PDF_NAME(A));

	fix_dest_array(ctx, fix, pdf_dict_get(ctx, holder, PDF_NAME(Dest)));
	if (pdf_name_eq(ctx, pdf_dict_get(ctx, action, PDF_NAME(S)), PDF_NAME(GoTo)))
	{
		fix_dest_array(ctx, fix, pdf_dict_get(ctx, action, PDF_NAME(D)));
	}
}

static void fix_named_dest(fz_context *ctx, pdf_obj *key, pdf_obj *val, void *arg)
{
	(void)key;
	fix_dest_value(ctx, arg, val);
}

static void fix_outline(fz_context *ctx, dest_fix *fix, pdf_obj *item, int depth)
{
	// Siblings are followed in a loop and children by recursion, so the
	// depth limit bounds the stack and the visited set bounds the loop.
	while (pdf_is_dict(ctx, item) && depth < 64)
	{
		if (pdf_is_indirect(ctx, item) && !gomupdf_num_set_add(ctx, &fix->visited, pdf_to_num(ctx, item)))
		{
			return;
		}
		fix_dest_holder(ctx, fix, item);
		fix_outline(ctx, fix, pdf_dict_get(ctx, item, PDF_NAME(First)), depth + 1);
		item = pdf_dict_get(ctx, item, PDF_NAME(Next));
	}
}

// Rewrite every explicit destination in the document that points at a page
// that was turned.
static void fix_destinations(fz_context *ctx, pdf_document *doc, const turned_pages *pages)
{
	dest_fix fix = { 0 };
	pdf_obj *root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));

	fix.pages = pages;

	fz_try(ctx)
	{
		pdf_obj *open_action = pdf_dict_gets(ctx, root, "OpenAction");
		pdf_obj *old_dests = pdf_dict_get(ctx, root, PDF_NAME(Dests));
		int i, j, n, count;

		// Links, on every page: a link on an upright page can point at a
		// turned one.
		count = pdf_count_pages(ctx, doc);
		for (i = 0; i < count; i++)
		{
			pdf_obj *annots = pdf_dict_get(ctx, pdf_lookup_page_obj(ctx, doc, i), PDF_NAME(Annots));

			n = pdf_array_len(ctx, annots);
			for (j = 0; j < n; j++)
			{
				fix_dest_holder(ctx, &fix, pdf_array_get(ctx, annots, j));
			}
		}

		// Bookmarks.
		fix_outline(ctx, &fix, pdf_dict_get(ctx, pdf_dict_get(ctx, root, PDF_NAME(Outlines)), PDF_NAME(First)), 0);

		// Named destinations: the name tree, and the PDF 1.1 dictionary.
		gomupdf_walk_name_tree(ctx, pdf_dict_get(ctx, pdf_dict_get(ctx, root, PDF_NAME(Names)), PDF_NAME(Dests)), fix_named_dest, &fix);
		n = pdf_dict_len(ctx, old_dests);
		for (i = 0; i < n; i++)
		{
			fix_dest_value(ctx, &fix, pdf_dict_get_val(ctx, old_dests, i));
		}

		// Where the document opens.
		if (pdf_is_array(ctx, open_action))
		{
			fix_dest_array(ctx, &fix, open_action);
		}
		else if (pdf_name_eq(ctx, pdf_dict_get(ctx, open_action, PDF_NAME(S)), PDF_NAME(GoTo)))
		{
			fix_dest_array(ctx, &fix, pdf_dict_get(ctx, open_action, PDF_NAME(D)));
		}
	}
	fz_always(ctx)
	{
		gomupdf_num_set_free(ctx, &fix.arrays);
		gomupdf_num_set_free(ctx, &fix.visited);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
}

int gomupdf_upright_pages(fz_context *ctx, const char *in, const char *out, int64_t max_bytes, int *changed, char *err)
{
	pdf_document *doc = NULL;
	gomupdf_budget budget = { max_bytes, 0 };
	turned_pages turned = { NULL, 0, 0 };

	fz_var(doc);

	*changed = 0;

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;
		int i, n;

		doc = pdf_open_document(ctx, in);
		n = pdf_count_pages(ctx, doc);
		for (i = 0; i < n; i++)
		{
			pdf_obj *page = pdf_lookup_page_obj(ctx, doc, i);
			int rot = gomupdf_page_rotation(ctx, page);

			// A page that is already upright is left exactly as it is,
			// apart from a malformed /Rotate being written as the 0 it
			// rendered as.
			if (rot == 0)
			{
				if (pdf_dict_get_inheritable_int(ctx, page, PDF_NAME(Rotate)) != 0)
				{
					pdf_dict_put_int(ctx, page, PDF_NAME(Rotate), 0);
				}
				continue;
			}
			// The same page object can sit in the page tree twice; it must
			// be turned once.
			if (turned_pages_find(&turned, pdf_to_num(ctx, page)))
			{
				continue;
			}
			upright_page(ctx, doc, page, rot, &budget, &turned);
		}
		if (turned.len > 0)
		{
			fix_destinations(ctx, doc, &turned);
		}

		opts.do_compress = 1;
		pdf_save_document(ctx, doc, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		fz_free(ctx, turned.v);
		gomupdf_set_err(ctx, err);
		return -1;
	}
	*changed = turned.len;
	fz_free(ctx, turned.v);
	return 0;
}
