#!/usr/bin/env bash
set -euo pipefail

latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Strip leading 'v', split on '.', bump patch
IFS='.' read -r major minor patch <<< "${latest#v}"
patch=$(( patch + 1 ))

echo "v${major}.${minor}.${patch}"
