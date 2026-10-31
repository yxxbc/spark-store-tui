#!/usr/bin/env bash
# Build every release artifact from one source of truth.
#
#   scripts/build-dist.sh                 # deb + rpm + source + AUR + checksums
#   scripts/build-dist.sh --skip-rpm      # Debian-only host
#   scripts/build-dist.sh --skip-deb --skip-source   # RPM-only rebuild
#
# Requires: go, tar, gzip; dpkg-deb for Deb; rpmbuild for RPM.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

VERSION=${VERSION:-$(tr -d '[:space:]' < VERSION)}
# shellcheck source=../packaging/revisions.env
. packaging/revisions.env

OUT=${OUT:-dist}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}

DEB_ARCHS=${DEB_ARCHS:-"amd64 arm64 loong64"}
RPM_TARGETS=${RPM_TARGETS:-"x86_64 aarch64 loongarch64"}
SKIP_DEB=0
SKIP_RPM=0
SKIP_SOURCE=0
AUR_ONLY=0

usage() {
  cat <<'EOF'
Usage: scripts/build-dist.sh [options]

  --skip-deb             do not build Debian packages
  --skip-rpm             do not build RPM packages
  --skip-source          do not build the source archive
  --aur-only             render dist/aur from an existing source archive
  --deb-archs "..."      Debian architectures (default: amd64 arm64 loong64)
  --rpm-targets "..."    RPM targets (default: x86_64 aarch64 loongarch64)
  --out DIR              output directory (default: dist)
  --version VERSION      override the VERSION file
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --skip-deb) SKIP_DEB=1 ;;
    --skip-rpm) SKIP_RPM=1 ;;
    --skip-source) SKIP_SOURCE=1 ;;
    --aur-only) AUR_ONLY=1 ;;
    --deb-archs) DEB_ARCHS="$2"; shift ;;
    --rpm-targets) RPM_TARGETS="$2"; shift ;;
    --out) OUT="$2"; shift ;;
    --version) VERSION="$2"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

PKGNAME=spark-store-tui
PKGROOT=packaging/debian/package-root
SOURCE_DIR="$PKGNAME-source-$VERSION"
SOURCE_TAR="$SOURCE_DIR-r$SOURCE_REVISION.tar.gz"
LDFLAGS="-s -w \
  -X github.com/yxxbc/spark-store-tui/internal/version.Version=$VERSION \
  -X github.com/yxxbc/spark-store-tui/internal/version.Commit=$COMMIT \
  -X github.com/yxxbc/spark-store-tui/internal/version.Date=$BUILD_DATE"

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

goarch_for_deb() {
  case "$1" in
    amd64) echo amd64 ;;
    arm64) echo arm64 ;;
    loong64) echo loong64 ;;
    riscv64) echo riscv64 ;;
    *) echo "$1" ;;
  esac
}

if [ "$AUR_ONLY" -eq 1 ]; then
  SKIP_DEB=1
  SKIP_RPM=1
  SKIP_SOURCE=1
fi

mkdir -p "$OUT"
OUT_ABS=$(cd "$OUT" && pwd)
if [ "$AUR_ONLY" -eq 0 ]; then
  [ "$SKIP_DEB" -eq 1 ] || rm -f "$OUT"/spark-store-tui_*.deb
  [ "$SKIP_RPM" -eq 1 ] || rm -f "$OUT"/spark-store-tui-*.rpm
  [ "$SKIP_SOURCE" -eq 1 ] || rm -f "$OUT"/spark-store-tui-source-*.tar.gz
  rm -f "$OUT/SHA256SUMS"
  rm -rf "$OUT/aur"
fi

render_aur() {
  [ -f "$OUT/$SOURCE_TAR" ] || { echo "source archive missing: $OUT/$SOURCE_TAR" >&2; exit 1; }
  source_sha256=$(sha256_of "$OUT/$SOURCE_TAR")
  mkdir -p "$OUT/aur"
  sed -e "s|@VERSION@|$VERSION|g" \
    -e "s|@AUR_RELEASE@|$AUR_PKGREL|g" \
    -e "s|@SOURCE_REVISION@|$SOURCE_REVISION|g" \
    -e "s|@SOURCE_SHA256@|$source_sha256|g" \
    packaging/aur/PKGBUILD.in > "$OUT/aur/PKGBUILD"
  sed -e "s|@VERSION@|$VERSION|g" \
    -e "s|@AUR_RELEASE@|$AUR_PKGREL|g" \
    -e "s|@SOURCE_REVISION@|$SOURCE_REVISION|g" \
    -e "s|@SOURCE_SHA256@|$source_sha256|g" \
    packaging/aur/SRCINFO.in > "$OUT/aur/.SRCINFO"
}

if [ "$AUR_ONLY" -eq 1 ]; then
  render_aur
  exit 0
fi

