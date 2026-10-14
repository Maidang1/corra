#!/bin/sh
# Installs the GitHub Actions workflows into .github/workflows/.
# The GitHub Contents API refuses workflow files without the `workflow`
# OAuth scope, so the workflows live here and are installed by this script
# (or copy setup/workflows/*.yml into .github/workflows/ yourself).
cd "$(dirname "$0")/.." || exit 1
mkdir -p .github/workflows
cp setup/workflows/ci.yml setup/workflows/release.yml .github/workflows/
echo "workflows installed; commit and push to enable CI"
