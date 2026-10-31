#!/usr/bin/env bash
# Verify that every place that declares a version agrees with VERSION.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
# shellcheck source=../packaging/revisions.env
. packaging/revisions.env

case "$version" in
  ''|*[!0-9.]*) echo "invalid VERSION: $version" >&2; exit 2 ;;
esac

status=0
expect_line() {
  file="$1" line="$2"
  if ! grep -Fqx "$line" "$file"; then
    echo "version mismatch: $file should contain: $line" >&2
    status=1
  fi
}

expect_line packaging/rpm/spark-store-tui.spec "Version:        $version"
expect_line packaging/rpm/spark-store-tui.spec "Release:        ${RPM_RELEASE}%{?dist}"
expect_line packaging/debian/package-root/DEBIAN/control "Version: ${version}-${DEB_REVISION}"

grep -Fq 'pkgver=@VERSION@' packaging/aur/PKGBUILD.in || {
  echo "version mismatch: packaging/aur/PKGBUILD.in must use @VERSION@" >&2
  status=1
}
grep -Fq 'pkgrel=@AUR_RELEASE@' packaging/aur/PKGBUILD.in || {
  echo "version mismatch: packaging/aur/PKGBUILD.in must use @AUR_RELEASE@" >&2
  status=1
}
grep -Fq 'pkgver = @VERSION@' packaging/aur/SRCINFO.in || {
  echo "version mismatch: packaging/aur/SRCINFO.in must use @VERSION@" >&2
  status=1
}
grep -Fq 'pkgrel = @AUR_RELEASE@' packaging/aur/SRCINFO.in || {
  echo "version mismatch: packaging/aur/SRCINFO.in must use @AUR_RELEASE@" >&2
  status=1
}

# The 0.8.3 compatibility branch in the installer is the only allowed reference
# to the old entrypoint.
stale=$(grep -Rn --exclude-dir=.git --exclude-dir=legacy --exclude-dir=dist --exclude-dir=build \
  'cmd/spark-store-tui' . 2>/dev/null | grep -v 'scripts/install-sparkstore.sh' || true)
if [ -n "$stale" ]; then
  echo "stale entrypoint reference:" >&2
  echo "$stale" >&2
  status=1
fi

stale=$(grep -RnE --exclude-dir=.git --exclude-dir=legacy --exclude-dir=dist --exclude-dir=build \
  '(^|[^/])package-root/' . 2>/dev/null | grep -v 'packaging/debian/package-root' | grep -v 'scripts/install-sparkstore.sh' || true)
if [ -n "$stale" ]; then
  echo "stale package-root reference:" >&2
  echo "$stale" >&2
  status=1
fi

exit "$status"
