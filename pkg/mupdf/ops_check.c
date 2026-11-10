#include "ops_internal.h"

static void check_warning(void *user, const char *message)
{
	gomupdf_check_result *res = user;
	size_t len = strlen(message);

	res->warning_count++;
	// Keep whole messages only, newline-separated; drop what does not fit.
	// The + 2 is the newline and the terminating NUL.
	if (res->used + len + 2 <= sizeof(res->warnings))
	{
		memcpy(res->warnings + res->used, message, len);
		res->used += len;
		res->warnings[res->used++] = '\n';
		res->warnings[res->used] = 0;
	}
}

// Errors MuPDF recovers from internally (a broken xref it then rebuilds, for
// example) are reported through the error callback rather than thrown. They
// explain why a file needed repair, so they are collected with the warnings.
static void check_error(void *user, const char *message)
{
	check_warning(user, message);
}

// Load one object and, if it is a stream, measure it against its /Length.
// An object that cannot be loaded is recorded as a warning: one bad object
// among thousands makes a file damaged, not unreadable.
static void check_object(fz_context *ctx, pdf_document *doc, int num, gomupdf_check_result *res)
{
	pdf_obj *obj = NULL;
	fz_buffer *buf = NULL;

	fz_var(obj);
	fz_var(buf);

	fz_try(ctx)
	{
		// Free entries are normal (deleted objects, and the gaps a
		// linearised or incrementally saved file leaves); loading one throws.
		if (pdf_object_exists(ctx, doc, num))
		{
			obj = pdf_load_object(ctx, doc, num);
			if (pdf_obj_num_is_stream(ctx, doc, num))
			{
				int64_t want = pdf_dict_get_int64(ctx, obj, PDF_NAME(Length));
				size_t got;

				// Streams are read raw: MuPDF itself reads a short stream
				// without complaint, and a short stream is what a truncated
				// file looks like. Nothing is decoded, so a scanned
				// document's images cost a read, not a decompression.
				buf = pdf_load_raw_stream_number(ctx, doc, num);
				got = fz_buffer_storage(ctx, buf, NULL);
				// In an encrypted file /Length counts the encrypted bytes
				// (AES adds an IV and padding) while the buffer holds the
				// decrypted ones, so the two cannot be compared. doc->crypt
				// is a struct field, not an accessor: MuPDF 1.27 has no
				// public "is this document encrypted" call for a
				// pdf_document, so re-check this when the library is bumped.
				if (!doc->crypt && want > 0 && (int64_t)got < want)
				{
					// MuPDF's formatter reads %ld as int64_t.
					fz_warn(ctx, "stream %d holds %zu bytes, fewer than its /Length of %ld", num, got, want);
				}
			}
		}
	}
	fz_always(ctx)
	{
		fz_drop_buffer(ctx, buf);
		pdf_drop_obj(ctx, obj);
	}
	fz_catch(ctx)
	{
		char message[GOMUPDF_ERRLEN];
		int code = fz_caught(ctx);

		if (gomupdf_is_fatal(code))
		{
			fz_rethrow(ctx);
		}
		fz_snprintf(message, sizeof(message), "object %d: %s", num, fz_convert_error(ctx, &code));
		check_warning(res, message);
	}
}

int gomupdf_check(fz_context *ctx, const char *path, gomupdf_check_result *res, fz_cookie *cancel, char *err)
{
	pdf_document *doc = NULL;
	void *old_warn_user = NULL;
	void *old_err_user = NULL;
	fz_warning_cb *old_warn = fz_warning_callback(ctx, &old_warn_user);
	fz_error_cb *old_err = fz_error_callback(ctx, &old_err_user);

	fz_var(doc);

	memset(res, 0, sizeof(*res));
	fz_flush_warnings(ctx);
	fz_set_warning_callback(ctx, check_warning, res);
	fz_set_error_callback(ctx, check_error, res);

	fz_try(ctx)
	{
		int i, n, len;

		doc = pdf_open_document(ctx, path);

		// Walk the page tree, then every object in the xref.
		n = pdf_count_pages(ctx, doc);
		len = pdf_xref_len(ctx, doc);
		for (i = 0; i < n && i < len; i++)
		{
			gomupdf_check_cancel(ctx, cancel);
			pdf_lookup_page_obj(ctx, doc, i);
		}
		for (i = 1; i < len; i++)
		{
			gomupdf_check_cancel(ctx, cancel);
			check_object(ctx, doc, i, res);
		}
		res->repaired = pdf_was_repaired(ctx, doc);
	}
	fz_always(ctx)
	{
		pdf_drop_document(ctx, doc);
		// res lives in the caller's memory and is about to go out of scope.
		// The callbacks must stop pointing at it on every way out, and they
		// are put back here, before fz_catch, so that the fatal error is
		// returned to the caller instead of also being collected.
		fz_flush_warnings(ctx);
		fz_set_warning_callback(ctx, old_warn, old_warn_user);
		fz_set_error_callback(ctx, old_err, old_err_user);
	}
	fz_catch(ctx)
	{
		gomupdf_set_err(ctx, err);
		return -1;
	}
	return 0;
}
