#include "ops_internal.h"

#include <stdlib.h>

// Whether dict is shaped like a name-tree node rather than a PDF 1.1
// name-to-destination dictionary: it holds only /Kids (an array of indirect
// references), /Names (an array alternating string keys and values) and
// /Limits. In a flat /Dests every key is a destination name and every value
// an array or dictionary, so even a destination literally called "Kids" or
// "Names" fails: a destination array is never all references, and never
// starts with a string.
static int is_name_tree_node(fz_context *ctx, pdf_obj *dict)
{
	pdf_obj *kids = pdf_dict_get(ctx, dict, PDF_NAME(Kids));
	pdf_obj *names = pdf_dict_get(ctx, dict, PDF_NAME(Names));
	int i, n;

	if (!pdf_is_dict(ctx, dict) || (!kids && !names))
	{
		return 0;
	}
	n = pdf_dict_len(ctx, dict);
	for (i = 0; i < n; i++)
	{
		pdf_obj *key = pdf_dict_get_key(ctx, dict, i);

		if (!pdf_name_eq(ctx, key, PDF_NAME(Kids)) && !pdf_name_eq(ctx, key, PDF_NAME(Names)) && !pdf_name_eq(ctx, key, PDF_NAME(Limits)))
		{
			return 0;
		}
	}
	if (kids)
	{
		if (!pdf_is_array(ctx, kids))
		{
			return 0;
		}
		n = pdf_array_len(ctx, kids);
		for (i = 0; i < n; i++)
		{
			if (!pdf_is_indirect(ctx, pdf_array_get(ctx, kids, i)))
			{
				return 0;
			}
		}
	}
	if (names)
	{
		if (!pdf_is_array(ctx, names))
		{
			return 0;
		}
		n = pdf_array_len(ctx, names);
		if (n % 2 != 0)
		{
			return 0;
		}
		for (i = 0; i < n; i += 2)
		{
			pdf_obj *key = pdf_array_get(ctx, names, i);

			if (pdf_is_indirect(ctx, key) || !pdf_is_string(ctx, key))
			{
				return 0;
			}
		}
	}
	return 1;
}

// One named destination. key and val are borrowed from the document; bytes
// and len are the key's string value, order is the position it was found
// at, which breaks ties so that the first tree collected wins.
typedef struct
{
	pdf_obj *key;
	pdf_obj *val;
	const char *bytes;
	size_t len;
	int order;
} dest_entry;

typedef struct
{
	dest_entry *v;
	int len;
	int cap;
} dest_list;

// More names than any real document has; a hostile one could otherwise make
// the merge allocate without bound.
#define MAX_DEST_ENTRIES 1000000

static void collect_dest(fz_context *ctx, pdf_obj *key, pdf_obj *val, void *arg)
{
	dest_list *list = arg;
	dest_entry *e;

	if (pdf_is_indirect(ctx, key) || !pdf_is_string(ctx, key))
	{
		return;
	}
	if (list->len >= MAX_DEST_ENTRIES)
	{
		fz_throw(ctx, FZ_ERROR_LIMIT, "more than %d named destinations", MAX_DEST_ENTRIES);
	}
	if (list->len == list->cap)
	{
		int cap = list->cap ? list->cap * 2 : 64;

		list->v = fz_realloc_array(ctx, list->v, cap, dest_entry);
		list->cap = cap;
	}
	e = &list->v[list->len];
	e->key = key;
	e->val = val;
	e->bytes = pdf_to_string(ctx, key, &e->len);
	e->order = list->len;
	list->len++;
}

static int compare_dest_names(const dest_entry *a, const dest_entry *b)
{
	size_t n = a->len < b->len ? a->len : b->len;
	int c = n ? memcmp(a->bytes, b->bytes, n) : 0;

	if (c != 0)
	{
		return c;
	}
	if (a->len != b->len)
	{
		return a->len < b->len ? -1 : 1;
	}
	return 0;
}

static int compare_dest_entries(const void *pa, const void *pb)
{
	const dest_entry *a = pa;
	const dest_entry *b = pb;
	int c = compare_dest_names(a, b);

	if (c != 0)
	{
		return c;
	}
	return a->order - b->order;
}