# 1. Source archive. A clean tree uses git archive for reproducibility; a dirty
# tree falls back to tar so release candidates can be exercised locally.
if [ "$SKIP_SOURCE" -eq 0 ]; then
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1 \
    && [ -z "$(git status --porcelain 2>/dev/null)" ]; then
    git archive --format=tar.gz --prefix="$SOURCE_DIR/" -o "$OUT/$SOURCE_TAR" HEAD
  else
    echo "warning: worktree is dirty; building the source archive from the worktree" >&2
    stage=$(mktemp -d)
    mkdir -p "$stage/$SOURCE_DIR"
    tar --exclude='./.git' --exclude='./dist' --exclude='./build' \
      --exclude='./.github' --exclude='./.codex-tools' --exclude='./packaging/aur' \
      --exclude='*.deb' --exclude='*.rpm' --exclude='*.tar.gz' \
      -cf - . | tar -xf - -C "$stage/$SOURCE_DIR"
    (cd "$stage" && tar -czf "$OUT_ABS/$SOURCE_TAR" "$SOURCE_DIR")
    rm -rf "$stage"
  fi
fi

# 2. Debian packages. Each architecture gets its own staged package root so the
# control file and md5sums always describe the binary actually shipped.
if [ "$SKIP_DEB" -eq 0 ]; then
  command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required for Deb packages" >&2; exit 1; }
  for arch in $DEB_ARCHS; do
    stage=$(mktemp -d)
    cp -a "$PKGROOT" "$stage/package-root"
    find "$stage/package-root" -type d -exec chmod 0755 {} +
    find "$stage/package-root" -type f -exec chmod 0644 {} +
    mkdir -p "$stage/package-root/usr/lib/sparkstore"
    CGO_ENABLED=0 GOOS=linux GOARCH="$(goarch_for_deb "$arch")" \
      go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" \
      -o "$stage/package-root/usr/lib/sparkstore/sparkstore" ./cmd/sparkstore
    sed -e "s/^Version: .*/Version: $VERSION-$DEB_REVISION/" \
      -e "s/^Architecture: .*/Architecture: $arch/" \
      "$stage/package-root/DEBIAN/control" > "$stage/package-root/DEBIAN/control.new"
    mv "$stage/package-root/DEBIAN/control.new" "$stage/package-root/DEBIAN/control"
    chmod 0755 "$stage/package-root/usr/bin/sparkstore" \
      "$stage/package-root/usr/lib/sparkstore/sparkstore" \
      "$stage/package-root/DEBIAN/postinst" "$stage/package-root/DEBIAN/postrm"
    (cd "$stage/package-root" && find usr -type f -print0 | sort -z | xargs -0 md5sum > DEBIAN/md5sums)
    dpkg-deb --build --root-owner-group "$stage/package-root" \
      "$OUT/${PKGNAME}_${VERSION}-${DEB_REVISION}_${arch}.deb"
    rm -rf "$stage"
  done
fi

# 3. RPM packages. The spec cross-compiles with GOARCH, so one Linux runner can
# produce every target.
if [ "$SKIP_RPM" -eq 0 ]; then
  command -v rpmbuild >/dev/null 2>&1 || { echo "rpmbuild is required for RPM packages" >&2; exit 1; }
  [ -f "$OUT/$SOURCE_TAR" ] || { echo "source archive missing: $OUT/$SOURCE_TAR" >&2; exit 1; }
  for target in $RPM_TARGETS; do
    topdir=$(mktemp -d)
    mkdir -p "$topdir/SOURCES" "$topdir/SPECS" "$topdir/BUILD" "$topdir/BUILDROOT" "$topdir/RPMS" "$topdir/SRPMS"
    cp "$OUT/$SOURCE_TAR" "$topdir/SOURCES/"
    sed -e "s|^Version:.*|Version:        $VERSION|" \
      -e "s|^Release:.*|Release:        $RPM_RELEASE%{?dist}|" \
      -e "s|^Source0:.*|Source0:        https://github.com/yxxbc/spark-store-tui/releases/download/v$VERSION/$SOURCE_TAR|" \
      packaging/rpm/spark-store-tui.spec > "$topdir/SPECS/$PKGNAME.spec"
    rpmbuild -bb --nodeps --target "$target" --define "_topdir $topdir" "$topdir/SPECS/$PKGNAME.spec"
    find "$topdir/RPMS" -name '*.rpm' -exec cp {} "$OUT/" \;
    rm -rf "$topdir"
  done
fi

# 4. AUR files are rendered from the same source archive, so the checksum can
# never drift from the published tarball.
if [ -f "$OUT/$SOURCE_TAR" ]; then
  render_aur
fi

# 5. Release notes are the current CHANGELOG section.
bash scripts/release-notes.sh > "$OUT/RELEASE_NOTES.md"

# 6. Checksums over every binary artifact.
(
  cd "$OUT"
  : > SHA256SUMS
  for file in *.deb *.rpm *.tar.gz; do
    [ -f "$file" ] || continue
    printf '%s  %s\n' "$(sha256_of "$file")" "$file" >> SHA256SUMS
  done
)

echo "artifacts in $OUT:"
ls -1 "$OUT"
if [ -d "$OUT/aur" ]; then
  echo "AUR files in $OUT/aur:"
  ls -1 "$OUT/aur"
fi
