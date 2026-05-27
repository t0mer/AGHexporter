#!/usr/bin/env bash
set -euo pipefail

# Calendar versioning: YYYY.M.N — N resets to 0 each new month and increments within it.
year=$(date +%Y)
month=$(date +%-m)

latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -n "$latest" ]; then
  IFS='.' read -r t_year t_month t_patch <<< "$latest"
  if [ "$t_year" = "$year" ] && [ "$t_month" = "$month" ]; then
    patch=$(( t_patch + 1 ))
  else
    patch=0
  fi
else
  patch=0
fi

echo "${year}.${month}.${patch}"
