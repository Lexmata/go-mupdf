#!/bin/bash
# Install cpdf, which the test suite compares several operations against
# (GetPageGeometry, OverlayPages, UprightPages, CompressPDF).
#
# cpdf is not packaged for Debian, so the binary is fetched from the
# coherentgraphics/cpdf-binaries repository, pinned to an exact commit and
# SHA-256 (the same pin lexmata-ocr's Dockerfile uses). To bump: pick the
# cpdf-binaries commit for the new release, then `sha256sum` the
# Linux-Intel-64bit/cpdf file at that commit.

set -euo pipefail

CPDF_COMMIT="${CPDF_COMMIT:-4d44a0da36bd9acd21917c0bd437d542a91c14d1}"
CPDF_SHA256="${CPDF_SHA256:-37511a498ae58df68ba1f52003435743f23c285f634f552da267df1b74e822b9}"
DEST="${1:-/usr/local/bin/cpdf}"

if [ "$(uname -s)" != "Linux" ] || [ "$(uname -m)" != "x86_64" ]; then
    echo "install-cpdf.sh: only a Linux x86_64 binary is pinned; got $(uname -s) $(uname -m)" >&2
    exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

curl -fsSL "https://raw.githubusercontent.com/coherentgraphics/cpdf-binaries/${CPDF_COMMIT}/Linux-Intel-64bit/cpdf" -o "$tmp"
echo "${CPDF_SHA256}  ${tmp}" | sha256sum -c - >/dev/null

install -m 0755 "$tmp" "$DEST"
echo "Installed cpdf to ${DEST}"
