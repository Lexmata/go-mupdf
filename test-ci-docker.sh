#!/bin/bash
# Legacy test script - redirects to new Docker testing infrastructure
# This script is kept for backwards compatibility

echo "⚠️  This script has been replaced with a new Docker testing infrastructure"
echo ""
echo "Please use one of the following instead:"
echo ""
echo "  make docker-test      # Run all tests in Docker"
echo "  make docker-quick     # Run tests without rebuilding"
echo "  make docker-coverage  # Generate coverage report"
echo "  make docker-shell     # Open interactive shell"
echo ""
echo "For more options, run:"
echo "  ./scripts/docker-test.sh help"
echo ""
echo "Or see the Docker Testing Guide:"
echo "  docs/DOCKER_TESTING.md"
echo ""
echo "Redirecting to: ./scripts/docker-test.sh test"
echo ""

exec ./scripts/docker-test.sh test

