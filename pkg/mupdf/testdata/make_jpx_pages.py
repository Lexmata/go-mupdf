"""Generate jpx_pages.pdf, the fixture for TestConcurrentJPXRendering.

Each page is one JPEG 2000 (/JPXDecode) image. MuPDF decodes JPX through
OpenJPEG, whose allocator hooks are process-wide, so decoding JPX on several
contexts at once is what exposes contexts created without shared locks
(MPDF-139). The image content is deterministic noise: it only has to be big
enough that the decodes overlap in time.

Run with Pillow installed (JPEG 2000 support needs OpenJPEG):

    python3 make_jpx_pages.py jpx_pages.pdf
"""
import io
import random
import sys

from PIL import Image

PAGES = 4
WIDTH, HEIGHT = 400, 500


def jpx_image(seed):
    rng = random.Random(seed)
    img = Image.new("L", (WIDTH, HEIGHT), 255)
    px = img.load()
    for y in range(0, HEIGHT, 3):
        for x in range(0, WIDTH, 3):
            px[x, y] = rng.randrange(256)
    out = io.BytesIO()
    img.save(out, "JPEG2000")
    return out.getvalue()


def main(path):
    objects = [b"<< /Type /Catalog /Pages 2 0 R >>", None]
    kids = []
    for i in range(PAGES):
        data = jpx_image(i)
        page = len(objects) + 1
        kids.append(b"%d 0 R" % page)
        content = b"q 612 0 0 792 0 0 cm /Im0 Do Q"
        objects.append(b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "
                       b"/Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>" % (page + 2, page + 1))
        objects.append(b"<< /Length %d >>\nstream\n%s\nendstream" % (len(content), content))
        objects.append(b"<< /Type /XObject /Subtype /Image /Width %d /Height %d /Filter /JPXDecode "
                       b"/Length %d >>\nstream\n%s\nendstream" % (WIDTH, HEIGHT, len(data), data))
    objects[1] = b"<< /Type /Pages /Kids [%s] /Count %d >>" % (b" ".join(kids), PAGES)

    out = io.BytesIO()
    out.write(b"%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
    offsets = []
    for n, body in enumerate(objects, start=1):
        offsets.append(out.tell())
        out.write(b"%d 0 obj\n%s\nendobj\n" % (n, body))
    xref = out.tell()
    out.write(b"xref\n0 %d\n0000000000 65535 f \n" % (len(objects) + 1))
    for off in offsets:
        out.write(b"%010d 00000 n \n" % off)
    out.write(b"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n" % (len(objects) + 1, xref))
    with open(path, "wb") as f:
        f.write(out.getvalue())


if __name__ == "__main__":
    main(sys.argv[1])
