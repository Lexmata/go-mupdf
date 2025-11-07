#!/bin/bash

# Install Git hooks for Go MuPDF Wrapper project
# This script sets up git to use hooks from the .githooks directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GITHOOKS_DIR="$SCRIPT_DIR/.githooks"
GIT_HOOKS_DIR="$SCRIPT_DIR/.git/hooks"

echo "🔧 Installing Git hooks for Go MuPDF Wrapper..."

# Check if .git directory exists
if [ ! -d "$SCRIPT_DIR/.git" ]; then
    echo "❌ Error: .git directory not found. Are you in the repository root?"
    exit 1
fi

# Check if .githooks directory exists
if [ ! -d "$GITHOOKS_DIR" ]; then
    echo "❌ Error: .githooks directory not found!"
    exit 1
fi

# Configure git to use hooks from .githooks directory
echo "📝 Configuring git to use hooks from .githooks directory..."
git config core.hooksPath .githooks

# Verify the configuration
if [ "$(git config core.hooksPath)" = ".githooks" ]; then
    echo "✅ Git hooks path configured successfully"
else
    echo "❌ Failed to configure git hooks path"
    exit 1
fi

# Make sure all hook scripts are executable
echo "🔐 Making hook scripts executable..."
find "$GITHOOKS_DIR" -type f -name "*" ! -name "*.md" ! -name "*.txt" -exec chmod +x {} \;

# List installed hooks
echo ""
echo "📋 Installed hooks:"
for hook in "$GITHOOKS_DIR"/*; do
    if [ -f "$hook" ] && [ -x "$hook" ]; then
        hook_name=$(basename "$hook")
        echo "   ✅ $hook_name"
    fi
done

echo ""
echo "✅ Git hooks installation complete!"
echo ""
echo "The following hooks are now active:"
echo "  - pre-commit: Runs linting and tests before commits"
echo "  - commit-msg: Validates commit message format (Conventional Commits)"
echo "  - prepare-commit-msg: Updates VERSION file when merging release branches"
echo "  - post-merge: Creates git tags when release branches merge to main"
echo ""
echo "To uninstall, run: git config --unset core.hooksPath"

