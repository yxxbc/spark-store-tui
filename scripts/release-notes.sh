#!/usr/bin/env bash
# Print the CHANGELOG section for the current VERSION. Release notes use the
# same text as the changelog, truncated to this release.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
if ! grep -q "^## \[$version\]" CHANGELOG.md; then
  echo "CHANGELOG.md has no section for $version" >&2
  exit 1
fi

awk -v version="$version" '
  $0 ~ "^## \\[" version "\\]" { printing = 1 }
  printing && /^## \[/ && $0 !~ "^## \\[" version "\\]" { exit }
  printing { print }
' CHANGELOG.md
