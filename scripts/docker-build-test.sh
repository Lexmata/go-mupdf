#!/bin/bash
# Quick script to validate Docker setup
# This script checks if Docker is available and validates the Dockerfile syntax

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "=== Docker Setup Validation ==="

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed"
    exit 1
fi
echo "✅ Docker is installed: $(docker --version)"

# Check Dockerfile exists
if [ ! -f "Dockerfile" ]; then
    echo "❌ Dockerfile not found"
    exit 1
fi
echo "✅ Dockerfile found"

# Validate Dockerfile syntax with the best available validator
if docker buildx version > /dev/null 2>&1 && docker buildx build --help 2>/dev/null | grep -q -- '--check'; then
    echo "Validating Dockerfile with 'docker buildx build --check'..."
    if docker buildx build --check -f Dockerfile .; then
        echo "✅ Dockerfile syntax is valid (buildx --check)"
    else
        echo "❌ Dockerfile syntax error detected"
        exit 1
    fi
elif command -v hadolint > /dev/null 2>&1; then
    echo "Validating Dockerfile with hadolint..."
    if hadolint Dockerfile; then
        echo "✅ Dockerfile passed hadolint"
    else
        echo "❌ hadolint reported Dockerfile issues"
        exit 1
    fi
else
    echo "⚠️  Dockerfile syntax validation skipped (no validator available: needs 'docker buildx build --check' or hadolint)"
fi

# Check docker-compose if available
if command -v docker-compose &> /dev/null; then
    echo "✅ docker-compose is available: $(docker-compose --version)"
elif docker compose version &> /dev/null; then
    echo "✅ docker compose (v2) is available"
else
    echo "⚠️  docker-compose not found (optional)"
fi

echo ""
echo "=== Ready to build ==="
echo "Run: ./scripts/docker-test.sh"
echo "Or: docker build -t go-mupdf-test:latest ."

