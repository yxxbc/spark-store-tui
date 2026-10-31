#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
aur_source="$root/dist/aur"
if [ ! -f "$aur_source/PKGBUILD" ] || [ ! -f "$aur_source/.SRCINFO" ]; then
  echo "dist/aur is missing; run scripts/build-dist.sh first." >&2
  exit 1
fi

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

git clone ssh://aur@aur.archlinux.org/spark-store-tui.git "$workdir/spark-store-tui"
install -m 0644 "$aur_source/PKGBUILD" "$workdir/spark-store-tui/PKGBUILD"
install -m 0644 "$aur_source/.SRCINFO" "$workdir/spark-store-tui/.SRCINFO"

cd "$workdir/spark-store-tui"
git config user.name "${GIT_AUTHOR_NAME:-Xynrin}"
git config user.email "${GIT_AUTHOR_EMAIL:-xynrin@163.com}"
if git diff --quiet; then
  echo 'AUR package is already current.'
  exit 0
fi
git add PKGBUILD .SRCINFO
version=$(awk -F= '/^pkgver=/ { print $2; exit }' PKGBUILD)
git commit -m "spark-store-tui $version"
git push
