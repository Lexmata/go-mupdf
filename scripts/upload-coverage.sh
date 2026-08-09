#!/bin/bash
# Upload a Go coverage profile to Codecov.
#
# Replaces the `curl -s https://codecov.io/bash | bash` pattern, which piped an
# unpinned, unverified script straight into a shell. This uses a pinned uploader
# version and verifies its SHA-256 checksum before executing it.
#
# Usage: scripts/upload-coverage.sh <coverage-file>
#
# CODECOV_TOKEN must be set; if it is not, the upload is skipped (not an error,
# so forks and local runs are unaffected).

set -euo pipefail

# Pinned uploader release. To bump: pick a new version, fetch
#   https://uploader.codecov.io/<version>/linux/codecov.SHA256SUM
# and update both values together.
CODECOV_UPLOADER_VERSION="v0.8.0"
CODECOV_UPLOADER_SHA256="b37359013b48fbc3b0790d59fc474a52a260fb96e28e1b2c2ae001dc9b9cc996"

COVERAGE_FILE="${1:-}"

if [ -z "$COVERAGE_FILE" ]; then
    echo "Usage: $0 <coverage-file>" >&2
    exit 1
fi

# Check the token before the file so builds without the secret (forks, PRs
# from outside the workspace) skip cleanly regardless of what was produced.
if [ -z "${CODECOV_TOKEN:-}" ]; then
    echo "CODECOV_TOKEN not set, skipping Codecov upload"
    exit 0
fi

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "Coverage file not found: $COVERAGE_FILE" >&2
    exit 1
fi

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

BASE_URL="https://uploader.codecov.io/${CODECOV_UPLOADER_VERSION}/linux"

echo "Downloading Codecov uploader ${CODECOV_UPLOADER_VERSION}..."
if ! curl -fsSL --retry 3 --retry-delay 2 -o "$WORK_DIR/codecov" "$BASE_URL/codecov"; then
    echo "Failed to download the Codecov uploader; skipping upload" >&2
    exit 0
fi

echo "Verifying uploader checksum..."
ACTUAL_SHA256=$(sha256sum "$WORK_DIR/codecov" | awk '{print $1}')
if [ "$ACTUAL_SHA256" != "$CODECOV_UPLOADER_SHA256" ]; then
    echo "Codecov uploader checksum mismatch!" >&2
    echo "  expected: $CODECOV_UPLOADER_SHA256" >&2
    echo "  actual:   $ACTUAL_SHA256" >&2
    echo "Refusing to execute an unverified binary." >&2
    exit 1
fi

chmod +x "$WORK_DIR/codecov"

echo "Uploading $COVERAGE_FILE to Codecov..."
# A failed upload must not fail the build; coverage reporting is advisory.
"$WORK_DIR/codecov" -f "$COVERAGE_FILE" -t "$CODECOV_TOKEN" || echo "Codecov upload failed"
