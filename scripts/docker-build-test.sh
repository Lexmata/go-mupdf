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

# Validate Dockerfile syntax (basic check)
if docker build --dry-run -f Dockerfile . 2>&1 | grep -q "error"; then
    echo "❌ Dockerfile syntax error detected"
    exit 1
fi
echo "✅ Dockerfile syntax appears valid"

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