// Fold the names of the misplaced tree into the proper one.
//
// The two trees are not always copies of each other: `cpdf -combine-pages`
// writes the overlay's destinations into the catalog while /Names /Dests
// holds the input's. Dropping the misplaced tree unread would lose any name
// only it has. Names present in both keep the value from /Names /Dests.
//
// Returns 1 if the misplaced tree contributed a name, in which case
// /Names /Dests has been replaced by one flat, sorted leaf; 0 if it was a
// true subset and /Names /Dests was left exactly as it was.
static int merge_dests(fz_context *ctx, pdf_document *doc, pdf_obj *names_dict, pdf_obj *proper, pdf_obj *misplaced)
{
	dest_list list = { NULL, 0, 0 };
	pdf_obj *leaf = NULL;
	pdf_obj *pairs = NULL;
	int proper_count;
	int added = 0;
	int i;

	fz_var(leaf);
	fz_var(pairs);
	fz_var(added);

	fz_try(ctx)
	{
		gomupdf_walk_name_tree(ctx, proper, collect_dest, &list);
		proper_count = list.len;
		gomupdf_walk_name_tree(ctx, misplaced, collect_dest, &list);

		if (list.len > 0)
		{
			qsort(list.v, list.len, sizeof(*list.v), compare_dest_entries);
		}
		// After a stable-by-order sort the first of each run of equal names
		// is the one found earliest. If any such survivor came from the
		// misplaced tree, that tree had a name of its own.
		for (i = 0; i < list.len; i++)
		{
			int first_of_run = i == 0 || compare_dest_names(&list.v[i - 1], &list.v[i]) != 0;

			if (first_of_run && list.v[i].order >= proper_count)
			{
				added = 1;
				break;
			}
		}

		if (added)
		{
			leaf = pdf_new_dict(ctx, doc, 1);
			pairs = pdf_new_array(ctx, doc, list.len * 2);
			for (i = 0; i < list.len; i++)
			{
				if (i > 0 && compare_dest_names(&list.v[i - 1], &list.v[i]) == 0)
				{
					continue;
				}
				pdf_array_push(ctx, pairs, list.v[i].key);
				pdf_array_push(ctx, pairs, list.v[i].val);
			}
			pdf_dict_put(ctx, leaf, PDF_NAME(Names), pairs);
			// The borrowed keys and values stay alive: the new array holds
			// its own references to them before the old tree is let go.
			pdf_dict_put(ctx, names_dict, PDF_NAME(Dests), leaf);
		}
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, pairs);
		pdf_drop_obj(ctx, leaf);
		fz_free(ctx, list.v);
	}
	fz_catch(ctx)
	{
		fz_rethrow(ctx);
	}
	return added;
}

int gomupdf_repair_catalog_dests(fz_context *ctx, const char *in, const char *out, int *outcome, char *err)
{
	pdf_document *doc = NULL;
	pdf_obj *new_names = NULL;
	int result = GOMUPDF_DESTS_UNCHANGED;

	fz_var(doc);
	fz_var(new_names);
	fz_var(result);

	*outcome = GOMUPDF_DESTS_UNCHANGED;

	fz_try(ctx)
	{
		pdf_write_options opts = pdf_default_write_options;
		pdf_obj *root, *dests, *names;

		doc = pdf_open_document(ctx, in);
		root = pdf_dict_get(ctx, pdf_trailer(ctx, doc), PDF_NAME(Root));
		if (!pdf_is_dict(ctx, root))
		{
			fz_throw(ctx, FZ_ERROR_FORMAT, "document has no catalog");
		}

		dests = pdf_dict_get(ctx, root, PDF_NAME(Dests));
		if (dests && !pdf_is_null(ctx, dests) && is_name_tree_node(ctx, dests))
		{
			pdf_obj *existing;

			names = pdf_dict_get(ctx, root, PDF_NAME(Names));
			existing = pdf_is_dict(ctx, names) ? pdf_dict_get(ctx, names, PDF_NAME(Dests)) : NULL;
			if (existing && !pdf_is_null(ctx, existing))
			{
				result = merge_dests(ctx, doc, names, existing, dests) ? GOMUPDF_DESTS_MERGED : GOMUPDF_DESTS_REMOVED_DUPLICATE;
			}
			else if (pdf_is_dict(ctx, names))
			{
				pdf_dict_put(ctx, names, PDF_NAME(Dests), dests);
				result = GOMUPDF_DESTS_MOVED_TO_NAMES;
			}
			else
			{
				// /Names is absent, or not a dictionary, which no reader
				// could use anyway.
				new_names = pdf_new_dict(ctx, doc, 1);
				pdf_dict_put(ctx, new_names, PDF_NAME(Dests), dests);
				pdf_dict_put(ctx, root, PDF_NAME(Names), new_names);
				result = GOMUPDF_DESTS_MOVED_TO_NAMES;
			}
			pdf_dict_del(ctx, root, PDF_NAME(Dests));
			opts.do_garbage = 1;
		}

		// The output is written whether or not anything changed, so a
		// caller can always carry on with it and can never mistake a file
		// left at the output path by an earlier run for this run's result.
		pdf_save_document(ctx, doc, out, &opts);
	}
	fz_always(ctx)
	{
		pdf_drop_obj(ctx, new_names);
		pdf_drop_document(ctx, doc);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	*outcome = result;
	return 0;
}
