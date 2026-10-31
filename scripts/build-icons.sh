#!/usr/bin/env bash
# Render the SVG sources into the PNG files used by the app and packages.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

if ! command -v rsvg-convert >/dev/null 2>&1; then
  echo "rsvg-convert is required (librsvg2-bin / librsvg)." >&2
  exit 1
fi

rsvg-convert -w 512 -h 512 assets/icon.svg -o icon.png
rsvg-convert -w 512 -h 512 assets/icon-official.svg -o assets/icon-official-512.png
rsvg-convert -w 256 -h 256 assets/icon.svg \
  -o packaging/debian/package-root/usr/share/icons/hicolor/256x256/apps/sparkstore.png

echo "icons rendered: icon.png, assets/icon-official-512.png, packaging icon"
