#!/bin/bash
# Verify that mupdf.lock matches the committed MuPDF submodule pointer.
#
# mupdf.lock is part of the Bitbucket Pipelines cache key for the pre-built
# MuPDF libraries (see the mupdf-libs cache in bitbucket-pipelines.yml).
# .gitmodules does not change when the submodule pointer moves, so without
# this pin the cache would serve libraries built from a different MuPDF
# commit. Nothing about that failure is visible at build time, so the pin is
# checked explicitly here.
#
# Usage:
#   scripts/check-mupdf-lock.sh            # check HEAD's submodule pointer
#   scripts/check-mupdf-lock.sh --staged   # check the staged pointer (pre-commit)
#   scripts/check-mupdf-lock.sh --fix      # rewrite mupdf.lock to match
#
# Exits non-zero when the pin is stale or missing.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOCK_FILE="$PROJECT_ROOT/mupdf.lock"

MODE="head"
case "${1:-}" in
    --staged) MODE="staged" ;;
    --fix)    MODE="fix" ;;
    "")       ;;
    *)
        echo "Usage: $0 [--staged|--fix]" >&2
        exit 2
        ;;
esac

cd "$PROJECT_ROOT" || exit 1

# Read the submodule pointer (a commit SHA) from the tree, not the working
# directory: this works even when the submodule has never been initialised.
if [ "$MODE" = "staged" ]; then
    # "160000 <sha> 0<TAB>third_party/mupdf" for a gitlink entry.
    ACTUAL=$(git ls-files --stage -- third_party/mupdf 2>/dev/null | awk '$1 == "160000" {print $2; exit}')
else
    ACTUAL=$(git rev-parse "HEAD:third_party/mupdf" 2>/dev/null)
fi

if [ -z "$ACTUAL" ]; then
    echo "⚠️  Could not determine the MuPDF submodule pointer; skipping mupdf.lock check"
    exit 0
fi

if [ "$MODE" = "fix" ]; then
    echo "$ACTUAL" > "$LOCK_FILE"
    echo "✅ mupdf.lock updated to $ACTUAL"
    exit 0
fi

if [ ! -f "$LOCK_FILE" ]; then
    echo "❌ mupdf.lock is missing."
    echo "   Create it with: scripts/check-mupdf-lock.sh --fix"
    exit 1
fi

PINNED=$(tr -d '[:space:]' < "$LOCK_FILE")

if [ "$PINNED" != "$ACTUAL" ]; then
    echo "❌ mupdf.lock is stale."
    echo "   mupdf.lock pins:        $PINNED"
    echo "   submodule points at:    $ACTUAL"
    echo ""
    echo "   The MuPDF submodule moved without the pin being regenerated, so the"
    echo "   CI library cache would serve libraries from the old MuPDF commit."
    echo "   Fix with:  scripts/check-mupdf-lock.sh --fix && git add mupdf.lock"
    exit 1
fi

echo "✅ mupdf.lock matches the MuPDF submodule pointer ($ACTUAL)"
exit 0
